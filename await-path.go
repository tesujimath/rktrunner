package rktrunner

import (
	"os"
	"path/filepath"

	"github.com/rjeczalik/notify"
)

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// awaitPath waits until the path appears
func awaitPath(path string) error {
	awaitDirEvents := make(chan notify.EventInfo, 2)
	err := notify.Watch(filepath.Dir(path), awaitDirEvents, notify.InCloseWrite)
	if err != nil {
		return err
	}
	defer notify.Stop(awaitDirEvents)

	// check after creating awaitDirEvents, to avoid race
	if exists(path) {
		return nil
	}

	for {
		switch ei := <-awaitDirEvents; ei.Event() {
		case notify.InCloseWrite:
			if exists(path) {
				return nil
			}
		}
	}
	// unreached
	return nil
}
