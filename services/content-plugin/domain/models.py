"""Domain value objects for the Content Plugin Service.

Pure Python — no dependency on FastAPI, aio-pika, or any adapter.
"""

from dataclasses import dataclass


@dataclass(frozen=True)
class Scene:
    """A single scene extracted from a Creator's script.

    `category_hint` is set by the Creator via the GUI (Story B2) — this
    service validates it, it does not infer category from `narration_text`.
    """

    scene_index: int
    narration_text: str
    category_hint: str
    illustration_hint: str | None = None
    code_snippet: str | None = None


@dataclass(frozen=True)
class ClassificationResult:
    """Result of classifying a single Scene against a plugin."""

    scene_index: int
    category: str
    animation_template_id: str


@dataclass(frozen=True)
class PluginInfo:
    """Metadata describing a registered content-type plugin."""

    plugin_id: str
    name: str
    supported_categories: tuple[str, ...]
