"""Unit tests for AnimationTemplateRegistry (ADR-0015).

Constructs the registry directly from fakes rather than calling
discover() against the real templates/ package, so these tests don't
require Manim to be importable — discover()'s real behavior against the
actual template plugins is verified manually/in Docker once Manim is
installed (see code/README.md).
"""

from __future__ import annotations

from adapters.rendering.registry import AnimationTemplateRegistry
from domain.models import SceneRenderRequest
from domain.ports import AnimationTemplatePort


class FakeTemplate(AnimationTemplatePort):
    def __init__(self, template_id: str) -> None:
        self._template_id = template_id

    @property
    def template_id(self) -> str:
        return self._template_id

    def build_scene(self, request: SceneRenderRequest):
        return object()


def test_get_returns_registered_template():
    registry = AnimationTemplateRegistry([FakeTemplate("concept_illustration")])

    template = registry.get("concept_illustration")

    assert template is not None
    assert template.template_id == "concept_illustration"


def test_get_returns_none_for_unknown_template():
    registry = AnimationTemplateRegistry([FakeTemplate("concept_illustration")])

    assert registry.get("nonexistent") is None


def test_list_all_returns_every_registered_template():
    registry = AnimationTemplateRegistry(
        [FakeTemplate("concept_illustration"), FakeTemplate("algorithm_visualization")]
    )

    ids = {t.template_id for t in registry.list_all()}

    assert ids == {"concept_illustration", "algorithm_visualization"}
