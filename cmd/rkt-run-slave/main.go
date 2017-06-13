package main

import (
	"fmt"
	"github.com/droundy/goopt"
	"os"
	"os/exec"
	"path/filepath"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run-slave: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func main() {
	cwd := goopt.String([]string{"--cwd"}, "", "run with current working directory")
	attachStdio := goopt.String([]string{"--attach-stdio"}, "", "directory containing host file descriptors to attach for stdio")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Summary = "Slave program to run within rkt container"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)
	args := goopt.Args

	// TODO remove:
	fmt.Fprintf(os.Stderr, "rkt-run-slave: running %v\n", args)

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

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Path = argv0
		if *attachStdio != "" {
			cmd.Stdin, err = os.OpenFile(filepath.Join(*attachStdio, "0"), os.O_RDONLY, 0)
			if err != nil {
				die("%v", err)
			}
			cmd.Stdout, err = os.OpenFile(filepath.Join(*attachStdio, "1"), os.O_WRONLY, 0)
			if err != nil {
				die("%v", err)
			}
			cmd.Stderr, err = os.OpenFile(filepath.Join(*attachStdio, "2"), os.O_WRONLY, 0)
			if err != nil {
				die("%v", err)
			}
		} else {
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		err = cmd.Run()
		if err != nil {
			die("%v", err)
		}
	} else {
		die("warning: %s", "nothing to execute")
	}
}
