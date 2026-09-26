"""Error classification for LLM calls.

Carried over from the orchestrator's llm_provider.go (CR-027 FR76.6 / D11):
the fix for a dead key and the fix for an exhausted token budget have nothing
in common, so a failed call says which one it was.
"""

from __future__ import annotations

from dataclasses import dataclass, field

AUTH = "auth"
BALANCE = "balance"
RATE_LIMIT = "rate_limit"
SERVER = "server"
TIMEOUT = "timeout"
# Stopped on length with NOTHING written: on a reasoning model the whole
# allowance went on thinking (CR-027 D13). The fix is a bigger budget.
BUDGET = "budget"
# Stopped on length mid-answer. Same fix as BUDGET.
TRUNCATED = "truncated"
# Finished cleanly and still said nothing: a real model/prompt problem.
EMPTY = "empty"
MALFORMED = "malformed"
NOT_CONFIGURED = "not_configured"

RETRYABLE = {RATE_LIMIT, SERVER}


@dataclass
class Usage:
    model: str = ""
    prompt_tokens: int = 0
    completion_tokens: int = 0
    reasoning_tokens: int = 0
    cached_tokens: int = 0

    def to_dict(self) -> dict:
        return {
            "model": self.model,
            "prompt_tokens": self.prompt_tokens,
            "completion_tokens": self.completion_tokens,
            "reasoning_tokens": self.reasoning_tokens,
            "cached_tokens": self.cached_tokens,
        }

    def __add__(self, other: Usage) -> Usage:
        return Usage(
            model=self.model or other.model,
            prompt_tokens=self.prompt_tokens + other.prompt_tokens,
            completion_tokens=self.completion_tokens + other.completion_tokens,
            reasoning_tokens=self.reasoning_tokens + other.reasoning_tokens,
            cached_tokens=self.cached_tokens + other.cached_tokens,
        )


@dataclass
class LLMError(Exception):
    kind: str
    provider: str
    message: str
    # Set when the provider billed us despite the failure: a truncated answer
    # still costs tokens and the usage screen must record it.
    usage: Usage = field(default_factory=Usage)
    # Text that did arrive before a TRUNCATED cut-off.
    partial: str = ""
    # The provider's own account of the call (status, request id,
    # finish_reason, body) for the error log. Not shown to the Creator.
    diag: str = ""

    def __str__(self) -> str:
        return f"{self.provider}: {self.kind}: {self.message}"

    def to_dict(self) -> dict:
        return {
            "kind": self.kind,
            "provider": self.provider,
            "message": self.message,
            "usage": self.usage.to_dict(),
            "partial": self.partial,
            "diag": self.diag,
        }
