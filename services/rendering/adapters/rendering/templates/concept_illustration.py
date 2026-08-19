"""ConceptIllustrationTemplate — animation_template_id "concept_illustration"
(ADR-0015). Renders a concept/idea scene: the narration/illustration hint
as on-screen text, plus the code snippet in a fixed corner when present
(Business Rule 3-4, via adapters/rendering/templates/_code_display.py).

MVP scope: content is intentionally generic (title + supporting text +
optional code), not concept-specific animation — the architecturally
significant part of this unit is the pluggable template mechanism
(ADR-0015) and duration-matching logic, not bespoke animation art for
each possible programming concept.
"""

from __future__ import annotations

from manim import DOWN, RIGHT, UP, FadeIn, Scene, Text, VGroup

from adapters.rendering.templates._code_display import build_code_mobject
from domain.models import SceneRenderRequest
from domain.ports import AnimationTemplatePort

_ENTRANCE_SECONDS = 1.0


class ConceptIllustrationTemplate(AnimationTemplatePort):
    @property
    def template_id(self) -> str:
        return "concept_illustration"

    def build_scene(self, request: SceneRenderRequest) -> Scene:
        return _ConceptIllustrationScene(request)


class _ConceptIllustrationScene(Scene):
    def __init__(self, request: SceneRenderRequest) -> None:
        self._request = request
        super().__init__()

    def construct(self) -> None:
        request = self._request
        code_mobject = build_code_mobject(request)

        content_column = VGroup()
        if request.illustration_hint:
            content_column.add(Text(request.illustration_hint, font_size=36).set_color("#FFD54F"))
        content_column.add(Text(request.narration_text, font_size=28))
        content_column.arrange(direction=DOWN, buff=0.5)

        if code_mobject is not None:
            content_column.to_edge(RIGHT)
        else:
            content_column.move_to((0, 0, 0))

        elapsed = _ENTRANCE_SECONDS
        self.play(FadeIn(content_column, shift=UP), run_time=_ENTRANCE_SECONDS)
        if code_mobject is not None:
            self.play(FadeIn(code_mobject), run_time=_ENTRANCE_SECONDS)
            elapsed += _ENTRANCE_SECONDS

        remaining = max(request.duration_seconds - elapsed, 0.1)
        self.wait(remaining)
