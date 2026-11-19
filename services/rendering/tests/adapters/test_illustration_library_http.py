"""Route của thư viện hình: mã trạng thái và thân JSON, với use case thật chạy trên bản giả trong bộ nhớ."""

from __future__ import annotations

import base64
import io
import json
import zipfile
from datetime import datetime

import httpx
import pytest
from fastapi import FastAPI

from adapters.http.check_server import create_check_app
from adapters.http.illustrations import create_illustration_router
from application.illustration_backup import BACKUP_FORMAT, IllustrationBackup
from application.illustration_drawer import IllustrationDrawer
from application.illustration_library import IllustrationLibrary
from domain.illustration import (
    MAX_EXEMPLARS,
    Illustration,
    IllustrationStatus,
    LLMError,
    builtin_illustrations,
)
from tests.illustration_fakes import (
    BUS_CODE,
    REF,
    FakeFiles,
    FakePrompts,
    FakeReferences,
    FakeRenderer,
    FakeRepo,
    ScriptedChat,
    reply,
)

MOTO = "export function Motorbike({color = '#E8453C', ...fig}) { return null; }"


class Env:
    def __init__(self, replies=None, with_drawer=True):
        self.repo, self.files, self.rend, self.refs = (
            FakeRepo(),
            FakeFiles(),
            FakeRenderer(),
            FakeReferences(),
        )
        self.library = IllustrationLibrary(self.repo, self.files, self.rend, self.refs)
        self.chat = ScriptedChat(replies or [reply("Motorbike", MOTO)])
        drawer = (
            IllustrationDrawer(self.library, self.chat, FakePrompts(), self.refs, 1000)
            if with_drawer
            else None
        )
        router = create_illustration_router(
            self.library,
            drawer,
            IllustrationBackup(self.library, self.files.read),
            self.refs,
            clock=lambda: datetime(2026, 9, 27, 8, 30),
        )
        app = FastAPI()
        app.include_router(router)
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")


async def create_bus(env: Env, **extra) -> dict:
    r = await env.client.post(
        "/v1/admin/illustrations",
        json={"name": "Bus", "folder_id": "phuong-tien", "code": BUS_CODE, "ref_id": REF, **extra},
    )
    assert r.status_code == 201, r.text
    return r.json()


async def test_folders_create_list_delete_and_their_errors():
    env = Env()
    r = await env.client.get("/v1/illustration-folders")
    assert r.status_code == 200 and r.json()["folders"][0]["id"] == "hinh-mau"
    r = await env.client.post("/v1/admin/illustration-folders", json={"id": "do-choi", "name": "Đồ chơi"})
    assert r.status_code == 201 and r.json()["is_system"] is False and r.json()["position"] == 0
    assert (
        await env.client.post("/v1/admin/illustration-folders", json={"id": "do-choi", "name": "x"})
    ).status_code == 409, "an id already taken"
    assert (
        await env.client.post("/v1/admin/illustration-folders", json={"id": "đồ chơi", "name": "x"})
    ).status_code == 400
    assert (await env.client.delete("/v1/admin/illustration-folders/con-nguoi")).status_code == 403
    assert (await env.client.delete("/v1/admin/illustration-folders/khong-co")).status_code == 404
    await create_bus(env, folder_id="do-choi")
    assert (await env.client.delete("/v1/admin/illustration-folders/do-choi")).status_code == 409
    assert (await env.client.delete("/v1/admin/illustration-folders/do-choi")).json()["error"] == (
        "illustration folder still holds drawings"
    )


async def test_a_body_that_is_not_json_is_a_400_with_the_same_message_on_every_route():
    env = Env()
    for method, path in [
        ("POST", "/v1/admin/illustration-folders"),
        ("POST", "/v1/illustration-tries"),
        ("POST", "/v1/admin/illustrations"),
        ("PUT", "/v1/admin/illustrations/x"),
        ("POST", "/v1/admin/illustrations/x/status"),
        ("POST", "/v1/admin/illustrations/draw"),
        ("POST", "/v1/admin/illustrations/x/redraw"),
    ]:
        r = await env.client.request(method, path, content=b"{not json")
        assert (r.status_code, r.json()) == (400, {"error": "invalid JSON body"}), (method, path)


