package main

import (
	"fmt"
	"github.com/tesujimath/rktrunner"
	"os"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-attach: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		die("%s", "usage: rkt-attach-by-name <app-name> <done-path>")
	}
	environ := append(os.Environ(), "RKT_EXPERIMENT_ATTACH=true")
	appName := os.Args[1]
	donePath := os.Args[2]
	attach := rktrunner.NewAttacher(donePath, environ)
	attach.ByName(appName)
	err := attach.Wait()
	if err != nil {
		fmt.Fprintf(os.Stderr, "rkt-attach-by-name: %v\n", err)
	}
}
