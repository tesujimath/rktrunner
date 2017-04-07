package rktrunner

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type ConfigT struct {
	Rkt                   string
	DefaultInteractiveCmd string `toml:"default-interactive-cmd"`
	Options               map[string][]string
	Volumes               map[string]map[string]string
}

// valid options
const GeneralOptions = "general"
const RunOptions = "run"
const ImageOptions = "image"

// volumes
const VolumeHost = "host"
const VolumeTarget = "target"

// The first config file found is the one used.
// There is a serious security hole if any of these files are writable
// by other than root.
var configFiles []string = []string{
	"/etc/rktrunner.toml",
	"/home/guestsi/go/src/github.com/tesujimath/rktrunner/examples/rktrunner.toml",
}

func GetConfig() (*ConfigT, error) {
	var c ConfigT
	var err error
	var path string
configFileAttempts:
	for _, path = range configFiles {
		_, err = toml.DecodeFile(path, &c)
		if err != nil && os.IsNotExist(err) {
			continue configFileAttempts
		}
		break configFileAttempts
	}
	if err != nil {
		if !os.IsNotExist(err) {
			// provide some context
			err = fmt.Errorf("%s %v", path, err)
		}
		return nil, err
	}

	// validate options
	if c.Rkt == "" {
		return nil, fmt.Errorf("missing rkt")
	}

	type validOptionsT map[string]bool
	validOptions := validOptionsT{
		GeneralOptions: true,
		RunOptions:     true,
		ImageOptions:   true,
	}

	for optKey := range c.Options {
		if !validOptions[optKey] {
			return nil, fmt.Errorf("unknown option: %s", optKey)
		}
	}

	// validate volumes
	type validVolAttrsT map[string]bool
	validVolAttrs := validVolAttrsT{
		VolumeHost:   true,
		VolumeTarget: true,
	}

	for volKey, volVal := range c.Volumes {
		for attrKey := range volVal {
			if !validVolAttrs[attrKey] {
				return nil, fmt.Errorf("unknown attr %s for volume %s", attrKey, volKey)
			}
		}
		for attrKey, _ := range validVolAttrs {
			if volVal[attrKey] == "" {
				return nil, fmt.Errorf("missing attr %s for volume %s", attrKey, volKey)
			}
		}
	}

	return &c, nil
}
