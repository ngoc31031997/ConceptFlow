"""Unit tests for HandleOAuthCallbackUseCase (business-rules.md Rule 6)."""

from __future__ import annotations

from datetime import UTC, datetime

import pytest

from application.handle_oauth_callback import HandleOAuthCallbackUseCase, UnknownOAuthAppError
from domain.models import OAuthApp, OAuthCredential
from tests.fakes import (
    TEST_CLIENT_ID,
    FakeOAuthAppRegistry,
    InMemoryCredentialStore,
    make_app,
)


class FakeOAuthFlow:
    def __init__(self, fail_with: Exception | None = None) -> None:
        self._fail_with = fail_with
        self.exchanged_with: OAuthApp | None = None

    def exchange_code(self, code: str, app: OAuthApp) -> OAuthCredential:
        if self._fail_with is not None:
            raise self._fail_with
        self.exchanged_with = app
        return OAuthCredential(
            access_token="token",
            refresh_token="refresh",
            expires_at=datetime.now(UTC),
            channel_id="UC1",
            client_id=app.client_id,
            channel_title="Kênh Một",
        )


def _use_case(flow, store=None, registry=None):
    return HandleOAuthCallbackUseCase(
        flow, store or InMemoryCredentialStore(), registry or FakeOAuthAppRegistry()
    )


def test_saves_credential_on_success():
    store = InMemoryCredentialStore()
    use_case = _use_case(FakeOAuthFlow(), store)

    credential = use_case.handle("valid-code", TEST_CLIENT_ID)

    assert store.get("UC1") is not None
    assert credential.channel_id == "UC1"
    # Returned so the router can tell the Creator which channel was picked on
    # Google's chooser — "connected" alone does not say.
    assert credential.channel_title == "Kênh Một"


def test_propagates_exchange_error():
    store = InMemoryCredentialStore()
    use_case = _use_case(FakeOAuthFlow(fail_with=ValueError("bad code")), store)

    with pytest.raises(ValueError):
        use_case.handle("bad-code", TEST_CLIENT_ID)

    assert store.list() == []


def test_exchanges_with_the_app_named_in_the_callback():
    """The token exchange must use the client that issued the code — any
    other client's secret is rejected by Google."""
    second = make_app(client_id="client-b", client_secret="secret-b", project_id="other-project")
    flow = FakeOAuthFlow()
    use_case = _use_case(flow, registry=FakeOAuthAppRegistry([make_app(), second]))

    use_case.handle("code", "client-b")

    assert flow.exchanged_with is not None
    assert flow.exchanged_with.client_id == "client-b"


def test_rejects_a_client_id_that_is_no_longer_configured():
    use_case = _use_case(FakeOAuthFlow())

    with pytest.raises(UnknownOAuthAppError):
        use_case.handle("code", "client-removed")


def test_falls_back_to_the_only_app_when_state_carries_no_client_id():
    """A pre-CR-012 state string still in flight is unambiguous while only
    one app exists (FR33.3)."""
    flow = FakeOAuthFlow()
    use_case = _use_case(flow)

    use_case.handle("code", "")

    assert flow.exchanged_with.client_id == TEST_CLIENT_ID


def test_refuses_to_guess_the_app_when_several_are_configured():
    registry = FakeOAuthAppRegistry([make_app(), make_app(client_id="client-b")])
    use_case = _use_case(FakeOAuthFlow(), registry=registry)

    with pytest.raises(UnknownOAuthAppError):
        use_case.handle("code", "")
