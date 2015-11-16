SHELL := /bin/bash
PKG = github.com/Clever/pathio
PKGS := $(PKG) github.com/Clever/pathio/cmd
READMES = $(addsuffix /README.md, $(PKGS))

.PHONY: test golint README

GOVERSION := $(shell go version | grep 1.5)
ifeq "$(GOVERSION)" ""
  $(error must be running Go version 1.5)
endif

export GO15VENDOREXPERIMENT = 1

golint:
	@go get github.com/golang/lint/golint

bin: $(PKGS)
	@go build -o p3 cmd/p3.go

test: $(PKGS)
docs: $(READMES)
%/README.md:
	@go get github.com/robertkrimen/godocdown/godocdown
	@$(GOPATH)/bin/godocdown $(shell dirname $@) > $(GOPATH)/src/$@

$(PKGS): golint docs
	@gofmt -w=true $(GOPATH)/src/$@*/**.go
	@echo "LINTING..."
	@PATH=$(PATH):$(GOPATH)/bin golint $(GOPATH)/src/$@*/**.go
	@echo ""
ifeq ($(COVERAGE),1)
	@go test -cover -coverprofile=$(GOPATH)/src/$@/c.out $@ -test.v
	@go tool cover -html=$(GOPATH)/src/$@/c.out
else
	@echo "TESTING..."
	@go test $@ -test.v
endif

GODEP=$(GOPATH)/bin/godep

$(GODEP):
	@go get -u github.com/tools/godep

vendor: $(GODEP)
	$(GODEP) save $(PKGS)
	find vendor/ -path '*/vendor' -type d | xargs -IX rm -r X # remove any nested vendor directories
