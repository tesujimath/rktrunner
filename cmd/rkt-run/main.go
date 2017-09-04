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
	"os/signal"
	"syscall"

	"github.com/tesujimath/rktrunner"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run: %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	r, err := rktrunner.NewRunner("/etc/rktrunner.toml")
	// for testing:
	// r, err := rktrunner.NewRunner("/home/guestsi/go/src/github.com/tesujimath/rktrunner/examples/rktrunner-biocontainers.toml")
	if err != nil {
		die("%v", err)
	}

	// ensure we cleanup on interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		for {
			s := <-c
			// We can't simply signal.Ignore(syscall.SIGINT), as that would
			// inhibit child processes from receiving it, so we disregard it here.
			if s != syscall.SIGINT {
				r.RemoveTempFiles()
				os.Exit(1)
			}
		}
	}()

	// set real uid same as effective
	err = syscall.Setreuid(syscall.Geteuid(), syscall.Geteuid())
	if err != nil {
		die("failed to set real uid: %v", err)
	}

	err = r.Execute()
	if err != nil {
		_, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			os.Exit(1)
		} else {
			die("failed: %v", err)
		}
	}
}
