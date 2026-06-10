SHELL := /bin/bash

EXAMPLES_MK_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
include $(EXAMPLES_MK_DIR)variables.mk

.PHONY: run

# Usage: make example-run example=<name>  (e.g. make example-run example=kubernetes-status)
run:
	@if [ -z "$(example)" ]; then \
		echo "Usage: make example-run example=<name>"; \
		echo "Available:"; \
		ls examples/; \
		exit 1; \
	fi
	cd examples/$(example) && LOG_LEVEL=$(LOG_LEVEL) go run main.go
