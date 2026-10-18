"""A valid scene light for storyboard fixtures: every colour is one palette role."""

from __future__ import annotations


def light(role: str) -> dict:
    """The `light` of a scene whose colours are all the palette role `role`."""
    return {
        "source": "cửa sổ",
        "x": 0.7,
        "y": 0.2,
        "color": role,
        "intensity": 0.8,
        "ambient": role,
        "atmosphere": role,
        "haze": 0.3,
        "focus": "figures",
        "grade": {"warmth": 0.1, "contrast": 0, "saturation": 0},
    }
