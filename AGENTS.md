# Agent Guidelines for go.bryk.io/pkg

This file provides essential information for AI agents working with this Go monorepo.

## Project Overview

- **Language**: Go 1.25.4
- **Module**: `go.bryk.io/pkg`
- **Type**: Public library collection (crypto, logging, CLI tools, networking, etc.)
- **Status**: Under heavy development - APIs may change

## Build, Test, and Lint Commands

### Running Tests

```bash
# Run all tests with race detection and coverage
make test

# Run tests for a specific package
make test pkg=errors

# Run a single test function
go test -v -run TestNew ./errors/

# Run benchmarks
make bench

# Run fuzz tests (30 seconds)
make fuzz
```

### Linting and Formatting

```bash
# Run full linting suite (golangci-lint)
make lint

# Lint specific package
make lint pkg=errors

# Format Go code
gofmt -s -w .
goimports -w .
```

### Building

```bash
# Build for current architecture
make build

# Tidy dependencies
make deps
```

### Protocol Buffers

```bash
# Compile protobuf definitions
make protos

# Validate protobuf (format, lint, breaking changes)
make proto-test
```

### Security Scanning

```bash
# Scan dependencies for vulnerabilities
make scan-deps

# Scan for leaked secrets
make scan-secrets

# Scan CI workflows
make scan-ci

# Run CodeQL analysis
make codeql
```

## Code Style Guidelines

### General Formatting

- **Line length**: 120 characters maximum (enforced by linter)
- **Indentation**: Tabs for Go files, 2 spaces for other files
- **Newlines**: LF (Unix-style), final newline required
- **Charset**: UTF-8

### Imports

- Group imports: standard library, third-party, then local packages
- Use named imports when necessary to avoid collisions (e.g., `lib "net/http"`)
- Run `goimports` to organize imports automatically

### Naming Conventions

- **Packages**: Short, lowercase, single word (e.g., `errors`, `log`, `cli`)
- **Exported identifiers**: PascalCase (e.g., `NewServer`, `SimpleLogger`)
- **Unexported identifiers**: camelCase (e.g., `getStack`, `isWrapper`)
- **Interface names**: Noun phrases describing capability (e.g., `SimpleLogger`, `HasStack`)
- **Test functions**: `TestXxx`, table-driven tests with `t.Run()` subtests
- **Constants**: Use `any` instead of `interface{}` for generic types

### Error Handling

- Use the internal `go.bryk.io/pkg/errors` package for error creation
- Always check errors: `if err != nil { return err }`
- Wrap errors with context: `errors.Wrap(err, "context")`
- Use `errors.New()` for simple errors, `errors.Errorf()` for formatted errors
- Preserve stack traces when wrapping errors
- Return `nil` for nil error inputs in error helpers

### Code Quality Rules

- **Function length**: Max 90 lines or 70 statements
- **Cyclomatic complexity**: Max 18
- **Nesting depth**: Avoid deep nesting (enforced by `nestif`)
- **Exported functions**: Must have documentation comments
- **Comments**: Use `//` style, end with period for complete sentences
- **Type assertions**: Always check `ok` return value

### Testing Conventions

- Use `github.com/stretchr/testify/assert` (aliased as `tdd`)
- Table-driven tests with descriptive names
- Use `t.Run()` for subtests
- Test files: `*_test.go` in same package
- Benchmarks: `BenchmarkXxx` functions
- Fuzz tests: `FuzzXxx` functions
- Race detection enabled in CI

### Protocol Buffers

- Use buf for linting and breaking change detection
- Place proto files in `proto/{package}/v{N}/`
- Use `buf validate` for field validation rules
- Service suffix: `API` (e.g., `FooAPI`)

### Git and CI

- Pre-commit hooks run lint and test automatically
- Commit messages can include `[skip scan-deps]`, `[skip scan-secrets]`, `[skip buf-breaking]`
- All code must pass `golangci-lint` with the project configuration

## Project Structure

```txt
/opt/go/src/go.bryk.io/pkg/
├── amqp/          # AMQP/RabbitMQ utilities
├── cli/           # CLI tools and configuration
├── crypto/        # Cryptographic utilities
├── errors/        # Enhanced error handling
├── jose/          # JWT/JWK/JWA implementations
├── log/           # Structured logging adapters
├── metadata/      # Service metadata utilities
├── net/           # Network utilities (HTTP, etc.)
├── otel/          # OpenTelemetry integration
├── prometheus/    # Prometheus metrics
├── proto/         # Protocol buffer definitions
├── storage/       # Database/storage utilities
└── ulid/          # ULID implementation
```

## Important Notes

- This is a public library - maintain backward compatibility when possible
- Cryptographic software notice: This distribution includes encryption software
- All exported APIs must be documented
- Use `any` type instead of `interface{}`
- Prefer composition over inheritance
- Always handle context cancellation in long-running operations
