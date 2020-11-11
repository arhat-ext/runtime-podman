/*
Copyright 2020 The arhat.dev Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"

	"arhat.dev/aranya-proto/aranyagopb/runtimepb"
	"arhat.dev/pkg/log"
	"arhat.dev/pkg/wellknownerrors"
	imagetypes "github.com/containers/image/v5/types"
	libpodimage "github.com/containers/podman/v2/libpod/image"
	libpodutil "github.com/containers/podman/v2/pkg/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

// EnsureImages ensure container images.
func (r *libpodRuntime) EnsureImages(ctx context.Context, options *runtimepb.ImageEnsureCmd) (*runtimepb.ImageStatusListMsg, error) {
	logger := r.logger.WithFields(log.String("action", "ensureImages"), log.Any("options", options))
	logger.D("ensuring pod container image(s)")

	allImages := map[string]*runtimepb.ImagePullSpec{
		r.pauseImage: {PullPolicy: runtimepb.IMAGE_PULL_IF_NOT_PRESENT},
	}

	for imageName, opt := range options.Images {
		allImages[imageName] = opt
	}
	pulledImages, err := r.ensureImages(ctx, allImages)
	if err != nil {
		logger.I("failed to ensure container images", log.Error(err))
		return nil, err
	}

	images := make([]*runtimepb.ImageStatusMsg, len(pulledImages))
	for _, img := range pulledImages {
		var sha256Hash string
		digests, err := img.RepoDigests()
		if err != nil {
			return nil, err
		}
		for _, digest := range digests {
			idx := strings.LastIndex(digest, "sha256:")
			if idx > -1 {
				sha256Hash = digest[idx+7:]
			}
		}

		if sha256Hash == "" {
			continue
		}

		images = append(images, &runtimepb.ImageStatusMsg{
			Sha256: sha256Hash,
			Refs:   []string{img.Tag},
		})
	}

	return &runtimepb.ImageStatusListMsg{Images: images}, nil
}

func (r *libpodRuntime) DeleteImages(
	ctx context.Context, options *runtimepb.ImageDeleteCmd,
) (*runtimepb.ImageStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *libpodRuntime) ListImages(
	ctx context.Context, options *runtimepb.ImageListCmd,
) (*runtimepb.ImageStatusListMsg, error) {
	return nil, wellknownerrors.ErrNotSupported
}

func (r *libpodRuntime) getImageConfig(ctx context.Context, image *libpodimage.Image) (*imagespec.ImageConfig, error) {
	imageData, err := image.Inspect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	if imageData == nil {
		imageData = &image.ImageData
	}

	if imageData.Config == nil {
		imageData.Config = &imagespec.ImageConfig{}
	}

	return imageData.Config, nil
}

var pullTypeMapping = map[runtimepb.ImagePullPolicy]libpodutil.PullType{
	runtimepb.IMAGE_PULL_ALWAYS:         libpodutil.PullImageAlways,
	runtimepb.IMAGE_PULL_IF_NOT_PRESENT: libpodutil.PullImageMissing,
	runtimepb.IMAGE_PULL_NEVER:          libpodutil.PullImageNever,
}

func (r *libpodRuntime) ensureImages(
	ctx context.Context,
	images map[string]*runtimepb.ImagePullSpec,
) (map[string]*libpodimage.Image, error) {
	pulledImages := make(map[string]*libpodimage.Image)

	ctx, cancel := r.ImageActionContext(ctx)
	defer cancel()

	for imageName, spec := range images {
		var dockerRegistryOptions *libpodimage.DockerRegistryOptions
		if spec.AuthConfig != nil {
			dockerRegistryOptions = &libpodimage.DockerRegistryOptions{
				DockerRegistryCreds: &imagetypes.DockerAuthConfig{
					Username: spec.AuthConfig.Username,
					Password: spec.AuthConfig.Password,
				},
				DockerCertPath:              "",
				DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(false),
			}
		}

		image, err := r.imageClient.New(
			ctx, imageName, "", "",
			os.Stderr, dockerRegistryOptions,
			libpodimage.SigningOptions{},
			nil,
			pullTypeMapping[spec.PullPolicy],
		)
		if err != nil {
			return nil, err
		}
		pulledImages[imageName] = image
	}
	return pulledImages, nil
}
