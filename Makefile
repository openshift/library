.DEFAULT_GOAL := help

verify: ## Run core verification. Example: make verify
	hack/verify-python.sh
	hack/verify-yaml.sh
	hack/verify-pullrequest.sh
.PHONY: verify

import: ## Run the import script. Example: make import
	python import_content.py
.PHONY: import

dep: ## Install Dependencies. Example: make dep 
	pip install -r requirements.txt
.PHONY: dependencies

help: ## [Default Target] - Print this help. Example: make help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
.PHONY: help

