package main

import (
	"fmt"
	"github.com/droundy/goopt"
	"github.com/tesujimath/rktrunner"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

type optionsT struct {
	exec          *string
	interactive   *bool
	verbose       *bool
	noImagePrefix *bool
}

type runT struct {
	options optionsT
	image   string
	cmd     string
	cmdArgs []string
}

func die(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "rkt-run: %s\n", fmt.Sprintf(format, args))
	os.Exit(1)
}

func autoPrefix(image string, c *rktrunner.ConfigT) string {
	for key, val := range c.AutoImagePrefix {
		if strings.HasPrefix(image, key) {
			return strings.Replace(image, key, val, 1)
		}
	}
	return image
}

func parseArgs(c *rktrunner.ConfigT) (r runT, err error) {
	r.options.exec = goopt.String([]string{"-e", "--exec"}, "", "command to run instead of image default")
	r.options.interactive = goopt.Flag([]string{"-i", "--interactive"}, []string{}, "run image interactively", "")
	r.options.verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{}, "show full rkt run command", "")
	r.options.noImagePrefix = goopt.Flag([]string{"-n", "--no-image-prefix"}, []string{}, "disable auto image prefix", "")
	goopt.RequireOrder = true
	goopt.Author = "Simon Guest <simon.guest@tesujimath.org>"
	goopt.Description = func() string {
		return `Run rkt containers with user mapping, and volume mounting
as defined by the system administrator.

$ rkt-run <options> <image> [<args>]
`
	}
	goopt.Summary = "Enable unprivileged users to run containers using rkt"
	goopt.Suite = "rktrunner"
	goopt.Parse(nil)
	args := goopt.Args

	// image
	if len(args) == 0 || args[0] == "" {
		err = fmt.Errorf("missing image")
		return
	}
	if args[0][0] == '-' {
		err = fmt.Errorf("image cannot start with -")
		return
	}
	if !*r.options.noImagePrefix {
		r.image = autoPrefix(args[0], c)
	} else {
		r.image = args[0]
	}

	// command
	if *r.options.exec != "" {
		if (*r.options.exec)[0] == '-' {
			err = fmt.Errorf("command cannot start with -")
			return
		}
		r.cmd = *r.options.exec
	} else if *r.options.interactive && c.DefaultInteractiveCmd != "" {
		r.cmd = c.DefaultInteractiveCmd
	}

	// args, check for ---
	for _, arg := range args[1:] {
		if arg == "---" {
			err = fmt.Errorf("%s invalid", arg)
			return
		}
	}
	if len(args) > 1 {
		r.cmdArgs = args[1:]
	}

	return
}

func formatVolumes(f *rktrunner.FragmentsT) []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Volume != "" {
			s = append(s, "--volume", fmt.Sprintf("%s,%s", key, vol.Volume))
		}
	}
	return s
}

func formatMounts(f *rktrunner.FragmentsT) []string {
	var s []string
	for key, vol := range f.Volume {
		if vol.Mount != "" {
			s = append(s, "--mount", fmt.Sprintf("volume=%s,%s", key, vol.Mount))
		}
	}
	return s
}

func printEnviron(environ map[string]string) {
	// get keys in order
	var keys []string
	for key := range environ {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("%s=%s ", key, environ[key])
	}
}

// augmentEnviron overrides and/or augments the base environment with the extra
func augmentEnviron(base []string, extra map[string]string) []string {
	environ := make(map[string]string)
	for _, keyval := range base {
		i := strings.IndexRune(keyval, '=')
		if i != -1 {
			key := keyval[:i]
			val := keyval[i+1:]
			environ[key] = val
		}
	}
	for key, val := range extra {
		environ[key] = val
	}
	var result []string
	for key, val := range environ {
		result = append(result, fmt.Sprintf("%s=%s", key, val))
	}
	return result
}

func execute(c *rktrunner.ConfigT, f *rktrunner.FragmentsT, r *runT) error {
	//rkt --insecure-options=image run --set-env=HOME=/home/guestsi --volume home,kind=host,source=/home/guestsi --volume data,kind=host,source=/data docker://quay.io/biocontainers/blast:2.6.0--boost1.61_0 --mount volume=home,target=/home/guestsi --mount volume=data,target=/hostdata --user=511 --group=511 --exec ~/scripts/myblast -- /hostdata/myfile
	args := make([]string, 1)
	args[0] = filepath.Base(c.Rkt)
	args = append(args, f.Options[rktrunner.GeneralOptions]...)
	args = append(args, "run")
	if *r.options.interactive {
		args = append(args, "--interactive")
	}
	args = append(args, f.Options[rktrunner.RunOptions]...)
	args = append(args, formatVolumes(f)...)
	args = append(args, r.image)
	args = append(args, formatMounts(f)...)
	args = append(args, f.Options[rktrunner.ImageOptions]...)
	if r.cmd != "" {
		args = append(args, "--exec", r.cmd)
	}
	if len(r.cmdArgs) > 0 {
		args = append(args, "--")
		args = append(args, r.cmdArgs...)
	}
	if *r.options.verbose {
		printEnviron(f.Environment)
		fmt.Printf("%s %s\n", c.Rkt, strings.Join(args[1:], " "))
	}

	environ := augmentEnviron(os.Environ(), f.Environment)
	return syscall.Exec(c.Rkt, args, environ)
}

func main() {
	c, err := rktrunner.GetConfig()
	if err != nil {
		die("configuration error: %v", err)
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
		die("configuration error: %v", err)
	}

	// set real uid same as effective
	err = syscall.Setreuid(syscall.Geteuid(), syscall.Geteuid())
	if err != nil {
		die("failed to set real uid: %v", err)
	}

	err = execute(c, f, &r)
	if err != nil {
		die("failed: %v", err)
	}
}
