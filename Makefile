# Global variables
PROJECTNAME=pluggo
# Go related variables.
GOBASE=$(shell pwd)
GOPATH="$(GOBASE)/vendor:$(GOBASE)"
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

## clean-build: Clean up, install dependencies and build the project
clean-build: clean install-dependencies build

## clean: Clean the projects build cache
clean:
	@echo " > Cleaning build cache..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

## run: Run the project using the local config.toml
run:
	@echo " > Running code..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go run $(GOFILES) config.toml

## build: Build the project
build:
	@echo " > Building binary..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -ldflags "-w -s" -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

## install-dependencies: Install all necessary dependencies for this project
install-dependencies:
	@echo " > Checking if there is any missing dependencies..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go get $(get)

.PHONY: help
help: Makefile
	@echo
	@echo "Choose a command run in "$(PROJECTNAME)":"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo
