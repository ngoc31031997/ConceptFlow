"""Domain-specific exceptions for the Content Plugin Service."""


class ContentPluginDomainError(Exception):
    """Base class for all domain errors in this service."""


class PluginNotFoundError(ContentPluginDomainError):
    """Raised when a requested plugin_id is not registered."""

    def __init__(self, plugin_id: str) -> None:
        self.plugin_id = plugin_id
        super().__init__(f"Plugin '{plugin_id}' not found")


class InvalidCategoryError(ContentPluginDomainError):
    """Raised when a scene's category_hint is not supported by the plugin."""

    def __init__(self, category_hint: str, plugin_id: str, supported: tuple[str, ...]) -> None:
        self.category_hint = category_hint
        self.plugin_id = plugin_id
        self.supported = supported
        super().__init__(
            f"Category '{category_hint}' is not supported by plugin '{plugin_id}' "
            f"(supported: {', '.join(supported)})"
        )


class InvalidSceneError(ContentPluginDomainError):
    """Raised when a Scene fails basic validation (e.g. empty narration_text)."""

    def __init__(self, scene_index: int, reason: str) -> None:
        self.scene_index = scene_index
        self.reason = reason
        super().__init__(f"Scene {scene_index} is invalid: {reason}")
