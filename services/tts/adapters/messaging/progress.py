"""Progress publisher for synthesize_speech (CR-029).

Mirrors services/rendering/adapters/messaging/progress.py: a batch of scenes
can take long enough (many scenes, a metered engine with real network calls)
that silence between the "synthesizing_speech" status and the final
speech_synthesized event reads as a hang rather than work in progress.

Published to `progress.fanout` (ADR-0017), fire-and-forget and deliberately
NOT routed through the Outbox — same reasoning as rendering's heartbeat: a
progress ping is UX-only, never part of the durable, exactly-once state
Outbox/Inbox exist to protect (messaging-design.md).
"""

from __future__ import annotations

import json
import logging

import aio_pika

logger = logging.getLogger(__name__)

PROGRESS_EXCHANGE = "progress.fanout"
SYNTHESIZE_SPEECH_STEP = "synthesize_speech"


class ProgressPublisher:
    def __init__(self, exchange: aio_pika.abc.AbstractExchange) -> None:
        self._exchange = exchange

    async def publish_scene_progress(
        self, project_id: str, scene_index: int, scene_total: int
    ) -> None:
        """Reports that one more scene's narration has finished synthesizing.

        Bắn theo đơn vị hoàn thành (1 câu xong = 1 event), không theo tick
        thời gian — CR-029: giữ tần suất event thấp (tối đa scene_total event
        cho cả bước), không tạo tải thêm lên RabbitMQ/SSE.
        """
        message = {
            "project_id": project_id,
            "step": SYNTHESIZE_SPEECH_STEP,
            "status": "in_progress",
            "scene_index": scene_index,
            "scene_total": scene_total,
        }
        try:
            await self._exchange.publish(
                aio_pika.Message(json.dumps(message).encode()), routing_key=""
            )
        except Exception:  # noqa: BLE001 — progress is UX-only, never fail a batch
            logger.exception("Could not publish speech progress for %s", project_id)
