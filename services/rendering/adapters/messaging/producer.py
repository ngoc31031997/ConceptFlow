"""Event envelope builders for the Rendering Service's 2 event types
(rendering_completed / rendering_failed — Manim-script input mode).

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


def rendering_completed_envelope(
    saga_id: str,
    project_id: str,
    video_path: str,
    wait_offsets: list[float],
    video_duration_seconds: float,
    layout_marks: list[dict] | None = None,
) -> dict:
    """The `rendering_completed` event.

    wait_offsets[i] is where narration segment i actually starts in the
    rendered video. The Orchestrator validates it against its own scene count
    before passing it to Video Assembly.

    layout_marks: the on-screen geometry at each
    narration mark, which the Orchestrator stores and hands to `qc_video`.
    Best-effort upstream, so an empty list is a normal value, not an error.
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "rendering_completed",
            "video_path": video_path,
            "wait_offsets": wait_offsets,
            "video_duration_seconds": video_duration_seconds,
            "layout_marks": layout_marks or [],
        },
    )


def rendering_failed_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "rendering_failed", "error_message": error_message}
    )


def script_validated_envelope(
    saga_id: str,
    project_id: str,
    narrations: list[str],
    visuals: list[str],
    beats: list[tuple[int, str]],
    chapters: list[tuple[int, str]],
    warnings: list[str],
    layout_warnings: list[dict] | None = None,
    scene_class_name: str = "",
    engine: str = "manim",
) -> dict:
    """Kết quả cổng kiểm tra, chạy trước TTS.

    `scenes` mang đúng hình dạng mà `script_parsed` từng mang, để Orchestrator
    và các bước phía sau không phải đổi cách đọc. Khác biệt nằm ở nguồn: danh
    sách này đến từ việc **chạy** script (thứ tự runtime), không phải từ việc
    quét comment (thứ tự dòng).
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "script_validated",
            # Orchestrator stores the scene class and narrations from this event.
            "scene_class_name": scene_class_name,
            "engine": engine,
            "scenes": [
                {
                    "scene_index": index,
                    "narration_text": text,
                    # Cái gì trên khung hình lúc câu này được nói.
                    "visual": visuals[index] if index < len(visuals) else "",
                    "illustration_hint": None,
                    "code_snippet": None,
                    "code_language": None,
                }
                for index, text in enumerate(narrations)
            ],
            "beats": [{"scene_index": i, "id": value} for i, value in beats],
            "chapters": [{"scene_index": i, "title": value} for i, value in chapters],
            # Cảnh báo không chặn Saga; Orchestrator chuyển tiếp để GUI hiện ra.
            "warnings": warnings,
            # Bug report (2026-09-12): chồng lấn hình ảnh phát hiện ở lượt dry
            # (ConceptFlowScene._check_overlaps) — không chặn Saga, cùng nguyên
            # tắc như `warnings` ở trên, chỉ tách field vì nguồn gốc khác
            # (hình học runtime, không phải lint tĩnh).
            "layout_warnings": layout_warnings or [],
        },
    )


def validation_failed_envelope(saga_id: str, project_id: str, reason: str) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {"event_type": "validation_failed", "error_message": reason, "reason": reason},
    )


def channel_asset_rendered_envelope(
    saga_id: str,
    project_id: str,
    kind: str,
    video_path: str,
    video_duration_seconds: float,
    render_quality: str,
) -> dict:
    """Kết quả dựng intro/outro cố định.

    `project_id` ở đây không phải một project thật: intro/outro thuộc về kênh,
    không thuộc project nào, nên đây là id của lượt gọi admin flow, giữ
    lại để khớp hình dạng envelope chung và cho `inbox`/`outbox` có khoá.
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "channel_asset_rendered",
            "kind": kind,
            "video_path": video_path,
            "video_duration_seconds": video_duration_seconds,
            # One asset per quality, so the consumer must not have to
            # guess which quality this file was rendered at.
            "render_quality": render_quality,
        },
    )


def channel_asset_render_failed_envelope(
    saga_id: str, project_id: str, kind: str, error_message: str
) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "channel_asset_render_failed",
            "kind": kind,
            "error_message": error_message,
        },
    )
