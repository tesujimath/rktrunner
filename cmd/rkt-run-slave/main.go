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
	"os/exec"
	"syscall"

	"github.com/droundy/goopt"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run-slave: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func waitForever() error {
	r, _, err := os.Pipe()
	if err != nil {
		return err
	}
	buf := make([]byte, 1, 1)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "warning: waitForever() returned unexpectedly\n")
	return nil
}

func main() {
	wait := goopt.Flag([]string{"--wait"}, []string{}, "wait forever", "")
	cwd := goopt.String([]string{"--cwd"}, "", "run with current working directory")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Summary = "Slave program to run within rkt container"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)
	args := goopt.Args

	if *wait {
		err := waitForever()
		if err != nil {
			die("%v", err)
		}
	}

	if *cwd != "" {
		err := os.Chdir(*cwd)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "warning: directory %s does not exist in container\n", *cwd)
			} else {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		}
	}

	if len(args) > 0 {
		argv0, err := exec.LookPath(args[0])
		if err != nil {
			die("%v PATH=%s", err, os.Getenv("PATH"))
		}
		err = syscall.Exec(argv0, args, os.Environ())
		if err != nil {
			die("%v", err)
		}
	} else {
		die("warning: %s", "nothing to execute")
	}
}
