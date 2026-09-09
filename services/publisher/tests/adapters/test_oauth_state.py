"""Unit tests for the OAuth state codec and nonce store (CR-012 FR33)."""

from __future__ import annotations

from adapters.youtube.oauth_state import NonceStore, decode_state, encode_state


def test_state_round_trips_the_app_project_and_nonce():
    state = encode_state("client-a", "project-1", "nonce-1")

    decoded = decode_state(state)

    assert (decoded.client_id, decoded.project_id, decoded.nonce) == (
        "client-a",
        "project-1",
        "nonce-1",
    )


def test_encoded_state_carries_no_padding():
    """Unpadded base64url — '=' would need escaping in the URL Google
    round-trips this through."""
    assert "=" not in encode_state("client-a", "project-1", "nonce-1")


def test_state_survives_a_null_project():
    decoded = decode_state(encode_state("client-a", None, "nonce-1"))

    assert decoded.project_id is None


def test_a_bare_project_id_still_decodes(): 
    """A pre-CR-012 consent already in flight when this deploys (FR33.3)."""
    decoded = decode_state("project-1")

    assert decoded.project_id == "project-1"
    assert decoded.client_id == ""
    assert decoded.nonce == ""


def test_empty_state_decodes_to_none():
    assert decode_state(None) is None
    assert decode_state("") is None


def test_a_nonce_is_accepted_once_then_rejected():
    store = NonceStore()
    nonce = store.issue()

    assert store.consume(nonce) is True
    assert store.consume(nonce) is False


def test_an_unissued_nonce_is_rejected():
    assert NonceStore().consume("never-issued") is False


def test_an_expired_nonce_is_rejected():
    store = NonceStore(ttl_seconds=0)
    nonce = store.issue()

    assert store.consume(nonce) is False
