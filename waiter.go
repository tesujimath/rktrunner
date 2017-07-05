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

type Waitable interface {
	Wait() error
}

// NewWaiter wraps a simple Wait() call in a goroutine,
// so multiple events can be awaited using select.
func NewWaiter(w Waitable) chan error {
	c := make(chan error)
	go func() {
		err := w.Wait()
		if err != nil {
			c <- err
		}
		close(c)
	}()
	return c
}
