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
	"os"
	"path/filepath"
)

const slaveBinVolume = "rktrunner-bin"
const slaveBinDir = "/usr/lib/rktrunner"

const masterRoot = "/var/lib/rktrunner"

const slaveRunner = "rkt-run-slave"

func masterRunDir() string {
	return filepath.Join(masterRoot, fmt.Sprintf("runner-%d", os.Getpid()))
}

func WorkerPodDir(uuid string) string {
	return filepath.Join(masterRoot, fmt.Sprintf("pod-%s", uuid))
}

func envFilePath() string {
	return filepath.Join(masterRunDir(), "env")
}

func uuidFilePath() string {
	return filepath.Join(masterRunDir(), "uuid")
}
