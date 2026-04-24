REPO := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GO = go

.PHONY: all build time-mcp clean

all: deps build

build: time-mcp

deps:
	$(GO) mod tidy
	$(GO) mod download

get:
	$(GO) get -v ./...

time-mcp:
	$(GO) build -o bin/time-mcp -v

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

check: fmt vet test

clean:
	$(GO) clean -v
	rm -rf $(REPO)/bin
