"""Bộ vẽ AI: model giả, bộ dựng giả, prompt giả."""

from __future__ import annotations

import pytest

from application.illustration_drawer import (
    MAX_DRAW_ATTEMPTS,
    MAX_OWN_REFERENCES,
    DrawRequest,
    IllustrationDrawer,
)
from application.illustration_library import IllustrationLibrary
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    ExemplarFolderError,
    Illustration,
    IllustrationError,
    IllustrationKind,
    IllustrationReadOnlyError,
    IllustrationStatus,
    InvalidIllustrationError,
    PromptUnavailableError,
    RefStyleRequiredError,
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


def make(replies):
    repo, rend, refs, prompts = FakeRepo(), FakeRenderer(), FakeReferences(), FakePrompts()
    library = IllustrationLibrary(repo, FakeFiles(), rend, refs)
    chat = ScriptedChat(replies)
    return IllustrationDrawer(library, chat, prompts, refs, 1000), library, repo, rend, chat, prompts


def draw_request(**kw) -> DrawRequest:
    return DrawRequest(**{"description": "xe máy màu đỏ", "folder_id": "phuong-tien", "ref_id": REF, **kw})


async def test_draw_retries_with_the_renderers_line_numbered_errors_then_stores_a_draft():
    drawer, _, _, _, chat, _ = make([reply("Motorbike", "BROKEN"), reply("Motorbike", MOTO)])
    out = await drawer.draw(draw_request())
    assert len(chat.calls) == 2
    second = chat.calls[1]["user"]
    assert "dòng 3" in second and "BROKEN" in second, "second turn must carry the failed code and its errors"
    assert out.name == "Motorbike" and out.title == "Xe máy" and out.status == IllustrationStatus.DRAFT
    assert out.has_preview and out.tags == ("xe", "xe máy") and out.ref_id == REF
    assert [c["phase"] for c in chat.calls] == ["attempt_1", "attempt_2"]
    assert chat.calls[0]["role"] == "illustration_drawer" and chat.calls[0]["step"] == "illustrations"


async def test_draw_gives_up_after_three_attempts_and_saves_nothing():
    drawer, _, repo, _, chat, _ = make([reply("Motorbike", "BROKEN")])
    with pytest.raises(InvalidIllustrationError) as exc:
        await drawer.draw(draw_request())
    assert len(chat.calls) == MAX_DRAW_ATTEMPTS
    assert str(exc.value).startswith(f"AI vẽ {MAX_DRAW_ATTEMPTS} lần vẫn chưa qua kiểm tra")
    assert exc.value.diagnostics[0].line == 3
    assert not any(r.name == "Motorbike" for r in repo.rows.values())


async def test_a_malformed_reply_is_asked_again_in_the_right_shape():
    drawer, _, _, _, chat, _ = make(["không có khối code", reply("Motorbike", MOTO)])
    await drawer.draw(draw_request())
    assert "sai khuôn" in chat.calls[1]["user"] and "```tsx" in chat.calls[1]["user"]


async def test_the_step_default_model_is_used_unless_the_request_names_one():
    drawer, _, _, _, chat, prompts = make([reply("Motorbike", MOTO)])
    await drawer.draw(draw_request())
    assert chat.calls[0]["model"] == prompts.model
    drawer, _, _, _, chat, _ = make([reply("Motorbike", MOTO)])
    await drawer.draw(draw_request(model="model-rieng"))
    assert chat.calls[0]["model"] == "model-rieng"


