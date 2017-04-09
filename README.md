This package provides the `rkt-run` command, which is intended to be
installed setuid root, to enabled unprivileged users to run containers
using `rkt`, in a controlled fashion.

All `rkt run` options are controlled by the config file,
`/etc/rktrunner.toml`, which should be carefully setup by the local
sysadmin.

Example use:
```
rkt-run --interactive --verbose docker://quay.io/biocontainers/blast:2.6.0--boost1.61_0
```

The `--verbose` option prints the full `rkt run` command which is
being run, as follows:
```
/usr/bin/rkt --insecure-options=image run --interactive --set-env=HOME=/home/guestsi --volume home,kind=host,source=/home/guestsi --volume dataset,kind=host,source=/dataset --volume bifo,kind=host,source=/bifo docker://quay.io/biocontainers/blast:2.6.0--boost1.61_0 --mount volume=dataset,target=/dataset --mount volume=bifo,target=/bifo --mount volume=home,target=/home/guestsi --user=511 --user=511 --exec bash
```

This is an early version of the program, which basically works, but
hasn't yet seen much usage.
