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
	"text/template"
)

type aliasFragmentsT struct {
	Environment map[string]string
	Passwd      []string
	Group       []string
}

type fragmentsT struct {
	Environment map[string]string
	Options     ModeOptionsT
	Volume      map[string]VolumeT
	Alias       map[string]aliasFragmentsT
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

	f.Alias = make(map[string]aliasFragmentsT)
	for aliasKey, aliasVal := range c.Alias {
		envMap := make(map[string]string)
		for envKey, envVal := range aliasVal.Environment {
			envFrag, err := expandFragments(fmt.Sprintf("alias %s environ %s", aliasKey, envKey), envVal, vars)
			if err != nil {
				return err
			}
			envMap[envKey] = envFrag
		}

		passwd := make([]string, len(aliasVal.Passwd), len(aliasVal.Passwd))
		for i, passwdVal := range aliasVal.Passwd {
			passwdFrag, err := expandFragments(fmt.Sprintf("alias %s passwd %d", aliasKey, i), passwdVal, vars)
			if err != nil {
				return err
			}
			passwd[i] = passwdFrag
		}

		group := make([]string, len(aliasVal.Group), len(aliasVal.Group))
		for i, groupVal := range aliasVal.Group {
			groupFrag, err := expandFragments(fmt.Sprintf("alias %s group %d", aliasKey, i), groupVal, vars)
			if err != nil {
				return err
			}
			group[i] = groupFrag
		}

		f.Alias[aliasKey] = aliasFragmentsT{Environment: envMap, Passwd: passwd, Group: group}
	}

	return nil
}

func (f *fragmentsT) getEnvironment(alias string) map[string]string {
	// merge general and image environment
	mergedEnviron := make(map[string]string)
	for key, val := range f.Environment {
		mergedEnviron[key] = val
	}
	if alias != "" {
		aliasFragments, ok := f.Alias[alias]
		if ok {
			for key, val := range aliasFragments.Environment {
				mergedEnviron[key] = val
			}
		}
	}
	return mergedEnviron
}

func (f *fragmentsT) passwd(alias string) []string {
	return f.Alias[alias].Passwd
}

func (f *fragmentsT) group(alias string) []string {
	return f.Alias[alias].Group
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
