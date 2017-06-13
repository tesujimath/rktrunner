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
	RecursiveMounts       bool              `toml:"recursive-mounts"`
	PreserveCwd           bool              `toml:"preserve-cwd"`
	UsePath               bool              `toml:"use-path"`
	ExecSlaveDir          string            `toml:"exec-slave-dir"`
	AutoImagePrefix       map[string]string `toml:"auto-image-prefix"`
	DefaultInteractiveCmd string            `toml:"default-interactive-cmd"`
	Environment           map[string]string `toml:"environment"`
	Options               ModeOptionsT
	Volume                map[string]VolumeT
	Alias                 map[string]ImageAliasT
}

type ModeOptionsT map[string]ClassOptionsT
type ClassOptionsT map[string][]string

type VolumeT struct {
	Volume    string
	Mount     string
	OnRequest bool `toml:"on-request"`
}

type ImageAliasT struct {
	Image string
	Exec  []string
}

const OptionsTable = "options"

// valid modes
const BatchMode = "batch"
const InteractiveMode = "interactive"
const CommonMode = "common"

// valid option classes
const GeneralClass = "general"
const FetchClass = "fetch"
const RunClass = "run"
const ImageClass = "image"

func validateOptionsForModes(modeOptions ModeOptionsT) error {
	type validModeT map[string]bool
	validMode := validModeT{
		BatchMode:       true,
		InteractiveMode: true,
		CommonMode:      true,
	}
	for mode, classOptions := range modeOptions {
		if !validMode[mode] {
			return fmt.Errorf("invalid %s.%s", OptionsTable, mode)
		}
		err := validateClassOptions(mode, classOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateClassOptions(mode string, classOptions ClassOptionsT) error {
	type validClassT map[string]bool
	validClass := validClassT{
		GeneralClass: true,
		FetchClass:   true,
		RunClass:     true,
		ImageClass:   true,
	}

	for class := range classOptions {
		if !validClass[class] {
			return fmt.Errorf("invalid %s.%s.%s", OptionsTable, mode, class)
		}
	}
	return nil
}

func GetConfig(path string, c *configT) error {
	_, err := toml.DecodeFile(path, c)
	if err != nil {
		if !os.IsNotExist(err) {
			// provide some context
			err = fmt.Errorf("%s %v", path, err)
		}
		return err
	}

	// validate
	if c.Rkt == "" {
		return fmt.Errorf("missing rkt")
	}

	if c.AttachStdio && c.ExecSlaveDir == "" {
		return fmt.Errorf("attach-stdio requires exec-slave-dir")
	}
	if c.PreserveCwd && c.ExecSlaveDir == "" {
		return fmt.Errorf("preserve-stdio requires exec-slave-dir")
	}
	if c.UsePath && c.ExecSlaveDir == "" {
		return fmt.Errorf("use-path requires exec-slave-dir")
	}
	if c.ExecSlaveDir != "" {
		p := filepath.Join(c.ExecSlaveDir, slaveRunner)
		_, err := os.Stat(p)
		if err != nil {
			return err
		}
	}

	return validateOptionsForModes(c.Options)
}