async def test_create_get_list_and_the_stored_images():
    env = Env()
    made = await create_bus(env, tags=["Xe"], detail_level="rich")
    assert (
        made["status"] == "draft" and made["version"] == 1 and made["has_preview"] and made["ref_id"] == REF
    )
    assert made["detail_level"] == "rich" and made["tags"] == ["xe"] and made["kind"] == "figure"
    got = (await env.client.get(f"/v1/illustrations/{made['id']}")).json()
    assert got["code"] == BUS_CODE and got["created_at"]
    listed = (
        await env.client.get("/v1/illustrations", params={"folder": "phuong-tien", "status": "draft"})
    ).json()
    assert [i["name"] for i in listed["illustrations"]] == ["Bus"]
    png = await env.client.get(f"/v1/illustrations/{made['id']}/preview.png")
    gif = await env.client.get(f"/v1/illustrations/{made['id']}/preview.gif")
    assert png.status_code == 200 and png.headers["content-type"] == "image/png"
    assert png.headers["cache-control"] == "private, max-age=86400"
    assert png.content == f"PNG:Bus{BUS_CODE}".encode() and gif.headers["content-type"] == "image/gif"
    assert (await env.client.get("/v1/illustrations/nope")).status_code == 404


async def test_a_builtin_preview_needs_a_reference_style_in_the_query():
    env = Env()
    r = await env.client.get("/v1/illustrations/builtin-Tooth/preview.png")
    assert r.status_code == 400 and "ref_id" in r.json()["error"]
    r = await env.client.get("/v1/illustrations/builtin-Tooth/preview.png", params={"ref_id": REF})
    assert r.status_code == 200 and r.content == b"PNG:Tooth"
    assert (
        await env.client.get("/v1/illustrations/builtin-Germ/preview.png", params={"ref_id": "R99"})
    ).status_code == 404


async def test_create_with_broken_code_is_a_422_with_diagnostics_and_saves_nothing():
    env = Env()
    r = await env.client.post(
        "/v1/admin/illustrations",
        json={"name": "Bus", "folder_id": "phuong-tien", "code": "BROKEN", "ref_id": REF},
    )
    body = r.json()
    assert r.status_code == 422 and body["diagnostics"] == [
        {"message": "TS2304: Cannot find name", "line": 3}
    ]
    assert "dòng 3" in body["error"]


async def test_create_error_codes():
    env = Env()
    base = {"name": "Bus", "folder_id": "phuong-tien", "code": BUS_CODE, "ref_id": REF}
    await create_bus(env)
    assert (await env.client.post("/v1/admin/illustrations", json=base)).status_code == 409
    assert (
        await env.client.post("/v1/admin/illustrations", json={**base, "name": "Van", "folder_id": "x"})
    ).status_code == 404
    assert (
        await env.client.post("/v1/admin/illustrations", json={**base, "name": "Van", "ref_id": ""})
    ).status_code == 400
    assert (
        await env.client.post("/v1/admin/illustrations", json={**base, "name": "Van", "ref_id": "R99"})
    ).status_code == 404
    assert (
        await env.client.post("/v1/admin/illustrations", json={**base, "name": "Van", "code": " "})
    ).status_code == 400
    assert (
        await env.client.post(
            "/v1/admin/illustrations", json={**base, "name": "Van", "folder_id": "hinh-mau"}
        )
    ).status_code == 422
    env.rend.unreachable = True
    assert (await env.client.post("/v1/admin/illustrations", json={**base, "name": "Van"})).status_code == 503


