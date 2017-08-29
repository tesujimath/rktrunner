# Rktrunner Workers

A rktrunner worker is a pod which is reused by several application instances.  Worker pods are selected by matching image and user (uid).  The motivation is to avoid the overhead of creating separate pods, and is relevant when very many instances of an containerized application may be started.

Each worker pod is started by `rkt run`, but does nothing, simply blocking until stopped by the rktrunner garbage collector.

Each application is run by `rkt enter`.  An application instance maintains a shared lock on the worker pod directory `/var/lib/rktrunner/pod-$uuid`.  A suitable worker pod is found in `rkt list` by matching image name, application name `worker-$uid`, and state `running`.

By default, the environment variables defined within `rkt enter` are the same as those defined within the original `rkt run`.  However, a certain class of applications may require to run with updated environment variables.  For example, a graphical application may be run once with a certain `$DISPLAY`, but then the user may want to run it with a revised `$DISPLAY`.  The current value of such an environment variable may be passed in to `rkt enter` by rktrunner on a per-alias basis by means of the following line within rktrunner.toml:

```
environment-update = ["DISPLAY"]
```

The rktrunner garbage collector seeks to acquire an exclusive lock on each worker pod directory, and for those that succeed, it ends them.
