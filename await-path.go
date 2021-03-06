// Copyright 2017 The rktrunner Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

// NewPathWaiter waits until the path appears
func NewPathWaiter(path string) chan error {
	c := make(chan error)
	go func() {
		awaitDirEvents := make(chan notify.EventInfo, 2)
		err := notify.Watch(filepath.Dir(path), awaitDirEvents, notify.InCloseWrite)
		if err != nil {
			c <- err
			close(c)
			return
		}
		defer notify.Stop(awaitDirEvents)

		// check after creating awaitDirEvents, to avoid race
		if exists(path) {
			close(c)
			return
		}

		for {
			switch ei := <-awaitDirEvents; ei.Event() {
			case notify.InCloseWrite:
				if exists(path) {
					close(c)
					return
				}
			}
		}
		// unreached
	}()
	return c
}
