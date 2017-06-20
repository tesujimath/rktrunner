package rktrunner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	if c.cmd.Process != nil {
		fmt.Fprintf(w, " (pid %d)\n", c.cmd.Process.Pid)
	} else {
		fmt.Fprintf(w, "\n")
	}
}

func (c *CommandT) create() {
	c.cmd = exec.Command(c.argv[0], c.argv[1:]...)
	c.cmd.Path = c.argv0
	c.cmd.Env = c.envv
	c.cmd.Stdin = os.Stdin
	c.cmd.Stdout = os.Stdout
	c.cmd.Stderr = os.Stderr
	c.cmd.ExtraFiles = c.extraFiles
}

func (c *CommandT) PreserveFile(f *os.File) {
	c.extraFiles = append(c.extraFiles, f)
}

func (c *CommandT) Run() error {
	c.create()
	return c.cmd.Run()
}

func (c *CommandT) Start() error {
	c.create()
	return c.cmd.Start()
}

func (c *CommandT) Wait() error {
	return c.cmd.Wait()
}
