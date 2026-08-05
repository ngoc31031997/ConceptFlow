"""Tests for ContentPluginRegistry dynamic discovery (ADR-0006)."""

from adapters.plugins.registry import ContentPluginRegistry


def test_discovers_programming_plugin() -> None:
    registry = ContentPluginRegistry.discover()

    programming = registry.get("programming")
    assert programming is not None
    assert programming.name == "Lập trình"
    assert set(programming.supported_categories) == {"algorithm", "concept"}


def test_list_all_returns_discovered_plugins() -> None:
    registry = ContentPluginRegistry.discover()
    plugin_ids = {p.plugin_id for p in registry.list_all()}
    assert "programming" in plugin_ids
