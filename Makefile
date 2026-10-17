BINARY_NAME=vcf-toolkit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DIR=bin

LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

.PHONY: all build test lint clean run run-bin help

all: lint
	@echo
	make test
	@echo
	make build
lint:
	@echo "linting code..."
	golangci-lint run

test:
	@echo "Testing codebase..."
	go test -v -race -count=1 ./...

build:
	@echo "building binary"
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .

clean:
	@echo "cleaning build artifacts..."
	rm -rf ${BUILD_DIR}

run:
	go run .

run-bin:
	@echo "running the binary..."
	./${BUILD_DIR}/${BINARY_NAME}

help:
	@echo "Available targets:"
	@echo "  all		- Lint, test, and build the project"
	@echo "  lint		- Lint the codebase"
	@echo "  test		- run tests"
	@echo "  run		- run main.go"
	@echo "  run-bin	- run the binary"
