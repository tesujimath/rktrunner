# NAME

/etc/rktrunner.toml - configuration file for rkt-run

# SYNTAX

`rkt = ` *string* `# path to rkt program`

`default-interactive-cmd = ` *string* `# shell for interactive containers`

## environment

*string* ` = ` *string* `# define environment variable`

## options

`general = ` *list-of-string* `# options passed to rkt program`

`run = ` *list-of-string* `# options passed to run subcommand`

`image = ` *list-of-string* `# options passed to image`

## auto-image-prefix

*string* ` = ` *string* `# substitution performed on image prefix`

## volume

`[volume.` *identifier* `]`
`volume = ` *string* `# parameters passed to --volume`
`mount = ` *string* `# parameters passed to --mount`

## alias

`[alias.` *identifier* `]`
`image = ` *string* `# image name`
`exec = ` *list-of-string* `# executables within image to expose as rkt-run aliases`

# TEMPLATE VARIABLES

The following template variables may be used.

`{{.HomeDir}}` user home directory

`{{.Username}}` user login name

`{{.Uid}}` numerical user id

`{{.Gid}}` numerical group id

# EXAMPLE
```
rkt = "/usr/bin/rkt"
default-interactive-cmd = "sh"

[environment]
# needed for per-application options --stdout, etc
# as per https://github.com/rkt/rkt/issues/3639
RKT_EXPERIMENT_ATTACH = "true"

[options]
general = ["--insecure-options=image"]
run = ["--net=host", "--set-env=HOME=/home/{{.Username}}"]
# --stdout=stream ought to work, but doesn't
# see https://github.com/rkt/rkt/issues/3639
image = ["--user={{.Uid}}", "--group={{.Gid}}", "--stdout=log"]

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

[alias.bbmap_]
image = "quay.io/biocontainers/bbmap:37.02--0"
exec = ["bbmap.sh"]

[alias.blast_]
image = "quay.io/biocontainers/blast:2.6.0--boost1.61_0"
exec = ["blastn","blastp","blastx","tblastn","tblastx"]

[alias.canu_]
image = "quay.io/biocontainers/canu:1.4--1"
exec = ["canu"]

[alias.julia]
image = "docker://julia"

[alias.mafft_]
image = "quay.io/biocontainers/mafft:7.305--0"
exec = ["mafft"]

[alias.qiime_]
image = "quay.io/biocontainers/qiime:1.9.1--py27_0"

[alias.ruby]
image = "docker://ruby"

[alias.spades_]
image = "quay.io/biocontainers/spades:3.10.1--py35_0"

[alias.stacks_]
image = "quay.io/biocontainers/stacks:1.44--0"

[alias.trimmomatic_]
image = "quay.io/biocontainers/trimmomatic:0.36--3"
```

# SEE ALSO

rkt-run(1)