# If no documents are specified, use these as the default
DOCUMENTS ?= official,community
LOGLEVEL ?= 0

ifeq ($(MATCHALL),true)
	MATCHALLTAGS=--match-all
endif

.DEFAULT_GOAL := help

verify: verify-gofmt verify-pullrequest ## Run verifications. Example: make verify
.PHONY: verify

verify-gofmt: ## Run gofmt verification. Example: make verify-gofmt
	hack/verify-gofmt.sh
.PHONY: verify-gofmt

verify-pullrequest: ## Run pull request verification. Example: make verify-pullrequest
	hack/verify-pullrequest.sh $(DOCUMENTS)
.PHONY: verify-pullrequest

verify-periodic: ## Run periodic job verification. Example: make verify-periodic
	hack/verify-periodic.sh
.PHONY: verify-periodic

verify-periodic-old: ## Run periodic job verification using the old syntax. Example: make verify-periodic-old
	hack/verify-periodic-old.sh
.PHONY: verify-periodic-old

# Using -race here since we are running concurrently
build: ## Build the library executable. Example: make build
	go version
	go build -race
.PHONY: build

import: ## Run the import script. Example: make import
	./library import --documents=$(DOCUMENTS) --tags=$(TAGS) $(MATCHALLTAGS) --dir=$(DIR) -v=$(LOGLEVEL)
.PHONY: import

vendor: ## Vendor Go Dependencies. Example: make vendor
	go mod vendor
.PHONY: vendor

clean: ## Clean up the workspace. Example: make clean
	rm -f library
	rm -rf _output/
.PHONY: clean

help: ## Print this help. Example: make help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.PHONY: help

