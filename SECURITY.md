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

helm-mcp implements multiple security layers, following the [MCP Security Best Practices](https://modelcontextprotocol.io/specification/2025-03-26/basic/security).

### Authentication

When serving over HTTP or SSE, helm-mcp can optionally require OIDC/OAuth2 authentication. Incoming JWTs are verified against the identity provider's JWKS endpoint, checking the signature, issuer, audience, expiry, and authorized party. Both Microsoft Entra ID v1 and v2 token formats are supported. The JWKS keys are cached locally and automatically refreshed when a previously unseen key ID appears (to handle key rotation).

Authorization decisions can additionally require specific OAuth2 scopes or app roles to be present in the token.

### Downstream Token Exchange (OBO)

When helm-mcp needs to call a downstream API on behalf of the calling user, it uses the On-Behalf-Of (OBO) flow rather than forwarding the original token. This means each hop in the call chain receives a token scoped specifically to that service's audience, preventing confused-deputy attacks. If Conditional Access policies block the exchange, the `interaction_required` error is surfaced to the caller for re-authentication.

### Session Cache

Validated tokens are held in a bounded in-memory cache keyed by the SHA-256 hash of the raw bearer token (raw tokens are not stored as map keys). Each access resets a sliding inactivity timer (configurable via `HELM_MCP_SESSION_TTL`, default 5 minutes), and tokens are never served past their `exp` claim. The cache is capped at 10,000 entries with oldest-first eviction.

### Audit Logging

Every authentication and authorization decision produces a structured `slog` event (`security_audit`), capturing the principal, tenant, client app, scopes/roles, action, outcome, and latency. These records are designed for ingestion into centralized log platforms.

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
