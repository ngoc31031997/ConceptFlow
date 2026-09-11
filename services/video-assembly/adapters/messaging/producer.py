"""Event envelope builders for video_assembled / assembly_failed
(interface-contracts.md).

Only builds envelope dicts — the consumer writes them to the Outbox, and
OutboxRelay (adapters/persistence/relay.py) is the only place that
actually publishes (ADR-0013).
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

EVENTS_EXCHANGE = "events.direct"
EVENTS_ROUTING_KEY = "orchestrator"
SCHEMA_VERSION = "1.0"


def build_envelope(saga_id: str, project_id: str, payload: dict) -> dict:
    return {
        "message_id": str(uuid.uuid4()),
        "saga_id": saga_id,
        "project_id": project_id,
        "schema_version": SCHEMA_VERSION,
        "timestamp": datetime.now(UTC).isoformat(),
        "payload": payload,
    }


def video_assembled_envelope(
    saga_id: str, project_id: str, video_path: str, caption_path: str | None = None
) -> dict:
    payload = {"event_type": "video_assembled", "video_path": video_path}
    if caption_path:
        # Absent rather than null when there is no caption track (CR-015
        # FR38.4) — mirrors how thumbnail_path already flows downstream.
        payload["caption_path"] = caption_path
    return build_envelope(saga_id, project_id, payload)


def assembly_failed_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "assembly_failed", "error_message": error_message}
    )


def qc_completed_envelope(
    saga_id: str,
    project_id: str,
    status: str,
    findings: list[dict],
    reason: str | None = None,
) -> dict:
    """CR-021 — the ONLY event `qc_video` ever produces.

    There is deliberately no `qc_failed` counterpart (FR61.4 / LLD D2): a
    technical failure (no layout marks, ffmpeg error, unreadable file) still
    publishes this event with status="not_scored" and a reason, so the project
    still reaches ready_to_publish. A broken gate must not become a locked gate.

    `status` is one of "passed" (no findings), "has_findings", "not_scored".
    `reason` explains the "not_scored" case and is null otherwise.
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "qc_completed",
            "status": status,
            "reason": reason,
            "findings": findings,
        },
    )


def channel_asset_normalized_envelope(
    saga_id: str, project_id: str, kind: str, asset_id: str, render_quality: str, version: int
) -> dict:
    """CR-023 D1/D8 correction — the one event shaped enough for Orchestrator's
    channel_asset_pointers projection (handle_step_event.go's
    handleChannelAssetProjection reads exactly these field names: kind,
    render_quality, asset_id, version). Published both after ingesting
    rendering's channel_asset_rendered and after normalizing a Creator
    upload — either path ends with a new active row in this service's own
    channel_assets table, and this is how Orchestrator finds out about it.
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "channel_asset_normalized",
            "kind": kind,
            "asset_id": asset_id,
            "render_quality": render_quality,
            "version": version,
        },
    )
