# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |
| < latest | :x:               |

Only the latest release receives security updates. We recommend always running the most recent version.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them privately via [GitHub Security Advisories](https://github.com/SCGIS-Wales/helm-mcp/security/advisories/new).

Include as much of the following information as possible:

- Type of issue (e.g., buffer overflow, credential exposure, path traversal, SSRF)
- Full paths of source file(s) related to the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit it

You should receive a response within 48 hours. If the issue is confirmed, a patch will be released as soon as possible depending on complexity.

## Security Measures

helm-mcp implements several layers of security:

### Process Hardening (Linux)

- `PR_SET_DUMPABLE(0)` blocks ptrace attach, core dumps, and `/proc/pid/mem` reads
- All Linux capabilities are dropped from the bounding set
- Credential memory is zeroed via `defer` after every tool handler completes

### Input Validation

- Release names validated against DNS-1123
- Kubeconfig paths checked for path traversal, symlinks, and sensitive paths
- URLs validated with SSRF protection (private IP blocking)
- Plugin names restricted to prevent argument injection
- Timeout durations capped at 24 hours

### Credential Protection

- Bearer tokens, basic auth, and URL-embedded passwords are scrubbed from error messages
- Repository config files written with `0600` permissions
- Config directories created with `0700` permissions

### HTTP Server Hardening

- Read/Write/Idle timeouts prevent slow client and connection exhaustion attacks
- MaxHeaderBytes limit prevents header-based DoS
- Graceful shutdown with timeout

## Dependencies

We monitor dependencies for known vulnerabilities using `govulncheck` (Go) and keep Python dependencies up to date. Our CI pipeline runs security checks on every pull request.
