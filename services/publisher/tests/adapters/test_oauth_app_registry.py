"""Unit tests for FileOAuthAppRegistry (CR-012 FR29)."""

from __future__ import annotations

import json

import pytest

from adapters.config.oauth_app_registry import FileOAuthAppRegistry


def _write_client_secret(directory, name, **overrides):
    config = {
        "client_id": "client-a.apps.googleusercontent.com",
        "client_secret": "secret-a",
        "project_id": "concer-508105",
        "redirect_uris": ["http://localhost:3000/oauth/youtube/callback"],
    }
    config.update(overrides)
    path = directory / name
    path.write_text(json.dumps({"web": config}))
    return path


@pytest.fixture
def secrets_dir(tmp_path, monkeypatch):
    monkeypatch.setenv("GOOGLE_OAUTH_CLIENT_SECRETS_DIR", str(tmp_path))
    monkeypatch.delenv("GOOGLE_OAUTH_CLIENT_ID", raising=False)
    monkeypatch.delenv("GOOGLE_OAUTH_CLIENT_SECRET", raising=False)
    return tmp_path


def test_loads_every_client_secret_file(secrets_dir):
    _write_client_secret(secrets_dir, "a.json")
    _write_client_secret(secrets_dir, "b.json", client_id="client-b", project_id="second-project")

    apps = FileOAuthAppRegistry.from_environment().list()

    assert {app.project_id for app in apps} == {"concer-508105", "second-project"}


def test_reads_the_installed_section_too(secrets_dir):
    """Console emits "installed" for Desktop clients and "web" for Web ones."""
    (secrets_dir / "desktop.json").write_text(
        json.dumps({"installed": {"client_id": "client-d", "client_secret": "s", "project_id": "p"}})
    )

    apps = FileOAuthAppRegistry.from_environment().list()

    assert [app.client_id for app in apps] == ["client-d"]


def test_ignores_a_malformed_file_without_taking_the_service_down(secrets_dir):
    """A stray file in secrets/ must not stop the Publisher from starting
    (FR29.4)."""
    _write_client_secret(secrets_dir, "good.json")
    (secrets_dir / "broken.json").write_text("{not json")
    (secrets_dir / "wrong-shape.json").write_text(json.dumps({"something": "else"}))
    (secrets_dir / "no-secret.json").write_text(json.dumps({"web": {"client_id": "x"}}))

    apps = FileOAuthAppRegistry.from_environment().list()

    assert [app.project_id for app in apps] == ["concer-508105"]


def test_falls_back_to_the_legacy_env_vars_when_the_directory_is_empty(secrets_dir, monkeypatch):
    """A pre-CR-012 .env must keep working untouched (FR29.3)."""
    monkeypatch.setenv("GOOGLE_OAUTH_CLIENT_ID", "legacy-client")
    monkeypatch.setenv("GOOGLE_OAUTH_CLIENT_SECRET", "legacy-secret")
    monkeypatch.setenv("GOOGLE_OAUTH_REDIRECT_URI", "http://localhost:3000/oauth/youtube/callback")

    apps = FileOAuthAppRegistry.from_environment().list()

    assert [app.client_id for app in apps] == ["legacy-client"]
    assert apps[0].accepts_redirect("http://localhost:3000/oauth/youtube/callback")


def test_files_win_over_the_legacy_env_vars(secrets_dir, monkeypatch):
    monkeypatch.setenv("GOOGLE_OAUTH_CLIENT_ID", "legacy-client")
    monkeypatch.setenv("GOOGLE_OAUTH_CLIENT_SECRET", "legacy-secret")
    _write_client_secret(secrets_dir, "a.json")

    apps = FileOAuthAppRegistry.from_environment().list()

    assert [app.client_id for app in apps] == ["client-a.apps.googleusercontent.com"]


def test_no_apps_configured_is_survivable(secrets_dir):
    registry = FileOAuthAppRegistry.from_environment()

    assert registry.list() == []
    assert registry.get("anything") is None


def test_accepts_redirect_matches_exactly(secrets_dir):
    """Google compares redirect URIs byte-for-byte, so a trailing slash is a
    real mismatch — this is exactly the file the Creator supplied."""
    _write_client_secret(secrets_dir, "a.json", redirect_uris=["http://localhost:3000/"])

    app = FileOAuthAppRegistry.from_environment().list()[0]

    assert app.accepts_redirect("http://localhost:3000/") is True
    assert app.accepts_redirect("http://localhost:3000/oauth/youtube/callback") is False
