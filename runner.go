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

var ErrNotRoot = errors.New("must run as root")
var ErrRktRunFailed = errors.New("rkt run failed")
var ErrRktEnterFailed = errors.New("rkt enter failed")

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

type aliasT struct {
	image string
	exec  string
}

type RunnerT struct {
	config           configT
	environ          map[string]string
	alias            map[string]aliasT
	requestedVolumes map[string]bool
	fragments        fragmentsT
	args             argsT
	image            string
	exec             string
	fetchCommand     *CommandT
	runCommand       *CommandT
	enterCommand     *CommandT
	worker           *Worker
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
		err = r.validateCmdArgs()
		if err == nil {
			err = r.resolveImage()
		}
		if err == nil && r.config.WorkerPods {
			r.worker, err = NewWorker(u, r.image)
		}
		// separate fetch is not working reliably, so hide it
		_, separateFetch := os.LookupEnv("RKTRUNNER_SEPARATE_FETCH")
		if err == nil && separateFetch {
			err = r.buildFetchCommand(mode)
		}
		if err == nil {
			if r.worker == nil || !r.worker.FoundPod() {
				err = r.buildRunCommand(mode)
			} else {
				// reuse worker pod we found
				r.buildEnterCommand()
			}
		}
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

// runSlave returns whether we need to run the slave
func (r *RunnerT) runSlave() bool {
	return r.config.PreserveCwd || r.config.UsePath || r.config.WorkerPods
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
	}
	return volumes
}

func (r *RunnerT) formatMounts() []string {
	mounts := r.fragments.formatMounts(r.requestedVolumes)
	if r.runSlave() {
		mounts = append(mounts,
			"--mount", fmt.Sprintf("volume=%s,target=%s", slaveBinVolume, slaveBinDir))
	}
	return mounts
}

func (r *RunnerT) buildFetchCommand(mode string) error {
	r.fetchCommand = NewCommand(r.config.Rkt)
	r.fetchCommand.AppendArgs(r.fragments.Options[mode][GeneralClass]...)
	r.fetchCommand.AppendArgs("fetch")
	r.fetchCommand.AppendArgs(r.fragments.Options[mode][FetchClass]...)
	r.fetchCommand.AppendArgs(r.image)
	r.fetchCommand.SetEnviron(os.Environ())
	return nil
}

func (r *RunnerT) validateCmdArgs() error {
	// check for ---
	for _, arg := range r.args.cmdArgs {
		if arg == "---" {
			return fmt.Errorf("%s invalid", arg)
		}
	}
	return nil
}

func (r *RunnerT) buildRunCommand(mode string) error {
	r.runCommand = NewCommand(r.config.Rkt)
	r.runCommand.AppendArgs(r.fragments.formatOptions(mode, GeneralClass)...)
	r.runCommand.AppendArgs("run")

	r.runCommand.AppendArgs("--uuid-file-save", uuidFilePath())
	r.runCommand.AppendArgs("--set-env-file", envFilePath())
	r.runCommand.AppendArgs(r.fragments.formatOptions(mode, RunClass)...)

	r.runCommand.AppendArgs(r.formatVolumes()...)
	r.runCommand.AppendArgs(r.image)

	if r.worker != nil {
		r.runCommand.AppendArgs("--name", r.worker.AppName)
	}

	r.runCommand.AppendArgs(r.formatMounts()...)
	r.runCommand.AppendArgs(r.fragments.formatOptions(mode, ImageClass)...)

	if r.runSlave() {
		r.runCommand.AppendArgs("--exec", filepath.Join(slaveBinDir, slaveRunner), "--")
		if r.config.PreserveCwd {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			r.runCommand.AppendArgs("--cwd", cwd)
		}
		if r.worker != nil {
			r.runCommand.AppendArgs("--wait")
		} else {
			if r.exec != "" {
				r.runCommand.AppendArgs(r.exec)
			}
		}
	} else {
		if r.exec != "" {
			r.runCommand.AppendArgs("--exec", r.exec, "--")
		}
	}

	if r.worker == nil && len(r.args.cmdArgs) > 0 {
		r.runCommand.AppendArgs(r.args.cmdArgs...)
	}

	r.runCommand.SetEnviron(r.buildEnviron())
	return nil
}

