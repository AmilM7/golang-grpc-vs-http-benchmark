GO ?= go
BINARY := bin/server
CMD := ./cmd/server

.PHONY: all build run clean

all: build

build:
	$(GO) build -o $(BINARY) $(CMD)

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)
