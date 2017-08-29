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
	"io"
	"os"
	"sort"
	"strings"
)

// ParseEnviron extracts all environment variables into a map
func ParseEnviron(env []string) map[string]string {
	environ := make(map[string]string)
	for _, keyval := range os.Environ() {
		i := strings.IndexRune(keyval, '=')
		if i != -1 {
			key := keyval[:i]
			val := keyval[i+1:]
			environ[key] = val
		}
	}
	return environ
}

// UpdateEnviron updates the map with a name=value
func UpdateEnviron(environ map[string]string, keyval string) {
	i := strings.IndexRune(keyval, '=')
	if i != -1 {
		key := keyval[:i]
		val := keyval[i+1:]
		environ[key] = val
	}
}

// BuildEnviron turns the environ map into a list of strings
func BuildEnviron(environ map[string]string) []string {
	var env []string
	for key, val := range environ {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	return env
}

func PrintEnvironment(w io.Writer, environ map[string]string) {
	// get keys in order
	keys := make([]string, 0, len(environ))
	for key := range environ {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s=%s\n", key, environ[key])
	}
}
