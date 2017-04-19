package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run-helper: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	const runner string = "rkt-run"
	runnerPath, err := exec.LookPath(runner)
	if err != nil {
		die("%v", err)
	}

	// new args look like the old ones, except for the program and first argument (alias)
	args := append(os.Args, "")
	copy(args[2:], args[1:])

	// basename of what we were invoked as determines the alias
	args[1] = filepath.Base(args[0])
	args[0] = runner
	err = syscall.Exec(runnerPath, args, os.Environ())
	if err != nil {
		die("%v", err)
	}
}
