.PHONY: help build run test lint clean
.DEFAULT_GOAL := help

build: ## Build the binary
	go build -o openpnp-tools .

run: ## Run with example job (set JOB=path/to/job.xml)
	go run . $(JOB)

test: ## Run tests
	go test ./...

lint: ## Lint source code
	golangci-lint run

clean: ## Clean build artifacts
	rm -f openpnp-tools
	go clean -testcache

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
