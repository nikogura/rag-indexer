.PHONY: build clean lint test install-tools help docker-build

# Build variables
BINARY_NAME=code-indexer
BUILD_DIR=.
INSTALL_DIR=/usr/local/bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Linter parameters
GOLANGCI_LINT=golangci-lint
NAMEDRETURNS_PKG=github.com/nikogura/namedreturns/cmd/namedreturns

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	$(GOBUILD) -o $(BINARY_NAME) -v

clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

lint: ## Run namedreturns linter then golangci-lint
	@echo "Running namedreturns linter..."
	@namedreturns ./...
	@echo ""
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run --timeout 5m

test: ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

coverage: test ## Run tests and show coverage
	$(GOCMD) tool cover -html=coverage.out

tidy: ## Tidy go.mod and go.sum
	$(GOMOD) tidy

install-tools: ## Install required development tools
	@echo "Installing namedreturns linter..."
	@$(GOGET) -u $(NAMEDRETURNS_PKG)
	@echo "Installing golangci-lint..."
	@which $(GOLANGCI_LINT) > /dev/null || (echo "Please install golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)

install: build ## Install binary to system
	install -m 755 $(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

run-serve: build ## Build and run in serve mode
	./$(BINARY_NAME) -mode serve

run-index: build ## Build and run in index mode
	./$(BINARY_NAME) -mode index

docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):latest .

all: clean tidy lint test build ## Clean, tidy, lint, test, and build
