"""Route Storyboard: use case thật chạy trên kho giả, thư mục tạm và bộ dựng khung giả."""

from __future__ import annotations

import asyncio
import hashlib

import httpx
import pytest
from fastapi import FastAPI

from adapters.http.storyboard import create_storyboard_router
from adapters.storage.storyboard_files import StoryboardFiles
from application.storyboard import Storyboard
from domain.project_asset import AssetStatus, ProjectAsset
from tests.kit_fixtures import phatsang_kit
from tests.project_asset_fakes import FakeAssetRepo
from tests.storyboard_fakes import PNG, FakeFrameRenderer, FakeStoryboardRepo

PROJECT = "7c1f2a9e-1111-2222-3333-444455556666"
BASE = f"/internal/v1/projects/{PROJECT}/storyboard"
HASH = "a" * 64


def frame_body(**over) -> dict:
    body = {
        "shot_id": "SH-001", "unit_id": "VP-001", "beat_id": "B1", "spec_hash": HASH, "kit": phatsang_kit(),
        "size": {"w": 1280, "h": 720}, "background": {"asset_id": "AST-SPACE-001", "color": "#0F1B3D"},
        "placements": [
            {"asset_id": "AST-HERO-001", "role": "subject", "position": "lower_left"},
            {"asset_id": "AST-GERM-001", "role": "main", "position": "upper_right"}],
        "camera_view": "wide", "on_screen_text": [{"text": "Vi khuẩn tấn công", "min_px": 44}],
        "text_color": "#FFFFFF",
    }
    body.update(over)
    return body


class Env:
    def __init__(self, tmp_path):
        self.repo, self.assets = FakeStoryboardRepo(), FakeAssetRepo()
        self.files = StoryboardFiles(tmp_path)
        self.root = tmp_path
        self.renderer = FakeFrameRenderer()
        app = FastAPI()
        app.include_router(create_storyboard_router(
            Storyboard(self.repo, self.assets, self.files, self.renderer, asyncio.Semaphore(1))))
        self.client = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://r")

    def seed_asset(
        self, asset_id, name, kind="figure", tier="support", status=AssetStatus.APPROVED, code="CODE",
    ):
        self.assets.rows[(PROJECT, asset_id)] = ProjectAsset(
            project_id=PROJECT, asset_id=asset_id, concept_id="C1", name=name, kind=kind, tier=tier,
            code=code, version=1, status=status)

    def seed_all(self):
        self.seed_asset("AST-SPACE-001", "Space", kind="backdrop", tier="background")
        self.seed_asset("AST-HERO-001", "Hero", tier="hero")
        self.seed_asset("AST-GERM-001", "Germ", tier="support")


@pytest.fixture
def env(tmp_path):
    return Env(tmp_path)


async def test_frame_is_drawn_with_assets_codes_and_tiers_from_their_records(env):
    env.seed_all()
    r = await env.client.post(f"{BASE}/frames", json=frame_body())
    assert r.status_code == 200, r.text
    out = r.json()
    assert out["frame_path"] == "storyboard/SH-001.png" and out["cache_hit"] is False
    assert out["spec_hash"] == HASH and out["frame_hash"] == hashlib.sha256(PNG).hexdigest()
    assert (env.root / PROJECT / "storyboard" / "SH-001.png").read_bytes() == PNG
    spec = env.renderer.specs[0]
    assert [(p["name"], p["tier"], p["position"]) for p in spec["placements"]] == [
        ("Hero", "hero", "lower_left"), ("Germ", "support", "upper_right")]
    assert spec["background"] == {"color": "#0F1B3D", "backdrop": {"code": "CODE", "name": "Space"}}
    assert all("scale" not in p for p in spec["placements"]), "rendering alone decides sizes"
    assert not [p for p in (env.root / PROJECT / "storyboard").iterdir() if p.name.endswith(".tmp.png")]
    assert env.repo.shots[(PROJECT, "SH-001")].frame_path == "storyboard/SH-001.png"


