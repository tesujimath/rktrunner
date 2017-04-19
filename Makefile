# Makefile for rktrunner

.PHONY: all program doc
.INTERMEDIATE: doc/rkt-run.1 doc/rktrunner.toml.5

all: program doc

program:
	go install github.com/tesujimath/rktrunner/cmd/rkt-run

doc: doc/rkt-run.1.gz doc/rktrunner.toml.5.gz

doc/%.gz: doc/%
	gzip -f $<

doc/rkt-run.1: doc/rkt-run.md
	pandoc -f markdown_github $< -V section=1 -V header="RKT-RUN" -s -t man -o $@

doc/rktrunner.toml.5: doc/rktrunner.toml.md
	pandoc -f markdown_github $< -V section=5 -V header="RKTRUNNER.TOML" -s -t man -o $@
