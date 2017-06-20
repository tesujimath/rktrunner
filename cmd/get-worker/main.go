package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/tesujimath/rktrunner"
)

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "get-worker: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 {
		die("%s", "usage: get-worker <image> <uid>")
	}

	image := os.Args[1]
	uid, err := strconv.Atoi(os.Args[2])
	if err != nil {
		die("expected uid, got %s", os.Args[2])
	}

	_, err = rktrunner.GetWorker(image, uid)
	if err != nil {
		die("%v", err)
	}
}
