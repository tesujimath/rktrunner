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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"github.com/appc/spec/schema"
)

const WORKER_APPNAME_PREFIX = "rktrunner-"

type Worker struct {
	uid     int
	image   string
	AppName string
	UUID    string
	Podlock *os.File
}

func NewWorker(u *user.User, image string) (*Worker, error) {
	var err error
	w := &Worker{}

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

// LockPod attempts to acquire a shared lock on the pod, without blocking.
func (w *Worker) LockPod(uuid string) error {
	podlock, err := os.Open(WorkerPodDir(uuid))
	if err != nil {
		return err
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
func (w *Worker) InitializePod(uuidPath string) error {
	// determine the pod UUID
	err := awaitPath(uuidPath)
	if err != nil {
		return err
	}
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
	warn := VisitPods(func(pod *VisitedPod) bool {
		if pod.AppName == w.AppName && pod.Image == w.image && pod.Status == "running" {
			warn := w.verifyPodUser(pod.UUID)
			if warn != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
			} else {
				warn := w.LockPod(pod.UUID)
				if warn != nil {
					fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
				}
			}
		}
		return !w.FoundPod()
	})
	if warn != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
	}
}
