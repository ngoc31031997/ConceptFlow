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
    # Which provider serves the light tasks (suggest-metadata, suggest-short-script).
    light_provider: str
    # Code pipeline (CR-039).
    code_chunk_shots: int
    code_chunk_concurrency: int
    code_repair_max_rounds: int
    rendering_url: str
    rendering_check_timeout: int

    @classmethod
    def from_env(cls) -> Config:
        light = os.environ.get("LIGHT_TASKS_PROVIDER", "ollama")
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
            code_chunk_shots=_int("CODE_CHUNK_SHOTS", 5),
            code_chunk_concurrency=_int("CODE_CHUNK_CONCURRENCY", 10),
            code_repair_max_rounds=_int("CODE_REPAIR_MAX_ROUNDS", 3),
            rendering_url=os.environ.get("RENDERING_URL", "http://rendering:8000"),
            rendering_check_timeout=_int("RENDERING_CHECK_TIMEOUT_SECONDS", 180),
        )
