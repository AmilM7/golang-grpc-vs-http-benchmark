# --- Config -------------------------------------------------------
GO      ?= $(shell which go)
BINARY  := bin/server
TESTBIN := bin/testclient
CMD_DIR := ./cmd/server
TESTCMD := ./cmd/testclient

HTTP_HOST ?= 127.0.0.1
HTTP_PORT ?= 8087
GRPC_HOST ?= 127.0.0.1
GRPC_PORT ?= 50055
BENCH_ITERATIONS ?= 100
BENCH_HTTP_BASE_URL ?= http://$(HTTP_HOST):$(HTTP_PORT)
BENCH_GRPC_ADDR ?= $(GRPC_HOST):$(GRPC_PORT)
BENCH_CONCURRENCY ?= 5
BENCH_WARMUP ?= 20
BENCH_RPC_TIMEOUT_MS ?= 2000

.PHONY: all build run clean build-test run-test deps fmt lint proto

all: build

build:
	@echo "Building server binary..."
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) $(CMD_DIR)

run: build
	@echo "Starting server (HTTP=$(HTTP_HOST):$(HTTP_PORT), gRPC=$(GRPC_HOST):$(GRPC_PORT))..."
	HTTP_HOST=$(HTTP_HOST) HTTP_PORT=$(HTTP_PORT) GRPC_HOST=$(GRPC_HOST) GRPC_PORT=$(GRPC_PORT) ./$(BINARY)

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY) $(TESTBIN)

build-test:
	@echo "Building test client..."
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(TESTBIN) $(TESTCMD)

run-test: build-test
	@echo "Running benchmark test client (HTTP=$(BENCH_HTTP_BASE_URL), gRPC=$(BENCH_GRPC_ADDR), iterations=$(BENCH_ITERATIONS), concurrency=$(BENCH_CONCURRENCY))..."
	BENCH_HTTP_BASE_URL=$(BENCH_HTTP_BASE_URL) BENCH_GRPC_ADDR=$(BENCH_GRPC_ADDR) BENCH_ITERATIONS=$(BENCH_ITERATIONS) BENCH_CONCURRENCY=$(BENCH_CONCURRENCY) BENCH_WARMUP=$(BENCH_WARMUP) BENCH_RPC_TIMEOUT_MS=$(BENCH_RPC_TIMEOUT_MS) ./$(TESTBIN)

deps:
	@echo "Syncing dependencies..."
	$(GO) mod tidy
	$(GO) mod verify

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

proto:
	@echo "Generating gRPC and protobuf stubs..."
	protoc --go_out=pkg/gen --go-grpc_out=pkg/gen proto/user/v1/user.proto
