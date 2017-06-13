package rktrunner

import (
	"fmt"
	"os"
	"path/filepath"
)

const slaveBinVolume = "rktrunner-bin"
const slaveBinDir = "/usr/lib/rktrunner"

const slaveFdVolume = "rktrunner-fd"
const slaveFdDir = "/var/run/rktrunner"

const slaveRunner = "rkt-run-slave"

func masterRunDir() string {
	return fmt.Sprintf("/tmp/rktrunner%d", os.Getpid())
}

func masterFdDir() string {
	return fmt.Sprintf("/proc/%d/fd", os.Getpid())
}

func envFilePath() string {
	return filepath.Join(masterRunDir(), "env")
}
