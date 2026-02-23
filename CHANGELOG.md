# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- CODE_OF_CONDUCT.md (Contributor Covenant v2.1)
- CHANGELOG.md (this file)
- GitHub issue templates for bug reports and feature requests
- Expanded PyPI metadata: author name, maintainer, keywords, classifiers
- Python versions and downloads badges on both READMEs

### Changed
- Updated LICENSE copyright holder to Dejan Gregor
- Updated glama.json maintainer to PyPI username

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

[Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.18...HEAD
[0.1.18]: https://github.com/SCGIS-Wales/helm-mcp/releases/tag/v0.1.18
