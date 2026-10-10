# Contributing to helm-mcp

Thank you for your interest in contributing to helm-mcp! This project is open source under the [MIT License](LICENSE).

## Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/<your-username>/helm-mcp.git
   cd helm-mcp
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feat/your-feature
   ```

## Development Setup

### Go (MCP Server)

Requires Go 1.27.2+ (the version in `go.mod`).

```bash
# Build
make build

# Run tests
make test

# Lint (requires golangci-lint v2; CI pins v2.14.0)
make lint

# Security scan
make security
```

### Python (FastMCP Wrapper)

Requires Python 3.12+. CI tests 3.12, 3.13 and 3.14.

```bash
cd python

# Install with dev dependencies
pip install -e ".[dev]"

# Run tests
pytest -v tests/

# Lint and type check
ruff check src/ tests/
ruff format --check src/ tests/
mypy src/
```

## Making Changes

### Adding a New Helm Tool

1. Add the engine method to `internal/helmengine/engine.go`
2. Implement in both `internal/helmengine/v3/` and `internal/helmengine/v4/`
3. Create the MCP tool handler in the appropriate `internal/tools/` package
4. Register the tool in `internal/server/server.go`
5. Add tests for both engine implementations and the tool handler

The Python package will automatically discover new tools via the MCP protocol.

### Adding an Argument to an Existing Tool

1. Add the field to the options type in `internal/helmengine/types.go` and to
   the tool input struct. Shared groups (chart source, value sets,
   post-renderer) live in `internal/tools/sources.go`.
2. Wire it into the Helm action in **both** engines. If Helm v3 has no
   equivalent, reject it in the v3 engine with a clear error rather than
   ignoring it.
3. Validate any file path or URL it accepts in the tool handler.
4. Add an engine test that proves the field changes Helm's behaviour (render
   or dry run install a chart), not only that it is passed along. Several
   arguments used to be accepted and silently dropped.

### Code Style

- **Go**: Follow standard Go conventions. Run `make lint` before submitting.
- **Python**: Follow PEP 8. Run `ruff check`, `ruff format --check` and `mypy src/` before submitting.

### Commit Messages

Use conventional commit prefixes:
- `feat:` — New features
- `fix:` — Bug fixes
- `docs:` — Documentation changes
- `refactor:` — Code refactoring without behavior changes
- `test:` — Adding or updating tests
- `chore:` — Build, CI, or tooling changes

### Tests

- All new features must include tests
- Run the full test suite before submitting:
  ```bash
  make test                # Go tests
  cd python && pytest -v   # Python tests
  ```
- Go tests should use race detection (`-race` flag, included in `make test`)

## Pull Requests

1. Ensure all CI checks pass (Go lint, Go test, Go build, Python lint, Python test)
2. Keep PRs focused on a single change
3. Update documentation if your change affects the public API or user-facing behavior
4. PRs are automatically merged once all checks pass

## Reporting Issues

- Use [GitHub Issues](https://github.com/SCGIS-Wales/helm-mcp/issues) for bug reports and feature requests
- Include steps to reproduce for bug reports
- Include Go/Python version and OS information

## Releases

Releases are automated. Every merge to `main` that passes CI will:
1. Increment the version: the patch by default, or the minor (while on 0.x)
   when a commit is marked breaking (`feat!:`, `fix!:` or a
   `BREAKING CHANGE:` trailer)
2. Create a git tag
3. Build and publish Go binaries to GitHub Releases
4. Publish the Python package to PyPI and the container image to GHCR
5. Publish the server to the MCP Registry

For a major version bump, create the tag manually before the next merge.

## License

By contributing to helm-mcp, you agree that your contributions will be licensed under the [MIT License](LICENSE).
