package rktrunner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/droundy/goopt"
)

var ErrRktRunFailed = errors.New("rkt run failed")

type optionsT struct {
	config        *string
	exec          *string
	volumes       *[]string
	setenvs       *[]string
	printEnv      *bool
	interactive   *bool
	verbose       *bool
	dryRun        *bool
	listAlias     *bool
	noImagePrefix *bool
}

type argsT struct {
	options optionsT
	image   string
	cmdArgs []string
}

type commandT struct {
	argv0 string
	argv  []string
	envv  []string
}

type aliasT struct {
	image string
	exec  string
}

type RunnerT struct {
	config           configT
	environ          map[string]string
	environExtra     map[string]string
	alias            map[string]aliasT
	requestedVolumes map[string]bool
	fragments        fragmentsT
	args             argsT
	image            string
	exec             string
	fetchCommand     commandT
	runCommand       commandT
}

func NewRunner(configFile string) (*RunnerT, error) {
	var r RunnerT

	err := r.parseArgs()
	if err != nil {
		return nil, fmt.Errorf("bad usage: %v", err)
	}

	if *r.args.options.config != "" {
		configFile = *r.args.options.config
	}
	err = GetConfig(configFile, &r.config)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v", err)
	}

	err = r.validateRequestedVolumes()
	if err != nil {
		return nil, err
	}

	r.parseEnviron()

	err = r.registerAliases(os.Stderr, true)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v", err)
	}

	u, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	err = GetFragments(&r.config, r.templateVariables(u), &r.fragments)
	if err != nil {
		return nil, fmt.Errorf("configuration error: %v", err)
	}

	if r.attachStdio() {
		r.environExtra["RKT_EXPERIMENT_ATTACH"] = "true"
	}

	var mode string
	if *r.args.options.interactive {
		mode = InteractiveMode
	} else {
		mode = BatchMode
	}

	// different functionality depending on options, see Execute()
	switch {
	case *r.args.options.listAlias:
		// do nothing for now
	default:
		err = r.resolveImage()
		if err != nil {
			return nil, fmt.Errorf("bad usage: %v", err)
		}
		err = r.buildFetchCommand(mode)
		if err != nil {
			return nil, fmt.Errorf("bad usage: %v", err)
		}
		err = r.buildRunCommand(mode)
		if err != nil {
			return nil, fmt.Errorf("bad usage: %v", err)
		}
	}

	return &r, nil
}

// validateRequestedVolumes checks whether the user is requesting
// only what is allowed
func (r *RunnerT) validateRequestedVolumes() error {
	// check volumes passed on command line are in config file as on-request
	r.requestedVolumes = make(map[string]bool)
	for _, requested := range *r.args.options.volumes {
		valid := true
		vol, ok := r.config.Volume[requested]
		if ok {
			// We don't let user request default volumes,
			// only on-request ones.
			valid = vol.OnRequest
		} else {
			valid = false
		}
		if !valid {
			return fmt.Errorf("invalid volume: %s", requested)
		}
		r.requestedVolumes[requested] = true
	}
	return nil
}

// attachStdio returns whether we need to attach stdio,
// which we don't do in interactive mode
func (r *RunnerT) attachStdio() bool {
	return r.config.AttachStdio && !*r.args.options.interactive
}

// runSlave returns whether we need to run the slave
func (r *RunnerT) runSlave() bool {
	return r.attachStdio() || r.config.PreserveCwd || r.config.UsePath
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
	r.args.options.config = goopt.String([]string{"--config"}, "", "alternative config file, requires root or --dry-run")
	r.args.options.exec = goopt.String([]string{"-e", "--exec"}, "", "command to run instead of image default")
	r.args.options.volumes = goopt.Strings([]string{"--volume"}, "", "activate pre-defined volume")
	r.args.options.setenvs = goopt.Strings([]string{"--set-env"}, "", "environment variable")
	r.args.options.printEnv = goopt.Flag([]string{"--print-env"}, []string{}, "print environment variables passed into container", "")
	r.args.options.interactive = goopt.Flag([]string{"-i", "--interactive"}, []string{}, "run image interactively", "")
	r.args.options.verbose = goopt.Flag([]string{"-v", "--verbose"}, []string{}, "show full rkt run command", "")
	r.args.options.dryRun = goopt.Flag([]string{"--dry-run"}, []string{}, "don't execute anything", "")
	r.args.options.listAlias = goopt.Flag([]string{"-l", "--list-alias"}, []string{}, "list image aliases", "")
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

	// validate options
	if *r.args.options.config != "" && syscall.Getuid() != 0 && !*r.args.options.dryRun {
		return fmt.Errorf("alternate config file requires root or dry run")
	}

	// image
	if len(args) > 0 && args[0] != "" {
		r.args.image = args[0]
	}

	if len(args) > 1 {
		r.args.cmdArgs = args[1:]
	}

	return nil
}

