---
title: Contributing
description: How to build, test, and contribute to sctx
---

# Contributing

## Prerequisites

- Go 1.25 or later
- [golangci-lint](https://golangci-lint.run/welcome/install/) v2+

## Getting started

```bash
git clone https://github.com/gregology/sctx.git
cd sctx
make check
```

That runs formatting, vetting, linting, and tests with race detection. If it passes, you're set.

## Building

```bash
# Build the binary
make build

# Run without building
go run ./cmd/sctx version

# Install to $GOPATH/bin
go install ./cmd/sctx
```

## Running tests

```bash
# All tests with race detection
make test

# Specific package
go test ./internal/core/...

# Specific test
go test ./internal/core/... -run TestResolve_EditBefore

# With coverage
make cover
```

## Project structure

```
cmd/sctx/              CLI entry point. Thin dispatch layer.
internal/
  core/                Agent-agnostic engine. Discovers, parses,
                       filters, and merges context files.
  adapter/             Agent-specific translation layers. Each
                       adapter maps agent input to a ResolveRequest.
  validator/           Schema validation for context files.
docs/                  Documentation (you're reading it).
```

The key boundary: `internal/core` must never import from `internal/adapter`. Agent-specific logic stays in adapters.

## Making changes

1. Create a branch off `main`
2. Make your changes
3. Run `make check` -- fmt, vet, lint, and tests must all pass
4. Open a PR

### Adding a new adapter

Create a new file in `internal/adapter/` (e.g., `cursor.go`). Your adapter reads whatever input the agent provides, maps it to a `core.ResolveRequest`, calls `core.Resolve`, and formats the output. Look at `claude.go` for the pattern.

### Adding new context fields

1. Add the field to the struct in `internal/core/schema.go`
2. Set a default in `applyDefaults` in `engine.go` if needed
3. If the field acts as a filter, add filtering logic in `filterContext()` or `filterDecisions()` in `engine.go`
4. Add validation in `internal/validator/validate.go`
5. Update testdata fixtures
6. Update `docs/protocol.md`

### Test conventions

- Table-driven tests for unit logic
- Test fixtures go in `testdata/` directories
- Use `t.TempDir()` when the test needs dynamic file creation
- No assertion libraries -- plain `if` + `t.Errorf`
- Test names follow `TestFunctionName_Scenario`

## Linting

We use golangci-lint with a tuned config in `.golangci.yml`. Some linters are intentionally disabled with rationale in the config file. If you think the linter found a false positive, check the config before adding a `//nolint` directive. If you do add one, include the linter name and a reason:

```go
data, err := os.ReadFile(path) //nolint:gosec // path comes from directory walk, not user input
```

## Validating context files

The project uses its own AGENTS.yaml files. After editing them:

```bash
go run ./cmd/sctx validate .
```