async def test_same_spec_hash_with_the_file_present_is_a_cache_hit_and_a_new_hash_redraws(env):
    env.seed_all()
    await env.client.post(f"{BASE}/frames", json=frame_body())
    again = (await env.client.post(f"{BASE}/frames", json=frame_body())).json()
    assert again["cache_hit"] is True and len(env.renderer.specs) == 1
    (env.root / PROJECT / "storyboard" / "SH-001.png").unlink()
    assert (await env.client.post(f"{BASE}/frames", json=frame_body())).json()["cache_hit"] is False
    changed = (await env.client.post(f"{BASE}/frames", json=frame_body(spec_hash="b" * 64))).json()
    assert changed["cache_hit"] is False and changed["spec_hash"] == "b" * 64 and len(env.renderer.specs) == 3


async def test_missing_or_unapproved_assets_are_a_409_listing_them_and_nothing_is_drawn(env):
    env.seed_asset("AST-HERO-001", "Hero", status=AssetStatus.DRAWN)
    r = await env.client.post(f"{BASE}/frames", json=frame_body(background={"color": "#0F1B3D"}))
    assert r.status_code == 409
    assert r.json()["assets"] == [
        {"asset_id": "AST-HERO-001", "reason": "chưa duyệt (drawn)"},
        {"asset_id": "AST-GERM-001", "reason": "không tìm thấy"}]
    assert env.renderer.specs == [] and env.repo.shots == {}


@pytest.mark.parametrize("change", [
    {"camera_view": "drone"},
    {"placements": [{"asset_id": "AST-HERO-001", "role": "subject", "position": "nowhere"}]},
    {"placements": [{"asset_id": "AST-HERO-001", "role": "boss", "position": "center"}]},
    {"placements": []},
    {"placements": [{"asset_id": "AST-HERO-001", "role": "subject", "position": "center"},
                    {"asset_id": "AST-GERM-001", "role": "main", "position": "center"}]},
    {"kit": None}, {"kit": {"nope": 1}},
    {"shot_id": "shot-1"}, {"spec_hash": ""}, {"size": {"w": 640, "h": 360}},
    {"background": {"color": "navy"}}, {"text_color": "white"},
    {"on_screen_text": [{"text": "x", "min_px": 0}]}, {"on_screen_text": [{"text": "", "min_px": 30}]},
])
async def test_values_outside_the_closed_vocabularies_or_shape_are_a_400(env, change):
    env.seed_all()
    r = await env.client.post(f"{BASE}/frames", json=frame_body(**change))
    assert r.status_code == 400, r.text
    assert env.renderer.specs == []


async def test_a_figure_asset_as_background_is_a_400(env):
    env.seed_all()
    body = frame_body(background={"asset_id": "AST-HERO-001", "color": "#000"})
    r = await env.client.post(f"{BASE}/frames", json=body)
    assert r.status_code == 400 and "backdrop" in r.json()["error"]


async def test_renderer_failure_is_a_503_with_the_message_and_is_recorded_without_a_frame(env):
    env.seed_all()
    env.renderer.error = "Chromium không chạy được"
    r = await env.client.post(f"{BASE}/frames", json=frame_body())
    assert r.status_code == 503 and "Chromium không chạy được" in r.json()["error"]
    row = env.repo.shots[(PROJECT, "SH-001")]
    assert row.frame_path is None and "Chromium" in row.error and row.spec_hash == HASH
    assert not (env.root / PROJECT / "storyboard" / "SH-001.png").exists()
    assert (await env.client.get(f"{BASE}/frames/SH-001")).status_code == 404


async def test_get_frame_serves_the_png_and_404s_when_the_shot_was_never_drawn(env):
    env.seed_all()
    assert (await env.client.get(f"{BASE}/frames/SH-009")).status_code == 404
    await env.client.post(f"{BASE}/frames", json=frame_body())
    r = await env.client.get(f"{BASE}/frames/SH-001")
    assert r.status_code == 200 and r.headers["content-type"] == "image/png" and r.content == PNG


def document(*shot_ids) -> dict:
    return {"schema_version": "1.0", "project_id": PROJECT, "document_id": "storyboard-v1",
            "shots": [
                {"shot_id": s, "unit_id": "VP-001", "frame_path": f"storyboard/{s}.png"} for s in shot_ids]}


async def draw(env, shot_id):
    body = frame_body(shot_id=shot_id, spec_hash=shot_id.replace("-", "") * 2)
    r = await env.client.post(f"{BASE}/frames", json=body)
    assert r.status_code == 200, r.text