func (r *RunnerT) buildEnterCommand() error {
	r.enterCommand = NewCommand(r.config.Rkt)
	r.enterCommand.AppendArgs("enter")
	if r.worker.FoundPod() {
		r.enterCommand.AppendArgs(r.worker.UUID)
	} else {
		// placeholder, just for verbose output
		r.enterCommand.AppendArgs("$uuid")
	}

	if r.runSlave() {
		r.enterCommand.AppendArgs(filepath.Join(slaveBinDir, slaveRunner))
		if r.config.PreserveCwd {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			r.enterCommand.AppendArgs("--cwd", cwd)
		}
	}

	if r.exec != "" {
		r.enterCommand.AppendArgs(r.exec)
	}

	if len(r.args.cmdArgs) > 0 {
		r.enterCommand.AppendArgs(r.args.cmdArgs...)
	}

	return nil
}

func (r *RunnerT) Execute() error {
	// different functionality depending on options, see NewRunner()
	switch {
	case *r.args.options.listAlias:
		r.printAliases(os.Stdout)

	default:
		if !*r.args.options.dryRun {
			if syscall.Getuid() != 0 || syscall.Geteuid() != 0 {
				return ErrNotRoot
			}

			if r.runCommand != nil {
				err := r.fetchAndRun()
				if err != nil {
					return err
				}
			}
			if r.worker != nil {
				err := r.buildEnterCommand()
				if err != nil {
					return err
				}

				err = r.enter()
				if err != nil {
					return err
				}
			}
		} else if *r.args.options.verbose {
			r.printFetchAndRun()
			if r.worker != nil {
				err := r.buildEnterCommand()
				if err != nil {
					return err
				}
				r.enterCommand.Print(os.Stderr)
			}
		}
	}
	return nil
}

// printFetchAndRun just prints the commands which would be used
func (r *RunnerT) printFetchAndRun() {
	if r.fetchCommand != nil {
		r.fetchCommand.Print(os.Stderr)
	}
	if r.runCommand != nil {
		r.runCommand.Print(os.Stderr)
	}
}

// fetchAndRun fetches the image, and runs the command.
// If it is a worker pod, the run is done in the background,
// otherwise we wait for it to complete.
func (r *RunnerT) fetchAndRun() error {
	var err error

	if r.fetchCommand != nil {
		if *r.args.options.verbose {
			r.fetchCommand.Print(os.Stderr)
		}

		err = r.fetchCommand.Run()
		if err != nil {
			return err
		}
	}

	// the master rundir is used for:
	// - environment file
	// - uuid file
	err = os.MkdirAll(masterRunDir(), 0755)
	if err != nil {
		return err
	}
	defer r.RemoveTempFiles()

	envPath := envFilePath()
	err = r.createEnvFile(envPath)
	if err != nil {
		return err
	}

	err = r.runCommand.Start()
	if err == nil {
		if *r.args.options.verbose {
			r.runCommand.Print(os.Stderr)
		}

		if r.worker != nil {
			err = r.worker.InitializePod(uuidFilePath(), NewWaiter(r.runCommand))
		} else {
			// don't care about the UUID, just wait for the pod to exit
			err = r.runCommand.Wait()
		}

		// ensure we don't print an error message if rkt run already did
		if err != nil {
			_, isExitErr := err.(*exec.ExitError)
			if isExitErr {
				err = ErrRktRunFailed
			}
		}
	} else {
		if *r.args.options.verbose {
			r.runCommand.Print(os.Stderr)
		}
	}

	return err
}

func (r *RunnerT) RemoveTempFiles() {
	os.Remove(uuidFilePath())
	WarnOnFailure(os.Remove(envFilePath()))
	WarnOnFailure(os.Remove(masterRunDir()))
}

// enter enters the pod.  In the case of not having also started the pod,
// if successful, it does not return.
func (r *RunnerT) enter() error {
	if *r.args.options.verbose {
		r.enterCommand.Print(os.Stderr)
	}
	r.enterCommand.PreserveFile(r.worker.Podlock)
	// if we also started a pod, then simply run the enter command
	if r.runCommand != nil {
		// need to stay for the cleanup
		return r.enterCommand.Run()
	} else {
		return r.enterCommand.Exec()
	}
}
