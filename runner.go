package rktrunner

import (
	"bytes"
	"fmt"
	"github.com/droundy/goopt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
)

type optionsT struct {
	config        *string
	exec          *string
	setenvs       *[]string
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
	config       configT
	environ      map[string]string
	environExtra map[string]string
	alias        map[string]aliasT
	fragments    fragmentsT
	args         argsT
	appName      string
	command      commandT
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

	// different functionality depending on options, see Execute()
	switch {
	case *r.args.options.listAlias:
		// do nothing for now
	default:
		err = r.buildExec()
		if err != nil {
			return nil, fmt.Errorf("bad usage: %v", err)
		}
	}

	return &r, nil
}

// attachStdio returns whether we need to attach stdio,
// which we don't do in interactive mode
func (r *RunnerT) attachStdio() bool {
	return r.config.AttachStdio && !*r.args.options.interactive
}

// runSlave returns whether we need to run the slave
func (r *RunnerT) runSlave() bool {
	return r.attachStdio() || r.config.PreserveCwd
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
	r.args.options.config = goopt.String([]string{"--config"}, "", "alternative config file, requires --dry-run")
	r.args.options.exec = goopt.String([]string{"-e", "--exec"}, "", "command to run instead of image default")
	r.args.options.setenvs = goopt.Strings([]string{"--set-env"}, "", "environment variable")
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
	if *r.args.options.config != "" && !*r.args.options.dryRun {
		return fmt.Errorf("alternate config file requires dry run")
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
			err = r.registerAlias(w, warn, exec, &aliasT{image: imageAlias.Image, exec: exec})
			if err != nil && anyErr == nil {
				anyErr = err
			}
		}
	}
	return anyErr
}

