"""AnimationTemplateRegistry — dynamic plugin discovery (ADR-0015, mirrors
Content Plugin Service's ContentPluginRegistry, ADR-0006).

On startup, recursively scans adapters/rendering/templates/ for .py
files, imports them, and registers any class implementing
AnimationTemplatePort. A template module that fails to import or does
not implement the port is logged and skipped — it must never crash the
service.
"""

from __future__ import annotations

import importlib
import inspect
import logging
import pkgutil
from pathlib import Path

from domain.ports import AnimationTemplatePort

logger = logging.getLogger(__name__)


class AnimationTemplateRegistry:
    def __init__(self, templates: list[AnimationTemplatePort]) -> None:
        self._by_id = {t.template_id: t for t in templates}

    def get(self, template_id: str) -> AnimationTemplatePort | None:
        return self._by_id.get(template_id)

    def list_all(self) -> list[AnimationTemplatePort]:
        return list(self._by_id.values())

    @classmethod
    def discover(cls, package_name: str = "adapters.rendering.templates") -> AnimationTemplateRegistry:
        discovered: list[AnimationTemplatePort] = []
        package = importlib.import_module(package_name)
        package_path = Path(package.__file__).parent

        for module_info in pkgutil.walk_packages([str(package_path)], prefix=f"{package_name}."):
            if module_info.name.rsplit(".", 1)[-1].startswith("_"):
                continue  # skip private helper modules (e.g. _code_display)
            try:
                module = importlib.import_module(module_info.name)
            except Exception:
                logger.warning(
                    "Skipping template module '%s': import failed", module_info.name, exc_info=True
                )
                continue

            for _, obj in inspect.getmembers(module, inspect.isclass):
                if obj is AnimationTemplatePort or not issubclass(obj, AnimationTemplatePort):
                    continue
                if inspect.isabstract(obj):
                    continue
                try:
                    discovered.append(obj())
                except Exception:
                    logger.warning(
                        "Skipping template class '%s' in '%s': instantiation failed",
                        obj.__name__,
                        module_info.name,
                        exc_info=True,
                    )

        logger.info(
            "Discovered %d animation template(s): %s", len(discovered), [t.template_id for t in discovered]
        )
        return cls(discovered)
