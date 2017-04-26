package main

import (
	"fmt"
	"github.com/tesujimath/rktrunner"
	"os"
	"time"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-attach: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		die("%s", "usage: rkt-attach <app-name> <done-path>")
	}
	environ := append(os.Environ(), "RKT_EXPERIMENT_ATTACH=true")
	appName := os.Args[1]
	donePath := os.Args[2]
	attach := rktrunner.NewAttacher(donePath, environ)
	attach.ByName(appName)
	time.Sleep(time.Duration(10 * time.Second))
}
