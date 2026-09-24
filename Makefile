BINARY_NAME=rtlsdr2mqtt
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X rtlsdr2mqtt/pkg/version.Version=$(VERSION)"

.PHONY: help build build-all test test-coverage docker docker-test clean fmt lint lint-linux lint-fix install-tools check-all check-all-linux setup-dev deps run version

.DEFAULT_GOAL := help

build: ## Build the application
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/rtlsdr2mqtt

build-all: ## Build for multiple platforms
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/rtlsdr2mqtt
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/rtlsdr2mqtt
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 ./cmd/rtlsdr2mqtt
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 ./cmd/rtlsdr2mqtt

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

docker: ## Build Docker image
	docker build --build-arg VERSION=$(VERSION) -t rtlsdr2mqtt:$(VERSION) -f Containerfile .
	docker tag rtlsdr2mqtt:$(VERSION) rtlsdr2mqtt:latest

docker-test: ## Build and test with docker compose
	cd integration-tests && docker compose up --build

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)*
	rm -f coverage.out coverage.html
	cd integration-tests && docker compose down -v

fmt: ## Format code
	golangci-lint fmt

lint: ## Lint code with strict rules
	golangci-lint run

# internal/sdr/rtlsdr.go is guarded by "//go:build cgo && linux", so lint
# (and check-all) on macOS/Windows never see it. This target reproduces CI's
# linux lint for that file via Docker; run it before pushing cgo changes.
LINUX_LINT_IMAGE=golang:1.26-alpine

lint-linux: ## Lint linux/cgo code in Docker (required on macOS; CI parity)
	docker run --rm -v "$(CURDIR)":/app -v "$${GOMODCACHE:-$${HOME}/go/pkg/mod}:/go/pkg/mod" -w /app $(LINUX_LINT_IMAGE) \
		sh -c 'apk add --no-cache build-base librtlsdr-dev libusb-dev git >/dev/null && \
		GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 >/dev/null && \
		golangci-lint run'

lint-fix: ## Lint code and auto-fix issues
	golangci-lint run --fix

install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

check-all: fmt lint test ## Run all quality checks

# On macOS/Windows also run make lint-linux: the cgo/linux-only SDR file is
# invisible to the native lint. CI lints it on ubuntu-latest.
check-all-linux: fmt lint lint-linux test ## Run all quality checks incl. linux/cgo lint

setup-dev: install-tools deps ## Setup development environment
	@echo "Development environment setup complete!"
	@echo ""
	@echo "Available commands:"
	@$(MAKE) help

deps: ## Update dependencies
	go mod tidy
	go mod download

run: ## Run the application with sample config
	go run ./cmd/rtlsdr2mqtt -config integration-tests/sample-config.yaml

version: ## Show version
	@echo $(VERSION)

help: ## Show help
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
