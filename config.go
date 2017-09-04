// Copyright 2017 The rktrunner Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rktrunner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type configT struct {
	Rkt                   string
	PreserveCwd           bool              `toml:"preserve-cwd"`
	UsePath               bool              `toml:"use-path"`
	WorkerPods            bool              `toml:"worker-pods"`
	RestrictImages        bool              `toml:"restrict-images"`
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
	Image                string
	Exec                 []string
	Environment          map[string]string
	Passwd               []string
	Group                []string
	HostTimezone         bool     `toml:"host-timezone"`
	EnvironmentUpdate    []string `toml:"environment-update"`
	EnvironmentBlacklist []string `toml:"environment-blacklist"`
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

	for _, aliasVal := range c.Alias {
		if (aliasVal.Passwd != nil || aliasVal.Group != nil) && !c.WorkerPods {
			return fmt.Errorf("passwd/group requires worker-pods")
		}
		if aliasVal.HostTimezone && !c.WorkerPods {
			return fmt.Errorf("host-timezone requires worker-pods")
		}
		if aliasVal.EnvironmentUpdate != nil && c.ExecSlaveDir == "" {
			return fmt.Errorf("environment-update requires exec-slave-dir")
		}
	}

	return validateOptionsForModes(c.Options)
}
