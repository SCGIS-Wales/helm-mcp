"""Tests for helm_mcp.resilience — configuration and middleware builder."""

import os
from unittest.mock import patch

import pytest

from helm_mcp.resilience import (
    BulkheadConfig,
    CacheConfig,
    CircuitBreakerConfig,
    ErrorHandlingConfig,
    OTelConfig,
    RateLimitConfig,
    ResilienceConfig,
    RetryConfig,
    TenacityConfig,
    TimingConfig,
    _env_bool,
    _env_float,
    _env_int,
    build_middleware,
    setup_otel,
)

# ---------------------------------------------------------------------------
# Env-var helpers
# ---------------------------------------------------------------------------


class TestEnvBool:
    """Test _env_bool helper."""

    def test_default_when_unset(self):
        with patch.dict(os.environ, {}, clear=True):
            assert _env_bool("UNSET_VAR", True) is True
            assert _env_bool("UNSET_VAR", False) is False

    def test_true_values(self):
        for val in ("1", "true", "True", "TRUE", "yes", "YES", "on", "ON"):
            with patch.dict(os.environ, {"TEST_BOOL": val}):
                assert _env_bool("TEST_BOOL", False) is True

    def test_false_values(self):
        for val in ("0", "false", "False", "no", "off", "anything_else"):
            with patch.dict(os.environ, {"TEST_BOOL": val}):
                assert _env_bool("TEST_BOOL", True) is False

    def test_empty_string_uses_default(self):
        with patch.dict(os.environ, {"TEST_BOOL": ""}):
            assert _env_bool("TEST_BOOL", True) is True


class TestEnvFloat:
    """Test _env_float helper."""

    def test_default_when_unset(self):
        with patch.dict(os.environ, {}, clear=True):
            assert _env_float("UNSET_VAR", 1.5) == 1.5

    def test_parses_value(self):
        with patch.dict(os.environ, {"TEST_FLOAT": "42.5"}):
            assert _env_float("TEST_FLOAT", 0.0) == 42.5

    def test_invalid_raises(self):
        with (
            patch.dict(os.environ, {"TEST_FLOAT": "not_a_number"}),
            pytest.raises(ValueError),
        ):
            _env_float("TEST_FLOAT", 0.0)


class TestEnvInt:
    """Test _env_int helper."""

    def test_default_when_unset(self):
        with patch.dict(os.environ, {}, clear=True):
            assert _env_int("UNSET_VAR", 10) == 10

    def test_parses_value(self):
        with patch.dict(os.environ, {"TEST_INT": "42"}):
            assert _env_int("TEST_INT", 0) == 42

    def test_invalid_raises(self):
        with (
            patch.dict(os.environ, {"TEST_INT": "not_a_number"}),
            pytest.raises(ValueError),
        ):
            _env_int("TEST_INT", 0)


# ---------------------------------------------------------------------------
# Configuration dataclasses
# ---------------------------------------------------------------------------


