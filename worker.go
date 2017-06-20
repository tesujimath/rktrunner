package rktrunner

import (
	"bufio"
	"encoding/json"
	"fmt"
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
}

func NewWorker(u *user.User, image string) (*Worker, error) {
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}
	w := &Worker{
		uid:     uid,
		image:   image,
		AppName: fmt.Sprintf("worker-%s", u.Username),
	}

	err = w.findPod()
	if err != nil {
		return nil, err
	}

	return w, nil
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
	var uuid string
	for scanner.Scan() && uuid == "" {
		fields := strings.Fields(scanner.Text())
		if fields[1] == w.AppName && fields[2] == w.image && fields[4] == "running" {
			candidateUUID := fields[0]
			warn := w.verifyPodUser(candidateUUID)
			if warn != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
			} else {
				// if we can lock the workerPodDir, we can use it
				podlock, warn := os.Open(workerPodDir(candidateUUID))
				if warn != nil {
					fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
				} else {
					warn := syscall.Flock(int(podlock.Fd()), syscall.LOCK_SH)
					if warn != nil {
						fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
						podlock.Close()
					} else {
						// We found a suitable pod, and locked it for use.
						// Now we leave the podlock open, to retain the lock.
						uuid = candidateUUID
					}
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

	w.UUID = uuid
	return nil
}
