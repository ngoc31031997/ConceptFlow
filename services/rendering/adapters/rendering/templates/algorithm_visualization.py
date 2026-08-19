"""AlgorithmVisualizationTemplate — animation_template_id "algorithm_visualization"
(ADR-0015). Renders an algorithm-walkthrough scene: narration text
appears step-by-step (paced across the scene's duration) alongside the
code snippet, when present, in a fixed corner (Business Rule 3-4).

MVP scope: content is intentionally generic (narration split into
sentence-like steps that appear one at a time), not a bespoke
step-by-step visualization per algorithm — the architecturally
significant part of this unit is the pluggable template mechanism
(ADR-0015) and duration-matching logic (Business Rule 2), not
algorithm-specific animation art.
"""

from __future__ import annotations

import re

from manim import DOWN, LEFT, RIGHT, FadeIn, Scene, Text, VGroup

from adapters.rendering.templates._code_display import build_code_mobject
from domain.models import SceneRenderRequest
from domain.ports import AnimationTemplatePort

_SENTENCE_SPLIT_RE = re.compile(r"(?<=[.!?])\s+")


class AlgorithmVisualizationTemplate(AnimationTemplatePort):
    @property
    def template_id(self) -> str:
        return "algorithm_visualization"

    def build_scene(self, request: SceneRenderRequest) -> Scene:
        return _AlgorithmVisualizationScene(request)


class _AlgorithmVisualizationScene(Scene):
    def __init__(self, request: SceneRenderRequest) -> None:
        self._request = request
        super().__init__()

    def construct(self) -> None:
        request = self._request
        code_mobject = build_code_mobject(request)

        steps = [s.strip() for s in _SENTENCE_SPLIT_RE.split(request.narration_text) if s.strip()]
        if not steps:
            steps = [request.narration_text]

        step_texts = VGroup(*(Text(step, font_size=28) for step in steps)).arrange(direction=DOWN, buff=0.4)
        if code_mobject is not None:
            step_texts.to_edge(RIGHT)
            code_mobject.to_edge(LEFT)
        else:
            step_texts.move_to((0, 0, 0))

        if code_mobject is not None:
            self.play(FadeIn(code_mobject), run_time=0.5)

        # Spread each step's entrance evenly across the scene's target
        # duration_seconds so the pacing roughly matches the audio
        # (Business Rule 2) — leftover time is absorbed by the final wait.
        per_step_seconds = max((request.duration_seconds - 0.5) / max(len(steps), 1), 0.3)
        for step in step_texts:
            self.play(FadeIn(step), run_time=min(per_step_seconds, 1.0))

        elapsed = 0.5 + min(per_step_seconds, 1.0) * len(steps)
        remaining = max(request.duration_seconds - elapsed, 0.1)
        self.wait(remaining)
