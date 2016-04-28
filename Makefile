include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = github.com/Clever/pathio
PKGS := $(shell go list ./... | grep -v /vendor)
$(eval $(call golang-version-check,1.6))
.PHONY: build test

build:
	go build -o build/p3 $(PKG)/cmd

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	go get -d -t $@
	$(call golang-test-all-strict,$@)
