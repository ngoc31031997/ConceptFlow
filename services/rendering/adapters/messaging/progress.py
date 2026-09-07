"""Progress publisher for long-running renders (CR-003 FR11.4).

Rendering is the only Python service that publishes progress directly. Every
other step is short enough that the Orchestrator's own step-level "completed"
messages are timely, but a render is minutes of silence — long enough that a
Creator cannot tell a working render from a hung one.

Published to `progress.fanout` (ADR-0017), fire-and-forget: progress is UX-only
(messaging-design.md), so a failure to publish must never fail a render.

Deliberately NOT routed through the Outbox. The Outbox exists to make state
transitions durable and exactly-once; a heartbeat is the opposite — it is only
useful live, and replaying a stale one after a restart would be misleading.
"""

from __future__ import annotations

import json
import logging

import aio_pika

logger = logging.getLogger(__name__)

PROGRESS_EXCHANGE = "progress.fanout"
RENDER_STEP = "render_scenes"


class ProgressPublisher:
    def __init__(self, exchange: aio_pika.abc.AbstractExchange) -> None:
        self._exchange = exchange

    async def publish_render_heartbeat(
        self, project_id: str, elapsed_seconds: float, animation_index: int | None
    ) -> None:
        """Reports that a render is still running.

        There is no percentage here on purpose: Manim gives no reliable total
        animation count (animations inside loops make a static count of
        `self.play` calls wrong), and a made-up percentage that stalls or jumps
        backwards is worse than an honest elapsed time.
        """
        message = {
            "project_id": project_id,
            "step": RENDER_STEP,
            "status": "in_progress",
            "elapsed_seconds": round(elapsed_seconds, 1),
        }
        if animation_index is not None:
            message["animation_index"] = animation_index

        try:
            await self._exchange.publish(
                aio_pika.Message(json.dumps(message).encode()), routing_key=""
            )
        except Exception:  # noqa: BLE001 — progress is UX-only, never fail a render
            logger.exception("Could not publish render heartbeat for %s", project_id)
