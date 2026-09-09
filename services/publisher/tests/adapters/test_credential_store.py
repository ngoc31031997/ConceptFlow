"""Unit tests for PostgresCredentialStore (ADR-0016, ADR-0026).

Mocks psycopg2.connect so no real PostgreSQL instance is needed to run
the test suite. These tests therefore assert on the SQL issued, not on
its effect — the semantics the SQL encodes (upsert by channel, preserved
refresh token, single default) are exercised end-to-end against a real
database in the build-and-test stage.
"""

from __future__ import annotations

from datetime import UTC, datetime
from unittest.mock import MagicMock, patch

from adapters.persistence.credential_store import PostgresCredentialStore
from domain.models import OAuthCredential


def _mock_connect(fetchone_return=None, fetchall_return=()):
    mock_conn = MagicMock()
    mock_cursor = MagicMock()
    mock_cursor.fetchone.return_value = fetchone_return
    mock_cursor.fetchall.return_value = fetchall_return
    mock_conn.cursor.return_value.__enter__.return_value = mock_cursor
    mock_conn.__enter__.return_value = mock_conn
    return mock_conn, mock_cursor


def _row(channel_id="UC1", is_default=True, expires_at=None):
    return (
        channel_id,
        f"Channel {channel_id}",
        "client-a",
        "token",
        "refresh",
        expires_at or datetime.now(UTC),
        is_default,
    )


def test_get_returns_none_when_no_row():
    mock_conn, _ = _mock_connect(fetchone_return=None)
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        store = PostgresCredentialStore("postgresql://fake")
        assert store.get() is None


def test_get_returns_credential_when_row_exists():
    expires_at = datetime.now(UTC)
    mock_conn, _ = _mock_connect(fetchone_return=_row(expires_at=expires_at))
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        store = PostgresCredentialStore("postgresql://fake")
        credential = store.get()

    assert credential == OAuthCredential(
        access_token="token",
        refresh_token="refresh",
        expires_at=expires_at,
        channel_id="UC1",
        client_id="client-a",
        channel_title="Channel UC1",
        is_default=True,
    )


def test_get_without_channel_id_asks_for_the_default_channel():
    mock_conn, mock_cursor = _mock_connect(fetchone_return=_row())
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").get()

    sql, params = mock_cursor.execute.call_args[0]
    assert "is_default DESC" in sql
    assert params == ()


def test_get_with_channel_id_filters_on_that_channel():
    mock_conn, mock_cursor = _mock_connect(fetchone_return=_row(channel_id="UC_two"))
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").get("UC_two")

    sql, params = mock_cursor.execute.call_args[0]
    assert "WHERE channel_id = %s" in sql
    assert params == ("UC_two",)


def test_list_returns_every_connected_channel():
    mock_conn, _ = _mock_connect(fetchall_return=[_row("UC1"), _row("UC2", is_default=False)])
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        credentials = PostgresCredentialStore("postgresql://fake").list()

    assert [c.channel_id for c in credentials] == ["UC1", "UC2"]


def test_save_upserts_on_channel_id_not_a_fixed_row():
    """The pre-CR-012 statement wrote id=1 unconditionally, so connecting a
    second channel overwrote the first."""
    mock_conn, mock_cursor = _mock_connect()
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").save(
            OAuthCredential(
                access_token="token",
                refresh_token="refresh",
                expires_at=datetime.now(UTC),
                channel_id="UC1",
                client_id="client-a",
                channel_title="Kênh Một",
            )
        )

    sql = mock_cursor.execute.call_args[0][0]
    assert "INSERT INTO youtube_accounts" in sql
    assert "ON CONFLICT (channel_id) DO UPDATE" in sql
    mock_conn.commit.assert_called_once()


def test_save_keeps_the_stored_refresh_token_when_google_returns_none():
    """Google only issues a refresh token on the first consent, so a
    re-consent must not blank the stored one (FR31.6)."""
    mock_conn, mock_cursor = _mock_connect()
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").save(
            OAuthCredential(
                access_token="token",
                refresh_token="",
                expires_at=datetime.now(UTC),
                channel_id="UC1",
                client_id="client-a",
            )
        )

    sql = mock_cursor.execute.call_args[0][0]
    assert "COALESCE" in sql
    assert "youtube_accounts.refresh_token" in sql


def test_delete_promotes_another_channel_when_the_default_goes():
    mock_conn, mock_cursor = _mock_connect()
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").delete("UC1")

    statements = [call[0][0] for call in mock_cursor.execute.call_args_list]
    assert any("DELETE FROM youtube_accounts" in sql for sql in statements)
    assert any("SET is_default = TRUE" in sql for sql in statements)
    mock_conn.commit.assert_called_once()


def test_set_default_clears_the_previous_default_first():
    """The partial unique index rejects two defaults existing at once, so
    the order of these two statements is load-bearing."""
    mock_conn, mock_cursor = _mock_connect()
    with patch("adapters.persistence.credential_store.psycopg2.connect", return_value=mock_conn):
        PostgresCredentialStore("postgresql://fake").set_default("UC2")

    statements = [call[0][0] for call in mock_cursor.execute.call_args_list]
    assert "SET is_default = FALSE" in statements[0]
    assert "SET is_default = TRUE" in statements[1]
