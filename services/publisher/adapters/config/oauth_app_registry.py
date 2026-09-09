"""FileOAuthAppRegistry — implements OAuthAppRegistryPort by scanning a
directory of client_secret*.json files (ADR-0026, CR-012 FR29).

Loading from a *directory* rather than numbered env vars
(GOOGLE_OAUTH_CLIENT_ID_2, _3, ...) is deliberate: Google Cloud Console
emits the client secret as one whole JSON file, so splitting it back into
separate env vars is a manual copy step that can go wrong, and there is
no natural ceiling on how many apps should be allowed to exist.

The registry is read once at startup. Dropping in a new file therefore
takes a service restart, which is the right trade for not stat-ing the
filesystem on every request in a service whose app list changes about
once a quarter.
"""

from __future__ import annotations

import json
import logging
import os
from pathlib import Path

from domain.models import OAuthApp
from domain.ports import OAuthAppRegistryPort

logger = logging.getLogger(__name__)

DEFAULT_SECRETS_DIR = "/run/secrets/google-oauth"


class FileOAuthAppRegistry(OAuthAppRegistryPort):
    def __init__(self, apps: list[OAuthApp]) -> None:
        # Keyed by client_id: that is what a stored credential names, and
        # what /start receives from the GUI.
        self._by_client_id = {app.client_id: app for app in apps}

    def list(self) -> list[OAuthApp]:
        return sorted(self._by_client_id.values(), key=lambda app: app.label)

    def get(self, client_id: str) -> OAuthApp | None:
        return self._by_client_id.get(client_id)

    @classmethod
    def from_environment(cls) -> FileOAuthAppRegistry:
        """Builds the registry from GOOGLE_OAUTH_CLIENT_SECRETS_DIR, falling
        back to the pre-CR-012 single-app env vars (FR29.3) so an existing
        .env keeps working untouched."""
        directory = os.environ.get("GOOGLE_OAUTH_CLIENT_SECRETS_DIR", DEFAULT_SECRETS_DIR)
        apps = cls._load_directory(Path(directory))

        if not apps:
            legacy = cls._legacy_app_from_env()
            if legacy is not None:
                logger.info(
                    "No client_secret files in %s — using the legacy "
                    "GOOGLE_OAUTH_CLIENT_ID/SECRET env vars as a single app",
                    directory,
                )
                apps = [legacy]
            else:
                logger.warning(
                    "No OAuth apps configured: %s holds no usable client_secret*.json "
                    "and GOOGLE_OAUTH_CLIENT_ID/SECRET are unset. Connecting a YouTube "
                    "channel will be unavailable until one is added.",
                    directory,
                )

        return cls(apps)

    @staticmethod
    def _load_directory(directory: Path) -> list[OAuthApp]:
        if not directory.is_dir():
            return []

        apps: list[OAuthApp] = []
        for path in sorted(directory.glob("*.json")):
            app = FileOAuthAppRegistry._parse_file(path)
            if app is not None:
                apps.append(app)
        return apps

    @staticmethod
    def _parse_file(path: Path) -> OAuthApp | None:
        """Returns None (with a warning naming the file) for anything
        unusable. A stray or malformed file in the secrets directory must
        not take the whole Publisher down at startup (FR29.4)."""
        try:
            raw = json.loads(path.read_text())
        except (OSError, json.JSONDecodeError) as exc:
            logger.warning("Ignoring OAuth client file %s — cannot be read as JSON: %s", path.name, exc)
            return None

        # Console emits "web" for Web application clients and "installed"
        # for Desktop ones; both carry the same fields we need.
        config = raw.get("web") or raw.get("installed")
        if not isinstance(config, dict):
            logger.warning(
                "Ignoring OAuth client file %s — no 'web' or 'installed' section", path.name
            )
            return None

        client_id = config.get("client_id")
        client_secret = config.get("client_secret")
        if not client_id or not client_secret:
            logger.warning(
                "Ignoring OAuth client file %s — missing client_id or client_secret", path.name
            )
            return None

        return OAuthApp(
            client_id=client_id,
            client_secret=client_secret,
            project_id=config.get("project_id", ""),
            redirect_uris=tuple(config.get("redirect_uris") or ()),
            source_file=path.name,
        )

    @staticmethod
    def _legacy_app_from_env() -> OAuthApp | None:
        client_id = os.environ.get("GOOGLE_OAUTH_CLIENT_ID")
        client_secret = os.environ.get("GOOGLE_OAUTH_CLIENT_SECRET")
        if not client_id or not client_secret:
            return None

        # The legacy env vars carry no redirect_uris list of their own, so
        # the configured redirect is taken on trust — it is the same single
        # value the pre-CR-012 code used unconditionally.
        redirect_uri = os.environ.get("GOOGLE_OAUTH_REDIRECT_URI", "")
        return OAuthApp(
            client_id=client_id,
            client_secret=client_secret,
            project_id="(env)",
            redirect_uris=(redirect_uri,) if redirect_uri else (),
            source_file="<environment>",
        )
