import httpx

from adapters.http.check_server import create_check_app
from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.check_script import CheckDiagnostic, CheckOutcome


class Fake:
    def __init__(self, outcome=None, raises=None):
        self.outcome, self.raises, self.calls = outcome, raises, []

    def check(self, engine, code, scene_class_name, layout=None):
        self.calls.append((engine, code, scene_class_name))
        self.layouts = getattr(self, "layouts", []) + [layout]
        if self.raises:
            raise self.raises
        return self.outcome


def client(fake):
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=create_check_app(fake)), base_url="http://r")


async def test_returns_diagnostics():
    fake = Fake(CheckOutcome(False, [CheckDiagnostic("boom", 4)]))
    r = await client(fake).post("/v1/check/remotion", json={"kit_key": "phatsang", "code": "c", "scene_class_name": "creator"})
    assert r.status_code == 200
    assert r.json() == {
        "ok": False,
        "diagnostics": [{"message": "boom", "line": 4, "kind": "compile", "rule": ""}],
        "warnings": [],
    }
    assert fake.calls == [("remotion", "c", "creator")]
    assert fake.layouts[0].subtitle_band is None and fake.layouts[0].video_font == ""
    assert fake.layouts[0].kit_key == "phatsang"


async def test_layout_diagnostics_warnings_and_context_cross_the_wire():
    from domain.layout_rules import SubtitleBand

    overflow = CheckDiagnostic("Shot 1.2, frame 85%: nhãn tràn", 62, "layout", "text_overflow")
    fake = Fake(CheckOutcome(False, [overflow], ["Bố cục: Shot 1.3: vật lớn nhất nhỏ"]))
    r = await client(fake).post("/v1/check/remotion", json={
        "kit_key": "phatsang", "code": "c", "scene_class_name": "creator",
        "subtitle_band": {"edge": "bottom", "px": 240}, "video_font": "Montserrat"})
    assert r.json()["diagnostics"] == [
        {"message": "Shot 1.2, frame 85%: nhãn tràn", "line": 62, "kind": "layout", "rule": "text_overflow"}
    ]
    assert r.json()["warnings"] == ["Bố cục: Shot 1.3: vật lớn nhất nhỏ"]
    assert fake.layouts[0].subtitle_band == SubtitleBand("bottom", 240)
    assert fake.layouts[0].video_font == "Montserrat"


async def test_a_nonsense_subtitle_band_is_rejected():
    fake = Fake(CheckOutcome(True))
    r = await client(fake).post(
        "/v1/check/remotion", json={"kit_key": "phatsang", "code": "c", "subtitle_band": {"edge": "bottom", "px": 0}}
    )
    assert r.status_code == 400 and fake.calls == []
    r = await client(fake).post(
        "/v1/check/remotion", json={"kit_key": "phatsang", "code": "c", "subtitle_band": {"edge": "left", "px": 9}}
    )
    assert r.status_code == 422


async def test_there_is_no_manim_route():
    fake = Fake(CheckOutcome(True))
    r = await client(fake).post("/v1/check/manim", json={"kit_key": "phatsang", "code": "c", "scene_class_name": "XScene"})
    assert r.status_code == 404 and fake.calls == []


async def test_bad_input_is_400_and_a_broken_checker_is_503_never_ok():
    r = await client(Fake(raises=ValueError("code is empty"))).post("/v1/check/remotion", json={"kit_key": "phatsang", "code": ""})
    assert r.status_code == 400
    r = await client(Fake(raises=TypeScriptCheckError("tsc not found"))).post(
        "/v1/check/remotion", json={"kit_key": "phatsang", "code": "c"}
    )
    assert r.status_code == 503 and "ok" not in r.json()


async def test_metrics_report_wait_run_timeouts_and_overlap_with_render():
    from adapters.http.check_metrics import CheckMetrics

    metrics = CheckMetrics()
    app = create_check_app(Fake(CheckOutcome(True)), metrics=metrics)
    c = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")

    await c.post("/v1/check/remotion", json={"kit_key": "phatsang", "code": "c"})
    with metrics.command_running():
        await c.post("/v1/check/remotion", json={"kit_key": "phatsang", "code": "c"})
    app_broken = create_check_app(
        Fake(raises=TypeScriptCheckError("tsc timed out after 90s")), metrics=metrics
    )
    cb = httpx.AsyncClient(transport=httpx.ASGITransport(app=app_broken), base_url="http://r")
    await cb.post("/v1/check/remotion", json={"kit_key": "phatsang", "code": "c"})

    snap = (await c.get("/metrics/checks")).json()
    assert snap["checks_total"] == 3
    assert snap["checks_overlapping_render"] == 1
    assert snap["checks_timed_out"] == 1
    assert snap["renders_running"] == 0
    assert snap["wait_seconds"]["p95"] >= 0 and snap["window"] == 3


