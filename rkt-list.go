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
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

type VisitedPod struct {
	UUID    string
	AppName string
	Image   string
	State   string
	Created string
	Started string
}

func (p *VisitedPod) String() string {
	return fmt.Sprintf("%s %s pod %s for %s", p.AppName, p.State, p.UUID, p.Image)
}

// VisitPods visits all pods, until the walker returns false.
func VisitPods(walker func(*VisitedPod) bool) error {
	cmd := exec.Command("rkt", "list", "--full", "--no-legend")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	keepVisiting := true
	for scanner.Scan() && keepVisiting {
		fields := strings.FieldsFunc(scanner.Text(), func(c rune) bool { return c == '\t' })
		if len(fields) > 4 {
			pod := VisitedPod{
				UUID:    fields[0],
				AppName: fields[1],
				Image:   fields[2],
				State:   fields[4],
			}
			if len(fields) > 5 {
				pod.Created = fields[5]
			}
			if len(fields) > 6 {
				pod.Started = fields[6]
			}

			keepVisiting = walker(&pod)
		}
	}

	err = scanner.Err()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
