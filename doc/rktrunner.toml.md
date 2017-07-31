# NAME

/etc/rktrunner.toml - configuration file for rkt-run

# SYNTAX

`rkt = ` *string* `# path to rkt program`

`default-interactive-cmd = ` *string* `# shell for interactive containers`

`preserve-cwd = ` *bool* `# whether to change to the host working directory in the container`

`use-path = ` *bool* `# whether to use the container path to find the entry point`

Note: `use-path` is only useful when using *stage1-fly*, and is a work-around for
[this issue](https://github.com/rkt/rkt/issues/3662).

`worker-pods = ` *bool* `# run user/image applications within a single worker pod`

`host-timezone = ` *bool* `# set pod timezone from host`

`restrict-images = ` *bool* `# allow only images for which aliases have been defined`

`exec-slave-dir = ` *string* `# host directory containing rkt-run-slave program`

## environment

[environment]

*name* `=` *value* `# environment variable for container`

## options

[options.*mode*] ` # mode is one of interactive, batch, common`

`general = ` *list-of-string* `# options passed to rkt program`

`run = ` *list-of-string* `# options passed to run subcommand`

`image = ` *list-of-string* `# options passed to image`

## auto-image-prefix

[auto-image-prefix]

*string* ` = ` *string* `# substitution performed on image prefix`

## volume

`[volume.` *identifier* `]`

`volume = ` *string* `# parameters passed to --volume`

`mount = ` *string* `# parameters passed to --mount`

`on-request = ` *bool* `# only include this volume if requested by user`

## alias

`[alias.` *identifier* `]`

`image = ` *string* `# image name`

`exec = ` *list-of-string* `# executables within image to expose as rkt-run aliases`

`passwd = ` *list-of-string* `# entries to append to passwd file`

`group = ` *list-of-string* `# entries to append to group file`

`[alias.` *identifier* `.environment]`

*name* `=` *value* `# environment variable override for this image`


# TEMPLATE VARIABLES

The following template variables may be used, in addition to any environment variable.

`{{.HomeDir}}` user home directory

`{{.Username}}` user login name

`{{.Uid}}` numerical user id

`{{.Gid}}` numerical group id

# EXAMPLE
```
rkt = "/usr/bin/rkt"
preserve-cwd = true
exec-slave-dir = "/usr/libexec/rktrunner"
default-interactive-cmd = "sh"

[environment]
# these are passed in a file, not on the command line
HOME = "/home/{{.Username}}"
http_proxy = "{{.http_proxy}}"
https_proxy = "{{.https_proxy}}"

[options.common]
general = ["--insecure-options=image"]
run = [
    "--net=host",
]
image = [
    "--user={{.Uid}}",
    "--group={{.Gid}}",
    "--seccomp", "mode=retain,@docker/default-whitelist,mbind",  # for Julia, see https://github.com/rkt/rkt/issues/3651
]

[auto-image-prefix]
"biocontainers/" = "docker://biocontainers/"

[volume.home]
volume = "kind=host,source={{.HomeDir}}"
mount = "target=/home/{{.Username}}"

[volume.dataset]
volume = "kind=host,source=/dataset"
mount = "target=/dataset"

[volume.bifo]
volume = "kind=host,source=/bifo"
mount = "target=/bifo"

[volume.volume-config]
volume = "kind=empty,uid={{.Uid}},gid={{.Gid}}"

[volume.volume-data]
volume = "kind=empty,uid={{.Uid}},gid={{.Gid}}"

#
# Aliases - keep alphabetical
#
# By convention, we append an underscore to the aliases for images
# which don't have a default executable.
#

[alias.blast_]
image = "quay.io/biocontainers/blast:2.6.0--boost1.61_0"
exec = ["blastn","blastp","blastx","tblastn","tblastx"]

[alias.julia_]
image = "docker://julia"
exec = ["julia"]

[alias.ruby_]
image = "docker://ruby"
exec = ["ruby", "irb"]
```

# SEE ALSO

rkt-run(1)
