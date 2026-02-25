"""Resilience configuration and setup for helm-mcp.

All settings are configurable via environment variables with the ``HELM_MCP_``
prefix.  When no environment variables are set, the defaults provide sensible
production behaviour that is backward-compatible with the existing system.

Proxy-level patterns (applied via FastMCP middleware)
-----------------------------------------------------
- **RetryMiddleware** — automatic retry with exponential backoff
- **RateLimitingMiddleware** — token-bucket rate limiting
- **ResponseCachingMiddleware** — TTL-based response caching
- **ErrorHandlingMiddleware** — structured MCP error responses
- **TimingMiddleware** — request timing instrumentation

Tool-level patterns (applied in HelmClient)
--------------------------------------------
- **circuitbreaker** — circuit breaker on external subprocess calls
- **tenacity** — retry with exponential backoff + jitter
- **asyncio.Semaphore** — bulkhead / concurrency limiter

Observability
-------------
- **OpenTelemetry** — opt-in tracing via ``HELM_MCP_OTEL_ENABLED``
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Environment variable helpers
# ---------------------------------------------------------------------------


def _env_bool(key: str, default: bool) -> bool:
    """Read a boolean from an environment variable."""
    val = os.environ.get(key, "").strip().lower()
    if not val:
        return default
    return val in ("1", "true", "yes", "on")


def _env_float(key: str, default: float) -> float:
    """Read a float from an environment variable."""
    val = os.environ.get(key)
    if val is None:
        return default
    return float(val)


def _env_int(key: str, default: int) -> int:
    """Read an int from an environment variable."""
    val = os.environ.get(key)
    if val is None:
        return default
    return int(val)


# ---------------------------------------------------------------------------
# Configuration dataclasses
# ---------------------------------------------------------------------------


@dataclass(frozen=True)
class RetryConfig:
    """FastMCP RetryMiddleware configuration (proxy-level).

    Env vars::

        HELM_MCP_RETRY_ENABLED           (bool, default: true)
        HELM_MCP_RETRY_MAX_RETRIES       (int,  default: 2)
        HELM_MCP_RETRY_BASE_DELAY        (float, default: 1.0)
        HELM_MCP_RETRY_MAX_DELAY         (float, default: 30.0)
        HELM_MCP_RETRY_BACKOFF_MULTIPLIER (float, default: 2.0)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_RETRY_ENABLED", True))
    max_retries: int = field(default_factory=lambda: _env_int("HELM_MCP_RETRY_MAX_RETRIES", 2))
    base_delay: float = field(default_factory=lambda: _env_float("HELM_MCP_RETRY_BASE_DELAY", 1.0))
    max_delay: float = field(default_factory=lambda: _env_float("HELM_MCP_RETRY_MAX_DELAY", 30.0))
    backoff_multiplier: float = field(
        default_factory=lambda: _env_float("HELM_MCP_RETRY_BACKOFF_MULTIPLIER", 2.0)
    )


@dataclass(frozen=True)
class RateLimitConfig:
    """FastMCP RateLimitingMiddleware configuration (proxy-level).

    Env vars::

        HELM_MCP_RATE_LIMIT_ENABLED  (bool,  default: false)
        HELM_MCP_RATE_LIMIT_MAX_RPS  (float, default: 10.0)
        HELM_MCP_RATE_LIMIT_BURST    (int,   default: 20)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_RATE_LIMIT_ENABLED", False))
    max_requests_per_second: float = field(
        default_factory=lambda: _env_float("HELM_MCP_RATE_LIMIT_MAX_RPS", 10.0)
    )
    burst_capacity: int = field(default_factory=lambda: _env_int("HELM_MCP_RATE_LIMIT_BURST", 20))


@dataclass(frozen=True)
class CacheConfig:
    """FastMCP ResponseCachingMiddleware configuration (proxy-level).

    Env vars::

        HELM_MCP_CACHE_ENABLED   (bool, default: false)
        HELM_MCP_CACHE_TOOL_TTL  (int,  default: 300)
        HELM_MCP_CACHE_LIST_TTL  (int,  default: 60)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_CACHE_ENABLED", False))
    tool_ttl: int = field(default_factory=lambda: _env_int("HELM_MCP_CACHE_TOOL_TTL", 300))
    list_ttl: int = field(default_factory=lambda: _env_int("HELM_MCP_CACHE_LIST_TTL", 60))


@dataclass(frozen=True)
class ErrorHandlingConfig:
    """FastMCP ErrorHandlingMiddleware configuration (proxy-level).

    Env vars::

        HELM_MCP_ERROR_HANDLING_ENABLED   (bool, default: true)
        HELM_MCP_ERROR_INCLUDE_TRACEBACK  (bool, default: false)
    """

    enabled: bool = field(
        default_factory=lambda: _env_bool("HELM_MCP_ERROR_HANDLING_ENABLED", True)
    )
    include_traceback: bool = field(
        default_factory=lambda: _env_bool("HELM_MCP_ERROR_INCLUDE_TRACEBACK", False)
    )