async def test_the_prompt_template_is_read_from_llm_service_and_filled_from_the_reference_style():
    drawer, _, repo, _, chat, prompts = make([reply("Motorbike", MOTO)])
    repo.rows["mine"] = Illustration(
        id="mine",
        name="Scooter",
        title="Xe tay ga",
        folder_id="phuong-tien",
        code="export function Scooter() {}",
        status=IllustrationStatus.APPROVED,
    )
    repo.rows["other"] = Illustration(
        id="other",
        name="Dog",
        folder_id="dong-vat",
        code="export function Dog() {}",
        status=IllustrationStatus.APPROVED,
    )
    for n, name in enumerate(["SchoolBus", "Cat", "Microscope"]):
        repo.rows[f"ex{n}"] = Illustration(
            id=f"ex{n}",
            name=name,
            folder_id=EXEMPLAR_FOLDER_ID,
            exemplar=True,
            code=f"export function {name}() {{}}",
            status=IllustrationStatus.APPROVED,
            created_at=f"2026-09-2{n + 1}T00:00:00Z",
        )
    await drawer.draw(draw_request(tier="hero"))
    system = chat.calls[0]["system"]
    assert prompts.asked == ["illustration_drawer"]
    # Biến phong cách lấy nguyên chữ của phong cách tham chiếu.
    assert "PHONG CÁCH Khoa học phát sáng: Khối bo tròn, ánh sáng kể chuyện." in system
    assert (
        "vector 2D / khối bo tròn / không viền / phẳng / theo nguồn sáng / một nguồn / lớp / phẳng / vân nhẹ"
        in system
    )
    assert "BỐ CỤC: trung tâm / bốn lớp / \n" in system
    assert "HẠNG CHÍNH (hero)" in system
    assert "<Figure" in system and "useBlink" in system, "kit helpers are filled in"
    for want in (
        "export function SchoolBus",
        "export function Cat",
        "export function Microscope",
        "export function Scooter",
    ):
        assert want in system
    assert "export function Dog" not in system, "another folder's drawing"
    assert "{{style_name}}" not in system and "backdropProps={{color, ground}}" in system
    assert system.index("SchoolBus") < system.index("Microscope"), "Hình mẫu must come oldest first"


async def test_prompt_without_exemplars_says_so_and_keeps_at_most_two_own_drawings():
    drawer, _, repo, _, chat, _ = make([reply("Motorbike", MOTO)])
    for name in ("Scooter", "Truck", "Van"):
        repo.rows[name] = Illustration(
            id=name,
            name=name,
            folder_id="phuong-tien",
            code=f"export function {name}() {{}}",
            status=IllustrationStatus.APPROVED,
        )
    await drawer.draw(draw_request())
    assert chat.calls[0]["system"].count("hình Creator đã duyệt, cùng thư mục") == MAX_OWN_REFERENCES

    drawer, _, _, _, chat, _ = make([reply("Motorbike", MOTO)])
    await drawer.draw(draw_request())
    assert "Chưa có hình tham chiếu" in chat.calls[0]["system"]


async def test_a_backdrop_uses_its_own_prompt_and_learns_only_from_backdrops():
    drawer, _, repo, _, chat, prompts = make([reply("ForestBackdrop", "export function ForestBackdrop() {}")])
    repo.rows["ex"] = Illustration(
        id="ex",
        name="Cat",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
    )
    repo.rows["fig"] = Illustration(
        id="fig",
        name="Tree",
        folder_id="boi-canh",
        code="export function Tree() {}",
        status=IllustrationStatus.APPROVED,
    )
    repo.rows["bd"] = Illustration(
        id="bd",
        name="RoomBackdrop",
        folder_id="boi-canh",
        code="export function RoomBackdrop() {}",
        kind=IllustrationKind.BACKDROP,
        status=IllustrationStatus.APPROVED,
    )
    out = await drawer.draw(draw_request(folder_id="boi-canh", kind=IllustrationKind.BACKDROP))
    system = chat.calls[0]["system"]
    assert prompts.asked == ["illustration_backdrop_drawer"]
    assert chat.calls[0]["role"] == "illustration_backdrop_drawer"
    assert "export function RoomBackdrop" in system
    assert "export function Cat" not in system and "export function Tree" not in system
    assert out.kind == IllustrationKind.BACKDROP


