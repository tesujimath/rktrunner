# Makefile for rktrunner

.PHONY: all doc html rkt-run rkt-run-helper rkt-run-slave
.INTERMEDIATE: doc/rkt-run.1 doc/rktrunner.toml.5

all: rkt-run rkt-run-helper rkt-run-slave rktrunner-gc doc

# ensure executables are statically linked
GO := CGO_ENABLED=0 go

rkt-run:
	$(GO) install github.com/tesujimath/rktrunner/cmd/rkt-run

rkt-run-helper:
	$(GO) install github.com/tesujimath/rktrunner/cmd/rkt-run-helper

rkt-run-slave:
	$(GO) install github.com/tesujimath/rktrunner/cmd/rkt-run-slave

rktrunner-gc:
	$(GO) install github.com/tesujimath/rktrunner/cmd/rktrunner-gc

# test program:
get-worker:
	$(GO) install github.com/tesujimath/rktrunner/cmd/get-worker

doc: doc/rkt-run.1.gz doc/rktrunner.toml.5.gz

doc/%.gz: doc/%
	gzip -f $<

doc/rkt-run.1: doc/rkt-run.md
	pandoc -f markdown_github $< -V section=1 -V header="RKT-RUN" -s -t man -o $@

doc/rktrunner.toml.5: doc/rktrunner.toml.md
	pandoc -f markdown_github $< -V section=5 -V header="RKTRUNNER.TOML" -s -t man -o $@

# generate html from markdown, for testing
MD_FILES := $(shell ls *.md doc/*.md)
MD_HTML_TARGETS := $(foreach mdfile,$(MD_FILES),$(patsubst %.md,build/html/%.html,$(mdfile)))
html: $(MD_HTML_TARGETS)

build/html/%.html: %.md build/html/doc
	pandoc -f markdown_github $< -s -t html -o $@

build/html/doc:
	mkdir -p $@