async def test_lock_stores_the_version_copies_frames_and_rewrites_paths(env):
    env.seed_all()
    await draw(env, "SH-001")
    await draw(env, "SH-002")
    r = await env.client.post(f"{BASE}/versions", json={
        "document": document("SH-001", "SH-002"), "sequence_version": 3, "note": "bản đầu"})
    assert r.status_code == 201, r.text
    stored = r.json()
    assert stored["version"] == 1 and stored["sequence_version"] == 3 and stored["note"] == "bản đầu"
    assert [s["frame_path"] for s in stored["document"]["shots"]] == [
        "storyboard/v1/SH-001.png", "storyboard/v1/SH-002.png"]
    folder = env.root / PROJECT / "storyboard"
    assert (folder / "v1" / "SH-001.png").read_bytes() == PNG and (folder / "v1" / "SH-002.png").exists()
    assert '"storyboard/v1/SH-002.png"' in (folder / "storyboard.json").read_text(encoding="utf-8")
    assert (await env.client.get(f"{BASE}/versions/latest")).json()["version"] == 1


async def test_a_second_lock_is_version_2_and_old_versions_stay_readable_and_untouched(env):
    env.seed_all()
    await draw(env, "SH-001")
    await env.client.post(f"{BASE}/versions", json={"document": document("SH-001"), "sequence_version": 1})
    env.renderer.specs.clear()
    second = {"document": document("SH-001"), "sequence_version": 2}
    again = await env.client.post(f"{BASE}/versions", json=second)
    assert again.json()["version"] == 2
    v1 = (await env.client.get(f"{BASE}/versions/1")).json()
    assert v1["sequence_version"] == 1
    assert v1["document"]["shots"][0]["frame_path"] == "storyboard/v1/SH-001.png"
    assert (await env.client.get(f"{BASE}/versions/latest")).json()["version"] == 2
    assert (await env.client.get(f"{BASE}/versions/7")).status_code == 404


async def test_lock_with_a_shot_without_a_frame_is_a_409_listing_it_and_writes_nothing(env):
    env.seed_all()
    await draw(env, "SH-001")
    r = await env.client.post(f"{BASE}/versions", json={"document": document("SH-001", "SH-002", "SH-003")})
    assert r.status_code == 409 and r.json()["missing_frames"] == ["SH-002", "SH-003"]
    assert env.repo.versions == {} and not (env.root / PROJECT / "storyboard" / "v1").exists()
    assert (await env.client.get(f"{BASE}/versions/latest")).status_code == 404


@pytest.mark.parametrize("doc", [None, [], {}, {"shots": []}, {"shots": [{"shot_id": "x"}]},
                                 {"shots": [{"shot_id": "SH-001"}, {"shot_id": "SH-001"}]}])
async def test_a_malformed_document_is_a_400(env, doc):
    r = await env.client.post(f"{BASE}/versions", json={"document": doc})
    assert r.status_code == 400, r.text


async def test_a_version_number_already_taken_is_a_409_and_the_locked_frames_stay_untouched(env):
    env.seed_all()
    await draw(env, "SH-001")
    await env.client.post(f"{BASE}/versions", json={"document": document("SH-001")})
    v1_frame = env.root / PROJECT / "storyboard" / "v1" / "SH-001.png"
    v1_frame.write_bytes(b"locked-bytes")

    async def stale_next(project_id):
        return 1

    env.repo.next_version = stale_next
    r = await env.client.post(f"{BASE}/versions", json={"document": document("SH-001")})
    assert r.status_code == 409
    assert v1_frame.read_bytes() == b"locked-bytes", "a locked version must never be overwritten"
    assert len(env.repo.versions) == 1


async def test_delete_removes_rows_and_the_folder_and_is_idempotent(env):
    env.seed_all()
    await draw(env, "SH-001")
    await env.client.post(f"{BASE}/versions", json={"document": document("SH-001")})
    assert (await env.client.delete(BASE)).status_code == 204
    assert not (env.root / PROJECT / "storyboard").exists()
    assert env.repo.shots == {} and env.repo.versions == {}
    assert (await env.client.delete(BASE)).status_code == 204


async def test_bad_json_and_bad_project_ids_are_400(env):
    broken = await env.client.post(f"{BASE}/frames", content=b"{nope")
    assert broken.json() == {"error": "invalid JSON body"}
    assert (await env.client.post(f"{BASE}/versions", content=b"[1]")).status_code == 400
    assert (await env.client.get("/internal/v1/projects/..bad/storyboard/versions/latest")).status_code == 400
