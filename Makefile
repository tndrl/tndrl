.PHONY: test test-race test-verbose test-cover build clean lint

# Default test target: race detection + goroutine leak detection (via goleak)
test:
	go test -race ./...

# Verbose test output
test-verbose:
	go test -race -v ./...

# Test with coverage
test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run only unit tests (skip integration)
test-unit:
	go test -race ./pkg/...

# Run only integration tests
test-integration:
	go test -race ./test/integration/...

# Build binaries
build:
	go build -o bin/latis ./cmd/latis

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Generate protobuf code
generate:
	buf generate

# Lint
lint:
	go vet ./...
