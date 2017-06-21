package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

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

func stopPod(uuid string) error {
	args := []string{"rkt", "stop", uuid}
	argv0, err := exec.LookPath(args[0])
	if err != nil {
		die("%v PATH=%s", err, os.Getenv("PATH"))
	}
	cmd := exec.Command(argv0, args[1:]...)
	err = cmd.Run()
	if err == nil {
		fmt.Fprintf(os.Stderr, "stopping inactive pod %s\n", uuid)
	}
	return err
}

func main() {
	err := rktrunner.VisitPods(func(pod *rktrunner.VisitedPod) bool {
		if pod.Status == "running" && strings.HasPrefix(pod.AppName, rktrunner.WORKER_APPNAME_PREFIX) {
			podlock, err := lockPod(pod.UUID)
			if err != nil {
				errno, isErrno := err.(syscall.Errno)
				if isErrno && errno == syscall.EAGAIN {
					fmt.Fprintf(os.Stderr, "skipping active pod %s\n", pod.UUID)
				} else {
					fmt.Fprintf(os.Stderr, "pod %s warning: %v %T\n", pod.UUID, err, err)
				}
			} else if podlock != nil {
				err = stopPod(pod.UUID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "pod %s warning: %v\n", pod.UUID, err)
				}
			}
		}
		return true
	})
	if err != nil {
		die("%v", err)
	}
}