@dataclass(frozen=True)
class TimingConfig:
    """FastMCP TimingMiddleware configuration (proxy-level).

    Env vars::

        HELM_MCP_TIMING_ENABLED   (bool, default: true)
        HELM_MCP_TIMING_DETAILED  (bool, default: false)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_TIMING_ENABLED", True))
    detailed: bool = field(default_factory=lambda: _env_bool("HELM_MCP_TIMING_DETAILED", False))


@dataclass(frozen=True)
class CircuitBreakerConfig:
    """Circuit breaker configuration for ``HelmClient.call_tool()`` (tool-level).

    Uses the ``circuitbreaker`` library to protect against repeated Go
    subprocess failures.

    Env vars::

        HELM_MCP_CIRCUIT_BREAKER_ENABLED        (bool,  default: true)
        HELM_MCP_CIRCUIT_BREAKER_FAILURE_THRESHOLD (int,   default: 5)
        HELM_MCP_CIRCUIT_BREAKER_RESET_TIMEOUT   (float, default: 30.0)
    """

    enabled: bool = field(
        default_factory=lambda: _env_bool("HELM_MCP_CIRCUIT_BREAKER_ENABLED", True)
    )
    failure_threshold: int = field(
        default_factory=lambda: _env_int("HELM_MCP_CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5)
    )
    reset_timeout: float = field(
        default_factory=lambda: _env_float("HELM_MCP_CIRCUIT_BREAKER_RESET_TIMEOUT", 30.0)
    )


@dataclass(frozen=True)
class TenacityConfig:
    """Tenacity retry configuration for ``HelmClient.call_tool()`` (tool-level).

    Provides exponential backoff with jitter for transient failures within
    individual tool calls.

    Env vars::

        HELM_MCP_TENACITY_ENABLED       (bool,  default: true)
        HELM_MCP_TENACITY_MAX_ATTEMPTS  (int,   default: 3)
        HELM_MCP_TENACITY_MIN_WAIT      (float, default: 0.5)
        HELM_MCP_TENACITY_MAX_WAIT      (float, default: 10.0)
        HELM_MCP_TENACITY_MULTIPLIER    (float, default: 1.5)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_TENACITY_ENABLED", True))
    max_attempts: int = field(default_factory=lambda: _env_int("HELM_MCP_TENACITY_MAX_ATTEMPTS", 3))
    min_wait: float = field(default_factory=lambda: _env_float("HELM_MCP_TENACITY_MIN_WAIT", 0.5))
    max_wait: float = field(default_factory=lambda: _env_float("HELM_MCP_TENACITY_MAX_WAIT", 10.0))
    multiplier: float = field(
        default_factory=lambda: _env_float("HELM_MCP_TENACITY_MULTIPLIER", 1.5)
    )


