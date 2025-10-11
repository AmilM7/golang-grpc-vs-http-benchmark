# --- Config -------------------------------------------------------
GO      ?= $(shell which go)
BINARY  := bin/server
TESTBIN := bin/testclient
CMD_DIR := ./cmd/server
TESTCMD := ./cmd/testclient

# --- Targets ------------------------------------------------------
.PHONY: all build run clean build-test run-test deps fmt lint

## Default target
all: build

## Build the main server binary
build:
	@echo "Building server binary..."
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) $(CMD_DIR)

## Run the main server
run: build
	@echo "Starting server..."
	./$(BINARY)

## Clean built binaries
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY) $(TESTBIN)

## Build the test client
build-test:
	@echo "Building test client..."
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(TESTBIN) $(TESTCMD)

## Run the test client
run-test: build-test
	@echo "Running benchmark test client..."
	./$(TESTBIN)

## Download Go module dependencies
deps:
	@echo "Syncing dependencies..."
	$(GO) mod tidy
	$(GO) mod verify

## Format Go code (automatically fix formatting)
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

