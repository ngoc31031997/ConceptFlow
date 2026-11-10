"""The prompt library's rules and routes, over an in-memory store (no database, no AI)."""

import uuid
from datetime import UTC, datetime

import httpx
import pytest

from app.library import (
    Prompt,
    PromptConflict,
    PromptInvalid,
    PromptLibrary,
    PromptNotFound,
    PromptReadOnly,
    SystemPrompt,
)
from app.main import create_app
from app.registry import Providers
from tests.conftest import MemoryCallLog
from tests.test_api import OkChecker, config


class MemoryStore:
    def __init__(self) -> None:
        self.rows: dict[str, Prompt] = {}
        self.owners: dict[str, str] = {}
        self.models: dict[str, str] = {}

    def _now(self) -> datetime:
        return datetime.now(UTC)

    async def get_active(self, role):
        rows = [p for p in self.rows.values() if p.role == role]
        rows.sort(key=lambda p: (not p.is_active, not p.is_system))
        return rows[0] if rows else None

    async def get_system(self, role):
        return next((p for p in self.rows.values() if p.role == role and p.is_system), None)

    async def get(self, prompt_id):
        return self.rows.get(prompt_id)

    async def list(self, role):
        rows = [p for p in self.rows.values() if not role or p.role == role]
        return sorted(rows, key=lambda p: (p.role, not p.is_system, p.created_at, p.id))

    async def role_exists(self, role):
        return any(p.role == role and p.is_system for p in self.rows.values())

    async def create(self, role, name, template_text):
        p = Prompt(str(uuid.uuid4()), role, name, template_text, False, False, self._now(), self._now())
        self.rows[p.id] = p
        return p

    async def update(self, prompt_id, name, template_text):
        p = self.rows.get(prompt_id)
        if p is None or p.is_system:
            return None
        p.name, p.template_text = name, template_text
        return p

    async def delete(self, prompt_id):
        p = self.rows.get(prompt_id)
        if p is None or p.is_system:
            return False
        del self.rows[prompt_id]
        return True

    async def activate(self, prompt_id):
        p = self.rows.get(prompt_id)
        if p is None:
            return None
        for other in self.rows.values():
            if other.role == p.role:
                other.is_active = False
        p.is_active = True
        return p

    async def replace_system(self, owner, prompts):
        for sp in prompts:
            holder = self.owners.get(sp.role)
            if holder not in (None, "", owner):
                raise PromptConflict(sp.role)
            existing = self.rows.get("system-" + sp.role)
            if existing:
                existing.name, existing.template_text = sp.name, sp.template_text
            else:
                has_active = any(p.role == sp.role and p.is_active for p in self.rows.values())
                self.rows["system-" + sp.role] = Prompt(
                    "system-" + sp.role, sp.role, sp.name, sp.template_text, True, not has_active,
                    self._now(), self._now())
            self.owners[sp.role] = owner
        keep = {sp.role for sp in prompts}
        gone = [r for r, o in self.owners.items() if o == owner and r not in keep]
        for role in gone:
            self.rows.pop("system-" + role, None)
            del self.owners[role]
        return len(prompts), len(gone)

    async def saved_step_models(self):
        return dict(self.models)

    async def save_step_models(self, models):
        self.models.update(models)


def sys(role, text="T"):
    return SystemPrompt(role, "Mặc định " + role, text)


@pytest.fixture
def store():
    return MemoryStore()


@pytest.fixture
def lib(store):
    return PromptLibrary(store)


async def test_the_first_push_activates_each_system_prompt_and_a_second_keeps_the_choice(lib):
    await lib.replace_system("authoring", [sys("idea")])
    mine = await lib.create("idea", "Của tôi", "X")
    await lib.activate(mine.id)
    await lib.replace_system("authoring", [sys("idea", "T2")])
    assert (await lib.active("idea")).id == mine.id
    assert (await lib.list("idea"))[0].template_text == "T2"


async def test_a_push_removes_only_the_owners_unlisted_system_prompts(lib, store):
    await lib.replace_system("authoring", [sys("idea"), sys("brief")])
    await lib.replace_system("visual", [sys("shot")])
    creator = await lib.create("brief", "Của Creator", "X")
    out = await lib.replace_system("authoring", [sys("idea")])
    assert out == {"owner": "authoring", "written": 1, "removed": 1}
    assert "system-brief" not in store.rows and "system-shot" in store.rows and creator.id in store.rows


async def test_a_service_cannot_take_a_role_another_one_owns(lib):
    await lib.replace_system("authoring", [sys("idea")])
    with pytest.raises(PromptConflict):
        await lib.replace_system("visual", [sys("idea")])


async def test_a_push_needs_an_owner_and_whole_prompts(lib):
    with pytest.raises(PromptInvalid):
        await lib.replace_system(" ", [sys("idea")])
    with pytest.raises(PromptInvalid):
        await lib.replace_system("a", [SystemPrompt("idea", "n", " ")])
    with pytest.raises(PromptInvalid):
        await lib.replace_system("a", [sys("idea"), sys("idea")])


async def test_a_role_nobody_ships_is_unknown(lib):
    with pytest.raises(PromptInvalid):
        await lib.active("nope")
    with pytest.raises(PromptInvalid):
        await lib.create("nope", "n", "t")


