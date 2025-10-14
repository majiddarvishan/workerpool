# Project variables
BINARY_NAME=workerpool-example
EXAMPLE_DIR=examples/basic
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

.PHONY: all build run test tidy clean run-metrics

all: build

## Build the example binary
build:
	echo ">> Building example..."
	cd $(EXAMPLE_DIR) && go build -o ../../$(BINARY_NAME)

## Run the example directly
run:
	echo ">> Running example..."
	cd $(EXAMPLE_DIR) && go run .

## Run tests
test:
	echo ">> Running tests..."
	go test ./... -v

## Tidy up go.mod / go.sum
tidy:
	echo ">> Tidying modules..."
	go mod tidy

## Clean build artifacts
clean:
	echo ">> Cleaning..."
	rm -f $(BINARY_NAME)

## Run example with Prometheus metrics endpoint
run-metrics:
	echo ">> Running example with Prometheus metrics on :2112/metrics"
	cd $(EXAMPLE_DIR) && go run .
