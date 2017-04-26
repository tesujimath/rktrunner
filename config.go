package rktrunner

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

type configT struct {
	Rkt                   string
	AttachStdio           bool              `toml:"attach-stdio"`
	PreserveCwd           bool              `toml:"preserve-cwd"`
	ExecSlaveDir          string            `toml:"exec-slave-dir"`
	AutoImagePrefix       map[string]string `toml:"auto-image-prefix"`
	DefaultInteractiveCmd string            `toml:"default-interactive-cmd"`
	StripLogPrefix        bool              `toml:"strip-log-prefix"`
	Options               map[string][]string
	Volume                map[string]VolumeT
	Alias                 map[string]ImageAliasT
}

type VolumeT struct {
	Volume string
	Mount  string
}

type ImageAliasT struct {
	Image string
	Exec  []string
}

// valid options
const GeneralOptions = "general"
const RunOptions = "run"
const ImageOptions = "image"

func GetConfig(path string, c *configT) error {
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		if !os.IsNotExist(err) {
			// provide some context
			err = fmt.Errorf("%s %v", path, err)
		}
		return err
	}

	// validate options
	if c.Rkt == "" {
		return fmt.Errorf("missing rkt")
	}

	if c.AttachStdio && c.ExecSlaveDir == "" {
		return fmt.Errorf("attach-stdio requires exec-slave-dir")
	}
	if c.PreserveCwd && c.ExecSlaveDir == "" {
		return fmt.Errorf("preserve-stdio requires exec-slave-dir")
	}
	if c.ExecSlaveDir != "" {
		p := filepath.Join(c.ExecSlaveDir, slaveRunner)
		_, err := os.Stat(p)
		if err != nil {
			return err
		}
	}

	type validOptionsT map[string]bool
	validOptions := validOptionsT{
		GeneralOptions: true,
		RunOptions:     true,
		ImageOptions:   true,
	}

	for optKey := range c.Options {
		if !validOptions[optKey] {
			return fmt.Errorf("unknown option: %s", optKey)
		}
	}

	return nil
}
