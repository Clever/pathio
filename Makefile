include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

.PHONY: test golint
SHELL := /bin/bash
PKG = github.com/Clever/pathio
PKGS := $(shell go list ./... | grep -v /vendor)
$(eval $(call golang-version-check,1.6))

bin: $(PKGS)
	@go build -o p3 cmd/p3.go

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	@go get -d -t $@
	$(call golang-test-all-strict,$@)
