"""FastAPI router for Content Plugin Service (ADR-0008: /v1 prefix)."""

from fastapi import APIRouter, Request, Response

from adapters.api.schemas import ListPluginsResponse, PluginResponse
from adapters.logging.correlation import set_correlation_id
from application.list_plugins import ListPluginsUseCase


def create_v1_router(list_plugins_use_case: ListPluginsUseCase) -> APIRouter:
    router = APIRouter(prefix="/v1")

    @router.get("/plugins", response_model=ListPluginsResponse)
    def list_plugins(request: Request, response: Response) -> ListPluginsResponse:
        correlation_id = set_correlation_id(request.headers.get("X-Request-ID"))
        response.headers["X-Request-ID"] = correlation_id

        plugins = list_plugins_use_case.execute()
        return ListPluginsResponse(
            plugins=[
                PluginResponse(
                    plugin_id=p.plugin_id,
                    name=p.name,
                    supported_categories=list(p.supported_categories),
                )
                for p in plugins
            ]
        )

    return router


def create_health_router(is_ready: callable) -> APIRouter:
    """Unversioned /health endpoint — infra concern, not part of the
    public v1 API contract (Infrastructure Design, Question 3)."""
    router = APIRouter()

    @router.get("/health", include_in_schema=False)
    def health() -> dict:
        return {"status": "ok" if is_ready() else "not_ready"}

    return router
