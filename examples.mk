# Examples runner
.PHONY: run

run:
	@cd examples/$(example) && go run main.go