async def test_system_prompts_are_read_only_but_can_be_copied(lib):
    await lib.replace_system("authoring", [sys("idea")])
    with pytest.raises(PromptReadOnly):
        await lib.update("system-idea", "n", "t")
    with pytest.raises(PromptReadOnly):
        await lib.delete("system-idea")
    copy = await lib.copy("system-idea")
    assert copy.name.startswith("Bản sao của") and not copy.is_system and not copy.is_active


async def test_deleting_the_active_prompt_hands_the_role_back_to_the_system_one(lib):
    await lib.replace_system("authoring", [sys("idea")])
    mine = await lib.create("idea", "Của tôi", "X")
    await lib.activate(mine.id)
    await lib.delete(mine.id)
    assert (await lib.active("idea")).id == "system-idea"
    with pytest.raises(PromptNotFound):
        await lib.delete(mine.id)


async def test_a_blank_name_or_text_is_refused(lib):
    await lib.replace_system("authoring", [sys("idea")])
    with pytest.raises(PromptInvalid):
        await lib.create("idea", " ", "t")
    with pytest.raises(PromptInvalid):
        await lib.create("idea", "n", " ")


async def test_step_models_fall_back_to_the_catalog_and_refuse_strangers(lib):
    first = (await lib.step_models())["step_models"]
    assert set(first.values()) == {"ollama"} and "illustrations" in first
    await lib.set_step_model("code", "zai-org/glm-5.3-flash")
    assert (await lib.step_models())["step_models"]["code"] == "zai-org/glm-5.3-flash"
    with pytest.raises(PromptInvalid):
        await lib.set_step_model("nope", "ollama")
    with pytest.raises(PromptInvalid):
        await lib.set_step_model("code", "gpt-x")
    with pytest.raises(PromptInvalid):
        await lib.set_step_model("code", "")
    with pytest.raises(PromptInvalid):
        await lib.set_step_models({"code": "ollama", "nope": "ollama"})
    assert (await lib.step_models())["step_models"]["code"] == "zai-org/glm-5.3-flash"


@pytest.fixture
def client(store):
    cfg = config()
    app = create_app(cfg, Providers(cfg), OkChecker(), MemoryCallLog(), PromptLibrary(store))
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://svc")


async def push(client, owner="authoring", roles=("idea",)):
    body = {"owner": owner, "prompts": [{"role": r, "name": "Mặc định", "template_text": "T"} for r in roles]}
    return await client.put("/internal/v1/prompts/system", json=body)


async def test_routes_keep_the_shape_the_settings_page_reads(client):
    assert (await push(client)).json() == {"owner": "authoring", "written": 1, "removed": 0}
    got = (await client.get("/v1/prompts/idea")).json()
    assert set(got) == {"id", "role", "name", "template_text", "is_system", "is_active", "created_at", "updated_at"}
    assert got["is_system"] and got["is_active"]
    assert (await client.get("/internal/v1/prompts/idea")).json() == got

    made = await client.post("/v1/admin/prompts", json={"role": "idea", "name": "Mới", "template_text": "X"})
    assert made.status_code == 201 and made.json()["is_active"] is False
    pid = made.json()["id"]
    assert (await client.put(f"/v1/admin/prompts/{pid}", json={"name": "Sửa", "template_text": "Y"})).json()["name"] == "Sửa"
    assert (await client.post(f"/v1/admin/prompts/{pid}/activate")).json()["is_active"] is True
    listed = (await client.get("/v1/admin/prompts", params={"role": "idea"})).json()["prompts"]
    assert [p["is_system"] for p in listed] == [True, False]
    assert (await client.post(f"/v1/admin/prompts/{pid}/copy")).status_code == 201
    assert (await client.delete(f"/v1/admin/prompts/{pid}")).status_code == 204


async def test_routes_answer_refusals_with_the_old_status_codes(client):
    await push(client)
    assert (await client.get("/v1/prompts/nope")).status_code == 400
    assert (await client.put("/v1/admin/prompts/system-idea", json={"name": "n", "template_text": "t"})).status_code == 403
    assert (await client.delete("/v1/admin/prompts/system-idea")).json()["error"].startswith("system prompts")
    assert (await client.post("/v1/admin/prompts/zzz/activate")).status_code == 404
    await push(client, owner="other", roles=("idea",)) 
    assert (await push(client, owner="other", roles=("idea",))).status_code == 409


async def test_settings_routes_serve_the_step_defaults_and_the_catalog(client):
    body = (await client.get("/v1/settings/ai-model")).json()
    assert body["steps"][0] == "brief" and body["step_models"]["code"] == "ollama"
    assert (await client.put("/v1/settings/ai-model/steps/code", json={"model": "zai-org/glm-5.3-flash"})).status_code == 204
    assert (await client.get("/internal/v1/step-models")).json()["step_models"]["code"] == "zai-org/glm-5.3-flash"
    bad = await client.put("/v1/settings/ai-model/steps/code", json={"model": "gpt-x"})
    assert bad.status_code == 400 and "gpt-x" in bad.json()["error"]
    assert (await client.put("/internal/v1/step-models", json={"models": {"brief": "ollama"}})).status_code == 204
    ids = [m["id"] for m in (await client.get("/internal/v1/models")).json()["models"]]
    assert ids[0] == "ollama" and len(ids) == 3
