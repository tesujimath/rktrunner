package rktrunner

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"text/template"
)

type fragmentsT struct {
	Environment map[string]string
	Options     map[string][]string
	Volume      map[string]VolumeT
}

func parseAndExecute(desc, tstr string, vars map[string]string) (string, error) {
	var result string
	t, err := template.New(desc).Option("missingkey=zero").Parse(tstr)
	if err != nil {
		return result, err
	}
	var b bytes.Buffer
	err = t.Execute(&b, vars)
	if err != nil {
		return result, err
	}
	result = b.String()
	return result, nil
}

func GetFragments(c *configT, vars map[string]string, f *fragmentsT) error {
	var err error

	f.Environment = make(map[string]string)
	for envKey, envVal := range c.Environment {
		s, err := parseAndExecute(fmt.Sprintf("environment %v", envKey), envVal, vars)
		if err != nil {
			return err
		}
		f.Environment[envKey] = s
	}

	f.Options = make(map[string][]string)
	for optKey, optVals := range c.Options {
		for _, optVal := range optVals {
			s, err := parseAndExecute(fmt.Sprintf("option %v", optKey), optVal, vars)
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
			volFrag.Volume, err = parseAndExecute(fmt.Sprintf("volume %s volume", volKey), volVal.Volume, vars)
			if err != nil {
				return err
			}
		}
		if volVal.Mount != "" {
			volFrag.Mount, err = parseAndExecute(fmt.Sprintf("volume %s mount", volKey), volVal.Mount, vars)
			if err != nil {
				return err
			}
		}
		f.Volume[volKey] = volFrag
	}

	return nil
}

func (f *fragmentsT) printEnvironment(w io.Writer) {
	// get keys in order
	keys := make([]string, 0, len(f.Environment))
	for key := range f.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s=%s\n", key, f.Environment[key])
	}
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
