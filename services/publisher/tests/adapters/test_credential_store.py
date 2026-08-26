"""Unit tests for PostgresCredentialStore (ADR-0016).

Mocks psycopg2.connect so no real PostgreSQL instance is needed to run
the test suite.
"""

from __future__ import annotations

from datetime import UTC, datetime
from unittest.mock import MagicMock, patch

from adapters.persistence.credential_store import PostgresCredentialStore
from domain.models import OAuthCredential


def _mock_connect(fetchone_return=None):
    mock_conn = MagicMock()
    mock_cursor = MagicMock()
    mock_cursor.fetchone.return_value = fetchone_return
    mock_conn.cursor.return_value.__enter__.return_value = mock_cursor
    mock_conn.__enter__.return_value = mock_conn
    return mock_conn, mock_cursor


def test_get_returns_none_when_no_row():
    mock_conn, _ = _mock_connect(fetchone_return=None)
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        store = PostgresCredentialStore("postgresql://fake")
        assert store.get() is None


def test_get_returns_credential_when_row_exists():
    expires_at = datetime.now(UTC)
    mock_conn, _ = _mock_connect(fetchone_return=("token", "refresh", expires_at, "UC1"))
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        store = PostgresCredentialStore("postgresql://fake")
        credential = store.get()

    assert credential == OAuthCredential(
        access_token="token", refresh_token="refresh", expires_at=expires_at, channel_id="UC1"
    )


def test_save_executes_upsert():
    mock_conn, mock_cursor = _mock_connect()
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        store = PostgresCredentialStore("postgresql://fake")
        store.save(
            OAuthCredential(
                access_token="token", refresh_token="refresh", expires_at=datetime.now(UTC), channel_id="UC1"
            )
        )

    mock_cursor.execute.assert_called_once()
    assert "INSERT INTO oauth_credentials" in mock_cursor.execute.call_args[0][0]
    mock_conn.commit.assert_called_once()
