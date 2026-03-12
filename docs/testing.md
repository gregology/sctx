# Testing Guidelines

## Philosophy

sctx uses **behavior-driven integration tests** as its foundation, with **property-based tests** targeted at the core resolution engine. The goal is tests that catch real bugs without slowing down development or blocking refactors.

### Core principles

1. **Test at public API boundaries** — `core.Resolve()`, `adapter.HandleClaudeHook()`, `validator.ValidateTree()`. Never test private functions directly.
2. **Every test represents a real user scenario** — if you can't describe the test as "when a user does X, Y should happen," delete it.
3. **No mocks** — use real temp directories and real YAML files. The codebase is small enough that real I/O is fast.
4. **No coverage targets** — coverage percentages incentivize pointless tests. Focus on behavior coverage instead.

## Test types by package

### `internal/core` — Integration + Property-based

The resolution engine has combinatorial inputs: glob patterns, nested directories, action types, timing, exclude patterns, merge ordering. Example-based tests can't cover this space efficiently.

**Table-driven integration tests** for documenting expected behavior:

- One test per distinct user scenario (edit a Python file, create a new file, read a test file, etc.)
- Assert on the content strings returned, not internal data structures
- Use `testdata/` fixtures for stable scenarios, `t.TempDir()` for dynamic ones

**Property-based tests** (using `pgregory.net/rapid`) for invariants:

- `Resolve` never panics for any valid input
- Child context always merges with parent (never overrides)
- Exclude patterns always take precedence over match patterns
- Entries with `on: edit` never appear for `ActionRead` requests
- Parent entries always appear before child entries in results

Property tests belong only in `internal/core`. The adapter and validator don't have the combinatorial complexity to justify them.

### `internal/adapter` — Integration tests only

- Test `HandleClaudeHook()` with real JSON input on stdin, assert on formatted output
- Test `EnableClaude()` / `DisableClaude()` against real temp `.claude/settings.local.json` files
- Table-driven tests for tool-name-to-action mapping

### `internal/validator` — Integration tests only

- Test `ValidateTree()` against directories containing valid and invalid YAML
- One test per error class (missing content, invalid glob, bad action enum, etc.)

## What NOT to test

- **Private functions** — if `matchesGlobs` breaks, a `Resolve` test will catch it. Testing both just means two tests to update when the signature changes.
- **Go stdlib behavior** — don't test that `os.ReadFile` works or that `yaml.Unmarshal` parses valid YAML.
- **Formatting details** — don't assert on exact whitespace in output. Check for the presence of key content strings.
- **Constructor/getter boilerplate** — if a function just returns a field, it doesn't need a test.

## Conventions

- **Table-driven tests** for any function with more than two interesting inputs
- **No assertion libraries** — use `t.Errorf` / `t.Fatalf` with descriptive messages
- **`t.Helper()`** on all test helpers so failures point to the calling line
- **`t.TempDir()`** for tests that need filesystem state — automatically cleaned up
- **Test names describe the scenario**, not the function: `TestResolve_ExcludeOverridesMatch` not `TestResolve_Case7`
- **Race detection** — all tests run with `-race` via `make test`

## Running tests

```bash
make test          # Run with race detection
make check         # Full: fmt + vet + lint + test
make cover         # Coverage report (informational, not a gate)
```
