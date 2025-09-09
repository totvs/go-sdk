## Makefile para facilitar tarefas comuns do projeto go-sdk

.PHONY: test test-v test-race cover cover-html fmt vet build tidy ci

TESTPKGS := ./...
TESTFLAGS ?=

.PHONY: run-example


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

# Run the example in ./examples/logger
run-example:
	cd examples/logger && go run .
