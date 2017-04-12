package rktrunner

import (
	"fmt"
	"github.com/droundy/goopt"
	"io"
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

type argsT struct {
	options optionsT
	image   string
	cmd     string
	cmdArgs []string
}

type execT struct {
	argv0 string
	argv  []string
	envv  []string
}

type RunnerT struct {
	config    configT
	fragments fragmentsT
	args      argsT
	exec      execT
}

func NewRunner() (*RunnerT, error) {
	var r RunnerT

	err := GetConfig(&r.config)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v", err)
	}

	err = r.parseArgs()
	if err != nil {
		return nil, fmt.Errorf("bad usage: %v", err)
	}

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	err = GetFragments(&r.config, u, &r.fragments)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v", err)
	}

	r.buildExec()

	return &r, nil
}

func (r *RunnerT) autoPrefix(image string) string {
	for key, val := range r.config.AutoImagePrefix {
		if strings.HasPrefix(image, key) {
			return strings.Replace(image, key, val, 1)
		}
	}
	return image
}

func (r *RunnerT) parseArgs() error {
	r.args.options.exec = goopt.String([]string{"-e", "--exec"}, "", "command to run instead of image default")
	r.args.options.interactive = goopt.Flag([]string{"-i", "--interactive"}, []string{}, "run image interactively", "")
	r.args.options.verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{}, "show full rkt run command", "")
	r.args.options.noImagePrefix = goopt.Flag([]string{"-n", "--no-image-prefix"}, []string{}, "disable auto image prefix", "")
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
		return fmt.Errorf("missing image")
	}
	if args[0][0] == '-' {
		return fmt.Errorf("image cannot start with -")

	}
	if !*r.args.options.noImagePrefix {
		r.args.image = r.autoPrefix(args[0])
	} else {
		r.args.image = args[0]
	}

	// command
	if *r.args.options.exec != "" {
		if (*r.args.options.exec)[0] == '-' {
			return fmt.Errorf("command cannot start with -")

		}
		r.args.cmd = *r.args.options.exec
	} else if *r.args.options.interactive && r.config.DefaultInteractiveCmd != "" {
		r.args.cmd = r.config.DefaultInteractiveCmd
	}

	// args, check for ---
	for _, arg := range args[1:] {
		if arg == "---" {
			return fmt.Errorf("%s invalid", arg)
		}
	}
	if len(args) > 1 {
		r.args.cmdArgs = args[1:]
	}

	return nil
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

func (r *RunnerT) buildExec() {
	argv0 := r.config.Rkt
	argv := make([]string, 1)
	argv[0] = filepath.Base(argv0)
	argv = append(argv, r.fragments.Options[GeneralOptions]...)
	argv = append(argv, "run")
	if *r.args.options.interactive {
		argv = append(argv, "--interactive")
	}
	argv = append(argv, r.fragments.Options[RunOptions]...)
	argv = append(argv, r.fragments.formatVolumes()...)
	argv = append(argv, r.args.image)
	argv = append(argv, r.fragments.formatMounts()...)
	argv = append(argv, r.fragments.Options[ImageOptions]...)
	if r.args.cmd != "" {
		argv = append(argv, "--exec", r.args.cmd)
	}
	if len(r.args.cmdArgs) > 0 {
		argv = append(argv, "--")
		argv = append(argv, r.args.cmdArgs...)
	}
	r.exec.argv0 = argv0
	r.exec.argv = argv
	r.exec.envv = augmentEnviron(os.Environ(), r.fragments.Environment)
	if *r.args.options.verbose {
		r.printExec(os.Stdout)
	}
}

func (r *RunnerT) printExec(w io.Writer) {
	environ := r.fragments.Environment
	// get keys in order
	var keys []string
	for key := range environ {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s=%s ", key, environ[key])
	}

	fmt.Fprintf(w, "%s %s\n", r.exec.argv0, strings.Join(r.exec.argv[1:], " "))
}

func (r *RunnerT) Execute() error {
	return syscall.Exec(r.exec.argv0, r.exec.argv, r.exec.envv)
}
