"""The cancel registry: who is cancelled, what gets killed, what is not reported."""

import subprocess
import sys
import time
from datetime import UTC, datetime, timedelta

import pytest

from adapters.messaging.cancellation import REGISTRY, CancelAwareOutbox, CancelRegistry, run_cancellable


def iso(dt: datetime) -> str:
    return dt.astimezone(UTC).isoformat().replace("+00:00", "Z")


NOW = datetime(2026, 9, 26, 12, 0, 0, tzinfo=UTC)


def test_a_command_sent_before_the_cancel_is_cancelled_and_a_retry_after_it_is_not():
    reg = CancelRegistry()
    reg.cancel("p", NOW)
    assert reg.is_cancelled("p", iso(NOW - timedelta(seconds=5)))
    assert reg.is_cancelled("p", iso(NOW))
    assert not reg.is_cancelled("p", iso(NOW + timedelta(seconds=1)))  # the Retry


def test_other_projects_and_unstamped_commands_are_never_cancelled():
    reg = CancelRegistry()
    reg.cancel("p", NOW)
    assert not reg.is_cancelled("other", iso(NOW - timedelta(seconds=5)))
    assert not reg.is_cancelled("p", None)
    assert not reg.is_cancelled("p", "not a date")


def test_registering_a_process_inside_a_command_lets_cancel_kill_it():
    reg = CancelRegistry()
    proc = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(30)"], start_new_session=True)
    try:
        with reg.command("p", iso(NOW - timedelta(seconds=5))):
            reg.register(proc)
            reg.cancel("p", NOW)
            deadline = time.time() + 5
            while proc.poll() is None and time.time() < deadline:
                time.sleep(0.05)
        assert proc.poll() is not None, "the child must be dead after cancel"
    finally:
        if proc.poll() is None:
            proc.kill()


def test_a_process_started_after_the_cancel_is_killed_on_registration():
    reg = CancelRegistry()
    reg.cancel("p", NOW)
    proc = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(30)"], start_new_session=True)
    try:
        with reg.command("p", iso(NOW - timedelta(seconds=1))):
            reg.register(proc)
        deadline = time.time() + 5
        while proc.poll() is None and time.time() < deadline:
            time.sleep(0.05)
        assert proc.poll() is not None
    finally:
        if proc.poll() is None:
            proc.kill()


class _Recorder:
    def __init__(self):
        self.events = []

    async def enqueue(self, conn, *, aggregate_id, event_type, envelope):
        self.events.append(event_type)


@pytest.mark.asyncio
async def test_a_cancelled_command_reports_nothing_but_a_retry_does():
    reg = CancelRegistry()
    inner = _Recorder()
    outbox = CancelAwareOutbox(inner, reg)
    reg.cancel("p", NOW)

    with reg.command("p", iso(NOW - timedelta(seconds=3))):
        await outbox.enqueue(None, aggregate_id="p", event_type="rendering_failed", envelope={})
    with reg.command("p", iso(NOW + timedelta(seconds=3))):  # the Retry
        await outbox.enqueue(None, aggregate_id="p", event_type="rendering_completed", envelope={})

    assert inner.events == ["rendering_completed"]


def test_run_cancellable_is_plain_run_outside_a_command():
    result = run_cancellable([sys.executable, "-c", "print('hi')"])
    assert result.returncode == 0 and result.stdout.strip() == "hi"


def test_run_cancellable_inside_a_command_dies_on_cancel():
    import threading

    stamp = iso(NOW - timedelta(seconds=5))
    box = {}

    def work():
        with REGISTRY.command("run-cancel-p", stamp):
            box["result"] = run_cancellable([sys.executable, "-c", "import time; time.sleep(30)"])

    thread = threading.Thread(target=work)
    started = time.time()
    thread.start()
    time.sleep(0.4)
    REGISTRY.cancel("run-cancel-p", NOW)
    thread.join(timeout=5)
    try:
        assert not thread.is_alive(), "the run must return once its child is killed"
        assert box["result"].returncode != 0
        assert time.time() - started < 10
    finally:
        REGISTRY._cancelled_at.pop("run-cancel-p", None)
