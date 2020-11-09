package conf

import (
	"time"

	"github.com/spf13/pflag"

	"ext.arhat.dev/runtime-podman/pkg/constant"
)

type RuntimeConfig struct {
	DataDir string `json:"dataDir" yaml:"dataDir"`

	ManagementNamespace string `json:"managementNamespace" yaml:"managementNamespace"`

	PauseImage   string   `json:"pauseImage" yaml:"pauseImage"`
	PauseCommand []string `json:"pauseCommand" yaml:"pauseCommand"`

	ImageActionTimeout time.Duration `json:"imageActionTimeout" yaml:"imageActionTimeout"`
	PodActionTimeout   time.Duration `json:"podActionTimeout" yaml:"podActionTimeout"`

	AbbotRequestSubCmd string `json:"abbotRequestSubCmd" yaml:"abbotRequestSubCmd"`
}

func FlagsForRuntime(prefix string, config *RuntimeConfig) *pflag.FlagSet {
	fs := pflag.NewFlagSet("runtime", pflag.ExitOnError)

	fs.StringVar(&config.DataDir, prefix+"dataDir",
		constant.DefaultPodDataDir, "set pod data root dir")

	fs.StringVar(&config.PauseImage, prefix+"pauseImage",
		constant.DefaultPauseImage, "set pause image to use")

	fs.StringSliceVar(&config.PauseCommand, prefix+"pauseCommand",
		[]string{constant.DefaultPauseCommand}, "set pause command to pause image")

	fs.DurationVar(&config.ImageActionTimeout, prefix+"imageActionTimeout",
		constant.DefaultImageActionTimeout, "set image operation timeout")

	fs.DurationVar(&config.PodActionTimeout, prefix+"podActionTimeout",
		constant.DefaultPodActionTimeout, "set image operation timeout")

	fs.StringVar(&config.AbbotRequestSubCmd, prefix+"abbotRequestSubCmd",
		"process", "set abbot sub command to process requests")

	return fs
}
