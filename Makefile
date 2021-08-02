include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = github.com/Clever/pathio/v4
PKGS := $(shell go list ./... | grep -v /vendor | grep -v /tools)
$(eval $(call golang-version-check,1.16))
.PHONY: build test

build:
	go build -o build/p3 $(PKG)/cmd

audit-gen: gen
	$(if \
	$(shell git status -s), \
	$(error "Generated files are not up to date. Please commit the results of `make gen`") \
	@echo "")

gen:
	go generate

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	$(call golang-test-all-strict,$@)

install_deps:
	go mod vendor
	go build -o bin/mockgen ./vendor/github.com/golang/mock/mockgen
