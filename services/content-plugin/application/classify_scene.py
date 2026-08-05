"""Classify Scene use cases.

Implements the two core processes from business-logic-model.md:
- Classify a single Scene (validate + delegate to plugin)
- Classify a batch of Scenes with fail-fast semantics (used by the
  classify_scenes AMQP command handler)
"""

from dataclasses import dataclass

from domain.errors import InvalidCategoryError, InvalidSceneError, PluginNotFoundError
from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginRegistryPort


class ClassifySceneUseCase:
    """Classifies a single scene against a chosen plugin (Rule 1-4)."""

    def __init__(self, registry: ContentPluginRegistryPort) -> None:
        self._registry = registry

    def execute(self, plugin_id: str, scene: Scene) -> ClassificationResult:
        plugin = self._registry.get(plugin_id)
        if plugin is None:
            raise PluginNotFoundError(plugin_id)

        if not scene.narration_text.strip():
            raise InvalidSceneError(scene.scene_index, "narration_text must not be empty")

        if scene.category_hint not in plugin.supported_categories:
            raise InvalidCategoryError(scene.category_hint, plugin_id, plugin.supported_categories)

        return plugin.classify(scene)


@dataclass(frozen=True)
class BatchClassificationSuccess:
    results: list[ClassificationResult]


@dataclass(frozen=True)
class BatchClassificationFailure:
    error_message: str


BatchClassificationOutcome = BatchClassificationSuccess | BatchClassificationFailure


class ClassifyScenesBatchUseCase:
    """Classifies every scene in a project, fail-fast on the first error
    (Business Logic Model — "Handle classify_scenes Command")."""

    def __init__(self, single_scene_use_case: ClassifySceneUseCase) -> None:
        self._classify_scene = single_scene_use_case

    def execute(self, plugin_id: str, scenes: list[Scene]) -> BatchClassificationOutcome:
        results: list[ClassificationResult] = []
        for scene in scenes:
            try:
                results.append(self._classify_scene.execute(plugin_id, scene))
            except (PluginNotFoundError, InvalidCategoryError, InvalidSceneError) as exc:
                return BatchClassificationFailure(error_message=str(exc))
        return BatchClassificationSuccess(results=results)
