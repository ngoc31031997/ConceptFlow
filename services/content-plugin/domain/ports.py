"""Ports (abstract interfaces) that adapters must implement.

Per the Hexagonal architecture (ADR-0002), domain code depends only on
these abstractions — never on a concrete plugin implementation or
infrastructure library.
"""

from abc import ABC, abstractmethod

from domain.models import ClassificationResult, PluginInfo, Scene


class ContentPluginPort(ABC):
    """A content-type plugin: classifies scenes for one educational domain.

    Concrete implementations (e.g. ProgrammingPlugin) live under
    adapters/plugins/ and are discovered dynamically at startup (ADR-0006).
    """

    @property
    @abstractmethod
    def plugin_id(self) -> str: ...

    @property
    @abstractmethod
    def name(self) -> str: ...

    @property
    @abstractmethod
    def supported_categories(self) -> tuple[str, ...]: ...

    @abstractmethod
    def classify(self, scene: Scene) -> ClassificationResult:
        """Classify a single scene. Assumes scene.category_hint is already
        validated against supported_categories by the caller."""
        ...

    def info(self) -> PluginInfo:
        return PluginInfo(
            plugin_id=self.plugin_id,
            name=self.name,
            supported_categories=self.supported_categories,
        )


class ContentPluginRegistryPort(ABC):
    """Read-only lookup of registered plugins.

    Discovery/loading mechanics belong to the concrete adapter
    (adapters/plugins/registry.py) — the application layer only needs
    lookup capability, expressed here as an abstraction (Dependency
    Inversion — use cases depend on this, not on the concrete registry).
    """

    @abstractmethod
    def get(self, plugin_id: str) -> ContentPluginPort | None: ...

    @abstractmethod
    def list_all(self) -> list[ContentPluginPort]: ...
