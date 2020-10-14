.PHONY: help check test fmt vet lint shellcheck fix-fmt build acceptance

help: ## Show this help
	@echo "Help"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "    \033[36m%-20s\033[93m %s\n", $$1, $$2}'

##
### Code validation
check: ## Run all checks excepting the acceptance tests
	@bash scripts/check.sh

test: ## Run tests for all go packages
	@bash scripts/checks/test.sh

tidy: ## Ensures repository reports required deps and is ready to be published
	@bash scripts/checks/tidy.sh

lint: ## Run lint on the codebase, printing any style errors
	@bash scripts/checks/lint.sh

shellcheck:	## Lint shell scripts for potential errors
	@bash scripts/checks/shellcheck.sh

fix-fmt: ## Run goimports on all packages, fix files that don't match code-style
	@bash scripts/local/fix-fmt.sh

fix-tidy: ## Fix go.mod inconsistency
	@bash scripts/local/fix-tidy.sh

acceptance: ## Run golang acceptance tests
	@bash scripts/acceptance.sh