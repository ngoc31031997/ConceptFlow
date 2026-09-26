import httpx
import pytest

from adapters.http.check_server import create_check_app
from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.check_script import CheckDiagnostic, CheckOutcome


class Fake:
    def __init__(self, outcome=None, raises=None):
        self.outcome, self.raises, self.calls = outcome, raises, []

    def check(self, engine, code, scene_class_name):
        self.calls.append((engine, code, scene_class_name))
        if self.raises:
            raise self.raises
        return self.outcome


def client(fake):
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=create_check_app(fake)), base_url="http://r")


async def test_returns_diagnostics_and_raw():
    fake = Fake(CheckOutcome(False, [CheckDiagnostic("boom", 4)], "raw text"))
    r = await client(fake).post("/v1/check/remotion", json={"code": "c", "scene_class_name": "creator"})
    assert r.status_code == 200
    assert r.json() == {"ok": False, "diagnostics": [{"message": "boom", "line": 4}], "raw": "raw text"}
    assert fake.calls == [("remotion", "c", "creator")]


async def test_manim_route_uses_manim():
    fake = Fake(CheckOutcome(True))
    r = await client(fake).post("/v1/check/manim", json={"code": "c", "scene_class_name": "XScene"})
    assert r.json()["ok"] is True and fake.calls[0][0] == "manim"


async def test_bad_input_is_400_and_a_broken_checker_is_503_never_ok():
    r = await client(Fake(raises=ValueError("code is empty"))).post("/v1/check/remotion", json={"code": ""})
    assert r.status_code == 400
    r = await client(Fake(raises=TypeScriptCheckError("tsc not found"))).post("/v1/check/remotion", json={"code": "c"})
    assert r.status_code == 503 and "ok" not in r.json()


async def test_metrics_report_wait_run_timeouts_and_overlap_with_render():
    from adapters.http.check_metrics import CheckMetrics

    metrics = CheckMetrics()
    app = create_check_app(Fake(CheckOutcome(True)), metrics=metrics)
    c = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")

    await c.post("/v1/check/manim", json={"code": "c"})
    with metrics.command_running():
        await c.post("/v1/check/manim", json={"code": "c"})
    app_broken = create_check_app(Fake(raises=TypeScriptCheckError("tsc timed out after 90s")), metrics=metrics)
    cb = httpx.AsyncClient(transport=httpx.ASGITransport(app=app_broken), base_url="http://r")
    await cb.post("/v1/check/remotion", json={"code": "c"})

    snap = (await c.get("/metrics/checks")).json()
    assert snap["checks_total"] == 3
    assert snap["checks_overlapping_render"] == 1
    assert snap["checks_timed_out"] == 1
    assert snap["renders_running"] == 0
    assert snap["wait_seconds"]["p95"] >= 0 and snap["window"] == 3
