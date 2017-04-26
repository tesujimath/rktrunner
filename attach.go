package rktrunner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Attacher struct {
	donePath string
	environ  []string
	abort    chan bool
}

func NewAttacher(donePath string, environ []string) *Attacher {
	return &Attacher{
		donePath: donePath,
		environ:  environ,
		abort:    make(chan bool),
	}
}

func (a *Attacher) warn(err error) {
	fmt.Fprintf(os.Stderr, "rkt-run: warning: attach failure %v\n", err)
}

func (a *Attacher) ByName(containerName string) {
	go a.run(containerName)
}

func (a *Attacher) Abort() {
	a.abort <- true
}

func (a *Attacher) run(containerName string) {
	var uuid string
	var err error
loop:
	for uuid == "" && err == nil {
		uuid, err = findUuid(containerName)
		if err != nil {
			a.warn(err)
		}

		if uuid == "" {
			// wait for a while
			fmt.Printf("attacher: waiting ...\n")
			timeout := time.After(time.Duration(time.Second))
			select {
			case <-a.abort:
				a.warn(fmt.Errorf("abort"))
				break loop
			case <-timeout:
				// go around again
			}
		}
	}

	if uuid != "" {
		err = attachByUuid(uuid, a.environ)
		fmt.Printf("attachByUuid %v", err)
		if err != nil {
			a.warn(err)
		}
	}

	// signal we're done
	f, err := os.Create(a.donePath)
	if err != nil {
		a.warn(err)
		return
	}
	f.Close()
}

// findUuid returns the uuid for the named container,
// or an empty string if it isn't found
func findUuid(containerName string) (uuid string, err error) {
	cmd := exec.Command("rkt", "list", "--full", "--no-legend")
	var stdout io.ReadCloser
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return
	}

	err = cmd.Start()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if fields[1] == containerName {
			uuid = fields[0]
		}
	}

	scannerErr := scanner.Err()
	err = cmd.Wait()
	// ensure we return scanner error if something went wrong
	if err == nil && scannerErr != nil {
		err = scannerErr
	}
	return
}

// attachByUuid attaches to a container by UUID
func attachByUuid(uuid string, environ []string) error {
	args := []string{"rkt", "attach", "--mode", "stdin,stdout,stderr", uuid}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = environ
	fmt.Printf("%s\n", strings.Join(args, " "))
	return cmd.Run()
}
