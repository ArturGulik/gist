BINARY      := gist
PREFIX      ?= $(HOME)/.local
BINDIR      := $(PREFIX)/bin

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS     := -s -w \
               -X main.version=$(VERSION) \
               -X main.commit=$(COMMIT) \
               -X main.date=$(DATE)

GO          ?= go

.PHONY: all build install test test-race vet fmt lint vuln tidy clean release-snapshot help

all: build

build: ## Build the binary into the project root
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BINARY) .

install: ## Build and install to $(BINDIR), plus shell completions
	@mkdir -p $(BINDIR)
	$(GO) build -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(BINARY) .
	@echo "installed $(BINDIR)/$(BINARY)"
	@$(BINDIR)/$(BINARY) completion install --shell=bash 2>/dev/null || true
	@$(BINDIR)/$(BINARY) completion install --shell=zsh  2>/dev/null || true

test: ## Run tests
	$(GO) test ./...

test-race: ## Run tests with race detector
	$(GO) test -race ./...

vet: ## Run go vet
	$(GO) vet ./...

fmt: ## Run gofmt -s -w
	gofmt -s -w .

lint: ## Run golangci-lint (must be installed)
	golangci-lint run

vuln: ## Run govulncheck (install: go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

tidy: ## Tidy go.mod
	$(GO) mod tidy

release-snapshot: ## Build a snapshot release locally with goreleaser
	goreleaser release --snapshot --clean

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
