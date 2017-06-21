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
	"bytes"
	"fmt"
	"io"
	"sort"
	"text/template"
)

type fragmentsT struct {
	Environment map[string]string
	Options     ModeOptionsT
	Volume      map[string]VolumeT
}

func expandFragments(desc, tstr string, vars map[string]string) (string, error) {
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
		s, err := expandFragments(fmt.Sprintf("environment %v", envKey), envVal, vars)
		if err != nil {
			return err
		}
		if s != "" {
			f.Environment[envKey] = s
		}
	}

	f.Options = make(ModeOptionsT)
	for mode, options := range c.Options {
		f.Options[mode] = make(ClassOptionsT)

		for class, classOptions := range options {
			for _, option := range classOptions {
				s, err := expandFragments(fmt.Sprintf("%s.%s.%s", OptionsTable, mode, class), option, vars)
				if err != nil {
					return err
				}
				f.Options[mode][class] = append(f.Options[mode][class], s)
			}
		}

	}

	f.Volume = make(map[string]VolumeT)
	for volKey, volVal := range c.Volume {
		volFrag := VolumeT{OnRequest: volVal.OnRequest}
		if volVal.Volume != "" {
			volFrag.Volume, err = expandFragments(fmt.Sprintf("volume %s volume", volKey), volVal.Volume, vars)
			if err != nil {
				return err
			}
		}
		if volVal.Mount != "" {
			volFrag.Mount, err = expandFragments(fmt.Sprintf("volume %s mount", volKey), volVal.Mount, vars)
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

func (f *fragmentsT) formatOptions(mode, class string) []string {
	var s []string
	for _, option := range f.Options[CommonMode][class] {
		s = append(s, option)
	}
	for _, option := range f.Options[mode][class] {
		s = append(s, option)
	}
	return s
}

func (f *fragmentsT) formatVolumes(requested map[string]bool) []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Volume != "" && (!vol.OnRequest || requested[key]) {
			s = append(s, "--volume", fmt.Sprintf("%s,%s", key, vol.Volume))
		}
	}
	return s
}

func (f *fragmentsT) formatMounts(requested map[string]bool) []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Mount != "" && (!vol.OnRequest || requested[key]) {
			s = append(s, "--mount", fmt.Sprintf("volume=%s,%s", key, vol.Mount))
		}
	}
	return s
}
