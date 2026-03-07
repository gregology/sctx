.PHONY: build test lint vet fmt check clean

# Build the sctx binary
build:
	go build -o sctx ./cmd/sctx

# Run all tests with race detection
test:
	go test -race -count=1 ./...

# Run tests with coverage report
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm coverage.out

# Run golangci-lint
lint:
	golangci-lint run ./...

# Run go vet
vet:
	go vet ./...

# Check formatting (fails if any file isn't gofmt'd)
fmt:
	@test -z "$$(gofmt -l .)" || (echo "Files not formatted:"; gofmt -l .; exit 1)

# Run everything: fmt, vet, lint, test
check: fmt vet lint test
	@echo "All checks passed."

# Remove build artifacts
clean:
	rm -f sctx coverage.out
