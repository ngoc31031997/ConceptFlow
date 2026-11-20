"""Route tài sản dự án: use case thật chạy trên kho, ảnh, bộ dựng và model giả."""

from __future__ import annotations

import httpx
from fastapi import FastAPI

from adapters.http.project_assets import create_project_asset_router
from application.illustration_drawer import IllustrationDrawer
from application.illustration_library import IllustrationLibrary
from application.project_assets import ProjectAssets
from tests.illustration_fakes import (
    REF,
    FakeFiles,
    FakePrompts,
    FakeReferences,
    FakeRenderer,
    FakeRepo,
    ScriptedChat,
    reply,
)
from tests.kit_fixtures import phatsang_kit
from tests.project_asset_fakes import FakeAssetFiles, FakeAssetRepo

PROJECT = "7c1f2a9e-1111-2222-3333-444455556666"
BASE = f"/internal/v1/projects/{PROJECT}/assets"
SPACE = "export function Space() { return null; }"
SPACE_V2 = "export function Space({tint}) { return null; }"
NEW_ASSET = {
    "concept_id": "C1", "name": "Space", "kind": "backdrop", "tier": "background",
    "portrayal": "Vũ trụ tối", "must_not_imply": ["ban ngày"], "palette_tokens": ["navy"],
}


class Env:
    def __init__(self, replies=None, with_drawer=True):
        self.lib_repo, self.lib_files = FakeRepo(), FakeFiles()
        self.rend, self.refs = FakeRenderer(), FakeReferences()
        self.library = IllustrationLibrary(self.lib_repo, self.lib_files, self.rend, self.refs)
        self.chat = ScriptedChat(replies or [reply("Space", SPACE)])
        self.prompts = FakePrompts()
        drawer = None
        if with_drawer:
            drawer = IllustrationDrawer(self.library, self.chat, self.prompts, self.refs, 1000)
        self.repo, self.files = FakeAssetRepo(), FakeAssetFiles()
        app = FastAPI()
        use_case = ProjectAssets(self.repo, self.files, self.library, drawer)
        app.include_router(create_project_asset_router(use_case))
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")

    async def create(self, **extra) -> dict:
        r = await self.client.post(BASE, json={**NEW_ASSET, **extra})
        assert r.status_code == 201, r.text
        return r.json()

    async def draw(self, aid, **extra):
        body = {"kit": phatsang_kit(), "ref_id": REF, **extra}
        return await self.client.post(f"{BASE}/{aid}/draw", json=body)


async def test_create_issues_codes_from_a_per_project_counter_never_reused():
    env = Env()
    first, second = await env.create(), await env.create()
    assert (first["asset_id"], second["asset_id"]) == ("AST-SPACE-001", "AST-SPACE-002")
    assert first["status"] == "planned" and first["version"] == 0 and first["to_library"] is True
    assert (await env.client.delete(f"{BASE}/AST-SPACE-002")).status_code == 204
    assert (await env.create())["asset_id"] == "AST-SPACE-003", "a deleted code must not come back"
    listed = (await env.client.get(BASE)).json()["assets"]
    assert [a["asset_id"] for a in listed] == ["AST-SPACE-001", "AST-SPACE-003"]


async def test_bad_bodies_and_unknown_assets_map_to_400_and_404():
    env = Env()
    assert (await env.client.post(BASE, content=b"{nope")).json() == {"error": "invalid JSON body"}
    assert (await env.client.post(BASE, json={**NEW_ASSET, "name": "bad name"})).status_code == 400
    assert (await env.client.get(f"{BASE}/AST-NOPE-001")).status_code == 404
    assert (await env.client.put(f"{BASE}/AST-NOPE-001", json=NEW_ASSET)).status_code == 404
    assert (await env.client.post(f"{BASE}/AST-NOPE-001/approve", json={})).status_code == 404


async def test_draw_runs_the_drawer_with_the_projects_kit_and_stores_the_preview():
    env = Env()
    made = await env.create(palette_tokens=["navy"], expressions=["tĩnh"])
    r = await env.draw(made["asset_id"])
    assert r.status_code == 200, r.text
    drawn = r.json()
    assert drawn["status"] == "drawn" and drawn["version"] == 1 and drawn["code"].strip() == SPACE
    assert drawn["preview_path"] == f"/shared/{PROJECT}/storyboard/assets/AST-SPACE-001/v1.png"
    assert env.files.files[drawn["preview_path"]].startswith(b"PNG:Space")
    assert env.rend.last["kit"] == phatsang_kit() and env.rend.last["tier"] == "background"
    user = env.chat.calls[0]["user"]
    assert "Vũ trụ tối" in user and "ban ngày" in user and "navy" in user and "tĩnh" in user
    assert "Tên component bắt buộc: Space" in user
    assert env.lib_repo.rows.keys() == {k for k in env.lib_repo.rows if k.startswith("builtin-")}, \
        "drawing must not add anything to the library"


