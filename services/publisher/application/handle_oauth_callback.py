"""HandleOAuthCallbackUseCase — business-logic-model.md, Story E1.

Errors from the OAuth code exchange (bad/expired/reused code) propagate
up to the REST layer as-is (Business Rule 6) — this use case does not
catch them, since the router is responsible for mapping them to a 400
response.
"""

from __future__ import annotations

from typing import Protocol

from domain.models import OAuthCredential
from domain.ports import CredentialStorePort


class OAuthFlowPort(Protocol):
    def exchange_code(self, code: str) -> OAuthCredential: ...


class HandleOAuthCallbackUseCase:
    def __init__(self, oauth_flow: OAuthFlowPort, credential_store: CredentialStorePort) -> None:
        self._oauth_flow = oauth_flow
        self._credential_store = credential_store

    def handle(self, code: str) -> None:
        credential = self._oauth_flow.exchange_code(code)
        self._credential_store.save(credential)
