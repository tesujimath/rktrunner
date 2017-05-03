package rktrunner

import (
	"fmt"
	"github.com/rjeczalik/notify"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Attacher struct {
	uuidPath      string
	uuidDirEvents chan notify.EventInfo
	donePath      string
	environ       []string
	verbose       bool
	abort         chan bool
	errs          chan error
}

// NewAttacher creates an Attacher, after which if done with no error,
// the caller *must* call Wait().
func NewAttacher(uuidPath, donePath string, environ []string, verbose bool) (*Attacher, error) {
	uuidDirEvents := make(chan notify.EventInfo, 2)
	err := notify.Watch(filepath.Dir(uuidPath), uuidDirEvents, notify.InCloseWrite)
	if err != nil {
		return nil, err
	}

	return &Attacher{
		uuidPath:      uuidPath,
		uuidDirEvents: uuidDirEvents,
		donePath:      donePath,
		environ:       environ,
		verbose:       verbose,
		abort:         make(chan bool),
		errs:          make(chan error),
	}, nil
}

func attacherWarn(err error) {
	fmt.Fprintf(os.Stderr, "rkt-run: warning: attach failure %v\n", err)
}

func (a *Attacher) Abort() {
	close(a.abort)
}

func (a *Attacher) Wait() error {
	notify.Stop(a.uuidDirEvents)
	err := <-a.errs
	return err
}

func (a *Attacher) Attach() {
	var uuid string
	var err error
	attachAttempted := false
loop:
	for uuid == "" && err == nil {
		// wait for the uuid file, or an abort event
		select {
		case _, ok := <-a.abort:
			if !ok {
				break loop
			}
		case ei := <-a.uuidDirEvents:
			switch ei.Event() {
			case notify.InCloseWrite:
				var bytes []byte
				bytes, err = ioutil.ReadFile(a.uuidPath)
				if err != nil {
					break loop
				}
				uuid = string(bytes)
			}
		}
	}

	if uuid != "" {
		err = a.rktStatusWaitReady(uuid)
		if err == nil {
			go a.rktAttach(uuid)
			attachAttempted = true

			// Give the asynchronous rkt attach a chance to do its thing.
			// This is rather unsatisfactory.
			time.Sleep(time.Duration(1000 * time.Millisecond))
		}
	}

	if attachAttempted {
		// notify slave that attachment is ready
		var f io.ReadCloser
		f, err = os.Create(a.donePath)
		if err != nil {
			attacherWarn(err)
			return
		}
		f.Close()
	} else {
		if err != nil {
			a.errs <- err
		}
		close(a.errs)
	}
}

// rktStatusWaitReady waits for the container to be ready.
// Any error is returned on the errs channel.
func (a *Attacher) rktStatusWaitReady(uuid string) error {
	args := []string{"rkt", "status", "--wait-ready", uuid}
	cmd := exec.Command(args[0], args[1:]...)
	if a.verbose {
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(args, " "))
	}
	return cmd.Run()
}

// rktAttach attaches to a container by UUID.
// Any error is returned on the errs channel.
func (a *Attacher) rktAttach(uuid string) {
	args := []string{"rkt", "attach", "--mode", "stdin,stdout,stderr", uuid}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = a.environ
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err == nil {
		if a.verbose {
			fmt.Fprintf(os.Stderr, "%s (pid %d)\n", strings.Join(args, " "), cmd.Process.Pid)
		}

		err = cmd.Wait()
	}
	if err != nil {
		a.errs <- err
	}
	close(a.errs)
}