async def test_preview_serves_the_current_png_and_is_404_before_the_first_drawing():
    env = Env()
    aid = (await env.create())["asset_id"]
    assert (await env.client.get(f"{BASE}/{aid}/preview")).status_code == 404
    await env.draw(aid)
    r = await env.client.get(f"{BASE}/{aid}/preview")
    assert r.status_code == 200 and r.headers["content-type"] == "image/png"
    assert r.headers["cache-control"] == "private, no-cache" and r.content.startswith(b"PNG:Space")
    assert (await env.client.get(f"{BASE}/AST-NOPE-001/preview")).status_code == 404
    evil = await env.client.get(f"/internal/v1/projects/..%2Fx/assets/{aid}/preview")
    assert evil.status_code in (400, 404)


async def test_draw_without_a_valid_kit_is_a_400_and_without_a_drawer_a_503():
    env = Env()
    made = await env.create()
    r = await env.client.post(f"{BASE}/{made['asset_id']}/draw", json={"ref_id": REF})
    assert r.status_code == 400 and "kit" in r.json()["error"]
    off = Env(with_drawer=False)
    made = await off.create()
    assert (await off.draw(made["asset_id"])).status_code == 503


async def test_draw_uses_the_base_illustrations_code_as_an_exemplar():
    env = Env()
    base = await env.library.create(
        env.lib_repo.rows["builtin-Tooth"].with_(
            id="", name="Planet", code="export function Planet() { return null; }",
            folder_id="thien-nhien-thoi-tiet", builtin=False, kind="backdrop"), REF)
    made = await env.create(base_illustration_id=base.id)
    assert (await env.draw(made["asset_id"])).status_code == 200
    assert "export function Planet()" in env.chat.calls[0]["system"]


async def test_a_base_illustration_of_another_kind_is_refused():
    env = Env()
    r = await env.client.post(BASE, json={**NEW_ASSET, "base_illustration_id": "builtin-Tooth"})
    assert r.status_code == 400 and "cùng loại" in r.json()["error"]


async def test_redraw_needs_a_drawing_then_bumps_the_version_and_keeps_the_name():
    env = Env([reply("Space", SPACE), reply("Space", SPACE_V2)])
    made = await env.create()
    aid = made["asset_id"]
    r = await env.client.post(f"{BASE}/{aid}/redraw", json={"kit": phatsang_kit(), "ref_id": REF})
    assert r.status_code == 409
    await env.draw(aid)
    r = await env.client.post(
        f"{BASE}/{aid}/redraw", json={"kit": phatsang_kit(), "ref_id": REF, "note": "thêm sao"})
    assert r.status_code == 200 and r.json()["version"] == 2 and r.json()["code"].strip() == SPACE_V2
    assert "thêm sao" in env.chat.calls[-1]["user"] and SPACE in env.chat.calls[-1]["user"]
    only = f"/shared/{PROJECT}/storyboard/assets/{aid}/v2.png"
    assert list(env.files.files) == [only], "old preview removed"


async def test_a_drawer_that_renames_the_component_is_refused():
    env = Env([reply("Cosmos", "export function Cosmos() { return null; }")])
    made = await env.create()
    r = await env.draw(made["asset_id"])
    assert r.status_code == 400 and "Space" in r.json()["error"]


async def test_approve_needs_a_drawing_then_adds_the_drawing_to_the_project_folder():
    env = Env()
    aid = (await env.create())["asset_id"]
    assert (await env.client.post(f"{BASE}/{aid}/approve", json={})).status_code == 409
    await env.draw(aid)
    r = await env.client.post(f"{BASE}/{aid}/approve", json={"project_name": "Phát sáng", "ref_id": REF})
    assert r.status_code == 200, r.text
    approved = r.json()
    assert approved["status"] == "approved" and approved["library_illustration_id"]
    assert any(f.id == "tu-du-an" and f.name == "Từ dự án" for f in env.lib_repo.folders)
    row = env.lib_repo.rows[approved["library_illustration_id"]]
    assert row.folder_id == "tu-du-an" and row.status == "approved" and row.kind == "backdrop"
    assert row.name == "Space7c1f2a9e" and row.title == "Space-7c1f2a9e"
    assert "Phát sáng" in row.description and "export function Space7c1f2a9e(" in row.code
    assert (await env.library.preview(row.id))[0].startswith(b"PNG:Space")


