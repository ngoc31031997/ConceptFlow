"""Unit tests for the FastAPI REST layer using FastAPI's TestClient."""

from fastapi import FastAPI
from fastapi.testclient import TestClient

from adapters.api.router import create_health_router, create_v1_router
from application.list_plugins import ListPluginsUseCase
from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginPort, ContentPluginRegistryPort


class FakePlugin(ContentPluginPort):
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
        raise NotImplementedError


class FakeRegistry(ContentPluginRegistryPort):
    def get(self, plugin_id: str) -> ContentPluginPort | None:
        return None

    def list_all(self) -> list[ContentPluginPort]:
        return [FakePlugin()]


def build_test_app(ready: bool = True) -> FastAPI:
    app = FastAPI()
    app.include_router(create_v1_router(ListPluginsUseCase(FakeRegistry())))
    app.include_router(create_health_router(lambda: ready))
    return app


def test_list_plugins_returns_registered_plugins() -> None:
    client = TestClient(build_test_app())
    response = client.get("/v1/plugins")

    assert response.status_code == 200
    body = response.json()
    assert body["plugins"] == [
        {"plugin_id": "programming", "name": "Lập trình", "supported_categories": ["algorithm", "concept"]}
    ]


def test_list_plugins_echoes_request_id_header() -> None:
    client = TestClient(build_test_app())
    response = client.get("/v1/plugins", headers={"X-Request-ID": "test-correlation-id"})
    assert response.headers["X-Request-ID"] == "test-correlation-id"


def test_list_plugins_generates_request_id_when_missing() -> None:
    client = TestClient(build_test_app())
    response = client.get("/v1/plugins")
    assert response.headers["X-Request-ID"]  # non-empty, auto-generated


def test_health_reports_ready() -> None:
    client = TestClient(build_test_app(ready=True))
    assert client.get("/health").json() == {"status": "ok"}


def test_health_reports_not_ready() -> None:
    client = TestClient(build_test_app(ready=False))
    assert client.get("/health").json() == {"status": "not_ready"}
