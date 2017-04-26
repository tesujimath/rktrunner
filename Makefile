# Makefile for rktrunner

.PHONY: all doc rkt-run rkt-run-helper rkt-run-slave rkt-attach-by-name
.INTERMEDIATE: doc/rkt-run.1 doc/rktrunner.toml.5

all: rkt-run rkt-run-helper rkt-run-slave doc

rkt-run:
	go install github.com/tesujimath/rktrunner/cmd/rkt-run

rkt-run-helper:
	go install github.com/tesujimath/rktrunner/cmd/rkt-run-helper

rkt-run-slave:
	go install github.com/tesujimath/rktrunner/cmd/rkt-run-slave

# test program:
rkt-attach-by-name:
	go install github.com/tesujimath/rktrunner/cmd/rkt-attach-by-name

doc: doc/rkt-run.1.gz doc/rktrunner.toml.5.gz

doc/%.gz: doc/%
	gzip -f $<

doc/rkt-run.1: doc/rkt-run.md
	pandoc -f markdown_github $< -V section=1 -V header="RKT-RUN" -s -t man -o $@

doc/rktrunner.toml.5: doc/rktrunner.toml.md
	pandoc -f markdown_github $< -V section=5 -V header="RKTRUNNER.TOML" -s -t man -o $@