async def test_update_status_rerender_and_delete():
    env = Env()
    made = await create_bus(env)
    r = await env.client.post(f"/v1/admin/illustrations/{made['id']}/status", json={"status": "approved"})
    assert r.status_code == 200 and r.json()["status"] == "approved"
    assert (
        await env.client.post(f"/v1/admin/illustrations/{made['id']}/status", json={"status": "x"})
    ).status_code == 400
    r = await env.client.put(f"/v1/admin/illustrations/{made['id']}", json={"code": BUS_CODE + "\n// v2"})
    assert r.status_code == 200 and r.json()["version"] == 2 and r.json()["status"] == "draft"
    r = await env.client.post(f"/v1/admin/illustrations/{made['id']}/rerender")
    assert r.status_code == 200 and base64.b64decode(r.json()["png"]).startswith(b"PNG:Bus")
    assert base64.b64decode(r.json()["gif"]) == b"GIF"
    assert (await env.client.delete("/v1/admin/illustrations/builtin-Tooth")).status_code == 403
    assert (
        await env.client.put("/v1/admin/illustrations/builtin-Tooth", json={"code": BUS_CODE})
    ).status_code == 403
    assert (await env.client.delete(f"/v1/admin/illustrations/{made['id']}")).status_code == 204
    assert (await env.client.get(f"/v1/illustrations/{made['id']}")).status_code == 404


async def test_try_renders_without_saving_and_returns_base64_with_warnings():
    env = Env()
    r = await env.client.post(
        "/v1/illustration-tries",
        json={
            "name": "Bus",
            "code": BUS_CODE,
            "kind": "backdrop",
            "detail_level": "silhouette",
            "ref_id": REF,
        },
    )
    body = r.json()
    assert (
        r.status_code == 200
        and base64.b64decode(body["png"]).startswith(b"PNG:Bus")
        and body["warnings"] == []
    )
    assert env.rend.last["kind"] == "backdrop" and env.rend.last["tier"] == "background"
    assert len(env.repo.rows) == len(builtin_illustrations())
    assert (
        await env.client.post("/v1/illustration-tries", json={"name": "bad name", "code": "x", "ref_id": REF})
    ).status_code == 400
    r = await env.client.post("/v1/illustration-tries", json={"name": "Bus", "code": "BROKEN", "ref_id": REF})
    assert r.status_code == 422 and r.json()["diagnostics"][0]["line"] == 3


async def test_draw_returns_201_and_redraw_200():
    env = Env()
    r = await env.client.post(
        "/v1/admin/illustrations/draw",
        json={"description": "xe máy", "folder_id": "phuong-tien", "ref_id": REF, "detail_level": "rich"},
    )
    assert r.status_code == 201 and r.json()["name"] == "Motorbike" and r.json()["detail_level"] == "rich"
    env.chat.replies = [reply("Motorbike", MOTO + "\n// to hơn")]
    r2 = await env.client.post(f"/v1/admin/illustrations/{r.json()['id']}/redraw", json={"note": "to hơn"})
    assert r2.status_code == 200 and r2.json()["version"] == 2


async def test_draw_errors():
    env = Env()
    ok = {"description": "xe", "folder_id": "phuong-tien", "ref_id": REF}
    assert (
        await env.client.post("/v1/admin/illustrations/draw", json={**ok, "description": ""})
    ).status_code == 400
    assert (
        await env.client.post("/v1/admin/illustrations/draw", json={**ok, "folder_id": "hinh-mau"})
    ).status_code == 422
    assert (
        await env.client.post("/v1/admin/illustrations/draw", json={**ok, "folder_id": "x"})
    ).status_code == 404
    assert (await env.client.post("/v1/admin/illustrations/builtin-Tooth/redraw", json={})).status_code == 403
    env.chat.replies = [reply("Motorbike", "BROKEN")]
    r = await env.client.post("/v1/admin/illustrations/draw", json=ok)
    assert r.status_code == 422 and r.json()["diagnostics"][0]["line"] == 3
    assert r.json()["error"].startswith("AI vẽ 3 lần vẫn chưa qua kiểm tra")


async def test_draw_without_a_drawer_is_a_503():
    env = Env(with_drawer=False)
    r = await env.client.post(
        "/v1/admin/illustrations/draw", json={"description": "xe", "folder_id": "phuong-tien", "ref_id": REF}
    )
    assert r.status_code == 503 and "AI vẽ chưa được bật" in r.json()["error"]


