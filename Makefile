BINS    := hacktris hacktris-server
BINDIR  := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# Cross-compile targets for `make release`.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build both binaries into bin/
	@mkdir -p $(BINDIR)
	@for b in $(BINS); do \
		echo "building $(BINDIR)/$$b"; \
		go build -ldflags "$(LDFLAGS)" -o $(BINDIR)/$$b ./cmd/$$b || exit 1; \
	done

.PHONY: install
install: ## Install both binaries into GOBIN
	@for b in $(BINS); do go install -ldflags "$(LDFLAGS)" ./cmd/$$b || exit 1; done

.PHONY: run
run: ## Build and run a solo game
	go run ./cmd/hacktris

.PHONY: server
server: ## Build and run the relay server
	go run ./cmd/hacktris-server

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: cover
cover: ## Run tests with coverage report
	go test -cover ./...

.PHONY: race
race: ## Run all tests under the race detector
	go test -race ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go source
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-clean
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "not formatted:"; echo "$$out"; exit 1; fi

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: check
check: fmt-check vet test ## Run fmt-check, vet, and tests

.PHONY: release
release: ## Cross-compile both binaries for all platforms into bin/
	@mkdir -p $(BINDIR)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		for b in $(BINS); do \
			out=$(BINDIR)/$$b-$$os-$$arch$$ext; \
			echo "building $$out"; \
			GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o $$out ./cmd/$$b || exit 1; \
		done; \
	done

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BINDIR)
