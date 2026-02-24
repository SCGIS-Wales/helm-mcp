# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]


## [0.1.24] - 2026-02-24

### Added
- **Automated CHANGELOG updates**: Added `scripts/update_changelog.py` that extracts PR `## Summary` sections via the GitHub API and injects them into CHANGELOG.md under the correct version heading, with proper Keep a Changelog categorisation (Added/Fixed/Changed) based on PR title prefixes (feat/fix/docs/chore/refactor/perf) ([#31](https://github.com/SCGIS-Wales/helm-mcp/pull/31))
- **CI integration**: Added `update-changelog` job to CI pipeline that runs after `auto-tag` and `github-release`, automatically committing CHANGELOG updates with `[skip ci]` to prevent recursive triggers ([#31](https://github.com/SCGIS-Wales/helm-mcp/pull/31))
- **Comprehensive test coverage**: Added 22 unit tests in `scripts/test_update_changelog.py` covering PR classification, summary-to-changelog conversion (bullet extraction, table/code-block/header skipping, PR link injection), and CHANGELOG file manipulation (version insertion, link updates) ([#31](https://github.com/SCGIS-Wales/helm-mcp/pull/31))
- **Full CHANGELOG retrofit**: Backfilled meaningful entries for all 24 releases (v0.1.0 through v0.1.23) with accurate descriptions sourced from actual PR summaries, replacing the minimal placeholder entries ([#31](https://github.com/SCGIS-Wales/helm-mcp/pull/31))

## [0.1.23] - 2026-02-24

### Changed
- Backfilled CHANGELOG.md with entries for all releases from v0.1.0 through v0.1.22 ([#30](https://github.com/SCGIS-Wales/helm-mcp/pull/30))

## [0.1.22] - 2026-02-24

### Fixed
- `_extract_text()` now unwraps `CallToolResult` objects (which have `.content` list instead of `.text`) before processing, preventing the raw object from falling through unchanged ([#29](https://github.com/SCGIS-Wales/helm-mcp/pull/29))
- `call_tool()` now checks `isError` directly on `CallToolResult` objects at the top level; previously only checked per-item in lists, so errors from FastMCP >=3.0.1 were silently ignored ([#29](https://github.com/SCGIS-Wales/helm-mcp/pull/29))

### Changed
- Bumped `fastmcp` dependency from `>=3.0.1` to `>=3.0.2` ([#29](https://github.com/SCGIS-Wales/helm-mcp/pull/29))

## [0.1.21] - 2026-02-24

### Added
- **Response budget/truncation**: Automatic response size limiting (256KB default) with truncation metadata in `TextResult()`, preventing LLM context window overflow from large Helm outputs like manifests and values ([#28](https://github.com/SCGIS-Wales/helm-mcp/pull/28))
- **Circuit breaker**: Three-state (Closed/Open/HalfOpen) pattern to fail fast when Kubernetes API or Helm backends are unavailable, with configurable threshold and recovery timeout ([#28](https://github.com/SCGIS-Wales/helm-mcp/pull/28))
- **Retry with exponential backoff**: Context-aware retry helper with jitter, configurable max attempts, and retryable error filtering for idempotent operations ([#28](https://github.com/SCGIS-Wales/helm-mcp/pull/28))
- **Per-tool context timeouts**: Category-based defaults (query: 30s, mutate: 120s, chart: 60s, repo: 60s) that respect existing parent context deadlines ([#28](https://github.com/SCGIS-Wales/helm-mcp/pull/28))
- Response payload credential scrubbing in output ([#28](https://github.com/SCGIS-Wales/helm-mcp/pull/28))

## [0.1.20] - 2026-02-23

### Added
- CODE_OF_CONDUCT.md (Contributor Covenant v2.1) ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))
- SECURITY.md with vulnerability reporting policy and security architecture overview ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))
- CHANGELOG.md in Keep a Changelog format ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))
- GitHub issue templates for bug reports (with transport mode, Helm version, OS dropdowns) and feature requests ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))
- Python versions and monthly downloads badges on both READMEs ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))

### Changed
- Expanded PyPI keywords from 6 to 16 and added classifiers for better discoverability ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))
- Updated LICENSE copyright holder to Dejan Gregor ([#27](https://github.com/SCGIS-Wales/helm-mcp/pull/27))

## [0.1.19] - 2026-02-22

### Fixed
- **ValidatePath symlink vulnerability**: Now resolves symlinks before validation, preventing symlink-based file reads via `values_files` ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- **Upgrade DryRun default case**: Returns error for invalid values in both v3 and v4 (v3 previously fell through silently) ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- **mergeValues file validation**: Validates values files exist before passing to Helm SDK for clearer error messages ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- **Shared mcp.Server concurrency**: Each HTTP/SSE request now gets its own `mcp.Server` instance to prevent race conditions ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))

### Added
- `ValidateURL` now accepts optional context for cancellable DNS resolution with 10s default timeout ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- HTTP/SSE bearer token authentication via `HELM_MCP_AUTH_TOKEN` environment variable ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- Structured logging via `log/slog` in `main.go` and v3 engine ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))

### Changed
- Python `_reconnect` uses exponential backoff (`min(2^attempt, 30)s`) instead of immediate retry to avoid CPU burn on subprocess crashes ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))
- `_extract_text()` now attempts JSON parsing for single text blocks and warns on multi-block responses ([#26](https://github.com/SCGIS-Wales/helm-mcp/pull/26))

## [0.1.18] - 2026-02-22

### Fixed
- `helm_list` with `all_namespaces=true` returned empty results because the storage driver was scoped to the default namespace; now re-initialises the action config with empty namespace when `AllNamespaces` is true, matching `helm` CLI behaviour ([#25](https://github.com/SCGIS-Wales/helm-mcp/pull/25))

## [0.1.17] - 2026-02-22

### Fixed
- `helm_install`/`helm_upgrade` failing with "requires a wait strategy" on Helm v4 when `wait: false` — WaitStrategy must always be set in v4 SDK ([#22](https://github.com/SCGIS-Wales/helm-mcp/pull/22))
- `helm_list` with `all_namespaces: true` returning no releases when namespace is set — clear namespace in engine layer when AllNamespaces requested ([#22](https://github.com/SCGIS-Wales/helm-mcp/pull/22))
- Replaced `net.LookupHost` with `net.DefaultResolver.LookupHost(ctx)` in `ValidateURL`'s SSRF check to fix `noctx` lint failure ([#24](https://github.com/SCGIS-Wales/helm-mcp/pull/24))

### Added
- SSRF protection in `ValidateURL`: resolves hostnames via DNS and blocks private IP ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 169.254.0.0/16, ::1/128, fe80::/10, fc00::/7) ([#22](https://github.com/SCGIS-Wales/helm-mcp/pull/22))
- Sensitive path rejection: blocks `/etc/shadow`, `/etc/passwd`, `/proc/`, `/dev/`, `/sys/` as kubeconfig paths ([#22](https://github.com/SCGIS-Wales/helm-mcp/pull/22))
- `ValidateGlobalInput` wired to all 33 tool handlers (was only 11 after PR #20) ([#22](https://github.com/SCGIS-Wales/helm-mcp/pull/22))

### Changed
- `make build` now depends on `lint` so `golangci-lint` always runs before a local build ([#23](https://github.com/SCGIS-Wales/helm-mcp/pull/23))

## [0.1.16] - 2026-02-22

### Changed
- Bumped `fastmcp` minimum from `>=3.0.0` to `>=3.0.1` for non-serializable state preservation fixes and OIDC `verify_id_token` support ([#21](https://github.com/SCGIS-Wales/helm-mcp/pull/21))

## [0.1.15] - 2026-02-22

### Fixed
- **Plugin argument injection (P0)**: Added `ValidatePluginName()` and `--` separator before positional args in all plugin exec calls to prevent flag injection ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))
- **Unused security validators (P0)**: Wired `ValidateReleaseName`, `ValidateNamespace`, `ValidateTimeout`, `ValidatePluginName`, `ValidateURL`, `ValidatePath` into all 25+ tool handlers ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))
- **Password not zeroed (P1)**: Added `defer opts.ZeroPassword()` for repo/add, registry/login, chart/pull ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))
- Replaced `http.DefaultClient` with dedicated `artifactHubClient` (30s timeout) ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))
- Moved 3 credential scrubbing regexes to package-level `var` declarations (compiled once instead of per-call) ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))

### Changed
- `NewServer()` now accepts version parameter, wired from ldflags-injected `main.version` ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))
- Renamed `ZeroSensitiveFields()` to `ZeroBearerToken()` for clarity ([#20](https://github.com/SCGIS-Wales/helm-mcp/pull/20))

## [0.1.14] - 2026-02-22

### Added
- **Linux process hardening**: `PR_SET_DUMPABLE(0)` blocks ptrace attach, core dumps, and `/proc/pid/mem` reads, preventing credential inspection by other processes ([#19](https://github.com/SCGIS-Wales/helm-mcp/pull/19))
- **Capability dropping**: Drops all capabilities from the bounding set to protect against privilege escalation in misconfigured Docker/K8s environments ([#19](https://github.com/SCGIS-Wales/helm-mcp/pull/19))
- **Credential memory zeroing**: `ZeroCredentials()` called via `defer` after every tool handler, with `SecureBytes` type for explicit credential lifecycle management ([#19](https://github.com/SCGIS-Wales/helm-mcp/pull/19))
- `--no-harden` flag to disable hardening for debugging (strace, delve) ([#19](https://github.com/SCGIS-Wales/helm-mcp/pull/19))

## [0.1.13] - 2026-02-21

### Fixed
- `pip install helm-mcp` resulted in `permission denied` because `build_wheels.py` placed the binary in `.data/scripts/` with Windows create_system flag, causing pip to not preserve Unix execute permissions ([#18](https://github.com/SCGIS-Wales/helm-mcp/pull/18))
- Go binary now bundled inside `helm_mcp/bin/` package directory with a `console_scripts` entry point that finds the binary, `chmod` it if needed, and `os.execvp()` it ([#18](https://github.com/SCGIS-Wales/helm-mcp/pull/18))

### Added
- `wheel-install-test` CI job on Ubuntu 22.04 that builds a platform wheel, installs in a virtualenv, and verifies both `helm-mcp` and `helm-mcp-python` commands work ([#18](https://github.com/SCGIS-Wales/helm-mcp/pull/18))

## [0.1.12] - 2026-02-21

### Added
- Multi-platform Docker container images (`linux/amd64`, `linux/arm64`) published to `ghcr.io/scgis-wales/helm-mcp` on each release ([#17](https://github.com/SCGIS-Wales/helm-mcp/pull/17))
- Dockerfile uses `golang:1-alpine` builder and `alpine:latest` runtime ([#17](https://github.com/SCGIS-Wales/helm-mcp/pull/17))
- `.dockerignore` to exclude non-essential files from build context ([#17](https://github.com/SCGIS-Wales/helm-mcp/pull/17))
- Container tagged with `latest`, full semver, and major.minor on each release ([#17](https://github.com/SCGIS-Wales/helm-mcp/pull/17))

## [0.1.11] - 2026-02-21

### Fixed
- `publish-pypi` CI job downloaded platform wheels then ran `actions/checkout@v4` which wiped the workspace, destroying the `dist/` directory; reordered steps so checkout happens first ([#16](https://github.com/SCGIS-Wales/helm-mcp/pull/16))
- This is why v0.1.10 never appeared on PyPI despite the GitHub Release being created successfully ([#16](https://github.com/SCGIS-Wales/helm-mcp/pull/16))

## [0.1.10] - 2026-02-21

### Added
- **Platform-specific Python wheels**: Bundle pre-compiled Go binary directly inside wheels so `pip install helm-mcp` works in air-gapped environments with zero network calls and zero manual setup ([#15](https://github.com/SCGIS-Wales/helm-mcp/pull/15))
- **Resilient async tool wrappers**: `HelmClient` context manager with typed async functions for all 44 MCP tools, auto-reconnect, configurable timeouts, and structured error hierarchy (`HelmError` / `HelmTimeoutError` / `HelmConnectionError` / `HelmToolError`) ([#15](https://github.com/SCGIS-Wales/helm-mcp/pull/15))
- Wheels published for Linux (x86_64, aarch64), macOS (x86_64, arm64), and Windows (amd64) with universal `py3-none-any` fallback ([#15](https://github.com/SCGIS-Wales/helm-mcp/pull/15))

### Changed
- Promoted to Production/Stable status; Python >=3.12 required ([#15](https://github.com/SCGIS-Wales/helm-mcp/pull/15))

## [0.1.9] - 2026-02-21

### Fixed
- Extracted duplicate `parseDuration` to shared `helmengine.ParseDuration` (was copy-pasted in v3 and v4) ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))
- Silent error swallowing in `GetMetadata` — `strconv.Atoi` and `time.Parse` errors now returned instead of discarded ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))
- Deduplicated shutdown goroutine in `main.go` into `gracefulShutdown()` helper ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))

### Added
- `--debug` flag on Go MCP server for verbose stderr logging ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))
- `--verbose`/`-v` flag on Python CLI with structured logging for binary discovery and server creation ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))
- 8 unit tests for `SearchArtifactHub` (httptest mocking) and 3 for `ParseDuration` ([#14](https://github.com/SCGIS-Wales/helm-mcp/pull/14))

## [0.1.8] - 2026-02-21

### Added
- Comprehensive mapping table in README showing every `helm` CLI command and its corresponding MCP tool — 44 of 44 operational commands covered ([#13](https://github.com/SCGIS-Wales/helm-mcp/pull/13))

## [0.1.7] - 2026-02-21

### Added
- **OCI registry support**: Initialise `registry.NewClient()` in both v3/v4 engines so `oci://` chart references work with Docker Hub, GHCR, and other OCI registries ([#11](https://github.com/SCGIS-Wales/helm-mcp/pull/11))
- **Artifact Hub search**: `helm_search_hub` now queries the Artifact Hub REST API directly instead of returning a guidance error ([#11](https://github.com/SCGIS-Wales/helm-mcp/pull/11))
- **68 integration tests** covering all 44 MCP tools against a real k3d cluster, with CI job in GitHub Actions ([#11](https://github.com/SCGIS-Wales/helm-mcp/pull/11))

### Fixed
- v4 uninstall "wait strategy not set" — `WaitStrategy` now always set to `kube.StatusWatcherStrategy` ([#11](https://github.com/SCGIS-Wales/helm-mcp/pull/11))
- v4 rollback "invalid server-side apply method" — set `"false"` when SSA not enabled ([#11](https://github.com/SCGIS-Wales/helm-mcp/pull/11))
- `errcheck`, `gosec`, and `staticcheck` findings in `artifacthub.go` ([#12](https://github.com/SCGIS-Wales/helm-mcp/pull/12))

## [0.1.6] - 2026-02-21

### Changed
- Updated copyright year in LICENSE file

## [0.1.5] - 2026-02-21

### Added
- **Auto-download Go binary**: `pip install helm-mcp` now just works — binary auto-downloaded from GitHub Releases on first use with SHA256 checksum verification, installed to Python scripts directory (on PATH) ([#10](https://github.com/SCGIS-Wales/helm-mcp/pull/10))

## [0.1.4] - 2026-02-21

### Added
- PyPI badges on Python README for better package visibility

## [0.1.3] - 2026-02-21

### Fixed
- Resolved all 16 SonarQube code smells (duplicated string literals, cognitive complexity, empty functions) ([#9](https://github.com/SCGIS-Wales/helm-mcp/pull/9))

### Added
- Achieved 99.6% test coverage across all files (80%+ per file) with comprehensive unit tests for mock engine, v3/v4 engine utilities, chart/release error paths, and Python CLI ([#9](https://github.com/SCGIS-Wales/helm-mcp/pull/9))
- `sonar-project.properties` for local SonarQube scanning ([#9](https://github.com/SCGIS-Wales/helm-mcp/pull/9))

## [0.1.2] - 2026-02-21

### Fixed
- SonarQube code smells: duplicated string literals, cognitive complexity, empty functions ([#9](https://github.com/SCGIS-Wales/helm-mcp/pull/9))

## [0.1.1] - 2026-02-21

### Fixed
- Guard against nil `msg.Err` in Helm lint message processing (v3/v4) to prevent nil pointer dereference ([#8](https://github.com/SCGIS-Wales/helm-mcp/pull/8))
- Graceful error handling for missing binary in Python CLI (`FileNotFoundError` catch) ([#8](https://github.com/SCGIS-Wales/helm-mcp/pull/8))

### Changed
- Made README community-friendly: added table of contents, "Why helm-mcp?" section, community links, and welcoming contributor messaging ([#8](https://github.com/SCGIS-Wales/helm-mcp/pull/8))

## [0.1.0] - 2026-02-21

### Added
- Complete MCP server exposing **44 Helm tools** via the Model Context Protocol using the native Go Helm SDK ([#1](https://github.com/SCGIS-Wales/helm-mcp/pull/1))
- Dual **Helm v3 + v4** support — every tool input has a `helm_version` field to select the backend ([#1](https://github.com/SCGIS-Wales/helm-mcp/pull/1))
- **Python package** (`helm-mcp` on PyPI) wrapping the Go binary via FastMCP 3.0 proxy ([#1](https://github.com/SCGIS-Wales/helm-mcp/pull/1))
- CI/CD pipeline with Go lint/test/build, Python lint/test/build, and release workflow ([#1](https://github.com/SCGIS-Wales/helm-mcp/pull/1))
- Auto-tag and release on merge to main: semver patch bump, version embedding via ldflags, GitHub Release with binaries and checksums, PyPI publishing via OIDC ([#5](https://github.com/SCGIS-Wales/helm-mcp/pull/5))
- CONTRIBUTING.md with development setup, code style, and PR guidelines ([#6](https://github.com/SCGIS-Wales/helm-mcp/pull/6))
- Comprehensive README with installation instructions, Python/FastMCP integration examples, and MCP client configuration ([#6](https://github.com/SCGIS-Wales/helm-mcp/pull/6))

### Fixed
- Corrected golangci-lint v2 install path; made `govulncheck` advisory ([#2](https://github.com/SCGIS-Wales/helm-mcp/pull/2))
- Simplified embedded field selectors in v3 and v4 release/chart methods (staticcheck QF1008) ([#3](https://github.com/SCGIS-Wales/helm-mcp/pull/3), [#4](https://github.com/SCGIS-Wales/helm-mcp/pull/4))
- Auto-tag version bump no longer fails when version files already match the target version ([#7](https://github.com/SCGIS-Wales/helm-mcp/pull/7))

[Unreleased]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.24...HEAD
[0.1.24]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.23...v0.1.24
[0.1.23]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.22...v0.1.23
[0.1.22]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.21...v0.1.22
[0.1.21]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.20...v0.1.21
[0.1.20]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.19...v0.1.20
[0.1.19]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.18...v0.1.19
[0.1.18]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.17...v0.1.18
[0.1.17]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.16...v0.1.17
[0.1.16]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.15...v0.1.16
[0.1.15]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.14...v0.1.15
[0.1.14]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.13...v0.1.14
[0.1.13]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.12...v0.1.13
[0.1.12]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.11...v0.1.12
[0.1.11]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.10...v0.1.11
[0.1.10]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.9...v0.1.10
[0.1.9]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.8...v0.1.9
[0.1.8]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/SCGIS-Wales/helm-mcp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/SCGIS-Wales/helm-mcp/releases/tag/v0.1.0
