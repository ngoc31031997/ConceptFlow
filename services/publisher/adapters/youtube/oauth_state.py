"""OAuth `state` encoding plus the nonce store that backs it.

`state` carries a small JSON object: the project_id, echoed back to the GUI
so it knows where to navigate after connecting, and *which* OAuth app minted
the code, since the token exchange needs that app's client_secret.

The nonce is the CSRF defence: without it, anyone able to make the
Creator's browser hit /callback with an attacker-obtained code could
attach the attacker's YouTube channel to this installation, and the
Creator's videos would then publish to a channel they do not own.
"""

from __future__ import annotations

import base64
import json
import secrets
import time
from dataclasses import dataclass

NONCE_TTL_SECONDS = 600  # a consent screen the Creator leaves open longer than this is abandoned


@dataclass(frozen=True)
class OAuthState:
    client_id: str
    project_id: str | None
    nonce: str


def encode_state(client_id: str, project_id: str | None, nonce: str) -> str:
    """base64url of compact JSON. Unpadded because Google round-trips
    `state` through URLs where '=' would need escaping."""
    payload = json.dumps(
        {"a": client_id, "p": project_id, "n": nonce}, separators=(",", ":")
    ).encode()
    return base64.urlsafe_b64encode(payload).decode().rstrip("=")


def decode_state(raw: str | None) -> OAuthState | None:
    """Decodes a state produced by encode_state.

    Anything that is not one of ours — including a legacy bare project_id —
    comes back as an OAuthState with an empty client_id and nonce, so such a
    callback still lands somewhere sensible rather than erroring.
    """
    if not raw:
        return None

    padded = raw + "=" * (-len(raw) % 4)
    try:
        decoded = json.loads(base64.urlsafe_b64decode(padded))
    except (ValueError, TypeError):
        return OAuthState(client_id="", project_id=raw, nonce="")

    if not isinstance(decoded, dict) or "a" not in decoded:
        return OAuthState(client_id="", project_id=raw, nonce="")

    return OAuthState(
        client_id=decoded.get("a") or "",
        project_id=decoded.get("p"),
        nonce=decoded.get("n") or "",
    )


class NonceStore:
    """In-memory, single-process store of nonces issued by /start.

    In memory rather than in Postgres because a nonce is only meaningful
    between one /start and the /callback that follows it, seconds later, in
    the same process — and losing them all on restart just means an
    in-flight consent has to be retried. That matches ADR-0016's local,
    single-instance threat model; a multi-instance deployment would need to
    move this to shared storage.
    """

    def __init__(self, ttl_seconds: int = NONCE_TTL_SECONDS) -> None:
        self._ttl_seconds = ttl_seconds
        self._issued: dict[str, float] = {}

    def issue(self) -> str:
        self._evict_expired()
        nonce = secrets.token_urlsafe(16)
        self._issued[nonce] = time.monotonic()
        return nonce

    def consume(self, nonce: str) -> bool:
        """Single-use: a valid nonce is removed as it is checked, so a
        replayed callback fails even inside the TTL."""
        self._evict_expired()
        return self._issued.pop(nonce, None) is not None

    def _evict_expired(self) -> None:
        cutoff = time.monotonic() - self._ttl_seconds
        expired = [nonce for nonce, issued_at in self._issued.items() if issued_at < cutoff]
        for nonce in expired:
            del self._issued[nonce]