async def test_approve_without_to_library_creates_nothing_in_the_library():
    env = Env()
    aid = (await env.create(to_library=False))["asset_id"]
    await env.draw(aid)
    r = await env.client.post(f"{BASE}/{aid}/approve", json={})
    assert r.json()["status"] == "approved" and r.json()["library_illustration_id"] == ""
    assert not any(not k.startswith("builtin-") for k in env.lib_repo.rows)


async def test_approve_into_a_chosen_folder_and_the_exemplar_folder_is_refused():
    env = Env()
    aid = (await env.create(library_folder_id="boi-canh"))["asset_id"]
    await env.draw(aid)
    r = await env.client.post(f"{BASE}/{aid}/approve", json={})
    assert env.lib_repo.rows[r.json()["library_illustration_id"]].folder_id == "boi-canh"
    bad = (await env.create(library_folder_id="hinh-mau"))["asset_id"]
    await env.draw(bad)
    assert (await env.client.post(f"{BASE}/{bad}/approve", json={})).status_code == 422


async def test_unapprove_returns_to_drawn_and_keeps_the_library_drawing():
    env = Env()
    aid = (await env.create())["asset_id"]
    assert (await env.client.post(f"{BASE}/{aid}/unapprove")).status_code == 409
    await env.draw(aid)
    lib_id = (await env.client.post(f"{BASE}/{aid}/approve", json={})).json()["library_illustration_id"]
    r = await env.client.post(f"{BASE}/{aid}/unapprove")
    assert r.json()["status"] == "drawn" and r.json()["library_illustration_id"] == lib_id
    assert lib_id in env.lib_repo.rows


async def test_approving_again_updates_the_same_library_drawing():
    env = Env([reply("Space", SPACE), reply("Space", SPACE_V2)])
    aid = (await env.create())["asset_id"]
    await env.draw(aid)
    lib_id = (await env.client.post(f"{BASE}/{aid}/approve", json={})).json()["library_illustration_id"]
    await env.client.post(f"{BASE}/{aid}/redraw", json={"kit": phatsang_kit(), "ref_id": REF})
    again = (await env.client.post(f"{BASE}/{aid}/approve", json={})).json()
    assert again["library_illustration_id"] == lib_id and env.lib_repo.rows[lib_id].version == 2
    assert "tint" in env.lib_repo.rows[lib_id].code


async def test_editing_a_drawn_asset_sets_it_back_and_delete_removes_the_preview_only():
    env = Env()
    aid = (await env.create())["asset_id"]
    await env.draw(aid)
    r = await env.client.put(f"{BASE}/{aid}", json={**NEW_ASSET, "portrayal": "Vũ trụ sáng"})
    assert r.json()["status"] == "planned" and r.json()["version"] == 1 and r.json()["code"]
    assert env.files.files
    assert (await env.client.delete(f"{BASE}/{aid}")).status_code == 204
    assert not env.files.files and (await env.client.get(f"{BASE}/{aid}")).status_code == 404


async def test_a_listed_asset_has_units_and_no_detail_and_is_not_drawn_until_it_has_one():
    env = Env()
    listed = await env.client.post(BASE, json={
        "concept_id": "C1", "name": "Space", "kind": "backdrop", "tier": "background",
        "units": ["VP-001", "VP-002"]})
    assert listed.status_code == 201, listed.text
    made = listed.json()
    assert made["units"] == ["VP-001", "VP-002"] and made["has_detail"] is False and made["portrayal"] == ""
    refused = await env.draw(made["asset_id"])
    assert refused.status_code == 409 and refused.json()["code"] == "asset_no_detail"
    assert env.chat.calls == [] and env.files.files == {}, "nothing may be drawn or saved"
    written = await env.client.put(f"{BASE}/{made['asset_id']}", json={
        **NEW_ASSET, "units": ["VP-001", "VP-002"]})
    assert written.status_code == 200 and written.json()["has_detail"] is True
    assert (await env.draw(made["asset_id"])).status_code == 200


async def test_editing_the_units_of_a_drawn_asset_sets_it_back_to_planned():
    env = Env()
    made = await env.create(units=["VP-001"])
    aid = made["asset_id"]
    await env.draw(aid)
    r = await env.client.put(f"{BASE}/{aid}", json={**NEW_ASSET, "units": ["VP-001", "VP-003"]})
    assert r.status_code == 200 and r.json()["status"] == "planned"
    assert r.json()["units"] == ["VP-001", "VP-003"]
    assert r.json()["version"] == 1 and r.json()["code"].strip() == SPACE
    again = await env.client.put(f"{BASE}/{aid}", json={**NEW_ASSET, "units": ["VP-001", "VP-003"]})
    assert again.json()["status"] == "planned"
    assert (await env.client.get(f"{BASE}/{aid}")).json()["units"] == ["VP-001", "VP-003"]
