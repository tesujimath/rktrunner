package rktrunner

import (
	"bufio"
	"os/exec"
	"strings"
)

type VisitedPod struct {
	UUID    string
	AppName string
	Image   string
	Status  string
}

// VisitPods visits all pods, until the walker returns false.
func VisitPods(walker func(*VisitedPod) bool) error {
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
	keepVisiting := true
	for scanner.Scan() && keepVisiting {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 4 {
			pod := VisitedPod{
				UUID:    fields[0],
				AppName: fields[1],
				Image:   fields[2],
				Status:  fields[4],
			}
			keepVisiting = walker(&pod)
		}
	}

	err = scanner.Err()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
