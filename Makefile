include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = gopkg.in/Clever/pathio.v3
PKGS := $(shell go list ./... | grep -v /vendor)
$(eval $(call golang-version-check,1.10))
.PHONY: build test

build:
	go build -o build/p3 $(PKG)/cmd

audit-gen: gen
	$(if \
	$(shell git status -s), \
	$(error "Generated files are not up to date. Please commit the results of `make gen`") \
	@echo "")

gomock: install_deps
	go get -u github.com/golang/mock/gomock

gen: gomock
	go generate

test: $(PKGS)
$(PKGS): golang-test-all-strict-deps
	go get -d -t $@
	$(call golang-test-all-strict,$@)


install_deps: golang-dep-vendor-deps
	$(call golang-dep-vendor)
	go build -o bin/mockgen ./vendor/github.com/golang/mock/mockgen
	cp bin/mockgen $(GOPATH)/bin/mockgen
