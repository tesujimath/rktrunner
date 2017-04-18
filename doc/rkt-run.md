# NAME

rkt-run - enable unprivileged users to run containers using rkt

# SYNOPSIS

**rkt-run**\ [*options*] **image** [*args*]

# DESCRIPTION

Run rkt containers with user mapping, and volume mounting
as defined by the system administrator.

# OPTIONS

`--config` *config-file*
alternative config file, requires --dry-run

`-e`, `--exec` *command*
command to run instead of image default

`-i`, `--interactive`
run image interactively

`-v`, `--verbose`
show full rkt run command

`--dry-run`
don't execute anything

`-l`, `--list-alias`
list image aliases

`-n`, `--no-image-prefix`
disable auto image prefix

# AUTHOR
Simon Guest
