package main

import (
	"flag"
	"fmt"
	"github.com/tesujimath/rktrunner"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

type optionsT struct {
	interactive bool
	verbose     bool
}

type runT struct {
	options   optionsT
	container string
	cmd       string
	cmdArgs   []string
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func parseArgs(c *rktrunner.ConfigT) (r runT, err error) {
	flag.BoolVar(&r.options.interactive, "interactive", false, "run image interactively")
	flag.BoolVar(&r.options.verbose, "verbose", false, "show full rkt run command")
	flag.Parse()
	args := flag.Args()

	// container
	if len(args) == 0 || args[0] == "" {
		err = fmt.Errorf("missing container")
	}
	if args[0][0] == '-' {
		err = fmt.Errorf("container cannot start with -")
	}
	r.container = flag.Args()[0]

	// commands, check for ---
	for _, arg := range args[1:] {
		if arg == "---" {
			err = fmt.Errorf("%s invalid", arg)
			return
		}
	}
	if len(args) > 1 {
		r.cmd = args[1]

		if len(args) > 2 {
			r.cmdArgs = args[2:]
		}
	} else if r.options.interactive && c.DefaultInteractiveCmd != "" {
		r.cmd = c.DefaultInteractiveCmd
	}

	return
}

func formatVolumes(f *rktrunner.FragmentsT) []string {
	var s []string
	for key, vol := range f.Volume {
		s = append(s, "--volume", fmt.Sprintf("%s,kind=host,source=%s", key, vol[rktrunner.VolumeHost]))
	}
	return s
}

func formatMounts(f *rktrunner.FragmentsT) []string {
	var s []string
	for key, vol := range f.Volume {
		s = append(s, "--mount", fmt.Sprintf("volume=%s,target=%s", key, vol[rktrunner.VolumeTarget]))
	}
	return s
}

func formatUserMapping(u *user.User) []string {
	var s []string
	s = append(s, fmt.Sprintf("--user=%s", u.Uid))
	s = append(s, fmt.Sprintf("--user=%s", u.Uid))
	return s
}

func execute(c *rktrunner.ConfigT, u *user.User, f *rktrunner.FragmentsT, r *runT) error {
	//rkt --insecure-options=image run --set-env=HOME=/home/guestsi --volume home,kind=host,source=/home/guestsi --volume data,kind=host,source=/data docker://quay.io/biocontainers/blast:2.6.0--boost1.61_0 --mount volume=home,target=/home/guestsi --mount volume=data,target=/hostdata --user=511 --group=511 --exec ~/scripts/myblast -- /hostdata/myfile
	args := make([]string, 1)
	args[0] = filepath.Base(c.Rkt)
	args = append(args, f.Options[rktrunner.GeneralOptions]...)
	args = append(args, "run")
	if r.options.interactive {
		args = append(args, "--interactive")
	}
	args = append(args, f.Options[rktrunner.RunOptions]...)
	args = append(args, formatVolumes(f)...)
	args = append(args, r.container)
	args = append(args, formatMounts(f)...)
	args = append(args, formatUserMapping(u)...)
	if r.cmd != "" {
		args = append(args, "--exec", r.cmd)
		if len(r.cmdArgs) > 0 {
			args = append(args, "--")
			args = append(args, r.cmdArgs...)
		}
	}
	if r.options.verbose {
		fmt.Printf("%s %s\n", c.Rkt, strings.Join(args[1:], " "))
	}
	return syscall.Exec(c.Rkt, args, os.Environ())
}

func main() {
	c, err := rktrunner.GetConfig()
	if err != nil {
		die("failed on config: %v", err)
	}

	r, err := parseArgs(c)
	if err != nil {
		die("bad usage: %v", err)
	}

	u, err := user.Current()
	if err != nil {
		die("failed to get current user: %v", err)
	}

	f, err := rktrunner.GetFragments(c, u)
	if err != nil {
		die("failed to get fragments: %v", err)
	}

	err = execute(c, u, f, &r)
	if err != nil {
		die("failed: %v", err)
	}
}