func formatAlias(key string, val aliasT) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s = ", key)
	if val.exec != "" {
		fmt.Fprintf(&b, "-e %s ", val.exec)
	}
	fmt.Fprintf(&b, "%s", val.image)
	return b.String()
}

func (r *RunnerT) registerAlias(w io.Writer, warn bool, key string, val *aliasT) error {
	var err error
	dupVal, isDup := r.alias[key]
	if isDup {
		err = fmt.Errorf("duplicate alias: %s", key)
		if warn {
			fmt.Fprintf(w, "%s\n", formatAlias(key, dupVal))
			fmt.Fprintf(w, "%s\n", formatAlias(key, *val))
		}
	} else {
		r.alias[key] = *val
	}
	return err
}

func (r *RunnerT) registerAliases(w io.Writer, warn bool) error {
	var anyErr error
	r.alias = make(map[string]aliasT)
	for imageKey, imageAlias := range r.config.Alias {
		err := r.registerAlias(w, warn, imageKey, &aliasT{image: imageAlias.Image})
		if err != nil && anyErr == nil {
			anyErr = err
		}
		for _, exec := range imageAlias.Exec {
			err = r.registerAlias(w, warn, filepath.Base(exec), &aliasT{image: imageAlias.Image, exec: exec})
			if err != nil && anyErr == nil {
				anyErr = err
			}
		}
	}
	return anyErr
}

func (r *RunnerT) printAliases(w io.Writer) {
	// get keys in order
	keys := make([]string, 0, len(r.alias))
	for key := range r.alias {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s\n", formatAlias(key, r.alias[key]))
	}
}

// parseEnviron extracts all environment variables into a map
func (r *RunnerT) parseEnviron() {
	r.environ = make(map[string]string)
	r.environExtra = make(map[string]string)
	for _, keyval := range os.Environ() {
		i := strings.IndexRune(keyval, '=')
		if i != -1 {
			key := keyval[:i]
			val := keyval[i+1:]
			r.environ[key] = val
		}
	}
}

// templateVariables returns a new map, comprising the base environ,
// augmented by (most of) the user fields
func (r *RunnerT) templateVariables(u *user.User) map[string]string {
	vars := make(map[string]string)
	for k, v := range r.environ {
		vars[k] = v
	}
	vars["Uid"] = u.Uid
	vars["Gid"] = u.Gid
	vars["Username"] = u.Username
	vars["HomeDir"] = u.HomeDir
	return vars
}

// buildEnviron turns the environ maps into a list of strings
func (r *RunnerT) buildEnviron() []string {
	var result []string
	mergedEnviron := make(map[string]string)
	for key, val := range r.environ {
		mergedEnviron[key] = val
	}
	for key, val := range r.environExtra {
		mergedEnviron[key] = val
	}
	for key, val := range mergedEnviron {
		result = append(result, fmt.Sprintf("%s=%s", key, val))
	}
	return result
}

func (r *RunnerT) createEnvFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	r.fragments.printEnvironment(f)
	if *r.args.options.printEnv {
		r.fragments.printEnvironment(os.Stderr)
	}

	for _, setenv := range *r.args.options.setenvs {
		fmt.Fprintf(f, "%s\n", setenv)
		if *r.args.options.printEnv {
			fmt.Fprintf(os.Stderr, "%s\n", setenv)
		}
	}
	return nil
}

