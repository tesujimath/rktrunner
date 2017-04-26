package rktrunner

import (
	"fmt"
	"os"
)

const slaveBinVolume = "rktrunner-bin"
const slaveBinDir = "/usr/lib/rktrunner"
const slaveRunVolume = "rktrunner-run"
const slaveRunDir = "/var/run/rktrunner"

const slaveRunner = "rkt-run-slave"
const attachReadyFile = "attached"

func masterRunDir() string {
	return fmt.Sprintf("/tmp/rktrunner%d", os.Getpid())
}
