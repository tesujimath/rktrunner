# rkt-run

This package provides the [rkt-run](doc/rkt-run.md) command, which
is intended to be installed setuid root, to enabled unprivileged users
to run containers using `rkt`, in a controlled fashion.

There are also `rkt-run-helper` and `rkt-run-slave` commands - see below.

`rkt-run` provides the following features:

* enable unprivileged users to run rkt

* enable concurrent use of pods

* preservation of working directory of host within container

The system-wide configuration enables the system administrator to
control the following aspects of the `rkt run` command line:

* aliases for images and their executables

* volumes to be mounted

* automatic prefix re-writing of image names

* general, run, and image options

## Basic Usage

All `rkt run` options are controlled by the config file,
[/etc/rktrunner.toml](doc/rktrunner.toml.md), which should be carefully setup
by the local sysadmin.

Example use:
```
$ rkt-run -i -v qiime_
```

The `-v` option prints the full `rkt run` command which is
being run, as follows:
```
# /usr/bin/rkt --insecure-options=image run --interactive --net=host --set-env=HOME=/home/guestsi --volume volume-config,kind=empty,uid=511,gid=511 --volume volume-data,kind=empty,uid=511,gid=511 --volume home,kind=host,source=/home/guestsi quay.io/biocontainers/qiime:1.9.1--py27_0 --mount volume=home,target=/home/guestsi --user=511 --group=511 --exec sh
```

Note that the options are taken from the [config file](doc/rktrunner.toml.md), which in this case looks like this:
```
rkt = "/usr/bin/rkt"
default-interactive-cmd = "sh"

[options]
general = ["--insecure-options=image"]
run = ["--net=host", "--set-env=HOME=/home/{{.Username}}"]
image = ["--user={{.Uid}}", "--group={{.Gid}}"]

[volume.home]
volume = "kind=host,source={{.HomeDir}}"
mount = "target=/home/{{.Username}}"

[volume.volume-config]
volume = "kind=empty,uid={{.Uid}},gid={{.Gid}}"

[volume.volume-data]
volume = "kind=empty,uid={{.Uid}},gid={{.Gid}}"

[alias.qiime_]
image = "quay.io/biocontainers/qiime:1.9.1--py27_0"
```

For further information, see the manpages for [rkt-run](doc/rkt-run.md)
and [rktrunner.toml](doc/rktrunner.toml.md)

## Concurrent use of pods

The configuration option `worker-pods` may be used to enable concurrent use of pods.  This is an optimisation, useful when large numbers of concurrent application processes are required.  For each user and image, a single pod may be shared across all the application instances, by means of `rkt enter`.

When using `worker-pods`, it is important to remove idle workers using `rktrunner-gc`, which should be run regularly as root.

Note that this feature is unlikely to be useful without the following `rkt` issues being addressed.

* [fly: enter should honor uid/gid/supp-gid #3392](https://github.com/rkt/rkt/issues/3392)

* [fly: enter should honor env variables #3393](https://github.com/rkt/rkt/issues/3393)

These are expected to be fixed in rkt 1.28.0.

## rkt-run-helper

`rkt-run-helper` is a simple wrapper, which invokes `rkt-run` passing
as first argument the name it was invoked with, along with all the
other arguments.

The intended use is to have a directory on the system containing links
to `rkt-run-helper`, with names `ruby`, `julia`, etc.  Then, if this
directory is on the path, scripts starting with the standard shebang
line as below will use `rkt-run` to run the containerized interpreter.
This relies on aliases for these programs being defined in [rktrunner.toml](doc/rktrunner.toml.md).

```
#!/usr/bin/env ruby
puts 'Hello World from Ruby version ' + RUBY_VERSION
```

## rkt-run-slave

`rkt-run-slave` is another wrapper, which runs within the container.
It optionally changes to the working directory as on the host.
