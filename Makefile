.PHONY: build test unit-test integration-test clean run

# Build targets
build:
	go build -o hotdealworker

# Test targets
test: unit-test integration-test

unit-test:
	go test -v ./... -run "^Test[^I]" -race

integration-test:
	go test -v ./... -run "^TestIntegration" -race

# Clean targets
clean:
	rm -f hotdealworker
	go clean

# Run targets
run:
	go run main.go

# Install dependencies
deps:
	go mod tidy
	go mod download