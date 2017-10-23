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
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type CommandT struct {
	argv0      string
	argv       []string
	envv       []string
	extraFiles []*os.File
	cmd        *exec.Cmd
}

func NewCommand(argv0 string) *CommandT {
	c := &CommandT{argv0: argv0}
	c.argv = make([]string, 1)
	c.argv[0] = filepath.Base(argv0)
	return c
}

func (c *CommandT) AppendArgs(args ...string) {
	c.argv = append(c.argv, args...)
}

func (c *CommandT) SetEnviron(envv []string) {
	c.envv = envv
}

func (c *CommandT) Print(w io.Writer) {
	fmt.Fprintf(w, "%s %s", c.argv0, strings.Join(c.argv[1:], " "))
	if c.cmd != nil && c.cmd.Process != nil {
		fmt.Fprintf(w, " (pid %d)\n", c.cmd.Process.Pid)
	} else {
		fmt.Fprintf(w, "\n")
	}
}

func (c *CommandT) create(preserveStdio bool) {
	c.cmd = exec.Command(c.argv[0], c.argv[1:]...)
	c.cmd.Path = c.argv0
	c.cmd.Env = c.envv
	if preserveStdio {
		c.cmd.Stdin = os.Stdin
		c.cmd.Stdout = os.Stdout
		c.cmd.Stderr = os.Stderr
	}
	c.cmd.ExtraFiles = c.extraFiles
}

func (c *CommandT) PreserveFile(f *os.File) {
	c.extraFiles = append(c.extraFiles, f)
}

func (c *CommandT) Run() error {
	c.create(true)
	return c.cmd.Run()
}

func (c *CommandT) Start() error {
	c.create(true)
	return c.cmd.Start()
}

func (c *CommandT) StartDaemon() error {
	c.create(false)
	return c.cmd.Start()
}

func (c *CommandT) Wait() error {
	if c.cmd != nil {
		return c.cmd.Wait()
	}
	return nil
}

func (c *CommandT) Exec() error {
	for _, f := range c.extraFiles {
		// clear O_CLOEXEC which is set by default
		_, _, err := syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_SETFD, 0)
		if err != syscall.Errno(0x0) {
			WarnError(err)
		}
	}
	return syscall.Exec(c.argv0, c.argv, c.envv)
}
