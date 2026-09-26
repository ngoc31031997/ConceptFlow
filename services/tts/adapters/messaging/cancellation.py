"""Cancel requests from the Orchestrator (Creator pressed "Huỷ").

A worker with prefetch 1 is busy for the whole of a render, so a cancel queued
behind its own command would only arrive once it no longer matters. Cancels
therefore travel on a fanout exchange (`control.fanout`) that every worker
listens to through its own exclusive queue, independently of the command queue.

Cancelling is keyed by project and *time*, not by message id: the request
carries the moment it was made, and a command stamped at or before it is dead.
That kills the command that is running (its child processes are terminated),
kills a command still waiting in the queue, and leaves a later Retry — a new
command with a later timestamp — alone.

The worker that was stopped must not report the stop as a failure of the step:
the Orchestrator has already marked the step cancelled, and a late
`*_failed` event would land on a step the Creator may have retried by then.
`CancelAwareOutbox` drops events of a cancelled command for that reason.
"""

from __future__ import annotations

import contextlib
import contextvars
import json
import logging
import os
import signal
import subprocess
import threading
from datetime import datetime

logger = logging.getLogger(__name__)

CONTROL_EXCHANGE = "control.fanout"

# (project_id, command timestamp) of the command this task/thread is running.
_current: contextvars.ContextVar[tuple[str, str | None] | None] = contextvars.ContextVar(
    "current_command", default=None
)


def _parse(ts: str | None) -> datetime | None:
    if not ts:
        return None
    try:
        # Go writes RFC3339 with a "Z"; fromisoformat only takes it from 3.11.
        return datetime.fromisoformat(ts.replace("Z", "+00:00"))
    except ValueError:
        return None


def _kill(proc: subprocess.Popen) -> None:
    """Terminate a child and everything it spawned (Manim starts ffmpeg)."""
    try:
        os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
    except (ProcessLookupError, PermissionError):
        pass
    except OSError:
        with contextlib.suppress(OSError):
            proc.kill()


class CancelRegistry:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self._cancelled_at: dict[str, datetime] = {}
        self._procs: dict[str, set[subprocess.Popen]] = {}

    def cancel(self, project_id: str, at: datetime) -> None:
        with self._lock:
            self._cancelled_at[project_id] = at
            procs = list(self._procs.get(project_id, ()))
        logger.warning("cancel requested for project_id=%s, stopping %d process(es)", project_id, len(procs))
        for proc in procs:
            _kill(proc)

    def is_cancelled(self, project_id: str, command_ts: str | None) -> bool:
        with self._lock:
            cutoff = self._cancelled_at.get(project_id)
        ts = _parse(command_ts)
        return cutoff is not None and ts is not None and ts <= cutoff

    def current_cancelled(self) -> bool:
        cur = _current.get()
        return cur is not None and self.is_cancelled(cur[0], cur[1])

    def register(self, proc: subprocess.Popen) -> None:
        """Track a child process of the command running in this context, so a
        cancel can kill it. A cancel that landed before the process existed
        kills it at once."""
        cur = _current.get()
        if cur is None:
            return
        with self._lock:
            self._procs.setdefault(cur[0], set()).add(proc)
        if self.current_cancelled():
            _kill(proc)

    def unregister(self, proc: subprocess.Popen) -> None:
        cur = _current.get()
        if cur is None:
            return
        with self._lock:
            self._procs.get(cur[0], set()).discard(proc)

    @contextlib.contextmanager
    def command(self, project_id: str, command_ts: str | None):
        token = _current.set((project_id, command_ts))
        try:
            yield
        finally:
            _current.reset(token)


REGISTRY = CancelRegistry()


def run_cancellable(cmd: list[str], *, timeout: float | None = None) -> subprocess.CompletedProcess:
    """`subprocess.run(capture_output=True, text=True)` that a cancel can kill.

    Inside a command it starts the child in its own process group and registers
    it; outside one (tests, ad-hoc use) it is plain `subprocess.run`."""
    if _current.get() is None:
        return subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    proc = subprocess.Popen(
        cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, start_new_session=True
    )
    REGISTRY.register(proc)
    try:
        out, err = proc.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        _kill(proc)
        proc.communicate()
        raise
    finally:
        REGISTRY.unregister(proc)
    return subprocess.CompletedProcess(cmd, proc.returncode, out, err)


class CancelAwareOutbox:
    """Drops the events of a cancelled command instead of publishing them."""

    def __init__(self, inner, registry: CancelRegistry = REGISTRY) -> None:
        self._inner = inner
        self._registry = registry

    async def enqueue(self, conn, *, aggregate_id: str, event_type: str, envelope: dict) -> None:
        if self._registry.current_cancelled():
            logger.info("dropping %s for cancelled project_id=%s", event_type, aggregate_id)
            return
        await self._inner.enqueue(conn, aggregate_id=aggregate_id, event_type=event_type, envelope=envelope)


async def listen_for_cancels(channel, registry: CancelRegistry = REGISTRY):
    """Subscribe this worker to cancel requests. Returns the consumer tag."""
    import aio_pika

    exchange = await channel.declare_exchange(CONTROL_EXCHANGE, aio_pika.ExchangeType.FANOUT, durable=True)
    queue = await channel.declare_queue("", exclusive=True, auto_delete=True)
    await queue.bind(exchange)

    async def on_message(message) -> None:
        async with message.process(ignore_processed=True):
            try:
                body = json.loads(message.body)
                if body.get("type") != "cancel":
                    return
                at = _parse(body.get("at"))
                if body.get("project_id") and at is not None:
                    registry.cancel(body["project_id"], at)
            except (ValueError, TypeError, AttributeError):
                logger.warning("ignoring unreadable control message")

    return await queue.consume(on_message)
