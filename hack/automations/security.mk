SHELL := /bin/bash

REPORTS_DIR := reports

.PHONY: run gosec gitleaks govulncheck

run: gosec gitleaks govulncheck
	@echo "Security checks completed"

gosec:
	@echo "Running gosec..."
	@mkdir -p $(REPORTS_DIR)
	@go run github.com/securego/gosec/v2/cmd/gosec@latest \
		-fmt=json \
		-out=$(REPORTS_DIR)/gosec-report.json \
		-exclude-dirs=examples \
		./... 2>&1 || echo "Warning: Security issues detected"

gitleaks:
	@echo "Running gitleaks..."
	@mkdir -p $(REPORTS_DIR)
	@docker run --rm \
		-v $(PWD):/path \
		-v $(PWD)/$(REPORTS_DIR):/reports \
		zricethezav/gitleaks:latest detect \
		--source="/path" \
		--report-path="/reports/gitleaks-report.json" \
		-v || echo "Warning: Secrets detected"

govulncheck:
	@echo "Running govulncheck..."
	@mkdir -p $(REPORTS_DIR)
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./... \
		> $(REPORTS_DIR)/govulncheck-report.txt 2>&1 || echo "Warning: Vulnerabilities detected"