async def test_the_tier_decides_the_detail_the_drawing_remembers_and_the_renderer_checks_with():
    drawer, _, _, rend, chat, _ = make([reply("Motorbike", MOTO)])
    out = await drawer.draw(draw_request(tier="hero"))
    assert out.detail_level == "rich" and rend.last["tier"] == "hero"
    assert "HẠNG CHÍNH (hero)" in chat.calls[0]["system"]


async def test_draw_asks_the_reference_service_once_per_drawing_not_once_per_attempt():
    drawer, library, _, _, _, _ = make([reply("Motorbike", "BROKEN"), reply("Motorbike", MOTO)])
    await drawer.draw(draw_request())
    assert library._references.kit_calls == [REF]


async def test_draw_refuses_the_exemplar_folder_a_missing_style_and_an_empty_description():
    drawer, _, _, _, chat, _ = make([reply("Motorbike", MOTO)])
    with pytest.raises(ExemplarFolderError):
        await drawer.draw(draw_request(folder_id=EXEMPLAR_FOLDER_ID))
    with pytest.raises(RefStyleRequiredError):
        await drawer.draw(draw_request(ref_id=""))
    with pytest.raises(IllustrationError, match="bắt buộc"):
        await drawer.draw(draw_request(description="  "))
    assert chat.calls == [], "the model must not be called"


async def test_a_prompt_with_a_variable_the_code_cannot_fill_is_refused():
    drawer, _, _, _, chat, prompts = make([reply("Motorbike", MOTO)])
    prompts.templates["illustration_drawer"] = "xin {{khong_biet}} {{style_name}}"
    with pytest.raises(IllustrationError, match="khong_biet"):
        await drawer.draw(draw_request())
    assert chat.calls == []


async def test_a_missing_prompt_stops_the_drawing_with_a_clear_error():
    drawer, _, _, _, chat, prompts = make([reply("Motorbike", MOTO)])
    del prompts.templates["illustration_drawer"]
    with pytest.raises(PromptUnavailableError):
        await drawer.draw(draw_request())
    assert chat.calls == []


async def test_redraw_keeps_the_name_and_makes_a_new_draft_version():
    drawer, library, repo, _, chat, _ = make([reply("Motorbike", MOTO + "\n// bánh to hơn")])
    made = await library.create(Illustration(name="Motorbike", folder_id="phuong-tien", code=MOTO), REF)
    await library.set_status(made.id, "approved")
    out = await drawer.redraw(made.id, "bánh to hơn", "")
    assert (
        out.version == 2 and out.status == IllustrationStatus.DRAFT and "bánh to hơn" in chat.calls[0]["user"]
    )
    chat.replies = [reply("Scooter", MOTO.replace("Motorbike", "Scooter"))]
    with pytest.raises(IllustrationError, match="giữ tên Motorbike"):
        await drawer.redraw(made.id, "", "")
    with pytest.raises(IllustrationReadOnlyError):
        await drawer.redraw("builtin-Tooth", "", "")
    assert repo.rows[made.id].version == 2


async def test_redraw_of_a_drawing_without_a_style_needs_one():
    drawer, _, repo, *_ = make([reply("Bus", BUS_CODE)])
    repo.rows["bus"] = Illustration(
        id="bus", name="Bus", folder_id="phuong-tien", code=BUS_CODE, status=IllustrationStatus.APPROVED
    )
    with pytest.raises(RefStyleRequiredError):
        await drawer.redraw("bus", "", "")
    out = await drawer.redraw("bus", "", "", ref_id=REF)
    assert out.ref_id == REF


async def test_redraw_refuses_exemplars_before_calling_the_model():
    drawer, _, repo, _, chat, _ = make([reply("Bus", BUS_CODE)])
    repo.rows["ex"] = Illustration(
        id="ex",
        name="BusMau",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        code=BUS_CODE,
        status=IllustrationStatus.APPROVED,
    )
    with pytest.raises(IllustrationReadOnlyError):
        await drawer.redraw("ex", "", "")
    assert chat.calls == []