@pytest.mark.parametrize(
    "kind, status, text",
    [
        ("auth", 502, "API key bị từ chối"),
        ("balance", 502, "hết số dư"),
        ("rate_limit", 429, "gọi quá nhanh"),
        ("timeout", 504, "không trả lời trong thời gian cho phép"),
        ("budget", 502, "hết hạn mức token"),
        ("truncated", 502, "hết hạn mức token"),
        ("empty", 502, "AI trả về rỗng"),
        ("malformed", 502, "Kết quả AI không dùng được: boom (mã cuộc gọi c1)"),
        ("server", 502, "Không gọi được llm-service (boom (mã cuộc gọi c1))"),
        ("not_configured", 503, "Chưa cấu hình HIVE_API_KEY"),
    ],
)
async def test_a_failed_model_call_becomes_the_creator_facing_message(kind, status, text):
    env = Env()

    async def failing(**_):
        raise LLMError(kind, "llm-service", "boom", "c1")

    env.chat.chat = failing
    r = await env.client.post(
        "/v1/admin/illustrations/draw", json={"description": "xe", "folder_id": "phuong-tien", "ref_id": REF}
    )
    assert r.status_code == status and text in r.json()["error"]


async def test_the_provider_failing_on_their_side_is_not_blamed_on_llm_service():
    env = Env()

    async def failing(**_):
        raise LLMError("server", "hive", "502 bad gateway")

    env.chat.chat = failing
    r = await env.client.post(
        "/v1/admin/illustrations/draw", json={"description": "xe", "folder_id": "phuong-tien", "ref_id": REF}
    )
    assert r.status_code == 502 and r.json()["error"] == "Nhà cung cấp AI đang lỗi phía họ — thử lại sau."


async def test_exemplar_routes():
    env = Env()
    made = await create_bus(env)
    await env.client.post(f"/v1/admin/illustrations/{made['id']}/status", json={"status": "approved"})
    r = await env.client.post(f"/v1/admin/illustrations/{made['id']}/exemplar")
    assert r.status_code == 201 and r.json()["source_id"] == made["id"] and r.json()["exemplar"] is True
    copy_id = r.json()["id"]
    assert (await env.client.post(f"/v1/admin/illustrations/{made['id']}/exemplar")).status_code == 409
    assert (await env.client.post("/v1/admin/illustrations/builtin-Tooth/exemplar")).status_code == 422
    assert (await env.client.delete(f"/v1/admin/illustrations/{made['id']}/exemplar")).status_code == 409
    assert (await env.client.delete(f"/v1/admin/illustrations/{copy_id}/exemplar")).status_code == 204

    env.repo.rows["cat"] = Illustration(
        id="cat",
        name="Cat",
        folder_id="hinh-mau",
        exemplar=True,
        home_folder_id="dong-vat",
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
    )
    r = await env.client.delete("/v1/admin/illustrations/cat/exemplar")
    assert r.status_code == 200 and r.json()["folder_id"] == "dong-vat"


async def test_make_exemplar_when_the_folder_is_full_is_a_409():
    env = Env()
    made = await create_bus(env)
    await env.client.post(f"/v1/admin/illustrations/{made['id']}/status", json={"status": "approved"})
    for n in range(MAX_EXEMPLARS):
        env.repo.rows[f"x{n}"] = Illustration(id=f"x{n}", name=f"Ex{n}", folder_id="hinh-mau", exemplar=True)
    r = await env.client.post(f"/v1/admin/illustrations/{made['id']}/exemplar")
    assert r.status_code == 409 and "đã đủ" in r.json()["error"]


