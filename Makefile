BINARY := makakito
BUILD_DIR := bin
MODULE := github.com/CyberArgonaut/makakito
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X $(MODULE)/cmd.version=$(VERSION) -X $(MODULE)/cmd.commit=$(COMMIT) -X $(MODULE)/cmd.date=$(DATE)"

.PHONY: build clean fmt vet lint test test-integration test-all dev-setup

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

clean:
	rm -rf $(BUILD_DIR)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -count=1 -tags=integration ./...

test-all: test test-integration

dev-setup:
	go mod download
	@echo "Installing golangci-lint..."
	@which golangci-lint > /dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Dev setup complete."
