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
	donePath        string
	environ         []string
	abort           chan bool
	rktAttachStatus chan error
}

func NewAttacher(donePath string, environ []string) *Attacher {
	return &Attacher{
		donePath:        donePath,
		environ:         environ,
		abort:           make(chan bool),
		rktAttachStatus: make(chan error),
	}
}

func attacherWarn(err error) {
	fmt.Fprintf(os.Stderr, "rkt-run: warning: attach failure %v\n", err)
}

func (a *Attacher) ByName(appName string) {
	go a.run(appName)
}

func (a *Attacher) Abort() {
	close(a.abort)
}

func (a *Attacher) Wait() error {
	err := <-a.rktAttachStatus
	return err
}

func (a *Attacher) run(appName string) {
	var uuid string
	var err error
loop:
	for uuid == "" && err == nil {
		uuid, err = findUuid(appName)
		if err != nil {
			attacherWarn(err)
		}

		if uuid == "" {
			// wait for a while
			timeout := time.After(time.Duration(time.Second))
			select {
			case _, ok := <-a.abort:
				if !ok {
					attacherWarn(fmt.Errorf("abort"))
					break loop
				}
			case <-timeout:
				// go around again
			}
		}
	}

	if uuid != "" {
		go a.attachByUuid(uuid)
	}

	// Give the asynchronous rkt attach a chance to do its thing.
	// This is rather unsatisfactory.
	time.Sleep(time.Duration(1000 * time.Millisecond))

	// notify slave that attachment is ready
	f, err := os.Create(a.donePath)
	if err != nil {
		attacherWarn(err)
		return
	}
	f.Close()
}

// findUuid returns the uuid for the named container,
// or an empty string if it isn't found
func findUuid(appName string) (uuid string, err error) {
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
		if len(fields) >= 2 && fields[1] == appName {
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

// attachByUuid attaches to a container by UUID.
// Any error is just printed, as this must be run asynchronously.
func (a *Attacher) attachByUuid(uuid string) {
	args := []string{"rkt", "attach", "--mode", "stdin,stdout,stderr", uuid}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = a.environ
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		a.rktAttachStatus <- err
	}
	close(a.rktAttachStatus)
}