async def test_a_check_without_a_pinned_kit_is_refused_before_it_runs():
    fake = Fake(CheckOutcome(True))
    r = await client(fake).post("/v1/check/remotion", json={"code": "c", "scene_class_name": "creator"})
    assert r.status_code == 422 and fake.calls == []


# --- kits, stills and the kit-aware preview ----------------------------------------

import pytest  # noqa: E402

from application.stills import StillsUseCase  # noqa: E402


class StillWriter:
    def __init__(self):
        self.calls = []

    def still(self, kit_key, composition, output, frame=10, props=None):
        self.calls.append((kit_key, composition))
        with open(output, "wb") as f:
            f.write(b"png")


def app_with_stills(tmp_path, writer=None):
    writer = writer or StillWriter()
    kits = {"phatsang": 2}
    use_case = StillsUseCase(writer, kits, tmp_path)
    app = create_check_app(Fake(CheckOutcome(True)), stills=use_case, kits=kits)
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r"), writer


async def test_the_kits_endpoint_lists_every_kit_with_its_version(tmp_path):
    c, _ = app_with_stills(tmp_path)
    r = await c.get("/v1/kits")
    assert r.status_code == 200 and r.json() == {"kits": [{"key": "phatsang", "version": 2}]}


async def test_a_still_is_rendered_once_then_reported_as_a_cache_hit(tmp_path):
    c, writer = app_with_stills(tmp_path)
    body = {"composition": "sample-fig-wide", "kit_key": "phatsang"}
    first = (await c.post("/v1/stills", json=body)).json()
    second = (await c.post("/v1/stills", json=body)).json()
    assert first["cache_hit"] is False and second["cache_hit"] is True and first["file"] == second["file"]
    assert len(writer.calls) == 1 and (tmp_path / first["file"]).exists()


@pytest.mark.parametrize("kit", ["", "nope"])
async def test_a_still_needs_a_known_kit(tmp_path, kit):
    c, writer = app_with_stills(tmp_path)
    r = await c.post("/v1/stills", json={"composition": "sample-fig-wide", "kit_key": kit})
    assert r.status_code == 400 and "kit_key" in r.json()["error"] and writer.calls == []


async def test_a_renderer_that_cannot_run_is_a_503_never_a_file(tmp_path):
    from adapters.rendering.illustration_previewer import IllustrationPreviewError

    class Down(StillWriter):
        def still(self, *a, **k):
            raise IllustrationPreviewError("browser down")

    c, _ = app_with_stills(tmp_path, Down())
    r = await c.post("/v1/stills", json={"composition": "sample-fig-wide", "kit_key": "phatsang"})
    assert r.status_code == 503 and "browser down" in r.json()["error"]
    assert list(tmp_path.iterdir()) == []


async def test_pruning_reports_what_it_removed_and_refuses_a_negative_age(tmp_path):
    c, _ = app_with_stills(tmp_path)
    ok = await c.post("/v1/stills/prune", json={"keep": [], "min_age_hours": 24})
    assert ok.status_code == 200 and ok.json() == {"removed": []}
    bad = await c.post("/v1/stills/prune", json={"keep": [], "min_age_hours": -1})
    assert bad.status_code == 400


async def test_stills_are_off_when_not_configured():
    c = client(Fake(CheckOutcome(True)))
    assert (await c.post("/v1/stills", json={"composition": "x", "kit_key": "phatsang"})).status_code == 404
    assert (await c.get("/v1/kits")).json() == {"kits": []}


async def test_a_preview_without_a_pinned_kit_is_refused_before_it_runs():
    c = client(Fake(CheckOutcome(True)))
    r = await c.post("/v1/illustrations/preview", json={"name": "Bus"})
    assert r.status_code == 422
