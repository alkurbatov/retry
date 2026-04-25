SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c

MAKEFLAGS := --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules

SEED := on

.DEFAULT_GOAL := help
.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-38s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: deps
deps: ## Install project dependencies
	go mod tidy

.PHONY: test
test: ## Run unit-tests
	go test -v -race -shuffle=$(SEED) ./... -coverprofile=coverage.out -covermode atomic
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out
