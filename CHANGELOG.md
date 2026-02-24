# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.22] - 2026-02-24

### Fixed
- `_extract_text()` now handles `CallToolResult` objects with `.content` attribute (previously fell through returning the raw object)
- `call_tool()` error detection now checks `isError` on `CallToolResult` objects at the top level (previously only checked per-item in lists)

### Changed
- Bumped `fastmcp` dependency from `>=3.0.1` to `>=3.0.2`

## [0.1.21] - 2026-02-24

### Added
- Resilience primitives for production deployments: circuit breaker, retry with exponential backoff, error budgets, per-call timeouts
- Configurable `max_response_bytes` with manifest summarisation for large responses
- Response payload sanitisation (credential scrubbing in output)

### Changed
- Documented response payload management and resilience features in README

## [0.1.20] - 2026-02-23

### Changed
- Improved PyPI discoverability: expanded description, keywords, classifiers

## [0.1.19] - 2026-02-22

### Fixed
- Suppressed gosec G706 false positive with nolint directive for CLI flag address binding
- Resolved gosec G706 log injection lint finding
- Addressed security, correctness, and code quality audit findings (round 3)

## [0.1.18] - 2025-02-23

### Added
- SECURITY.md with vulnerability reporting policy and security architecture overview
- CONTRIBUTING.md with development setup, code style, and PR guidelines
- Comprehensive CI/CD pipeline (Go lint, test, build + Python lint, test, build + integration tests)
- Trusted Publishing for PyPI releases (no API tokens)
- Platform-specific Python wheels with bundled Go binary and SHA256 verification
- 44 MCP tools covering every operational Helm CLI command
- Dual Helm SDK support (v3 and v4) via native Go SDK
- Three transport modes: stdio, HTTP (Streamable HTTP), SSE
- Cloud provider support: EKS (AWS), GKE (Google Cloud), AKS (Azure)
- Linux process hardening (PR_SET_DUMPABLE, capability dropping, credential zeroing)
- Credential scrubbing for bearer tokens, basic auth, and URL-embedded passwords
- Input validation with SSRF protection, path traversal prevention, DNS-1123 validation
- HTTP server hardening (timeouts, max header bytes, graceful shutdown)
- Forward proxy support (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
- FastMCP-based Python wrapper with auto-discovery of all Go tools
- Docker support with non-root user and read-only kubeconfig mount
- Python client and server APIs with FastMCP composition support

[Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.22...HEAD
[0.1.22]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.21...v0.1.22
[0.1.21]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.20...v0.1.21
[0.1.20]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.19...v0.1.20
[0.1.19]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.18...v0.1.19
[0.1.18]: https://github.com/SCGIS-Wales/helm-mcp/releases/tag/v0.1.18
