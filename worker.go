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

	"github.com/appc/spec/schema"
)

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

	w.AppName = fmt.Sprintf("rktrunner-%s", u.Username)

	err = w.findPod()
	if err != nil {
		return nil, err
	}

	return w, nil
}

// FoundPod returns whether we found (and locked) a suitable pod.
func (w *Worker) FoundPod() bool {
	return w.UUID != ""
}

// LockPod attempts to acquire a shared lock on the pod, without blocking.
func (w *Worker) LockPod(uuid string) error {
	podlock, err := os.Open(workerPodDir(uuid))
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
	err = os.MkdirAll(workerPodDir(uuid), 0755)
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
func (w *Worker) findPod() error {
	cmd := exec.Command("rkt", "list", "--full", "--no-legend")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() && !w.FoundPod() {
		fields := strings.Fields(scanner.Text())
		if fields[1] == w.AppName && fields[2] == w.image && fields[4] == "running" {
			candidateUUID := fields[0]
			warn := w.verifyPodUser(candidateUUID)
			if warn != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
			} else {
				warn := w.LockPod(candidateUUID)
				if warn != nil {
					fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
				}
			}
		}
	}

	scannerErr := scanner.Err()
	warn := cmd.Wait()
	// ensure we warn if something went wrong
	if warn == nil && scannerErr != nil {
		warn = scannerErr
	}
	if warn != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
	}

	return nil
}
