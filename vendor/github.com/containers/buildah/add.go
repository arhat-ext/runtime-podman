package buildah

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/containers/buildah/copier"
	"github.com/containers/buildah/pkg/chrootuser"
	"github.com/containers/storage/pkg/fileutils"
	"github.com/containers/storage/pkg/idtools"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// AddAndCopyOptions holds options for add and copy commands.
type AddAndCopyOptions struct {
	// Chown is a spec for the user who should be given ownership over the
	// newly-added content, potentially overriding permissions which would
	// otherwise be set to 0:0.
	Chown string
	// PreserveOwnership, if Chown is not set, tells us to avoid setting
	// ownership of copied items to 0:0, instead using whatever ownership
	// information is already set.  Not meaningful for remote sources.
	PreserveOwnership bool
	// All of the data being copied will pass through Hasher, if set.
	// If the sources are URLs or files, their contents will be passed to
	// Hasher.
	// If the sources include directory trees, Hasher will be passed
	// tar-format archives of the directory trees.
	Hasher io.Writer
	// Excludes is the contents of the .dockerignore file.
	Excludes []string
	// ContextDir is the base directory for content being copied and
	// Excludes patterns.
	ContextDir string
	// ID mapping options to use when contents to be copied are part of
	// another container, and need ownerships to be mapped from the host to
	// that container's values before copying them into the container.
	IDMappingOptions *IDMappingOptions
	// DryRun indicates that the content should be digested, but not actually
	// copied into the container.
	DryRun bool
	// Clear the setuid bit on items being copied.  Has no effect on
	// archives being extracted, where the bit is always preserved.
	StripSetuidBit bool
	// Clear the setgid bit on items being copied.  Has no effect on
	// archives being extracted, where the bit is always preserved.
	StripSetgidBit bool
	// Clear the sticky bit on items being copied.  Has no effect on
	// archives being extracted, where the bit is always preserved.
	StripStickyBit bool
}

// sourceIsRemote returns true if "source" is a remote location.
func sourceIsRemote(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

// getURL writes a tar archive containing the named content
func getURL(src, mountpoint, renameTarget string, writer io.Writer) error {
	url, err := url.Parse(src)
	if err != nil {
		return errors.Wrapf(err, "error parsing URL %q", url)
	}
	response, err := http.Get(src)
	if err != nil {
		return errors.Wrapf(err, "error parsing URL %q", url)
	}
	defer response.Body.Close()
	// Figure out what to name the new content.
	name := renameTarget
	if name == "" {
		name = path.Base(url.Path)
	}
	// If there's a date on the content, use it.  If not, use the Unix epoch
	// for compatibility.
	date := time.Unix(0, 0).UTC()
	lastModified := response.Header.Get("Last-Modified")
	if lastModified != "" {
		d, err := time.Parse(time.RFC1123, lastModified)
		if err != nil {
			return errors.Wrapf(err, "error parsing last-modified time %q", lastModified)
		}
		date = d
	}
	// Figure out the size of the content.
	size := response.ContentLength
	responseBody := response.Body
	if size < 0 {
		// Create a temporary file and copy the content to it, so that
		// we can figure out how much content there is.
		f, err := ioutil.TempFile(mountpoint, "download")
		if err != nil {
			return errors.Wrapf(err, "error creating temporary file to hold %q", src)
		}
		defer os.Remove(f.Name())
		defer f.Close()
		size, err = io.Copy(f, response.Body)
		if err != nil {
			return errors.Wrapf(err, "error writing %q to temporary file %q", src, f.Name())
		}
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return errors.Wrapf(err, "error setting up to read %q from temporary file %q", src, f.Name())
		}
		responseBody = f
	}
	// Write the output archive.  Set permissions for compatibility.
	tw := tar.NewWriter(writer)
	defer tw.Close()
	hdr := tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     size,
		Mode:     0600,
		ModTime:  date,
	}
	err = tw.WriteHeader(&hdr)
	if err != nil {
		return errors.Wrapf(err, "error writing header")
	}
	_, err = io.Copy(tw, responseBody)
	return errors.Wrapf(err, "error writing content from %q to tar stream", src)
}

