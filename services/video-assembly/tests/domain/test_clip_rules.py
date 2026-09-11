"""Unit tests for domain/clip_rules.py (CR-007 FR19.6/19.7) — pure functions,
no ffmpeg, no I/O."""

from __future__ import annotations

from domain.clip_rules import ClipThresholds, validate_clip_duration


def test_short_preset_accepts_duration_within_max() -> None:
    thresholds = ClipThresholds()
    assert validate_clip_duration(45.0, "short", thresholds) is None


def test_short_preset_rejects_75_seconds() -> None:
    thresholds = ClipThresholds()
    error = validate_clip_duration(75.0, "short", thresholds)
    assert error is not None
    assert "short" in error
    assert "75" in error


def test_long_preset_accepts_75_seconds() -> None:
    thresholds = ClipThresholds()
    assert validate_clip_duration(75.0, "long", thresholds) is None


def test_long_preset_rejects_below_minimum() -> None:
    thresholds = ClipThresholds()
    error = validate_clip_duration(53.5, "long", thresholds)
    assert error is not None
    assert "long" in error


def test_long_preset_rejects_above_maximum() -> None:
    thresholds = ClipThresholds()
    error = validate_clip_duration(200.0, "long", thresholds)
    assert error is not None
    assert "long" in error


def test_long_preset_boundaries_are_inclusive() -> None:
    thresholds = ClipThresholds()
    assert validate_clip_duration(60.0, "long", thresholds) is None
    assert validate_clip_duration(180.0, "long", thresholds) is None


def test_short_preset_boundary_is_inclusive() -> None:
    thresholds = ClipThresholds()
    assert validate_clip_duration(60.0, "short", thresholds) is None


def test_unknown_preset_reports_error() -> None:
    thresholds = ClipThresholds()
    error = validate_clip_duration(30.0, "medium", thresholds)
    assert error is not None


def test_thresholds_read_from_env(monkeypatch) -> None:
    """C2b — ngưỡng preset là config, không hardcode."""
    monkeypatch.setenv("CLIP_PRESET_SHORT_MAX_SECONDS", "30")
    monkeypatch.setenv("CLIP_PRESET_LONG_MIN_SECONDS", "30")
    monkeypatch.setenv("CLIP_PRESET_LONG_MAX_SECONDS", "90")

    thresholds = ClipThresholds.from_env()

    assert thresholds.short_max_seconds == 30.0
    assert validate_clip_duration(45.0, "short", thresholds) is not None
    assert validate_clip_duration(45.0, "long", thresholds) is None


def test_thresholds_from_env_falls_back_on_unparseable_value(monkeypatch) -> None:
    monkeypatch.setenv("CLIP_PRESET_SHORT_MAX_SECONDS", "not-a-number")
    thresholds = ClipThresholds.from_env()
    assert thresholds.short_max_seconds == 60.0
