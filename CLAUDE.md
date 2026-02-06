This document provides guidelines for maintaining high-quality Go code. These rules MUST be followed by all AI coding agents and contributors.

## Your Core Principles

All code you write MUST be fully optimized.

"Fully optimized" includes:

- maximizing algorithmic big-O efficiency for memory and runtime
- using goroutines and concurrency primitives where appropriate
- following proper style conventions for Go (e.g. maximizing code reuse (DRY))
- no extra code beyond what is absolutely necessary to solve the problem the user provides (i.e. no technical debt)

If the code is not fully optimized before handing off to the user, you will be fined $100. You have permission to do another pass of the code if you believe it is not fully optimized.

## Preferred Tools

- Use `go` for building, testing, and dependency management.
- Use `mise` for task automation (see `mise.toml`).
- Use `golangci-lint` for linting.
- Use `encoding/json` (stdlib) for JSON serialization/deserialization.

## Project Layout

```
cmd/ciro/        # CLI entry point
pkg/pipeline/    # Reusable pipeline types (external consumers may import)
pkg/parser/      # Reusable KDL parser (external consumers may import)
internal/builder # BuildKit LLB builder (internal only)
internal/runner  # BuildKit runner (internal only)
```

- `pkg/` contains packages safe for external import.
- `internal/` contains implementation details not exported.

## Code Style and Formatting (CS)

- **CS-1**: **MUST** use `gofmt` for all formatting (tabs, not spaces).
- **CS-2**: No name stutter. `pipeline.Pipeline` is fine; `pipeline.CiroPipeline` is not.
- **CS-3**: **MUST** use meaningful, descriptive variable and function names.
- **CS-4**: Follow [Effective Go](https://go.dev/doc/effective_go) and Go idioms.
- **CS-5**: Functions with >5 parameters **MUST** use an input struct.
- **NEVER** use emoji, or unicode that emulates emoji (e.g. check marks, X marks). The only exception is when writing tests and testing the impact of multibyte characters.

## Documentation (API)

- **API-1**: **MUST** include doc comments for all exported functions, types, and methods.
- **API-2**: Accept interfaces, return concrete types.
- Keep comments up-to-date with code changes.

## Error Handling (ERR)

- **ERR-1**: All errors **MUST** be wrapped with `fmt.Errorf("context: %w", err)`.
- **ERR-2**: Use `errors.Is` and `errors.As` for error checking.
- **ERR-3**: Use sentinel errors (`var ErrFoo = errors.New(...)`) for expected failure modes.
- **ERR-5**: **NEVER** return `nil, nil`. If there's no error, return a valid value.
- **MUST** handle errors once: log OR return, never both.
- Error wrap messages **MUST NOT** start with "failed" or "error".

## Architecture (ARCH)

- **ARCH-9**: Use sentinel errors for expected failure modes.
- **ARCH-10**: Handle errors once (log OR return, not both).
- **ARCH-11**: Error wrap messages must not begin with "failed"/"error".
- **ARCH-29**: Underscore prefix for unexported package-level globals (e.g. `_defaultAddr`).

## Testing (T)

- **T-1**: **MUST** use table-driven tests with `t.Parallel()`.
- **T-2**: **MUST** use `-race` flag when running tests.
- **MUST** mock external dependencies (APIs, databases, file systems).
- Follow the Arrange-Act-Assert pattern.
- Do not commit commented-out tests.

## Observability (OBS)

- **OBS-1**: Use `log/slog` (stdlib) for structured logging.
- **NEVER** log sensitive information (passwords, tokens, PII).

## Concurrency (CC)

- **CC-1**: The sender closes the channel.
- **CC-4**: Use `golang.org/x/sync/errgroup` for concurrent goroutine management.
- Prefer channels for communication, mutexes for state protection.
- Always use `context.Context` for cancellation.

## Dependencies

- **MD-1**: Only add external dependencies when there is clear payoff over hand-rolling.
- **MUST** run `go mod tidy` after dependency changes.
- Prefer stdlib where feasible.

## Security

- **NEVER** store secrets, API keys, or passwords in code. Only store them in `.env`.
  - Ensure `.env` is declared in `.gitignore`.
- **MUST** use environment variables for sensitive configuration via `os.Getenv`.
- **NEVER** log sensitive information (passwords, tokens, PII).

## Version Control

- **MUST** write clear, descriptive commit messages.
- **NEVER** commit commented-out code; delete it.
- **NEVER** commit debug `fmt.Println` or `log.Println` statements.
- **NEVER** commit credentials or sensitive data.

## Tools

- **MUST** use `gofmt` for code formatting.
- **MUST** use `golangci-lint` for linting.
- **MUST** ensure code compiles with no warnings.
- Use `go build ./...` for building.
- Use `go test -race ./...` for running tests.
- Use `go vet ./...` for static analysis.
- Use `mise` for task automation.

## Before Committing

- [ ] All tests pass (`go test -race ./...`)
- [ ] No compiler warnings (`go build ./...`)
- [ ] Vet passes (`go vet ./...`)
- [ ] Code is formatted (`gofmt`)
- [ ] All exported items have doc comments
- [ ] No commented-out code or debug statements
- [ ] No hardcoded credentials

---

**Remember:** Prioritize clarity and maintainability over cleverness.
