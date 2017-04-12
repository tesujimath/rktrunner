package rktrunner

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type configT struct {
	Rkt                   string
	AutoImagePrefix       map[string]string `toml:"auto-image-prefix"`
	DefaultInteractiveCmd string            `toml:"default-interactive-cmd"`
	Environment           map[string]string
	Options               map[string][]string
	Volume                map[string]VolumeT
}

type VolumeT struct {
	Volume string
	Mount  string
}

// valid options
const GeneralOptions = "general"
const RunOptions = "run"
const ImageOptions = "image"

// The first config file found is the one used.
// There is a serious security hole if any of these files are writable
// by other than root.
var configFiles []string = []string{
	// during development:
	//"/home/guestsi/go/src/github.com/tesujimath/rktrunner/examples/rktrunner.toml",
	"/etc/rktrunner.toml",
}

func GetConfig(c *configT) error {
	var err error
	var path string
configFileAttempts:
	for _, path = range configFiles {
		_, err = toml.DecodeFile(path, c)
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
		return err
	}

	// validate options
	if c.Rkt == "" {
		return fmt.Errorf("missing rkt")
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
