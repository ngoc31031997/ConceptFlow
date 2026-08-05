"""Unit tests for ClassifySceneUseCase and ClassifyScenesBatchUseCase.

Uses a FakeContentPluginRegistry/FakePlugin — no FastAPI, no AMQP, no
real plugin discovery involved (per dependency-injection.md).
"""

import pytest

from application.classify_scene import (
    BatchClassificationFailure,
    BatchClassificationSuccess,
    ClassifyScenesBatchUseCase,
    ClassifySceneUseCase,
)
from application.list_plugins import ListPluginsUseCase
from domain.errors import InvalidCategoryError, InvalidSceneError, PluginNotFoundError
from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginPort, ContentPluginRegistryPort

TEMPLATE_BY_CATEGORY = {
    "algorithm": "algorithm_visualization",
    "concept": "concept_illustration",
}


class FakeProgrammingPlugin(ContentPluginPort):
    @property
    def plugin_id(self) -> str:
        return "programming"

    @property
    def name(self) -> str:
        return "Lập trình"

    @property
    def supported_categories(self) -> tuple[str, ...]:
        return ("algorithm", "concept")

    def classify(self, scene: Scene) -> ClassificationResult:
        return ClassificationResult(
            scene_index=scene.scene_index,
            category=scene.category_hint,
            animation_template_id=TEMPLATE_BY_CATEGORY[scene.category_hint],
        )


class FakeContentPluginRegistry(ContentPluginRegistryPort):
    def __init__(self, plugins: list[ContentPluginPort]) -> None:
        self._by_id = {p.plugin_id: p for p in plugins}

    def get(self, plugin_id: str) -> ContentPluginPort | None:
        return self._by_id.get(plugin_id)

    def list_all(self) -> list[ContentPluginPort]:
        return list(self._by_id.values())


@pytest.fixture
def registry() -> FakeContentPluginRegistry:
    return FakeContentPluginRegistry([FakeProgrammingPlugin()])


def make_scene(index: int = 0, category_hint: str = "algorithm", narration: str = "some text") -> Scene:
    return Scene(scene_index=index, narration_text=narration, category_hint=category_hint)


class TestClassifySceneUseCase:
    def test_classifies_valid_scene(self, registry: FakeContentPluginRegistry) -> None:
        use_case = ClassifySceneUseCase(registry)
        result = use_case.execute("programming", make_scene(category_hint="algorithm"))
        assert result.category == "algorithm"
        assert result.animation_template_id == "algorithm_visualization"

    def test_raises_when_plugin_not_found(self, registry: FakeContentPluginRegistry) -> None:
        use_case = ClassifySceneUseCase(registry)
        with pytest.raises(PluginNotFoundError):
            use_case.execute("unknown_plugin", make_scene())

    def test_raises_when_category_hint_invalid(self, registry: FakeContentPluginRegistry) -> None:
        use_case = ClassifySceneUseCase(registry)
        with pytest.raises(InvalidCategoryError):
            use_case.execute("programming", make_scene(category_hint="not_a_real_category"))

    def test_raises_when_narration_text_empty(self, registry: FakeContentPluginRegistry) -> None:
        use_case = ClassifySceneUseCase(registry)
        with pytest.raises(InvalidSceneError):
            use_case.execute("programming", make_scene(narration="   "))


class TestClassifyScenesBatchUseCase:
    def test_all_scenes_succeed(self, registry: FakeContentPluginRegistry) -> None:
        single = ClassifySceneUseCase(registry)
        batch = ClassifyScenesBatchUseCase(single)
        scenes = [make_scene(0, "algorithm"), make_scene(1, "concept")]

        outcome = batch.execute("programming", scenes)

        assert isinstance(outcome, BatchClassificationSuccess)
        assert len(outcome.results) == 2

    def test_fail_fast_on_first_invalid_scene(self, registry: FakeContentPluginRegistry) -> None:
        single = ClassifySceneUseCase(registry)
        batch = ClassifyScenesBatchUseCase(single)
        scenes = [make_scene(0, "algorithm"), make_scene(1, "invalid_category")]

        outcome = batch.execute("programming", scenes)

        assert isinstance(outcome, BatchClassificationFailure)
        assert "invalid_category" in outcome.error_message


class TestListPluginsUseCase:
    def test_lists_registered_plugins(self, registry: FakeContentPluginRegistry) -> None:
        use_case = ListPluginsUseCase(registry)
        plugins = use_case.execute()
        assert len(plugins) == 1
        assert plugins[0].plugin_id == "programming"
        assert plugins[0].supported_categories == ("algorithm", "concept")