func (r *RunnerT) resolveImage() error {
	if r.args.image == "" {
		return fmt.Errorf("missing image")
	} else if r.args.image[0] == '-' {
		return fmt.Errorf("image cannot start with -")
	}

	alias, ok := r.alias[r.args.image]
	if ok {
		r.image = alias.image
		r.exec = alias.exec
	} else {
		if *r.args.options.noImagePrefix {
			r.image = r.args.image
		} else {
			r.image = r.autoPrefix(r.args.image)
		}
	}

	switch {
	case r.exec != "":
		if *r.args.options.exec != "" {
			return fmt.Errorf("cannot specify executable with alias")
		}

	case *r.args.options.exec != "":
		r.exec = *r.args.options.exec

	case *r.args.options.interactive && r.config.DefaultInteractiveCmd != "":
		r.exec = r.config.DefaultInteractiveCmd
	}
	if r.exec != "" && r.exec[0] == '-' {
		return fmt.Errorf("command cannot start with -")
	}

	return nil
}

func (r *RunnerT) formatVolumes() []string {
	volumes := r.fragments.formatVolumes(r.requestedVolumes)
	if r.runSlave() {
		volumes = append(volumes,
			"--volume", fmt.Sprintf("%s,kind=host,source=%s", slaveBinVolume, r.config.ExecSlaveDir))
		if r.attachStdio() {
			volumes = append(volumes,
				"--volume", fmt.Sprintf("%s,kind=host,source=%s", slaveRunVolume, masterRunDir()))
		}
	}
	return volumes
}

func (r *RunnerT) formatMounts() []string {
	mounts := r.fragments.formatMounts(r.requestedVolumes)
	if r.runSlave() {
		mounts = append(mounts,
			"--mount", fmt.Sprintf("volume=%s,target=%s", slaveBinVolume, slaveBinDir))
		if r.attachStdio() {
			mounts = append(mounts,
				"--mount", fmt.Sprintf("volume=%s,target=%s", slaveRunVolume, slaveRunDir))
		}
	}
	return mounts
}

func (r *RunnerT) buildFetchCommand(mode string) error {
	argv0 := r.config.Rkt
	argv := make([]string, 1)
	argv[0] = filepath.Base(argv0)
	argv = append(argv, r.fragments.Options[mode][GeneralClass]...)
	argv = append(argv, "fetch")

	argv = append(argv, r.fragments.Options[mode][FetchClass]...)

	argv = append(argv, r.image)

	r.fetchCommand.argv0 = argv0
	r.fetchCommand.argv = argv
	r.fetchCommand.envv = os.Environ()
	return nil
}

func (r *RunnerT) buildRunCommand(mode string) error {
	argv0 := r.config.Rkt
	argv := make([]string, 1)
	argv[0] = filepath.Base(argv0)
	argv = append(argv, r.fragments.formatOptions(mode, GeneralClass)...)
	argv = append(argv, "run")
	if *r.args.options.interactive {
		argv = append(argv, "--interactive")
	}

	if r.attachStdio() {
		argv = append(argv, "--uuid-file-save", uuidFilePath())
	}
	argv = append(argv, "--set-env-file", envFilePath())
	argv = append(argv, r.fragments.formatOptions(mode, RunClass)...)

	argv = append(argv, r.formatVolumes()...)
	argv = append(argv, r.image)

	if r.attachStdio() {
		argv = append(argv, "--stdin=stream", "--stdout=stream", "--stderr=stream")
	}

	argv = append(argv, r.formatMounts()...)
	argv = append(argv, r.fragments.formatOptions(mode, ImageClass)...)

	if r.runSlave() {
		argv = append(argv, "--exec", filepath.Join(slaveBinDir, slaveRunner), "--")
		if r.attachStdio() {
			argv = append(argv, "--await-file", filepath.Join(slaveRunDir, attachReadyFile))
		}
		if r.config.PreserveCwd {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			argv = append(argv, "--cwd", cwd)
		}
		if r.exec != "" {
			argv = append(argv, r.exec)
		}
	} else {
		if r.exec != "" {
			argv = append(argv, "--exec", r.exec, "--")
		}
	}

	if len(r.args.cmdArgs) > 0 {
		// check for ---
		for _, arg := range r.args.cmdArgs {
			if arg == "---" {
				return fmt.Errorf("%s invalid", arg)
			}
		}

		argv = append(argv, r.args.cmdArgs...)
	}
	r.runCommand.argv0 = argv0
	r.runCommand.argv = argv
	r.runCommand.envv = r.buildEnviron()
	return nil
}

