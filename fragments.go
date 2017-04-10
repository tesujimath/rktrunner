package rktrunner

import (
	"bytes"
	"fmt"
	"os/user"
	"text/template"
)

type FragmentsT struct {
	Options map[string][]string
	Volume  map[string]VolumeT
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

func GetFragments(c *ConfigT, u *user.User) (*FragmentsT, error) {
	var f FragmentsT

	f.Options = make(map[string][]string)
	for optKey, optVals := range c.Options {
		for _, optVal := range optVals {
			s, err := parseAndExecute(fmt.Sprintf("option %v", optKey), optVal, u)
			if err != nil {
				return nil, err
			}
			f.Options[optKey] = append(f.Options[optKey], s)
		}
	}

	f.Volume = make(map[string]VolumeT)
	for volKey, volVal := range c.Volume {
		var err error
		var volFrag VolumeT
		if volVal.Volume != "" {
			volFrag.Volume, err = parseAndExecute(fmt.Sprintf("volume %s volume", volKey), volVal.Volume, u)
			if err != nil {
				return nil, err
			}
		}
		if volVal.Mount != "" {
			volFrag.Mount, err = parseAndExecute(fmt.Sprintf("volume %s mount", volKey), volVal.Mount, u)
			if err != nil {
				return nil, err
			}
		}
		f.Volume[volKey] = volFrag
	}

	return &f, nil
}
