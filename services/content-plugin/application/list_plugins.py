"""ListPluginsUseCase — returns metadata for every registered plugin."""

from domain.models import PluginInfo
from domain.ports import ContentPluginRegistryPort


class ListPluginsUseCase:
    def __init__(self, registry: ContentPluginRegistryPort) -> None:
        self._registry = registry

    def execute(self) -> list[PluginInfo]:
        return [plugin.info() for plugin in self._registry.list_all()]
