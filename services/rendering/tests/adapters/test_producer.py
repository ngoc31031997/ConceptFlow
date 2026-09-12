"""script_validated_envelope must carry clip_marks (bug report, 2026-09-12) —
otherwise Orchestrator/GUI have no way to warn "chưa có clip nào sẽ được tạo"
before TTS runs, only after."""

from adapters.messaging.producer import script_validated_envelope


def test_script_validated_envelope_carries_clip_marks():
    envelope = script_validated_envelope(
        "saga-1", "proj-1", ["dòng một"], ["visual"], [], [], [],
        clip_marks=[{"kind": "clip", "name": "vi du", "index": 0, "t_start": 1.0, "t_end": 5.0}],
    )

    assert envelope["payload"]["clip_marks"] == [
        {"kind": "clip", "name": "vi du", "index": 0, "t_start": 1.0, "t_end": 5.0},
    ]


def test_script_validated_envelope_clip_marks_defaults_to_empty_list():
    envelope = script_validated_envelope("saga-1", "proj-1", ["x"], [], [], [], [])

    assert envelope["payload"]["clip_marks"] == []
