"""ProgrammingPlugin — the content-type plugin for the programming domain.

Implements FR1.2: classifies scenes as either "algorithm" (algorithms &
data structures) or "concept" (general programming concepts). The
category is decided by the Creator via the GUI (category_hint) — this
plugin only maps a validated category to its animation template
(business-rules.md, Rule 1 & Rule 2).
"""

from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginPort

_TEMPLATE_BY_CATEGORY = {
    "algorithm": "algorithm_visualization",
    "concept": "concept_illustration",
}


class ProgrammingPlugin(ContentPluginPort):
    @property
    def plugin_id(self) -> str:
        return "programming"

    @property
    def name(self) -> str:
        return "Lập trình"

    @property
    def supported_categories(self) -> tuple[str, ...]:
        return tuple(_TEMPLATE_BY_CATEGORY.keys())

    def classify(self, scene: Scene) -> ClassificationResult:
        # scene.category_hint is guaranteed valid by ClassifySceneUseCase
        # before this is called (Rule 1: Category Source of Truth).
        return ClassificationResult(
            scene_index=scene.scene_index,
            category=scene.category_hint,
            animation_template_id=_TEMPLATE_BY_CATEGORY[scene.category_hint],
        )