func (r *RunnerT) printFetchCommand(w io.Writer) {
	fmt.Fprintf(w, "%s %s\n", r.fetchCommand.argv0, strings.Join(r.fetchCommand.argv[1:], " "))
}

func (r *RunnerT) printRunCommand(w io.Writer, pid int) {
	// get keys in order
	keys := make([]string, 0, len(r.environExtra))
	for key := range r.environExtra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s=%s ", key, r.environExtra[key])
	}

	fmt.Fprintf(w, "%s %s", r.runCommand.argv0, strings.Join(r.runCommand.argv[1:], " "))
	if pid > 0 {
		fmt.Fprintf(w, " (pid %d)\n", pid)
	} else {
		fmt.Fprintf(w, "\n")
	}
}

func (r *RunnerT) Execute() error {
	// different functionality depending on options, see NewRunner()
	switch {
	case *r.args.options.listAlias:
		r.printAliases(os.Stdout)

	default:
		if !*r.args.options.dryRun {
			return r.fetchAndRun()
		} else if *r.args.options.verbose {
			r.printFetchAndRun()
		}
	}
	return nil
}

// printFetchAndRun just prints the commands which would be used
func (r *RunnerT) printFetchAndRun() {
	// separate fetch is not working reliably, so hide it
	_, separateFetch := os.LookupEnv("RKTRUNNER_SEPARATE_FETCH")
	if separateFetch {
		r.printFetchCommand(os.Stderr)
	}
	r.printRunCommand(os.Stderr, 0)
}

// fetchAndRun fetches the image, and runs the command, waiting for it to complete
func (r *RunnerT) fetchAndRun() error {
	var err error

	// separate fetch is not working reliably, so hide it
	_, separateFetch := os.LookupEnv("RKTRUNNER_SEPARATE_FETCH")
	if separateFetch {
		if *r.args.options.verbose {
			r.printFetchCommand(os.Stderr)
		}

		fetchCmd := exec.Command(r.fetchCommand.argv[0], r.fetchCommand.argv[1:]...)
		fetchCmd.Path = r.fetchCommand.argv0
		fetchCmd.Env = r.fetchCommand.envv
		// for progress updates:
		fetchCmd.Stderr = os.Stderr
		err = fetchCmd.Run()
		if err != nil {
			return err
		}
	}

	// the master rundir is used for:
	// - environment file
	// - uuid file
	// - attached inotify to slave
	rundir := masterRunDir()
	err = os.MkdirAll(rundir, 0755)
	if err != nil {
		return err
	}
	defer func() {
		warn := os.Remove(rundir)
		if warn != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", warn)
		}
	}()

	envPath := envFilePath()
	err = r.createEnvFile(envPath)
	if err != nil {
		return err
	}
	defer os.Remove(envPath)

	attachReadyPath := filepath.Join(rundir, attachReadyFile)
	var attach *Attacher
	if r.attachStdio() {
		uuidPath := uuidFilePath()
		defer os.Remove(uuidPath)
		attach, err = NewAttacher(uuidPath, attachReadyPath, r.runCommand.envv, *r.args.options.verbose)
		if err != nil {
			return err
		}
		go attach.Attach()
		defer os.Remove(attachReadyPath)
	}

	runCmd := exec.Command(r.runCommand.argv[0], r.runCommand.argv[1:]...)
	runCmd.Path = r.runCommand.argv0
	runCmd.Env = r.runCommand.envv
	if !r.attachStdio() {
		runCmd.Stdin = os.Stdin
		runCmd.Stdout = os.Stdout
	}
	runCmd.Stderr = os.Stderr
	err = runCmd.Start()
	if err == nil {
		if *r.args.options.verbose {
			r.printRunCommand(os.Stderr, runCmd.Process.Pid)
		}

		err = runCmd.Wait()

		// ensure we don't print an error message if rkt run already did
		if err != nil {
			_, isExitErr := err.(*exec.ExitError)
			if isExitErr {
				err = ErrRktRunFailed
			}
		}
	} else {
		if *r.args.options.verbose {
			r.printRunCommand(os.Stderr, 0)
		}
	}

	if r.attachStdio() {
		if err != nil {
			attach.Abort()
		}

		// report any error from attach, then discard it
		warn := attach.Wait()
		if warn != nil {
			fmt.Fprintf(os.Stderr, "%v\n", warn)
		}
	}

	return err
}
