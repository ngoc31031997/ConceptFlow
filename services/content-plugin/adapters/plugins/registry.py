"""ContentPluginRegistry — dynamic plugin discovery (ADR-0006).

On startup, recursively scans this package's subdirectories for .py
files, imports them, and registers any class implementing
ContentPluginPort. A plugin file that fails to import or does not
implement the port is logged and skipped — it must never crash the
service (Low-Level Design, Question 3).
"""

from __future__ import annotations

import importlib
import inspect
import logging
import pkgutil
from pathlib import Path

from domain.ports import ContentPluginPort, ContentPluginRegistryPort

logger = logging.getLogger(__name__)


class ContentPluginRegistry(ContentPluginRegistryPort):
    def __init__(self, plugins: list[ContentPluginPort]) -> None:
        self._by_id = {p.plugin_id: p for p in plugins}

    def get(self, plugin_id: str) -> ContentPluginPort | None:
        return self._by_id.get(plugin_id)

    def list_all(self) -> list[ContentPluginPort]:
        return list(self._by_id.values())

    @classmethod
    def discover(cls, package_name: str = "adapters.plugins") -> ContentPluginRegistry:
        """Scan `package_name` (and subpackages) for ContentPluginPort
        implementations and instantiate one registry containing all of
        them.
        """
        discovered: list[ContentPluginPort] = []
        package = importlib.import_module(package_name)
        package_path = Path(package.__file__).parent

        for module_info in pkgutil.walk_packages([str(package_path)], prefix=f"{package_name}."):
            if module_info.name.endswith(".registry"):
                continue  # don't try to import this file itself
            try:
                module = importlib.import_module(module_info.name)
            except Exception:
                logger.warning("Skipping plugin module '%s': import failed", module_info.name, exc_info=True)
                continue

            for _, obj in inspect.getmembers(module, inspect.isclass):
                if obj is ContentPluginPort or not issubclass(obj, ContentPluginPort):
                    continue
                if inspect.isabstract(obj):
                    continue
                try:
                    discovered.append(obj())
                except Exception:
                    logger.warning(
                        "Skipping plugin class '%s' in '%s': instantiation failed",
                        obj.__name__,
                        module_info.name,
                        exc_info=True,
                    )

        logger.info("Discovered %d content plugin(s): %s", len(discovered), [p.plugin_id for p in discovered])
        return cls(discovered)
