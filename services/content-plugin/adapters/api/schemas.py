"""Pydantic request/response schemas for the REST API (adapters/api)."""

from pydantic import BaseModel


class PluginResponse(BaseModel):
    plugin_id: str
    name: str
    supported_categories: list[str]


class ListPluginsResponse(BaseModel):
    plugins: list[PluginResponse]
