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

	"github.com/appc/spec/schema"
)

type Worker struct {
}

func workerAppName(uid int) string {
	u, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return fmt.Sprintf("worker-%d", uid)
	} else {
		return fmt.Sprintf("worker-%s", u.Username)
	}
}

func verifyPodUser(uuid string, uid int) error {
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

	if ra.App.User != strconv.Itoa(uid) {
		return fmt.Errorf("unexpected pod manifest user %s, expected %d", ra.App.User, uid)
	}

	return nil
}

// findWorker finds the UUID for a worker, if any
func findWorker(image string, uid int) (string, error) {
	name := workerAppName(uid)
	cmd := exec.Command("rkt", "list", "--full", "--no-legend")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	err = cmd.Start()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(stdout)
	var uuid string
	for scanner.Scan() && uuid == "" {
		fields := strings.Fields(scanner.Text())
		if fields[1] == name && fields[2] == image && fields[4] == "running" {
			candidateUuid := fields[0]
			warn := verifyPodUser(candidateUuid, uid)
			if warn != nil {
				fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
			} else {
				uuid = candidateUuid
			}
		}
	}

	scannerErr := scanner.Err()
	err = cmd.Wait()
	// ensure we return scanner error if something went wrong
	if err == nil && scannerErr != nil {
		err = scannerErr
	}
	if err != nil {
		return "", err
	}

	return uuid, nil
}

func GetWorker(image string, uid int) (string, error) {
	uuid, err := findWorker(image, uid)
	if err != nil {
		return "", err
	}
	// if uuid == "" {
	// 	uuid, err := startWorker(uid, image)
	// }
	return uuid, nil
}
