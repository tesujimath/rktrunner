package main

import (
	"fmt"
	"github.com/droundy/goopt"
	"github.com/rjeczalik/notify"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run-slave: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// await waits until the path appears
func await(path string) error {
	awaitDirEvents := make(chan notify.EventInfo, 2)
	err := notify.Watch(filepath.Dir(path), awaitDirEvents, notify.InCloseWrite)
	if err != nil {
		return err
	}
	defer notify.Stop(awaitDirEvents)

	// check after creating awaitDirEvents, to avoid race
	if exists(path) {
		return nil
	}

	for {
		switch ei := <-awaitDirEvents; ei.Event() {
		case notify.InCloseWrite:
			if exists(path) {
				return nil
			}
		}
	}
	// unreached
	return nil
}

func main() {
	awaitFile := goopt.String([]string{"--await-file"}, "", "wait for file to exist before running")
	cwd := goopt.String([]string{"--cwd"}, "", "run with current working directory")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Summary = "Slave program to run within rkt container"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)
	args := goopt.Args

	if *awaitFile != "" {
		err := await(*awaitFile)
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
			die("%v", err)
		}
		err = syscall.Exec(argv0, args, os.Environ())
		if err != nil {
			die("%v", err)
		}
	} else {
		die("warning: %s", "nothing to execute")
	}
}
