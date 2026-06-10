SHELL := /bin/bash

SETUP_MK_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
include $(SETUP_MK_DIR)variables.mk

.PHONY: run install-lefthook asdf-add-plugins asdf-install

run: install-lefthook

install-lefthook:
ifeq ($(ASDF),true)
	@$(MAKE) -f $(SETUP_MK_DIR)setup.mk asdf-add-plugins
	@$(MAKE) -f $(SETUP_MK_DIR)setup.mk asdf-install
	@asdf exec lefthook install
else
	@go install github.com/evilmartians/lefthook@latest
	@lefthook install
endif
	@echo "Git hooks configured"

asdf-add-plugins:
	@echo "Adding asdf plugins..."
	@asdf plugin add golang   https://github.com/asdf-community/asdf-golang.git  2>/dev/null || true
	@asdf plugin add lefthook https://github.com/jtzero/asdf-lefthook.git        2>/dev/null || true

asdf-install:
	@echo "Installing asdf tools..."
	@asdf install
