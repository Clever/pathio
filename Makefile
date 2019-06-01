include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = gopkg.in/Clever/pathio.v3
PKGS := $(shell go list ./... | grep -v /vendor)
$(eval $(call golang-version-check,1.12))
.PHONY: build test

build:
	go build -o build/p3 $(PKG)/cmd

audit-gen: gen
	$(if \
	$(shell git status -s), \
	$(error "Generated files are not up to date. Please commit the results of `make gen`") \
	@echo "")

gomock:
	go get -u github.com/golang/mock/gomock
	go get -u github.com/golang/mock/mockgen

gen: gomock
	go generate

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	go get -d -t $@
	$(call golang-test-all-strict,$@)


install_deps: golang-dep-vendor-deps
	$(call golang-dep-vendor)