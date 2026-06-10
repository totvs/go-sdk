SHELL := /bin/bash

TEST_MK_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
include $(TEST_MK_DIR)variables.mk

.PHONY: run test test-v test-race cover cover-html

run: test

test:
	go test $(TESTFLAGS) $(TESTPKGS)

test-v:
	go test -v $(TESTPKGS)

test-race:
	go test -race -cover -covermode=atomic $(TESTPKGS)

cover:
	go test -coverprofile=coverage.out $(TESTPKGS)
	go tool cover -func=coverage.out

cover-html:
	go test -coverprofile=coverage.out $(TESTPKGS)
	go tool cover -html=coverage.out