// Add copies the contents of the specified sources into the container's root
// filesystem, optionally extracting contents of local files that look like
// non-empty archives.
func (b *Builder) Add(destination string, extract bool, options AddAndCopyOptions, sources ...string) error {
	mountPoint, err := b.Mount(b.MountLabel)
	if err != nil {
		return err
	}
	defer func() {
		if err2 := b.Unmount(); err2 != nil {
			logrus.Errorf("error unmounting container: %v", err2)
		}
	}()

	contextDir := options.ContextDir
	if contextDir == "" {
		contextDir = string(os.PathSeparator)
	}

	// Figure out what sorts of sources we have.
	var localSources, remoteSources []string
	for _, src := range sources {
		if sourceIsRemote(src) {
			remoteSources = append(remoteSources, src)
			continue
		}
		localSources = append(localSources, src)
	}

	// Check how many items our local source specs matched.  Each spec
	// should have matched at least one item, otherwise we consider it an
	// error.
	var localSourceStats []*copier.StatsForGlob
	if len(localSources) > 0 {
		statOptions := copier.StatOptions{
			CheckForArchives: extract,
		}
		localSourceStats, err = copier.Stat(contextDir, contextDir, statOptions, localSources)
		if err != nil {
			return errors.Wrapf(err, "error checking on sources %v under %q", localSources, contextDir)
		}
	}
	numLocalSourceItems := 0
	for _, localSourceStat := range localSourceStats {
		if localSourceStat.Error != "" {
			errorText := localSourceStat.Error
			rel, err := filepath.Rel(contextDir, localSourceStat.Glob)
			if err != nil {
				errorText = fmt.Sprintf("%v; %s", err, errorText)
			}
			if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				errorText = fmt.Sprintf("possible escaping context directory error: %s", errorText)
			}
			return errors.Errorf("error checking on source %v under %q: %v", localSourceStat.Glob, contextDir, errorText)
		}
		if len(localSourceStat.Globbed) == 0 {
			return errors.Wrapf(syscall.ENOENT, "error checking on source %v under %q: no glob matches", localSourceStat.Glob, contextDir)
		}
		numLocalSourceItems += len(localSourceStat.Globbed)
	}
	if numLocalSourceItems+len(remoteSources) == 0 {
		return errors.Wrapf(syscall.ENOENT, "no sources %v found", sources)
	}

	// Find out which user (and group) the destination should belong to.
	var chownDirs, chownFiles *idtools.IDPair
	var chmodDirs, chmodFiles *os.FileMode
	var user specs.User
	if options.Chown != "" {
		user, _, err = b.user(mountPoint, options.Chown)
		if err != nil {
			return errors.Wrapf(err, "error looking up UID/GID for %q", options.Chown)
		}
	}
	chownDirs = &idtools.IDPair{UID: int(user.UID), GID: int(user.GID)}
	chownFiles = &idtools.IDPair{UID: int(user.UID), GID: int(user.GID)}
	if options.Chown == "" && options.PreserveOwnership {
		chownDirs = nil
		chownFiles = nil
	}

	// If we have a single source archive to extract, or more than one
	// source item, or the destination has a path separator at the end of
	// it, and it's not a remote URL, the destination needs to be a
	// directory.
	if destination == "" || !filepath.IsAbs(destination) {
		tmpDestination := filepath.Join(string(os.PathSeparator)+b.WorkDir(), destination)
		if destination == "" || strings.HasSuffix(destination, string(os.PathSeparator)) {
			destination = tmpDestination + string(os.PathSeparator)
		} else {
			destination = tmpDestination
		}
	}
	destMustBeDirectory := (len(sources) > 1) || strings.HasSuffix(destination, string(os.PathSeparator))
	destCanBeFile := false
	if len(sources) == 1 {
		if len(remoteSources) == 1 {
			destCanBeFile = sourceIsRemote(sources[0])
		}
		if len(localSources) == 1 {
			item := localSourceStats[0].Results[localSourceStats[0].Globbed[0]]
			if item.IsDir || (item.IsArchive && extract) {
				destMustBeDirectory = true
			}
			if item.IsRegular {
				destCanBeFile = true
			}
		}
	}

	// We care if the destination either doesn't exist, or exists and is a
	// file.  If the source can be a single file, for those cases we treat
	// the destination as a file rather than as a directory tree.
	renameTarget := ""
	extractDirectory := filepath.Join(mountPoint, destination)
	statOptions := copier.StatOptions{
		CheckForArchives: extract,
	}
	destStats, err := copier.Stat(mountPoint, filepath.Join(mountPoint, b.WorkDir()), statOptions, []string{extractDirectory})
	if err != nil {
		return errors.Wrapf(err, "error checking on destination %v", extractDirectory)
	}
	if (len(destStats) == 0 || len(destStats[0].Globbed) == 0) && !destMustBeDirectory && destCanBeFile {
		// destination doesn't exist - extract to parent and rename the incoming file to the destination's name
		renameTarget = filepath.Base(extractDirectory)
		extractDirectory = filepath.Dir(extractDirectory)
	}
	if len(destStats) == 1 && len(destStats[0].Globbed) == 1 && destStats[0].Results[destStats[0].Globbed[0]].IsRegular {
		if destMustBeDirectory {
			return errors.Errorf("destination %v already exists but is not a directory", destination)
		}
		// destination exists - it's a file, we need to extract to parent and rename the incoming file to the destination's name
		renameTarget = filepath.Base(extractDirectory)
		extractDirectory = filepath.Dir(extractDirectory)
	}

	pm, err := fileutils.NewPatternMatcher(options.Excludes)
	if err != nil {
		return errors.Wrapf(err, "error processing excludes list %v", options.Excludes)
	}

	// Copy each source in turn.
	var srcUIDMap, srcGIDMap []idtools.IDMap
	if options.IDMappingOptions != nil {
		srcUIDMap, srcGIDMap = convertRuntimeIDMaps(options.IDMappingOptions.UIDMap, options.IDMappingOptions.GIDMap)
	}
	destUIDMap, destGIDMap := convertRuntimeIDMaps(b.IDMappingOptions.UIDMap, b.IDMappingOptions.GIDMap)

	for _, src := range sources {
		var multiErr *multierror.Error
		var getErr, closeErr, renameErr, putErr error
		var wg sync.WaitGroup
		if sourceIsRemote(src) {
			pipeReader, pipeWriter := io.Pipe()
			wg.Add(1)
			go func() {
				getErr = getURL(src, mountPoint, renameTarget, pipeWriter)
				pipeWriter.Close()
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				b.ContentDigester.Start("")
				hashCloser := b.ContentDigester.Hash()
				hasher := io.Writer(hashCloser)
				if options.Hasher != nil {
					hasher = io.MultiWriter(hasher, options.Hasher)
				}
				if options.DryRun {
					_, putErr = io.Copy(hasher, pipeReader)
				} else {
					putOptions := copier.PutOptions{
						UIDMap:     destUIDMap,
						GIDMap:     destGIDMap,
						ChownDirs:  chownDirs,
						ChmodDirs:  chmodDirs,
						ChownFiles: chownFiles,
						ChmodFiles: chmodFiles,
					}
					putErr = copier.Put(mountPoint, extractDirectory, putOptions, io.TeeReader(pipeReader, hasher))
				}
				hashCloser.Close()
				pipeReader.Close()
				wg.Done()
			}()
			wg.Wait()
			if getErr != nil {
				getErr = errors.Wrapf(getErr, "error reading %q", src)
			}
			if putErr != nil {
				putErr = errors.Wrapf(putErr, "error storing %q", src)
			}
			multiErr = multierror.Append(getErr, putErr)
			if multiErr != nil && multiErr.ErrorOrNil() != nil {
				if len(multiErr.Errors) > 1 {
					return multiErr.ErrorOrNil()
				}
				return multiErr.Errors[0]
			}
			continue
		}

		// Dig out the result of running glob+stat on this source spec.
		var localSourceStat *copier.StatsForGlob
		for _, st := range localSourceStats {
			if st.Glob == src {
				localSourceStat = st
				break
			}
		}
		if localSourceStat == nil {
			return errors.Errorf("internal error: should have statted %s, but we didn't?", src)
		}

		// Iterate through every item that matched the glob.
		itemsCopied := 0
		for _, glob := range localSourceStat.Globbed {
			rel, err := filepath.Rel(contextDir, glob)
			if err != nil {
				return errors.Wrapf(err, "error computing path of %q", glob)
			}
			if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
				return errors.Errorf("possible escaping context directory error: %q is outside of %q", glob, contextDir)
			}
			// Check for dockerignore-style exclusion of this item.
			if rel != "." {
				matches, err := pm.Matches(filepath.ToSlash(rel)) // nolint:staticcheck
				if err != nil {
					return errors.Wrapf(err, "error checking if %q(%q) is excluded", glob, rel)
				}
				if matches {
					continue
				}
			}
			st := localSourceStat.Results[glob]
			pipeReader, pipeWriter := io.Pipe()
			wg.Add(1)
			go func() {
				renamedItems := 0
				writer := io.WriteCloser(pipeWriter)
				if renameTarget != "" {
					writer = newTarFilterer(writer, func(hdr *tar.Header) (bool, bool, io.Reader) {
						hdr.Name = renameTarget
						renamedItems++
						return false, false, nil
					})
				}
				getOptions := copier.GetOptions{
					UIDMap:         srcUIDMap,
					GIDMap:         srcGIDMap,
					Excludes:       options.Excludes,
					ExpandArchives: extract,
					StripSetuidBit: options.StripSetuidBit,
					StripSetgidBit: options.StripSetgidBit,
					StripStickyBit: options.StripStickyBit,
				}
				getErr = copier.Get(contextDir, contextDir, getOptions, []string{glob}, writer)
				closeErr = writer.Close()
				if renameTarget != "" && renamedItems > 1 {
					renameErr = errors.Errorf("internal error: renamed %d items when we expected to only rename 1", renamedItems)
				}
				wg.Done()
			}()
			wg.Add(1)
			go func() {
				if st.IsDir {
					b.ContentDigester.Start("dir")
				} else {
					b.ContentDigester.Start("file")
				}
				hashCloser := b.ContentDigester.Hash()
				hasher := io.Writer(hashCloser)
				if options.Hasher != nil {
					hasher = io.MultiWriter(hasher, options.Hasher)
				}
				if options.DryRun {
					_, putErr = io.Copy(hasher, pipeReader)
				} else {
					putOptions := copier.PutOptions{
						UIDMap:     destUIDMap,
						GIDMap:     destGIDMap,
						ChownDirs:  chownDirs,
						ChmodDirs:  chmodDirs,
						ChownFiles: chownFiles,
						ChmodFiles: chmodFiles,
					}
					putErr = copier.Put(mountPoint, extractDirectory, putOptions, io.TeeReader(pipeReader, hasher))
				}
				hashCloser.Close()
				pipeReader.Close()
				wg.Done()
			}()
			wg.Wait()
			if getErr != nil {
				getErr = errors.Wrapf(getErr, "error reading %q", src)
			}
			if closeErr != nil {
				closeErr = errors.Wrapf(closeErr, "error closing %q", src)
			}
			if renameErr != nil {
				renameErr = errors.Wrapf(renameErr, "error renaming %q", src)
			}
			if putErr != nil {
				putErr = errors.Wrapf(putErr, "error storing %q", src)
			}
			multiErr = multierror.Append(getErr, closeErr, renameErr, putErr)
			if multiErr != nil && multiErr.ErrorOrNil() != nil {
				if len(multiErr.Errors) > 1 {
					return multiErr.ErrorOrNil()
				}
				return multiErr.Errors[0]
			}
			itemsCopied++
		}
		if itemsCopied == 0 {
			return errors.Wrapf(syscall.ENOENT, "no items matching glob %q copied (%d filtered)", localSourceStat.Glob, len(localSourceStat.Globbed))
		}
	}
	return nil
}

// user returns the user (and group) information which the destination should belong to.
func (b *Builder) user(mountPoint string, userspec string) (specs.User, string, error) {
	if userspec == "" {
		userspec = b.User()
	}

	uid, gid, homeDir, err := chrootuser.GetUser(mountPoint, userspec)
	u := specs.User{
		UID:      uid,
		GID:      gid,
		Username: userspec,
	}
	if !strings.Contains(userspec, ":") {
		groups, err2 := chrootuser.GetAdditionalGroupsForUser(mountPoint, uint64(u.UID))
		if err2 != nil {
			if errors.Cause(err2) != chrootuser.ErrNoSuchUser && err == nil {
				err = err2
			}
		} else {
			u.AdditionalGids = groups
		}

	}
	return u, homeDir, err
}