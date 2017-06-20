package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/tesujimath/rktrunner"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	r, err := rktrunner.NewRunner("/etc/rktrunner.toml")
	// for testing:
	// r, err := rktrunner.NewRunner("/home/guestsi/go/src/github.com/tesujimath/rktrunner/examples/rktrunner-experimental.toml")
	if err != nil {
		die("%v", err)
	}

	// set real uid same as effective
	err = syscall.Setreuid(syscall.Geteuid(), syscall.Geteuid())
	if err != nil {
		die("failed to set real uid: %v", err)
	}

	err = r.Execute()
	if err != nil {
		switch err {
		case rktrunner.ErrRktRunFailed:
			// don't output message, since rkt run already did
			os.Exit(1)
		default:
			die("failed: %v", err)
		}
	}
}
