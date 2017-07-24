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
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/appc/spec/schema"
)

const WORKER_APPNAME_PREFIX = "rktrunner-"

type Worker struct {
	rkt     string
	uid     int
	image   string
	verbose bool
	AppName string
	UUID    string
	Podlock *os.File
}

func NewWorker(u *user.User, image, rkt string, verbose bool) (*Worker, error) {
	var err error
	w := &Worker{rkt: rkt, verbose: verbose}

	w.uid, err = strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}

	// need version suffix on image name, to match output of rkt list
	if strings.ContainsRune(image, ':') {
		w.image = image
	} else {
		w.image = fmt.Sprintf("%s:latest", image)
	}

	w.AppName = fmt.Sprintf("%s%s", WORKER_APPNAME_PREFIX, u.Username)

	w.findPod()

	return w, nil
}

// FoundPod returns whether we found (and locked) a suitable pod.
func (w *Worker) FoundPod() bool {
	return w.UUID != ""
}

// awaitReady waits until the pod is running (or exited), which is necessary
// if we just created it.
func (w *Worker) awaitReady(uuid string) error {
	ready := false
	for !ready {
		cmd := exec.Command("rkt", "status", uuid)
		cmd.Path = w.rkt
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		err = cmd.Start()
		if err != nil {
			return fmt.Errorf("%s %s %s failed to start: ", w.rkt, "status", uuid, err)
		}

		foundState := false
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() && !foundState {
			fields := strings.SplitN(scanner.Text(), "=", 2)
			if len(fields) == 2 && fields[0] == "state" {
				foundState = true
				if fields[1] == "running" || fields[1] == "exited" {
					ready = true
				}
			}
		}
		if !foundState {
			fmt.Fprintf(os.Stderr, "warning: rkt status %s failed to list state\n", uuid)
		}

		warn := cmd.Wait()
		if warn != nil {
			// Simply warn about rkt status failure, since it does fail if
			// we call it too early.  And retry.
			if w.verbose {
				fmt.Fprintf(os.Stderr, "warning: rkt status %s failed: %v, retry\n", uuid, warn)
			}
			ready = false
		}
		err = scanner.Err()
		if err != nil {
			return err
		}
		if !ready {
			// not yet ready, so pause before retry
			if w.verbose {
				fmt.Fprintf(os.Stderr, "waiting for worker pod %s\n", uuid)
			}
			time.Sleep(1)
		}
	}
	return nil
}

// LockPod attempts to acquire a shared lock on the pod, without blocking.
func (w *Worker) LockPod(uuid string) error {
	podlock, err := os.Open(WorkerPodDir(uuid))
	if err != nil {
		return fmt.Errorf("LockPod attempt %v", err)
	}
	err = syscall.Flock(int(podlock.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err != nil {
		podlock.Close()
		return err
	}
	w.UUID = uuid
	w.Podlock = podlock
	return nil
}

// InitializePod sets up a new pod for use as a worker, and locks it.
func (w *Worker) InitializePod(uuidPath string, cmdWaiter chan error) error {
	// wait for the UUID file, or the cmd itself to finish (e.g. on failure)
	pathWaiter := NewPathWaiter(uuidPath)
	select {
	case err := <-pathWaiter:
		if err != nil {
			return err
		}

	case err := <-cmdWaiter:
		if err != nil {
			return err
		}
	}

	// determine the pod UUID
	uuidFile, err := os.Open(uuidPath)
	if err != nil {
		return err
	}
	defer uuidFile.Close()
	uuidBytes, err := ioutil.ReadAll(uuidFile)
	if err != nil {
		return err
	}
	uuid := string(uuidBytes)

	// wait for the pod to be actually running, or exited (in case of early failure)
	err = w.awaitReady(uuid)
	if err != nil {
		return fmt.Errorf("awaitReady(%s) failed: %v\n", uuid, err)
	}

	// create the worker pod dir, which can be locked by users of the worker
	err = os.MkdirAll(WorkerPodDir(uuid), 0755)
	if err != nil {
		return err
	}

	return w.LockPod(uuid)
}

func (w *Worker) verifyPodUser(uuid string) error {
	cmd := exec.Command("rkt", "cat-manifest", uuid)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	manifest := json.NewDecoder(stdout)
	var pm schema.PodManifest
	err = manifest.Decode(&pm)
	errWait := cmd.Wait()
	if err != nil {
		return err
	}
	if errWait != nil {
		return errWait
	}

	if len(pm.Apps) != 1 {
		return fmt.Errorf("unexpected pod manifest with %d apps", len(pm.Apps))
	}
	ra := pm.Apps[0]

	if ra.App.User != strconv.Itoa(w.uid) {
		return fmt.Errorf("unexpected pod manifest user %s, expected %d", ra.App.User, w.uid)
	}

	return nil
}

// findPod finds the UUID for a worker pod, if any
func (w *Worker) findPod() {
	imageName := CanonicalImageName(w.image)
	WarnOnFailure(VisitPods(func(pod *VisitedPod) bool {
		if pod.AppName == w.AppName && pod.Image == imageName && pod.State == "running" {
			err := w.verifyPodUser(pod.UUID)
			if err != nil {
				WarnError(err)
			} else {
				WarnOnFailure(w.LockPod(pod.UUID))
			}
		}
		return !w.FoundPod()
	}))
}
