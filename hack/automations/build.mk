SHELL := /bin/bash

.PHONY: run fmt vet build tidy ci

run: fmt vet build

fmt:
	gofmt -w .

vet:
	go vet ./...

build:
	go build ./...

tidy:
	go mod tidy

ci: fmt vet test
