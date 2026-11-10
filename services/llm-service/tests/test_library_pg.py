"""PgLibraryStore against a real Postgres. Runs only when LLM_TEST_DATABASE_URL points at a
throwaway database; the library tables are emptied before and after."""

import os

import pytest

from app.library import PgLibraryStore, PromptConflict, PromptLibrary, SystemPrompt

DSN = os.environ.get("LLM_TEST_DATABASE_URL", "")

pytestmark = pytest.mark.skipif(not DSN, reason="LLM_TEST_DATABASE_URL is not set")


@pytest.fixture
async def lib():
    store = PgLibraryStore(DSN)
    await store.start()
    await store.pool.execute("DELETE FROM prompts")
    await store.pool.execute("DELETE FROM step_default_models")
    yield PromptLibrary(store)
    await store.pool.execute("DELETE FROM prompts")
    await store.pool.execute("DELETE FROM step_default_models")
    await store.stop()


def sp(role, text="T"):
    return SystemPrompt(role, "Mặc định", text)


async def test_push_activate_delete_round_trip(lib):
    await lib.replace_system("authoring", [sp("idea"), sp("brief")])
    assert (await lib.active("idea")).id == "system-idea"
    mine = await lib.create("idea", "Của tôi", "X")
    await lib.activate(mine.id)
    assert (await lib.active("idea")).id == mine.id
    await lib.replace_system("authoring", [sp("idea", "T2")])
    assert (await lib.active("idea")).id == mine.id
    assert {p.id for p in await lib.list()} == {"system-idea", mine.id}
    await lib.delete(mine.id)
    assert (await lib.active("idea")).template_text == "T2"


async def test_a_push_leaves_other_owners_and_creator_rows(lib):
    await lib.replace_system("authoring", [sp("idea")])
    await lib.replace_system("visual", [sp("shot")])
    creator = await lib.create("idea", "Của Creator", "X")
    out = await lib.replace_system("authoring", [])
    assert out["removed"] == 1
    assert {p.id for p in await lib.list()} == {"system-shot", creator.id}
    with pytest.raises(PromptConflict):
        await lib.replace_system("authoring", [sp("shot")])


async def test_step_models_persist(lib):
    await lib.set_step_model("code", "zai-org/glm-5.3-flash")
    await lib.set_step_model("code", "ollama")
    assert (await lib.step_models())["step_models"]["code"] == "ollama"