@dataclass(frozen=True)
class BulkheadConfig:
    """Bulkhead (concurrency limiter) for ``HelmClient`` (tool-level).

    Uses ``asyncio.Semaphore`` to limit the number of concurrent tool calls.

    Env vars::

        HELM_MCP_BULKHEAD_ENABLED         (bool, default: true)
        HELM_MCP_BULKHEAD_MAX_CONCURRENT  (int,  default: 10)
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_BULKHEAD_ENABLED", True))
    max_concurrent: int = field(
        default_factory=lambda: _env_int("HELM_MCP_BULKHEAD_MAX_CONCURRENT", 10)
    )


@dataclass(frozen=True)
class OTelConfig:
    """OpenTelemetry instrumentation configuration.

    FastMCP depends on ``opentelemetry-api`` so traces are always emitted
    as no-ops.  To receive actual trace data, install the OpenTelemetry SDK
    (``pip install helm-mcp[otel]``) and set ``HELM_MCP_OTEL_ENABLED=true``.

    Env vars::

        HELM_MCP_OTEL_ENABLED       (bool, default: false)
        HELM_MCP_OTEL_SERVICE_NAME  (str,  default: "helm-mcp")
        HELM_MCP_OTEL_EXPORTER      (str,  default: "console")
    """

    enabled: bool = field(default_factory=lambda: _env_bool("HELM_MCP_OTEL_ENABLED", False))
    service_name: str = field(
        default_factory=lambda: os.environ.get("HELM_MCP_OTEL_SERVICE_NAME", "helm-mcp")
    )
    exporter: str = field(
        default_factory=lambda: os.environ.get("HELM_MCP_OTEL_EXPORTER", "console")
    )


@dataclass(frozen=True)
class ResilienceConfig:
    """Top-level resilience configuration aggregating all sub-configs.

    Instantiate with no arguments to read all settings from environment
    variables with sensible defaults::

        config = ResilienceConfig()

    Or provide explicit overrides for any sub-config::

        config = ResilienceConfig(
            rate_limit=RateLimitConfig(enabled=True, max_requests_per_second=50),
            circuit_breaker=CircuitBreakerConfig(failure_threshold=3),
        )
    """

    retry: RetryConfig = field(default_factory=RetryConfig)
    rate_limit: RateLimitConfig = field(default_factory=RateLimitConfig)
    cache: CacheConfig = field(default_factory=CacheConfig)
    error_handling: ErrorHandlingConfig = field(default_factory=ErrorHandlingConfig)
    timing: TimingConfig = field(default_factory=TimingConfig)
    circuit_breaker: CircuitBreakerConfig = field(default_factory=CircuitBreakerConfig)
    tenacity: TenacityConfig = field(default_factory=TenacityConfig)
    bulkhead: BulkheadConfig = field(default_factory=BulkheadConfig)
    otel: OTelConfig = field(default_factory=OTelConfig)


# ---------------------------------------------------------------------------
# Middleware builder
# ---------------------------------------------------------------------------


def build_middleware(config: ResilienceConfig) -> list:
    """Build a list of FastMCP middleware instances from the resilience config.

    Middleware is added in a deliberate order::

        1. TimingMiddleware      (outermost — measures total time)
        2. ErrorHandlingMiddleware (catches all exceptions from inner layers)
        3. RateLimitingMiddleware  (reject early before doing work)
        4. RetryMiddleware         (retry transient failures)
        5. ResponseCachingMiddleware (innermost — cache closest to execution)

    Returns:
        List of Middleware instances to pass to the FastMCP server.
    """
    middlewares: list = []

    if config.timing.enabled:
        if config.timing.detailed:
            from fastmcp.server.middleware.timing import DetailedTimingMiddleware

            middlewares.append(DetailedTimingMiddleware())
        else:
            from fastmcp.server.middleware.timing import TimingMiddleware

            middlewares.append(TimingMiddleware())
        logger.info("timing middleware enabled (detailed=%s)", config.timing.detailed)

    if config.error_handling.enabled:
        from fastmcp.server.middleware.error_handling import ErrorHandlingMiddleware

        middlewares.append(
            ErrorHandlingMiddleware(
                include_traceback=config.error_handling.include_traceback,
            )
        )
        logger.info("error handling middleware enabled")

    if config.rate_limit.enabled:
        from fastmcp.server.middleware.rate_limiting import RateLimitingMiddleware

        middlewares.append(
            RateLimitingMiddleware(
                max_requests_per_second=config.rate_limit.max_requests_per_second,
                burst_capacity=config.rate_limit.burst_capacity,
            )
        )
        logger.info(
            "rate limiting middleware enabled (%.1f rps, burst %d)",
            config.rate_limit.max_requests_per_second,
            config.rate_limit.burst_capacity,
        )

    if config.retry.enabled:
        from fastmcp.server.middleware.error_handling import RetryMiddleware

        middlewares.append(
            RetryMiddleware(
                max_retries=config.retry.max_retries,
                base_delay=config.retry.base_delay,
                max_delay=config.retry.max_delay,
                backoff_multiplier=config.retry.backoff_multiplier,
                retry_exceptions=(ConnectionError, TimeoutError, OSError),
            )
        )
        logger.info(
            "retry middleware enabled (max=%d, base_delay=%.1fs)",
            config.retry.max_retries,
            config.retry.base_delay,
        )

    if config.cache.enabled:
        from fastmcp.server.middleware.caching import ResponseCachingMiddleware

        middlewares.append(
            ResponseCachingMiddleware(
                call_tool_settings={"ttl": config.cache.tool_ttl, "enabled": True},
                list_tools_settings={"ttl": config.cache.list_ttl, "enabled": True},
            )
        )
        logger.info(
            "response caching middleware enabled (tool_ttl=%ds, list_ttl=%ds)",
            config.cache.tool_ttl,
            config.cache.list_ttl,
        )

    return middlewares


# ---------------------------------------------------------------------------
# OpenTelemetry setup
# ---------------------------------------------------------------------------


def setup_otel(config: OTelConfig) -> None:
    """Configure OpenTelemetry SDK if enabled and SDK packages are available.

    This is a best-effort setup: if the SDK is not installed, OTel remains
    a no-op (FastMCP only depends on ``opentelemetry-api``).
    """
    if not config.enabled:
        return

    try:
        from opentelemetry import trace
        from opentelemetry.sdk.resources import Resource
        from opentelemetry.sdk.trace import TracerProvider

        resource = Resource.create({"service.name": config.service_name})
        provider = TracerProvider(resource=resource)

        if config.exporter == "console":
            from opentelemetry.sdk.trace.export import (
                ConsoleSpanExporter,
                SimpleSpanProcessor,
            )

            provider.add_span_processor(SimpleSpanProcessor(ConsoleSpanExporter()))
        elif config.exporter == "otlp":
            from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import (
                OTLPSpanExporter,
            )
            from opentelemetry.sdk.trace.export import BatchSpanProcessor

            provider.add_span_processor(BatchSpanProcessor(OTLPSpanExporter()))

        trace.set_tracer_provider(provider)
        logger.info(
            "OpenTelemetry enabled (service=%s, exporter=%s)",
            config.service_name,
            config.exporter,
        )
    except ImportError:
        logger.warning(
            "HELM_MCP_OTEL_ENABLED=true but opentelemetry-sdk is not installed. "
            "Install with: pip install helm-mcp[otel]  — telemetry will remain no-op."
        )