class TestResilienceConfig:
    """Test ResilienceConfig reads from env vars with defaults."""

    def test_default_config(self):
        """All defaults are sensible when no HELM_MCP_ env vars set."""
        with patch.dict(os.environ, {}, clear=True):
            config = ResilienceConfig()
            # Retry enabled by default
            assert config.retry.enabled is True
            assert config.retry.max_retries == 2
            # Rate limiting disabled by default
            assert config.rate_limit.enabled is False
            # Cache disabled by default
            assert config.cache.enabled is False
            # Error handling enabled
            assert config.error_handling.enabled is True
            # Timing enabled
            assert config.timing.enabled is True
            # Circuit breaker enabled
            assert config.circuit_breaker.enabled is True
            assert config.circuit_breaker.failure_threshold == 5
            # Tenacity enabled
            assert config.tenacity.enabled is True
            assert config.tenacity.max_attempts == 3
            # Bulkhead enabled
            assert config.bulkhead.enabled is True
            assert config.bulkhead.max_concurrent == 10
            # OTel disabled
            assert config.otel.enabled is False

    def test_retry_env_override(self):
        env = {
            "HELM_MCP_RETRY_ENABLED": "false",
            "HELM_MCP_RETRY_MAX_RETRIES": "5",
            "HELM_MCP_RETRY_BASE_DELAY": "2.0",
        }
        with patch.dict(os.environ, env, clear=True):
            config = RetryConfig()
            assert config.enabled is False
            assert config.max_retries == 5
            assert config.base_delay == 2.0

    def test_rate_limit_env_override(self):
        env = {
            "HELM_MCP_RATE_LIMIT_ENABLED": "true",
            "HELM_MCP_RATE_LIMIT_MAX_RPS": "50.0",
            "HELM_MCP_RATE_LIMIT_BURST": "100",
        }
        with patch.dict(os.environ, env, clear=True):
            config = RateLimitConfig()
            assert config.enabled is True
            assert config.max_requests_per_second == 50.0
            assert config.burst_capacity == 100

    def test_circuit_breaker_env_override(self):
        env = {
            "HELM_MCP_CIRCUIT_BREAKER_ENABLED": "false",
            "HELM_MCP_CIRCUIT_BREAKER_FAILURE_THRESHOLD": "10",
            "HELM_MCP_CIRCUIT_BREAKER_RESET_TIMEOUT": "60.0",
        }
        with patch.dict(os.environ, env, clear=True):
            config = CircuitBreakerConfig()
            assert config.enabled is False
            assert config.failure_threshold == 10
            assert config.reset_timeout == 60.0

    def test_tenacity_env_override(self):
        env = {
            "HELM_MCP_TENACITY_MAX_ATTEMPTS": "5",
            "HELM_MCP_TENACITY_MIN_WAIT": "1.0",
        }
        with patch.dict(os.environ, env, clear=True):
            config = TenacityConfig()
            assert config.max_attempts == 5
            assert config.min_wait == 1.0

    def test_bulkhead_env_override(self):
        env = {"HELM_MCP_BULKHEAD_MAX_CONCURRENT": "25"}
        with patch.dict(os.environ, env, clear=True):
            config = BulkheadConfig()
            assert config.max_concurrent == 25

    def test_otel_env_override(self):
        env = {
            "HELM_MCP_OTEL_ENABLED": "true",
            "HELM_MCP_OTEL_SERVICE_NAME": "my-service",
            "HELM_MCP_OTEL_EXPORTER": "otlp",
        }
        with patch.dict(os.environ, env, clear=True):
            config = OTelConfig()
            assert config.enabled is True
            assert config.service_name == "my-service"
            assert config.exporter == "otlp"

    def test_explicit_override(self):
        """Direct constructor overrides bypass env vars."""
        config = ResilienceConfig(
            rate_limit=RateLimitConfig(enabled=True, max_requests_per_second=50.0),
        )
        assert config.rate_limit.enabled is True
        assert config.rate_limit.max_requests_per_second == 50.0

    def test_frozen(self):
        """Config dataclasses are immutable."""
        config = RetryConfig()
        with pytest.raises(AttributeError):
            config.enabled = False  # type: ignore[misc]


# ---------------------------------------------------------------------------
# Middleware builder
# ---------------------------------------------------------------------------


