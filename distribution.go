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
	"strings"
)

type imagePrefixT struct {
	raw       string
	canonical string
}

var imagePrefixes []imagePrefixT = []imagePrefixT{
	{
		raw:       "docker://",
		canonical: "registry-1.docker.io/library/",
	},
	{
		raw:       "docker:",
		canonical: "registry-1.docker.io/library/",
	},
}

// CanonicalImageName converts the convenience prefixes into official
// paths, and ensures there is a tag suffix, by appending :latest if required.
func CanonicalImageName(raw string) string {
	canonical := raw
	for _, prefix := range imagePrefixes {
		if strings.HasPrefix(raw, prefix.raw) {
			n := len(prefix.raw)
			canonical = prefix.canonical + raw[n:]
			break
		}
	}

	// ensure we have a tag
	colon := strings.IndexRune(canonical, ':')
	if colon < 0 {
		canonical = canonical + ":latest"
	}

	return canonical
}
