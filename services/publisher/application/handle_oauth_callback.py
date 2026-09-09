"""HandleOAuthCallbackUseCase — business-logic-model.md, Story E1.

Errors from the OAuth code exchange (bad/expired/reused code) propagate
up to the REST layer as-is (Business Rule 6) — this use case does not
catch them, since the router is responsible for mapping them to a 400
response.
"""

from __future__ import annotations

from typing import Protocol

from domain.models import OAuthApp, OAuthCredential
from domain.ports import CredentialStorePort, OAuthAppRegistryPort


class UnknownOAuthAppError(Exception):
    """The callback named a client_id that is no longer configured — e.g.
    its client_secret file was removed between /start and /callback."""


class OAuthFlowPort(Protocol):
    def exchange_code(self, code: str, app: OAuthApp) -> OAuthCredential: ...


class HandleOAuthCallbackUseCase:
    def __init__(
        self,
        oauth_flow: OAuthFlowPort,
        credential_store: CredentialStorePort,
        app_registry: OAuthAppRegistryPort,
    ) -> None:
        self._oauth_flow = oauth_flow
        self._credential_store = credential_store
        self._app_registry = app_registry

    def handle(self, code: str, client_id: str) -> OAuthCredential:
        """Exchanges the code with the app that issued it and stores the
        resulting channel. Returns the credential so the router can tell
        the Creator which channel just got connected — with several
        channels in play, "connected: true" alone is not enough to know
        whether the right one was picked on Google's chooser.
        """
        app = self._resolve_app(client_id)
        credential = self._oauth_flow.exchange_code(code, app)
        self._credential_store.save(credential)
        return credential

    def _resolve_app(self, client_id: str) -> OAuthApp:
        if client_id:
            app = self._app_registry.get(client_id)
            if app is None:
                raise UnknownOAuthAppError(
                    f"OAuth client {client_id!r} is not configured — its client_secret "
                    "file may have been removed from the secrets directory"
                )
            return app

        # No client_id in state: either a pre-CR-012 state string still in
        # flight, or a hand-built callback. Only unambiguous with exactly
        # one app configured — guessing among several would exchange the
        # code against the wrong client and fail confusingly.
        apps = self._app_registry.list()
        if len(apps) == 1:
            return apps[0]
        raise UnknownOAuthAppError(
            "callback did not identify which OAuth client it came from, and "
            f"{len(apps)} clients are configured — start the connection again "
            "from the app list"
        )