class TestBuildMiddleware:
    """Test build_middleware() produces correct middleware list."""

    def test_all_disabled(self):
        """Empty list when all middleware configs disabled."""
        config = ResilienceConfig(
            retry=RetryConfig(enabled=False),
            rate_limit=RateLimitConfig(enabled=False),
            cache=CacheConfig(enabled=False),
            error_handling=ErrorHandlingConfig(enabled=False),
            timing=TimingConfig(enabled=False),
        )
        middlewares = build_middleware(config)
        assert middlewares == []

    def test_default_config_has_middleware(self):
        """Default config enables timing, error handling, and retry."""
        with patch.dict(os.environ, {}, clear=True):
            config = ResilienceConfig()
            middlewares = build_middleware(config)
            # Timing + ErrorHandling + Retry = 3
            assert len(middlewares) == 3

    def test_all_enabled(self):
        """All five middleware types included when all enabled."""
        config = ResilienceConfig(
            retry=RetryConfig(enabled=True),
            rate_limit=RateLimitConfig(enabled=True),
            cache=CacheConfig(enabled=True),
            error_handling=ErrorHandlingConfig(enabled=True),
            timing=TimingConfig(enabled=True),
        )
        middlewares = build_middleware(config)
        assert len(middlewares) == 5

    def test_order_is_correct(self):
        """Middleware order: timing → error → rate_limit → retry → cache."""
        from fastmcp.server.middleware.caching import ResponseCachingMiddleware
        from fastmcp.server.middleware.error_handling import (
            ErrorHandlingMiddleware,
            RetryMiddleware,
        )
        from fastmcp.server.middleware.rate_limiting import RateLimitingMiddleware
        from fastmcp.server.middleware.timing import TimingMiddleware

        config = ResilienceConfig(
            retry=RetryConfig(enabled=True),
            rate_limit=RateLimitConfig(enabled=True),
            cache=CacheConfig(enabled=True),
            error_handling=ErrorHandlingConfig(enabled=True),
            timing=TimingConfig(enabled=True),
        )
        middlewares = build_middleware(config)
        assert isinstance(middlewares[0], TimingMiddleware)
        assert isinstance(middlewares[1], ErrorHandlingMiddleware)
        assert isinstance(middlewares[2], RateLimitingMiddleware)
        assert isinstance(middlewares[3], RetryMiddleware)
        assert isinstance(middlewares[4], ResponseCachingMiddleware)

    def test_detailed_timing(self):
        """DetailedTimingMiddleware used when detailed=True."""
        from fastmcp.server.middleware.timing import DetailedTimingMiddleware

        config = ResilienceConfig(
            timing=TimingConfig(enabled=True, detailed=True),
            error_handling=ErrorHandlingConfig(enabled=False),
            retry=RetryConfig(enabled=False),
        )
        middlewares = build_middleware(config)
        assert len(middlewares) == 1
        assert isinstance(middlewares[0], DetailedTimingMiddleware)

    def test_individual_toggle(self):
        """Each middleware can be independently enabled/disabled."""
        from fastmcp.server.middleware.rate_limiting import RateLimitingMiddleware

        config = ResilienceConfig(
            retry=RetryConfig(enabled=False),
            rate_limit=RateLimitConfig(enabled=True),
            cache=CacheConfig(enabled=False),
            error_handling=ErrorHandlingConfig(enabled=False),
            timing=TimingConfig(enabled=False),
        )
        middlewares = build_middleware(config)
        assert len(middlewares) == 1
        assert isinstance(middlewares[0], RateLimitingMiddleware)


# ---------------------------------------------------------------------------
# OpenTelemetry setup
# ---------------------------------------------------------------------------


class TestSetupOtel:
    """Test OpenTelemetry SDK configuration."""

    def test_disabled_noop(self):
        """No SDK setup when otel.enabled=False."""
        config = OTelConfig(enabled=False)
        # Should not raise or do anything
        setup_otel(config)

    def test_missing_sdk_warns(self, caplog):
        """Warning logged when SDK not installed but otel.enabled=True."""
        config = OTelConfig(enabled=True)
        with patch.dict("sys.modules", {"opentelemetry": None, "opentelemetry.sdk": None}):
            # Force ImportError by hiding the module
            original = __import__

            def mock_import(name, *args, **kwargs):
                if name.startswith("opentelemetry"):
                    raise ImportError("mocked")
                return original(name, *args, **kwargs)

            with patch("builtins.__import__", side_effect=mock_import):
                import logging

                with caplog.at_level(logging.WARNING, logger="helm_mcp.resilience"):
                    setup_otel(config)
                assert "opentelemetry-sdk is not installed" in caplog.text
