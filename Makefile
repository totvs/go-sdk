SHELL := /bin/bash

HACK_DIR := hack/automations

.PHONY: all help fmt vet build tidy ci test test-v test-race cover cover-html setup \
        security-% setup example-%

all: build

##@ General

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

fmt: ## Run gofmt
	@$(MAKE) -f $(HACK_DIR)/build.mk fmt

vet: ## Run go vet
	@$(MAKE) -f $(HACK_DIR)/build.mk vet

build: ## Build all packages
	@$(MAKE) -f $(HACK_DIR)/build.mk build

tidy: ## Run go mod tidy
	@$(MAKE) -f $(HACK_DIR)/build.mk tidy

ci: ## Run fmt + vet + test (CI pipeline)
	@$(MAKE) -f $(HACK_DIR)/build.mk ci

##@ Test

test: ## Run all tests
	@$(MAKE) -f $(HACK_DIR)/test.mk test

test-v: ## Run tests in verbose mode
	@$(MAKE) -f $(HACK_DIR)/test.mk test-v

test-race: ## Run tests with race detector
	@$(MAKE) -f $(HACK_DIR)/test.mk test-race

cover: ## Generate coverage report (func summary)
	@$(MAKE) -f $(HACK_DIR)/test.mk cover

cover-html: ## Generate coverage report (HTML)
	@$(MAKE) -f $(HACK_DIR)/test.mk cover-html

##@ Security

security: ## Run all security checks (gosec + gitleaks + govulncheck)
	@$(MAKE) -f $(HACK_DIR)/security.mk run

security-%: ## Run a specific security check (e.g. make security-gosec)
	@$(MAKE) -f $(HACK_DIR)/security.mk $(patsubst security-%,%,$@)

##@ Setup

setup: ## Install lefthook and configure git hooks
	@$(MAKE) -f $(HACK_DIR)/setup.mk run

setup-%: ## Run a specific setup step (e.g. make setup-install-lefthook)
	@$(MAKE) -f $(HACK_DIR)/setup.mk $(patsubst setup-%,%,$@)

##@ Examples

example-run: ## Run an example: make example-run example=<name>
	@$(MAKE) -f $(HACK_DIR)/examples.mk run example=$(example)
