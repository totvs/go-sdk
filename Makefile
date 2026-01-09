## Makefile para facilitar tarefas comuns do projeto go-sdk

.PHONY: test test-v test-race cover cover-html fmt vet build tidy ci setup

TESTPKGS := ./...
TESTFLAGS ?=

include examples.mk

.PHONY: run-example

# default log level for the example (can be overridden: `make run-example LOG_LEVEL=info`)
LOG_LEVEL ?= DEBUG

# Executa todos os testes
test:
	go test $(TESTFLAGS) $(TESTPKGS)

# Executa os testes em modo verbose
test-v:
	go test -v $(TESTPKGS)

# Testes com detector de race e cobertura mínima
test-race:
	go test -race -cover -covermode=atomic $(TESTPKGS)

## Gera arquivo de cobertura e mostra resumo
cover:
	go test -coverprofile=coverage.out $(TESTPKGS)
	go tool cover -func=coverage.out

## Gera relatório HTML de cobertura
cover-html:
	go test -coverprofile=coverage.out $(TESTPKGS)
	go tool cover -html=coverage.out

## Formata o código (gofmt)
fmt:
	gofmt -w .

## Executa go vet
vet:
	go vet $(TESTPKGS)

## Compila os pacotes
build:
	go build $(TESTPKGS)

## Ajusta dependências do módulo
tidy:
	go mod tidy

## Target para CI: formata, analisa e testa
ci: fmt vet test

## Setup: instala lefthook e configura git hooks
# Use ASDF=true para instalar via asdf ao invés de go install
ASDF ?= false

setup:
ifeq ($(ASDF),true)
	@asdf install
	@asdf exec lefthook install
else
	@go install github.com/evilmartians/lefthook@latest
	@lefthook install
endif
	@echo "Git hooks configured"

# Run the example in ./examples/logger (default: DEBUG level)
# You can override the level by calling e.g. `make run-example LOG_LEVEL=info`
run-example:
	cd examples/logger && LOG_LEVEL=$(LOG_LEVEL) go run .


# security
.PHONY: security
security: gosec gitleaks govulncheck
	@echo "Security checks completed"

.PHONY: gosec
gosec:
	@echo "Running gosec..."
	@mkdir -p reports
	@go run github.com/securego/gosec/v2/cmd/gosec@latest -fmt=json -out=reports/gosec-report.json -exclude-dirs=examples ./... 2>&1 || (echo "Security issues were detected!") 

.PHONY: gitleaks
gitleaks:
	@echo "Running gitleaks..."
	@docker run -v ${PWD}:/path -v ${PWD}/reports:/reports zricethezav/gitleaks:latest detect \
		--source="/path" \
		--report-path="/reports/gitleaks-report.json" \
		-v || (echo "Secrets were detected!")
	
.PHONY: govulncheck
govulncheck:
	@echo "Running govulncheck..."
	@mkdir -p reports
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./... > reports/govulncheck-report.txt 2>&1 || (echo "Vulnerabilities were detected!")
