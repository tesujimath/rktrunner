# Rktrunner Workers

A rktrunner worker is a pod which is reused by several application instances.  Worker pods are selected by matching image and user (uid).  The motivation is to avoid the overhead of creating separate pods, and is relevant when very many instances of an containerized application may be started.

Each worker pod is started by `rkt run`, but does nothing, simply blocking until stopped by the rktrunner garbage collector.

Each application is run by `rkt enter`.  An application instance maintains a shared lock on the worker pod directory `/var/lib/rktrunner/pod-$uuid`.  A suitable worker pod is found in `rkt list` by matching image name, application name `worker-$uid`, and state `running`.

The rktrunner garbage collector seeks to acquire an exclusive lock on each worker pod directory, and for those that succeed, it ends them.