func (r *RunnerT) printAliases(w io.Writer) {
	// get keys in order
	var keys []string
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

func (r *RunnerT) resolveCommand() aliasT {
	cmd, ok := r.alias[r.args.image]
	if !ok {
		if *r.args.options.noImagePrefix {
			cmd.image = r.args.image
		} else {
			cmd.image = r.autoPrefix(r.args.image)
		}
	}
	return cmd
}

func (r *RunnerT) formatVolumes() []string {
	volumes := r.fragments.formatVolumes()
	if r.runSlave() {
		volumes = append(volumes,
			"--volume", fmt.Sprintf("%s,kind=host,source=%s", slaveRunVolume, masterRunDir()),
			"--volume", fmt.Sprintf("%s,kind=host,source=%s", slaveBinVolume, r.config.ExecSlaveDir))
	}
	return volumes
}

func (r *RunnerT) formatMounts() []string {
	mounts := r.fragments.formatMounts()
	if r.runSlave() {
		mounts = append(mounts,
			"--mount", fmt.Sprintf("volume=%s,target=%s", slaveRunVolume, slaveRunDir),
			"--mount", fmt.Sprintf("volume=%s,target=%s", slaveBinVolume, slaveBinDir))
	}
	return mounts
}

func generateUniqueAppName(image string) string {
	var start int
	var basename string
	lastSlash := strings.LastIndex(image, "/")
	if lastSlash >= 0 {
		start = lastSlash + 1
	}
	firstColon := strings.Index(image[start:], ":")
	if firstColon >= 0 {
		basename = image[start : firstColon+start]
	} else {
		basename = image[start:]
	}
	return fmt.Sprintf("%s-%d", basename, os.Getpid())
}

func (r *RunnerT) buildExec() error {
	argv0 := r.config.Rkt
	argv := make([]string, 1)
	argv[0] = filepath.Base(argv0)
	argv = append(argv, r.fragments.Options[GeneralOptions]...)
	argv = append(argv, "run")
	if *r.args.options.interactive {
		argv = append(argv, "--interactive")
	}

	argv = append(argv, r.fragments.Options[RunOptions]...)

	for _, setenv := range *r.args.options.setenvs {
		argv = append(argv, fmt.Sprintf("--set-env=%s", setenv))
	}

	argv = append(argv, r.formatVolumes()...)

	if r.args.image == "" {
		return fmt.Errorf("missing image")
	} else if r.args.image[0] == '-' {
		return fmt.Errorf("image cannot start with -")
	}
	cmd := r.resolveCommand()
	argv = append(argv, cmd.image)

	r.appName = generateUniqueAppName(cmd.image)
	argv = append(argv, "--name", r.appName)

	if r.attachStdio() {
		argv = append(argv, "--stdin=stream", "--stdout=stream", "--stderr=stream")
	}

	argv = append(argv, r.formatMounts()...)
	argv = append(argv, r.fragments.Options[ImageOptions]...)

	switch {
	case cmd.exec != "":
		if *r.args.options.exec != "" {
			return fmt.Errorf("cannot specify executable with alias")
		}

	case *r.args.options.exec != "":
		cmd.exec = *r.args.options.exec

	case *r.args.options.interactive && r.config.DefaultInteractiveCmd != "":
		cmd.exec = r.config.DefaultInteractiveCmd
	}
	if cmd.exec != "" && cmd.exec[0] == '-' {
		return fmt.Errorf("command cannot start with -")
	}

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
		if cmd.exec != "" {
			argv = append(argv, cmd.exec)
		}
	} else {
		if cmd.exec != "" {
			argv = append(argv, "--exec", cmd.exec, "--")
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
	r.command.argv0 = argv0
	r.command.argv = argv
	r.command.envv = r.buildEnviron()
	return nil
}

func (r *RunnerT) printExec(w io.Writer) {
	// get keys in order
	var keys []string
	for key := range r.environExtra {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(w, "%s=%s ", key, r.environExtra[key])
	}

	fmt.Fprintf(w, "%s %s\n", r.command.argv0, strings.Join(r.command.argv[1:], " "))
}

func (r *RunnerT) Execute() error {
	// different functionality depending on options, see NewRunner()
	switch {
	case *r.args.options.listAlias:
		r.printAliases(os.Stdout)

	default:
		if *r.args.options.verbose {
			r.printExec(os.Stdout)
		}

		if !*r.args.options.dryRun {
			return r.exec()
		}
	}
	return nil
}

// exec runs the command, waiting for it to complete
func (r *RunnerT) exec() error {
	rundir := masterRunDir()
	attachReadyPath := filepath.Join(rundir, attachReadyFile)
	if r.runSlave() {
		err := os.Mkdir(rundir, os.ModeDir|0755)
		if err != nil {
			return err
		}
	}

	var attach *Attacher
	if r.attachStdio() {
		attach = NewAttacher(attachReadyPath, r.command.envv)
		attach.ByName(r.appName)
	}

	cmd := exec.Command(r.command.argv[0], r.command.argv[1:]...)
	cmd.Path = r.command.argv0
	cmd.Env = r.command.envv
	// We mustn't pass in stdin, as that would compete with
	// our rkt attach.
	// We choose not to pass in stdout, to avoid possible pollution
	// of application output with rkt wrapper stuff.
	// We pass in stderr, so that rkt errors are seen by the user.
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if r.attachStdio() {
		if err != nil {
			attach.Abort()
		} else {
			// ignore errors on cleanup
			warn := attach.Wait()
			if warn != nil {
				fmt.Fprintf(os.Stderr, "%v\n", warn)
			}
		}
	}

	if r.runSlave() {
		// ignore errors on cleanup
		os.Remove(attachReadyPath)
		warn := os.Remove(rundir)
		if warn != nil {
			fmt.Fprintf(os.Stderr, "%v\n", warn)
		}
	}

	if err != nil {
		_, isExitErr := err.(*exec.ExitError)
		if isExitErr {
			// don't report error to caller, as rkt run has already told user
			// what's wrong
			err = nil
		}
	}

	return err
}
