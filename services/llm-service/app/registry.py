"""Builds the providers once, sharing one rate limiter across all Hive calls."""

from __future__ import annotations

from app.config import Config
from app.limiter import RateLimiter
from app.provider import Provider


class Providers:
    def __init__(self, config: Config) -> None:
        limiter = RateLimiter(config.hive_rate_per_second)
        self.hive = Provider(
            "hive", config.hive_base_url, config.hive_api_key, config.hive_model,
            timeout=config.hive_timeout or None, max_retries=config.hive_max_retries, limiter=limiter,
        )
        # Ollama serves an OpenAI-compatible API under /v1 and ignores the key,
        # but the SDK insists on one: "ollama" is the documented placeholder.
        self.ollama = Provider(
            "ollama", config.ollama_url.rstrip("/") + "/v1", "ollama", config.ollama_model,
            timeout=config.ollama_timeout, max_retries=1,
        )
        self.light = self.ollama if config.light_provider == "ollama" else self.hive

    def get(self, name: str) -> Provider:
        if name == "hive":
            return self.hive
        if name == "ollama":
            return self.ollama
        raise KeyError(name)
