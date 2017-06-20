package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/droundy/goopt"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run-slave: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func waitForever() error {
	r, _, err := os.Pipe()
	if err != nil {
		return err
	}
	buf := make([]byte, 1, 1)
	_, err = r.Read(buf)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "warning: waitForever() returned unexpectedly\n")
	return nil
}

func main() {
	wait := goopt.Flag([]string{"--wait"}, []string{}, "wait forever", "")
	cwd := goopt.String([]string{"--cwd"}, "", "run with current working directory")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Summary = "Slave program to run within rkt container"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)
	args := goopt.Args

	if *wait {
		err := waitForever()
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
