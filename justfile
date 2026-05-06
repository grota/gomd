# gomd justfile

# Build the binary
build:
    go build ./cmd/gomd

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Run tests for a specific package (usage: just test-pkg ./internal/parser/)
test-pkg pkg:
    go test {{pkg}}

# Run go vet
vet:
    go vet ./...

# Format all Go files
fmt:
    gofmt -w .

# Check formatting (exits non-zero if files need formatting)
fmt-check:
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Run all checks (vet + fmt-check + test)
check: vet fmt-check test

# Clean build artifacts
clean:
    rm -f gomd

# Build and run with arguments (usage: just run README.md)
run *args:
    go build ./cmd/gomd && ./gomd {{args}}

# Tidy go modules
tidy:
    go mod tidy
