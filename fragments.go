package rktrunner

import (
	"bytes"
	"fmt"
	"os/user"
	"text/template"
)

type FragmentsT struct {
	Options map[string][]string
	Volumes map[string]map[string]string
}

func parseAndExecute(desc, tstr string, u *user.User) (string, error) {
	var result string
	t, err := template.New(desc).Parse(tstr)
	if err != nil {
		return result, fmt.Errorf("failed to parse template for %s: %v", desc, err)
	}
	var b bytes.Buffer
	err = t.Execute(&b, u)
	if err != nil {
		return result, fmt.Errorf("failed to execute template for %s: %v", desc, err)
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

	f.Volumes = make(map[string]map[string]string)
	for volKey, volVal := range c.Volumes {
		f.Volumes[volKey] = make(map[string]string)
		for attrKey, attrVal := range volVal {
			s, err := parseAndExecute(fmt.Sprintf("volume %s %s", volKey, attrKey), attrVal, u)
			if err != nil {
				return nil, err
			}
			f.Volumes[volKey][attrKey] = s
		}
	}

	return &f, nil
}
