"""Ports (abstract interfaces) that adapters must implement.

Per Hexagonal architecture (ADR-0002), domain code depends only on these
abstractions — never on Manim, aio-pika, or asyncpg directly.
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Any

from domain.models import SceneRenderRequest


class AnimationRendererPort(ABC):
    """Renders one scene to a video file. Concrete implementation
    (ManimAnimationRenderer) lives under adapters/rendering/."""

    @abstractmethod
    def render(self, request: SceneRenderRequest, output_path: str) -> float:
        """Renders animation to output_path, returns actual duration_seconds.

        Raises:
            domain.errors.AnimationEngineError: if the engine fails or times out.
        """


class AnimationTemplatePort(ABC):
    """An animation template: builds a configured Manim Scene for one
    render request. Concrete implementations live under
    adapters/rendering/templates/ and are discovered dynamically at
    startup (ADR-0015, mirrors ADR-0006's ContentPluginPort)."""

    @property
    @abstractmethod
    def template_id(self) -> str: ...

    @abstractmethod
    def build_scene(self, request: SceneRenderRequest) -> Any:
        """Constructs a configured manim.Scene instance ready to .render().

        Returns `Any` (not `manim.Scene`) so the domain layer stays free
        of the Manim import — the concrete return type is documented,
        not enforced, at this boundary.
        """
