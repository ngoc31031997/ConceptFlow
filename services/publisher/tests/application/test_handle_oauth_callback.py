"""Unit tests for HandleOAuthCallbackUseCase (business-rules.md Rule 6)."""

from __future__ import annotations

from datetime import UTC, datetime

import pytest

from application.handle_oauth_callback import HandleOAuthCallbackUseCase
from domain.models import OAuthCredential
from domain.ports import CredentialStorePort


class FakeCredentialStore(CredentialStorePort):
    def __init__(self) -> None:
        self.saved: OAuthCredential | None = None

    def get(self) -> OAuthCredential | None:
        return self.saved

    def save(self, credential: OAuthCredential) -> None:
        self.saved = credential


class FakeOAuthFlow:
    def __init__(self, fail_with: Exception | None = None) -> None:
        self._fail_with = fail_with

    def exchange_code(self, code: str) -> OAuthCredential:
        if self._fail_with is not None:
            raise self._fail_with
        return OAuthCredential(
            access_token="token", refresh_token="refresh", expires_at=datetime.now(UTC), channel_id="UC1"
        )


def test_saves_credential_on_success():
    store = FakeCredentialStore()
    use_case = HandleOAuthCallbackUseCase(FakeOAuthFlow(), store)

    use_case.handle("valid-code")

    assert store.saved is not None
    assert store.saved.channel_id == "UC1"


def test_propagates_exchange_error():
    store = FakeCredentialStore()
    use_case = HandleOAuthCallbackUseCase(FakeOAuthFlow(fail_with=ValueError("bad code")), store)

    with pytest.raises(ValueError):
        use_case.handle("bad-code")

    assert store.saved is None
