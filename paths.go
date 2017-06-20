package rktrunner

import (
	"fmt"
	"os"
	"path/filepath"
)

const slaveBinVolume = "rktrunner-bin"
const slaveBinDir = "/usr/lib/rktrunner"

const masterRoot = "/var/lib/rktrunner"

const slaveRunner = "rkt-run-slave"

func masterRunDir() string {
	return filepath.Join(masterRoot, fmt.Sprintf("runner-%d", os.Getpid()))
}

func envFilePath() string {
	return filepath.Join(masterRunDir(), "env")
}

func uuidFilePath() string {
	return filepath.Join(masterRunDir(), "uuid")
}