async def test_illustration_style_lists_the_exemplars_and_the_chosen_style():
    env = Env()
    env.repo.rows["e1"] = Illustration(
        id="e1", name="Cat", folder_id="hinh-mau", exemplar=True, created_at="2026-09-01T00:00:00Z"
    )
    env.repo.rows["e2"] = Illustration(
        id="e2", name="Dog", folder_id="hinh-mau", exemplar=True, created_at="2026-09-02T00:00:00Z"
    )
    r = await env.client.get("/v1/illustration-style")
    assert r.status_code == 200
    assert r.json() == {"exemplar_ids": ["e1", "e2"], "max_exemplars": MAX_EXEMPLARS, "style": None}
    r = await env.client.get("/v1/illustration-style", params={"ref_id": REF})
    assert r.json()["style"]["name"] == "Khoa học phát sáng"
    assert set(r.json()["style"]) == {"id", "name", "summary", "drawing", "layout"}
    assert (await env.client.get("/v1/illustration-style", params={"ref_id": "R99"})).status_code == 404


async def test_export_downloads_the_zip():
    env = Env()
    await create_bus(env)
    r = await env.client.get("/v1/admin/illustrations/export")
    assert r.status_code == 200 and r.headers["content-type"] == "application/zip"
    assert (
        r.headers["content-disposition"]
        == 'attachment; filename="conceptflow-thu-vien-hinh-20260927-0830.zip"'
    )
    assert r.headers["cache-control"] == "no-store"
    with zipfile.ZipFile(io.BytesIO(r.content)) as zf:
        assert "hinh/phuong-tien/Bus.tsx" in zf.namelist()


async def test_import_passes_the_body_and_conflict_mode():
    src = Env()
    await create_bus(src)
    blob = (await src.client.get("/v1/admin/illustrations/export")).content

    env = Env()
    r = await env.client.post("/v1/admin/illustrations/import", content=blob)
    assert r.status_code == 200 and r.json()["items"][0]["result"] == "created"
    assert r.json()["not_processed"] == 0 and "aborted" not in r.json()
    r = await env.client.post(
        "/v1/admin/illustrations/import", params={"on_conflict": "replace"}, content=blob
    )
    assert r.json()["items"][0]["result"] == "replaced"
    r = await env.client.post("/v1/admin/illustrations/import", params={"on_conflict": "skip"}, content=blob)
    assert r.json()["items"][0]["result"] == "skipped"


@pytest.mark.parametrize(
    "params, body, status, error",
    [
        ({"on_conflict": "merge"}, b"PK", 400, "on_conflict phải là skip hoặc replace"),
        ({}, b"", 400, "chưa gửi file sao lưu"),
        ({}, b"PK", 400, "illustration library backup is not valid: không phải file ZIP"),
    ],
)
async def test_import_refusals(params, body, status, error):
    env = Env()
    r = await env.client.post("/v1/admin/illustrations/import", params=params, content=body)
    assert r.status_code == status and r.json()["error"].startswith(error)


async def test_import_takes_a_reference_style_for_a_file_that_names_none():
    manifest = json.dumps(
        {
            "format": BACKUP_FORMAT,
            "format_version": 1,
            "folders": [],
            "illustrations": [{"name": "Kite", "folder_id": "do-vat", "version": 1, "code_file": "d.tsx"}],
        }
    )
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w") as zf:
        zf.writestr("manifest.json", manifest)
        zf.writestr("d.tsx", "export function Kite() {}")
    env = Env()
    r = await env.client.post(
        "/v1/admin/illustrations/import", params={"ref_id": REF}, content=buf.getvalue()
    )
    assert r.json()["items"][0]["result"] == "created"


async def test_an_unexpected_failure_is_logged_and_answered_with_a_500_not_swallowed():
    env = Env()

    async def boom():
        raise RuntimeError("db down")

    env.library.folders = boom
    r = await env.client.get("/v1/illustration-folders")
    assert r.status_code == 500 and "db down" in r.json()["error"]


async def test_the_routes_are_mounted_on_the_check_app_next_to_the_preview_route():
    env = Env()
    router = create_illustration_router(
        env.library, None, IllustrationBackup(env.library, env.files.read), env.refs
    )
    app = create_check_app(object(), library_router=router)
    client = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")
    assert (await client.get("/v1/illustration-folders")).status_code == 200
    assert (await client.get("/health")).json() == {"status": "ok"}
    assert (
        await client.post("/v1/illustrations/preview", json={"kit": {}, "name": "Bus"})
    ).status_code == 404
