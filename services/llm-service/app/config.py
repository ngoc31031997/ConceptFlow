from __future__ import annotations

import os
from dataclasses import dataclass


def _int(name: str, default: int) -> int:
    raw = os.environ.get(name, "")
    if raw == "":
        return default
    try:
        return int(raw)
    except ValueError as exc:
        raise ValueError(f"{name} must be an integer, got {raw!r}") from exc


def _non_negative(name: str, default: int) -> int:
    value = _int(name, default)
    if value < 0:
        raise ValueError(f"{name} must be >= 0 (0 = off), got {value}")
    return value


@dataclass(frozen=True)
class Config:
    hive_api_key: str
    hive_base_url: str
    hive_model: str
    # 0 = no timeout (wait for Hive as long as it takes).
    hive_timeout: int
    hive_max_retries: int
    hive_rate_per_second: int
    ollama_url: str
    ollama_model: str
    ollama_timeout: int
    # Which provider serves the light task (suggest-metadata).
    light_provider: str
    # Code pipeline.
    code_chunk_shots: int
    code_chunk_concurrency: int
    code_repair_max_rounds: int
    rendering_url: str
    rendering_check_timeout: int
    # Stop a call that has reasoned for more than this many
    # characters without writing any answer. 0 = no limit.
    # Every call of the code pipeline (layout/cast/chunk/repair).
    code_max_reasoning_chars: int
    # /v1/chat (steps 1a/1b legitimately think at length, so off by default).
    chat_max_reasoning_chars: int

    @classmethod
    def from_env(cls) -> Config:
        light = os.environ.get("LIGHT_TASKS_PROVIDER", "hive")
        if light not in ("ollama", "hive"):
            raise ValueError(f"LIGHT_TASKS_PROVIDER must be ollama or hive, got {light!r}")
        return cls(
            hive_api_key=os.environ.get("HIVE_API_KEY", ""),
            hive_base_url=os.environ.get("HIVE_BASE_URL", "https://api-cdn.thehive.ai/api/v3"),
            hive_model=os.environ.get("HIVE_MODEL", "deepseek-ai/deepseek-v4.1-flash"),
            hive_timeout=_int("HIVE_TIMEOUT_SECONDS", 0),
            hive_max_retries=_int("HIVE_MAX_RETRIES", 3),
            hive_rate_per_second=_int("HIVE_RATE_PER_SECOND", 5),
            ollama_url=os.environ.get("OLLAMA_URL", "http://ollama:11434"),
            ollama_model=os.environ.get("OLLAMA_MODEL", "llama3.2"),
            ollama_timeout=_int("OLLAMA_TIMEOUT_SECONDS", 120),
            light_provider=light,
            code_chunk_shots=_int("CODE_CHUNK_SHOTS", 3),
            code_chunk_concurrency=_int("CODE_CHUNK_CONCURRENCY", 10),
            code_repair_max_rounds=_int("CODE_REPAIR_MAX_ROUNDS", 3),
            rendering_url=os.environ.get("RENDERING_URL", "http://rendering:8000"),
            rendering_check_timeout=_int("RENDERING_CHECK_TIMEOUT_SECONDS", 180),
            code_max_reasoning_chars=_non_negative("CODE_MAX_REASONING_CHARS", 60000),
            chat_max_reasoning_chars=_non_negative("CHAT_MAX_REASONING_CHARS", 0),
        )
