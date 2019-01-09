# Makefile for picsplit
# -----------------------------------------------------------------
#
#        ENV VARIABLE
#
# -----------------------------------------------------------------

# go env vars
GO=$(firstword $(subst :, ,$(GOPATH)))
# list of pkgs for the project without vendor
PKGS=$(shell go list ./... | grep -v /vendor/)

# -----------------------------------------------------------------
#        Version
# -----------------------------------------------------------------

# version
VERSION=1.0.0
BUILDDATE=$(shell date -u '+%s')
BUILDHASH=$(shell git rev-parse --short HEAD)
VERSION_FLAG=-ldflags "-X main.Version=$(VERSION) -X main.GitHash=$(BUILDHASH) -X main.BuildStmp=$(BUILDDATE)"

# -----------------------------------------------------------------
#        Main targets
# -----------------------------------------------------------------

all: clean build

help:
	@echo
	@echo "----- BUILD ------------------------------------------------------------------------------"
	@echo "all                  clean and build the project"
	@echo "clean                clean the project"
	@echo "dependencies         download the dependencies"
	@echo "build                build all libraries and binaries"
	@echo "----- TESTS && LINT ----------------------------------------------------------------------"
	@echo "test                 test all packages"
	@echo "format               format all packages"
	@echo "lint                 lint all packages"
	@echo "help                 print this message"

clean:
	@go clean
	@rm -Rf .tmp
	@rm -Rf *.log
	@rm -Rf *.out
	@rm -Rf *.mem

build: format
	@go build -v $(VERSION_FLAG) -o $(GO)/bin/picsplit picsplit.go

format:
	@go fmt $(PKGS)

lint:
	@golint $(PKGS)
	@go vet $(PKGS)

test:
	go test ./...
