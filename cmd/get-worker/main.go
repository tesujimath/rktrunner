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

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/tesujimath/rktrunner"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "get-worker: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		die("%s", "usage: get-worker <image> <uid>")
	}

	image := os.Args[1]
	uid, err := strconv.Atoi(os.Args[2])
	if err != nil {
		die("expected uid, got %s", os.Args[2])
	}

	_, err = rktrunner.GetWorker(image, uid)
	if err != nil {
		die("%v", err)
	}
}
