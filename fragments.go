package rktrunner

import (
	"bytes"
	"fmt"
	"os/user"
	"text/template"
)

type fragmentsT struct {
	Environment map[string]string
	Options     map[string][]string
	Volume      map[string]VolumeT
}

func parseAndExecute(desc, tstr string, u *user.User) (string, error) {
	var result string
	t, err := template.New(desc).Parse(tstr)
	if err != nil {
		return result, err
	}
	var b bytes.Buffer
	err = t.Execute(&b, u)
	if err != nil {
		return result, err
	}
	result = b.String()
	return result, nil
}

func GetFragments(c *configT, u *user.User, f *fragmentsT) error {
	var err error

	f.Environment = make(map[string]string)
	for envKey, envVal := range c.Environment {
		f.Environment[envKey], err = parseAndExecute(fmt.Sprintf("environment %v", envKey), envVal, u)
		if err != nil {
			return err
		}
	}

	f.Options = make(map[string][]string)
	for optKey, optVals := range c.Options {
		for _, optVal := range optVals {
			s, err := parseAndExecute(fmt.Sprintf("option %v", optKey), optVal, u)
			if err != nil {
				return err
			}
			f.Options[optKey] = append(f.Options[optKey], s)
		}
	}

	f.Volume = make(map[string]VolumeT)
	for volKey, volVal := range c.Volume {
		var volFrag VolumeT
		if volVal.Volume != "" {
			volFrag.Volume, err = parseAndExecute(fmt.Sprintf("volume %s volume", volKey), volVal.Volume, u)
			if err != nil {
				return err
			}
		}
		if volVal.Mount != "" {
			volFrag.Mount, err = parseAndExecute(fmt.Sprintf("volume %s mount", volKey), volVal.Mount, u)
			if err != nil {
				return err
			}
		}
		f.Volume[volKey] = volFrag
	}

	return nil
}

func (f *fragmentsT) formatVolumes() []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Volume != "" {
			s = append(s, "--volume", fmt.Sprintf("%s,%s", key, vol.Volume))
		}
	}
	return s
}

func (f *fragmentsT) formatMounts() []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Mount != "" {
			s = append(s, "--mount", fmt.Sprintf("volume=%s,%s", key, vol.Mount))
		}
	}
	return s
}
