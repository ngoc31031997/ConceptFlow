"""CR-055: messaging/persistence plumbing copied between the Python services must stay identical.

tts, rendering, video-assembly and publisher each carry their own outbox, relay,
inbox, purge and test fake (separate Docker build contexts, no shared package —
ADR-0001). A fix made in one copy and forgotten in the others is the failure this
pins: every file in a group must match the first service's copy.

When a service genuinely needs its own version of one of these files, take that
service out of the group below and say why in a comment — do not loosen the check.
"""

from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[2] / "services"

ALL_FOUR = ["tts", "rendering", "video-assembly", "publisher"]

GROUPS = [
    ("adapters/persistence/outbox.py", ALL_FOUR),
    ("adapters/persistence/relay.py", ALL_FOUR),
    ("adapters/persistence/inbox.py", ALL_FOUR),
    # video-assembly and publisher extend db.py / cancellation.py with their own tables and steps.
    ("adapters/persistence/db.py", ["tts", "rendering"]),
    ("adapters/messaging/cancellation.py", ["tts", "rendering"]),
    # video-assembly's fake also models its channel-asset tables.
    ("tests/adapters/fake_postgres.py", ["tts", "rendering", "publisher"]),
]

# purge.py differs by design in its docstring's first line and SERVICE_NAME only.
PURGE = ("adapters/messaging/purge.py", ["tts", "rendering", "video-assembly"])


def _purge_body(text: str) -> str:
    lines = text.splitlines()[1:]
    return "\n".join(line for line in lines if not line.startswith("SERVICE_NAME = "))


def _mismatches(rel: str, services: list[str], normalize=lambda text: text) -> list[str]:
    reference = normalize((ROOT / services[0] / rel).read_text())
    return [s for s in services[1:] if normalize((ROOT / s / rel).read_text()) != reference]


@pytest.mark.parametrize(("rel", "services"), GROUPS, ids=[g[0] for g in GROUPS])
def test_copies_are_identical(rel, services):
    drifted = _mismatches(rel, services)
    assert not drifted, (
        f"{rel} in {drifted} differs from {services[0]}'s copy. "
        f"Apply the same change to every copy in {services}."
    )


def test_purge_copies_differ_only_by_service_name():
    rel, services = PURGE
    drifted = _mismatches(rel, services, _purge_body)
    assert not drifted, (
        f"{rel} in {drifted} differs from {services[0]}'s copy beyond the docstring's "
        f"first line and SERVICE_NAME. Apply the same change to every copy in {services}."
    )
