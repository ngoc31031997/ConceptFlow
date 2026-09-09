"""Unit tests for SubtitleCue.shifted_by (CR-015 / ADR-0027).

This is the single place a subtitle timestamp gets shifted — both the .ass
and .srt serializers receive already-shifted cues, so this is the one test
that has to be right for both of them to stay in sync.
"""

from __future__ import annotations

from domain.models import SubtitleCue


def test_shifted_by_moves_both_start_and_end():
    cue = SubtitleCue(scene_index=0, text="hi", start_time=10.0, end_time=12.0)

    shifted = cue.shifted_by(0.5)

    assert shifted.start_time == 10.5
    assert shifted.end_time == 12.5


def test_shifted_by_preserves_scene_index_and_text():
    cue = SubtitleCue(scene_index=3, text="hello", start_time=0.0, end_time=1.0)

    shifted = cue.shifted_by(2.0)

    assert shifted.scene_index == 3
    assert shifted.text == "hello"


def test_shifted_by_zero_returns_the_same_cue():
    """Not just an equal cue — the same object, so a caller that always calls
    shifted_by(lead_in) doesn't pay for a copy when there is no lead-in."""
    cue = SubtitleCue(scene_index=0, text="hi", start_time=1.0, end_time=2.0)

    assert cue.shifted_by(0.0) is cue


def test_original_cue_is_unchanged():
    cue = SubtitleCue(scene_index=0, text="hi", start_time=1.0, end_time=2.0)

    cue.shifted_by(5.0)

    assert cue.start_time == 1.0
    assert cue.end_time == 2.0
