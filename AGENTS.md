# AGENTS.md

## Overview

This document provides essential instructions, code style guidelines, and operational conventions for agentic contributors (code-writing AI agents and human maintainers) working in this repository. These rules are designed to maintain codebase consistency, reliability, and collaborative efficiency. Please adhere strictly unless a stronger project-specific rationale exists.

---

## Build, Lint, and Test Commands

### Bootstrapping and Setup
- The primary language is Go (see `go.mod`, minimum version 1.24.0).
- Dependencies are managed through `go.mod` and `go.sum`.
- No package.json or Makefile: use native Go/CI commands.

### Linting
- All linting is done using **golangci-lint**.
- To lint all code:
  ```sh
  golangci-lint run ./...
  ```
  - The official GitHub workflow runs this with the latest available golangci-lint.
- Fix lint issues as errors or warnings—never ignore linters unless approved in PR/review.

### Testing
- All substantive tests are written with **Ginkgo** and assertions use **Gomega** (see `go.mod`).
- To run all tests:
  ```sh
  go run github.com/onsi/ginkgo/v2/ginkgo ./...
  ```
- To run tests in a specific package:
  ```sh
  go run github.com/onsi/ginkgo/v2/ginkgo ./path/to/package
  ```
- To run a single test (by name, e.g. 'MyFeature'):
  ```sh
  go run github.com/onsi/ginkgo/v2/ginkgo -focus='MyFeature' ./path/to/package
  ```
- Write new tests in `*_test.go` files, colocated with the code, following the structure already in `cmd/`, `internal/` submodules.
- For end-to-end test orchestration, match/test `main.go`-based flows (see `cmd/`).

### Build
- Build the project binary with:
  ```sh
  go build -o home-gate main.go
  ```
- Container builds use Docker, see `Dockerfile` and `.github/workflows/docker.yml`.
- The CI uses the following Go version: 1.23 (workflow), but module sets 1.24 (use module if locally validating).

### Project Environment
- If your code needs custom environment variables, document these inline and in PRs. Viper provides binding for flags and environment variables (see `cmd/monitor.go`).
- No .env file is present by default, but scripts or the environment may export `FRITZBOX_USERNAME` and `FRITZBOX_PASSWORD` for Fritzbox access.

---

## Code Style & Conventions

### Imports
- Use grouped imports: standard, third-party, local (in that order).
- Avoid unnecessary aliasing unless required for clarity (see e.g. `fritzbox "home-gate/internal/fritzbox"`).
- Prefer explicit imports, not wildcard or dot imports (except as per Ginkgo idioms).

### Formatting
- Follow `gofmt` (automatically enforce with `gofmt -s -w .` before every commit).
- 4-space indentation (Go default), tabs are allowed as per Go conventions.
- Max 120 columns recommended for code lines.

### Types & Typedefs
- Clearly define struct and interface types with explanatory comments for exported types (see `type FritzboxLibClient interface`, etc.).
- Use concrete struct fields for package-level config (see `struct fritzboxClient {...}`).
- Prefer interfaces for dependencies, especially for testability (fakes/mocks).
- Place `//go:generate` directives for fakes/stubs directly above target interfaces.

### Naming Conventions
- Use `CamelCase` for exported names, `camelCase` for locals.
- Package names are short (avoid underscores/dashes).
- Test function names start with `TestXxx` (as required by Go), and BDD test specs can use Ginkgo's Describe/Context/It.
- Use `ErrXxx` or `errorXxx` for error variables/types.
- Do not abbreviate outside of well-established Go conventions (cfg, err, min, etc. are fine).

### Error Handling
- Prefer idiomatic Go error handling: `if err != nil { ... }`.
- Wrap errors with context for diagnosis:
  ```go
  return fmt.Errorf("fetch: %w", err)
  ```
- Fatal errors during CLI runtime should call `log.Fatal`.
- Function comments should indicate possible error returns for exported/public APIs.

### File and Package Structure
- CLI code: `cmd/`
- Business logic: `internal/` split into clear modules (e.g. `policy/`, `fritzbox/`)
- Fakes for testing (using `maxbrunsfeld/counterfeiter`): `_test.go` and `/fakes/` subdirs.

### Env, Flags and Config
- All command flags must be defined using Cobra, configuration values available to Viper.
- Bindings for environment variables must be clearly specified for agent consumption/testing.

### Comments and Documentation
- Use full sentences in comments for exported functions/types.
- Keep comments up-to-date—remove outdated/incorrect docstrings.

### Test Fakes, Mocks, and Stubs
- Use `counterfeiter/v6` for generating fakes—see `//go:generate` directives. Do not check in generated code unless explicitly instructed.
- Place fake types into `fakes/` subdirectories local to their implementation.

### General Practices
- Run all tests before submitting any PR/commit (and ensure CI is green).
- Document any deviations from these rules in PR descriptions and code comments.
- Agents must not commit secrets or .env files—warn if such files are detected in the diff.
- All agentic code should assume the latest Go idioms unless the codebase moves to a different version.
- There are currently no Cursor/Copilot/agent instruction rules in this repo.

---

## References
- [Go Style Guide](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [Ginkgo Docs](https://onsi.github.io/ginkgo/)
- [GolangCI-Lint](https://golangci-lint.run/)

---

*Update this file if the project’s structure, CI workflows, or coding conventions change significantly.*
