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
	"strings"
	"syscall"
	"time"

	"github.com/droundy/goopt"
	"github.com/tesujimath/rktrunner"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rktrunner-gc: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func lockPod(uuid string) (*os.File, error) {
	podlock, err := os.Open(rktrunner.WorkerPodDir(uuid))
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(podlock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		podlock.Close()
		return nil, err
	}
	return podlock, nil
}

func stopPod(pod *rktrunner.VisitedPod) error {
	args := []string{"rkt", "stop", pod.UUID}
	argv0, err := exec.LookPath(args[0])
	if err != nil {
		die("%v PATH=%s", err, os.Getenv("PATH"))
	}
	cmd := exec.Command(argv0, args[1:]...)
	err = cmd.Run()
	if err == nil {
		fmt.Fprintf(os.Stderr, "stop idle %s\n", pod)
	}
	return err
}

func main() {
	dryRun := goopt.Flag([]string{"--dry-run"}, []string{}, "don't execute anything", "")
	gracePeriodRaw := goopt.String([]string{"--grace-period"}, "", "duration to wait before collecting idle worker pods")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Summary = "rktrunner worker pod garbage collector"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)

	var gracePeriod time.Duration
	var err error
	if *gracePeriodRaw != "" {
		gracePeriod, err = time.ParseDuration(*gracePeriodRaw)
		if err != nil {
			die("%v", err)
		}
	}

	err = rktrunner.VisitPods(func(pod *rktrunner.VisitedPod) bool {
		if pod.State == "running" && strings.HasPrefix(pod.AppName, rktrunner.WORKER_APPNAME_PREFIX) {
			var expired bool
			if pod.Started != "" {
				started, err := time.Parse("2006-01-02 15:04:05.9 -0700 MST", pod.Started)
				if err != nil {
					die("failed to parse start time for pod %s: %v", pod.UUID, err)
				}
				expiry := started.Add(gracePeriod)
				expired = time.Now().After(expiry)
			}
			if !expired {
				fmt.Fprintf(os.Stderr, "skip baby %s\n", pod)
			} else {
				podlock, err := lockPod(pod.UUID)
				if err != nil {
					errno, isErrno := err.(syscall.Errno)
					if isErrno && errno == syscall.EAGAIN {
						fmt.Fprintf(os.Stderr, "skip busy %s\n", pod)
					} else {
						fmt.Fprintf(os.Stderr, "warning: %s %v %T\n", pod, err, err)
					}
				} else if podlock != nil {
					if *dryRun {
						fmt.Fprintf(os.Stderr, "stop idle %s\n", pod)
					} else {
						err = stopPod(pod)
						if err != nil {
							fmt.Fprintf(os.Stderr, "warning: %s %v\n", pod, err)
						} else {
							os.Remove(rktrunner.WorkerPodDir(pod.UUID))
						}
					}
					podlock.Close()
				}
			}
		}
		return true
	})
	if err != nil {
		die("%v", err)
	}
}
