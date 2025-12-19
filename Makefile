# Makefile for picsplit v2.0.0
# -----------------------------------------------------------------

# Go environment
GO=$(firstword $(subst :, ,$(GOPATH)))
PKGS=$(shell go list ./... | grep -v /vendor/)
GOFLAGS=-mod=readonly

# Binary
BINARY_NAME=picsplit
BIN_DIR=./bin
BINARY_PATH=$(BIN_DIR)/$(BINARY_NAME)
INSTALL_PATH=$(GO)/bin/$(BINARY_NAME)

# Version
VERSION=2.0.0
BUILDDATE=$(shell date -u '+%s')
BUILDHASH=$(shell git rev-parse --short HEAD)
VERSION_FLAG=-ldflags "-X main.Version=$(VERSION) -X main.GitHash=$(BUILDHASH) -X main.BuildStmp=$(BUILDDATE)"

# Coverage
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
COVERAGE_TARGET=80

# -----------------------------------------------------------------
#        Main targets
# -----------------------------------------------------------------

.PHONY: all
all: clean build test

.PHONY: help
help:
	@echo
	@echo "----- BUILD ----------------------------------------------------------------------"
	@echo "all                  clean, build and test the project"
	@echo "clean                clean the project (bin/, coverage files, temp files)"
	@echo "build                build the binary to ./bin/picsplit"
	@echo "install              install the binary to GOPATH/bin"
	@echo "----- TESTS && COVERAGE ----------------------------------------------------------"
	@echo "test                 run tests"
	@echo "test-coverage        run tests with coverage report"
	@echo "coverage-html        generate HTML coverage report"
	@echo "test-verbose         run tests with verbose output"
	@echo "----- CODE QUALITY ---------------------------------------------------------------"
	@echo "format               format all packages"
	@echo "lint                 lint all packages"
	@echo "tidy                 tidy go modules"
	@echo "----- UTILITIES ------------------------------------------------------------------"
	@echo "help                 print this message"

.PHONY: clean
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -Rf .tmp
	@rm -Rf *.log
	@rm -Rf *.out
	@rm -Rf *.mem
	@rm -f screenshot*.png screenshot*.jpg screenshot*.webp
	@rm -Rf $(BIN_DIR)
	@rm -f $(INSTALL_PATH)

.PHONY: build
build: format
	@echo "Building picsplit $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	@go build $(GOFLAGS) -v $(VERSION_FLAG) -o $(BINARY_PATH) picsplit.go
	@echo "Binary created: $(BINARY_PATH)"

.PHONY: install
install: build
	@echo "Installing picsplit to $(INSTALL_PATH)..."
	@mkdir -p $(GO)/bin
	@cp $(BINARY_PATH) $(INSTALL_PATH)
	@echo "Installed: $(INSTALL_PATH)"

.PHONY: format
format:
	@echo "Formatting code..."
	@go fmt $(PKGS)

.PHONY: lint
lint:
	@echo "Linting code..."
	@go vet $(PKGS)

.PHONY: tidy
tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@go mod verify

# -----------------------------------------------------------------
#        Tests
# -----------------------------------------------------------------

.PHONY: test
test:
	@echo "Running tests..."
	@go test -v -race ./...

.PHONY: test-verbose
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v -race -count=1 ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo
	@echo "=== Coverage Summary ==="
	@go tool cover -func=$(COVERAGE_FILE) | grep total
	@echo "========================"
	@echo
	@echo "Target coverage: $(COVERAGE_TARGET)%"
	@echo "Coverage file: $(COVERAGE_FILE)"

.PHONY: coverage-html
coverage-html: test-coverage
	@echo "Generating HTML coverage report..."
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "HTML report: $(COVERAGE_HTML)"
	@open $(COVERAGE_HTML) 2>/dev/null || xdg-open $(COVERAGE_HTML) 2>/dev/null || echo "Open $(COVERAGE_HTML) manually"
