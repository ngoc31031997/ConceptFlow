"""Adapter của thư viện hình: file ảnh, client llm-service và visual-service, bộ dựng trong tiến trình."""

from __future__ import annotations

import asyncio
import base64
import json

import httpx
import pytest

from adapters.llm.client import LLMServiceClient, push_system_prompts_until_accepted
from adapters.rendering.illustration_previewer import IllustrationPreviewError
from adapters.rendering.library_renderer import InProcessIllustrationRenderer
from adapters.storage.illustration_files import IllustrationFiles
from adapters.visual.client import VisualServiceClient
from application.preview_illustration import IllustrationPreview
from domain.drawer_prompts import SystemPrompt
from domain.illustration import (
    LLMError,
    PromptUnavailableError,
    ReferenceKitMissingError,
    ReferenceServiceError,
    ReferenceStyleNotFoundError,
    RendererUnavailableError,
)
from tests.illustration_fakes import STYLE
from tests.kit_fixtures import phatsang_kit

# --- file ảnh ---


def test_files_are_written_as_v_version_and_old_versions_are_removed(tmp_path):
    files = IllustrationFiles(tmp_path)
    png, gif = files.save("abc", 1, b"png1", b"gif1")
    assert png == str(tmp_path / "abc" / "v1.png") and gif == str(tmp_path / "abc" / "v1.gif")
    assert files.read(png) == b"png1"
    png2, gif2 = files.save("abc", 2, b"png2", b"")
    assert gif2 == "" and files.read(png2) == b"png2"
    assert sorted(p.name for p in (tmp_path / "abc").iterdir()) == [
        "v2.png"
    ], "no temp file or old version stays"


def test_remove_deletes_the_folder_and_a_missing_one_is_fine(tmp_path):
    files = IllustrationFiles(tmp_path)
    files.save("abc", 1, b"p", b"g")
    files.remove("abc")
    assert not (tmp_path / "abc").exists()
    files.remove("abc")


def test_paths_outside_the_root_and_ids_with_separators_are_refused(tmp_path):
    files = IllustrationFiles(tmp_path / "root")
    (tmp_path / "secret").write_text("x")
    with pytest.raises(ValueError):
        files.read(str(tmp_path / "secret"))
    for bad in ("", "..", "a/b", "a\\b"):
        with pytest.raises(ValueError):
            files.save(bad, 1, b"p", b"g")


# --- llm-service ---


def llm(handler) -> LLMServiceClient:
    return LLMServiceClient(
        "http://llm", client=httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://llm")
    )


def ndjson(*events: dict) -> httpx.Response:
    return httpx.Response(200, content="\n".join(json.dumps(e) for e in events) + "\n")


async def test_chat_sends_the_call_context_and_returns_the_result_of_the_stream():
    seen = {}

    def handler(request: httpx.Request) -> httpx.Response:
        seen["path"], seen["body"] = request.url.path, json.loads(request.content)
        return ndjson(
            {"type": "progress", "content_chars": 3},
            {"type": "result", "content": "xin chào", "call_id": "c9"},
        )

    out = await llm(handler).chat(
        system="S",
        user="U",
        model="m",
        max_tokens=99,
        temperature=0.5,
        role="illustration_drawer",
        step="illustrations",
        phase="attempt_1",
    )
    assert (out.content, out.call_id) == ("xin chào", "c9")
    assert seen["path"] == "/v1/chat"
    assert seen["body"] == {
        "provider": "hive",
        "system": "S",
        "user": "U",
        "model": "m",
        "max_tokens": 99,
        "temperature": 0.5,
        "role": "illustration_drawer",
        "step": "illustrations",
        "phase": "attempt_1",
        "expect_json": False,
    }


async def test_chat_errors_carry_the_kind_and_the_call_id():
    def error_event(_):
        return ndjson(
            {
                "type": "error",
                "error": {"kind": "balance", "provider": "hive", "message": "no credit", "call_id": "c1"},
            }
        )

    with pytest.raises(LLMError) as exc:
        await llm(error_event).chat(
            system="", user="", model="", max_tokens=1, temperature=0, role="r", step="s", phase="p"
        )
    assert (exc.value.kind, exc.value.provider, exc.value.call_id) == ("balance", "hive", "c1")

    def status_error(_):
        return httpx.Response(422, json={"error": {"kind": "malformed", "message": "bad json"}})

    with pytest.raises(LLMError) as exc:
        await llm(status_error).chat(
            system="", user="", model="", max_tokens=1, temperature=0, role="r", step="s", phase="p"
        )
    assert exc.value.kind == "malformed" and exc.value.provider == "llm-service"

    def closed(_):
        return ndjson({"type": "progress"})

    with pytest.raises(LLMError, match="closed the stream"):
        await llm(closed).chat(
            system="", user="", model="", max_tokens=1, temperature=0, role="r", step="s", phase="p"
        )


async def test_chat_when_llm_service_is_unreachable_is_a_server_error_naming_it():
    def down(request):
        raise httpx.ConnectError("refused", request=request)

    with pytest.raises(LLMError) as exc:
        await llm(down).chat(
            system="", user="", model="", max_tokens=1, temperature=0, role="r", step="s", phase="p"
        )
    assert exc.value.kind == "server" and exc.value.provider == "llm-service"


async def test_get_template_reads_the_active_prompt_of_the_role():
    def handler(request):
        assert request.url.path == "/internal/v1/prompts/illustration_drawer"
        return httpx.Response(200, json={"role": "illustration_drawer", "template_text": "TEMPLATE"})

    assert await llm(handler).get_template("illustration_drawer") == "TEMPLATE"
    with pytest.raises(PromptUnavailableError):
        await llm(lambda _: httpx.Response(400, json={"error": "unknown role"})).get_template("x")
    with pytest.raises(PromptUnavailableError):
        await llm(lambda _: httpx.Response(404)).get_template("x")
    with pytest.raises(PromptUnavailableError):
        await llm(lambda _: httpx.Response(200, json={"template_text": "  "})).get_template("x")
    with pytest.raises(LLMError):
        await llm(lambda _: httpx.Response(500, text="boom")).get_template("x")


async def test_step_default_model_is_empty_when_llm_service_cannot_say():
    ok = llm(lambda _: httpx.Response(200, json={"step_models": {"illustrations": "glm-5", "code": "x"}}))
    assert await ok.step_default_model("illustrations") == "glm-5"
    assert await ok.step_default_model("unknown") == ""
    assert await llm(lambda _: httpx.Response(500)).step_default_model("illustrations") == ""


async def test_push_system_prompts_sends_the_owner_and_every_prompt():
    seen = {}

    def handler(request):
        seen["method"], seen["path"], seen["body"] = (
            request.method,
            request.url.path,
            json.loads(request.content),
        )
        return httpx.Response(200, json={"owner": "rendering", "written": 2, "removed": 0})

    prompts = [SystemPrompt("illustration_drawer", "Bộ vẽ hình", "T1"), SystemPrompt("b", "Bộ vẽ nền", "T2")]
    await push_system_prompts_until_accepted(llm(handler), prompts)
    assert (seen["method"], seen["path"]) == ("PUT", "/internal/v1/prompts/system")
    assert seen["body"] == {
        "owner": "rendering",
        "prompts": [
            {"role": "illustration_drawer", "name": "Bộ vẽ hình", "template_text": "T1"},
            {"role": "b", "name": "Bộ vẽ nền", "template_text": "T2"},
        ],
    }


async def test_push_retries_with_a_growing_pause_until_llm_service_accepts():
    attempts = []

    def handler(_):
        attempts.append(1)
        return httpx.Response(503, text="starting") if len(attempts) < 3 else httpx.Response(200, json={})

    await asyncio.wait_for(
        push_system_prompts_until_accepted(llm(handler), [SystemPrompt("r", "n", "t")], first_pause=0.01), 5
    )
    assert len(attempts) == 3


# --- visual-service ---


def visual(handler) -> VisualServiceClient:
    return VisualServiceClient(
        "http://v", client=httpx.AsyncClient(transport=httpx.MockTransport(handler), base_url="http://v")
    )


async def test_get_kit_reads_the_kit_of_a_reference_style():
    def handler(request):
        assert request.url.path == "/internal/v1/reference-styles/R12/kit"
        return httpx.Response(200, json={"ref_id": "R12", "kit": phatsang_kit()})

    assert await visual(handler).get_kit("R12") == phatsang_kit()


async def test_get_kit_errors_are_told_apart():
    with pytest.raises(ReferenceStyleNotFoundError):
        await visual(lambda _: httpx.Response(404, json={"error": "x"})).get_kit("R99")
    with pytest.raises(ReferenceKitMissingError):
        await visual(lambda _: httpx.Response(409, json={"error": "x"})).get_kit("R05")
    with pytest.raises(ReferenceServiceError):
        await visual(lambda _: httpx.Response(500)).get_kit("R12")
    with pytest.raises(ReferenceServiceError):
        await visual(lambda _: httpx.Response(200, json={"ref_id": "R12", "kit": None})).get_kit("R12")

    def down(request):
        raise httpx.ConnectError("refused", request=request)

    with pytest.raises(ReferenceServiceError, match="visual-service"):
        await visual(down).get_kit("R12")


async def test_get_style_finds_the_style_in_the_list():
    def handler(request):
        assert request.url.path == "/v1/reference-styles"
        return httpx.Response(200, json={"styles": [{"id": "R01"}, STYLE]})

    assert (await visual(handler).get_style("R12"))["name"] == STYLE["name"]
    with pytest.raises(ReferenceStyleNotFoundError):
        await visual(handler).get_style("R99")


# --- bộ dựng trong tiến trình ---


class FakePreview:
    def __init__(self, result=None, error=None):
        self.result, self.error, self.calls = result, error, []

    def run(self, kit, name, code, props, gif, kind, tier):
        self.calls.append((kit, name, code, props, gif, kind, tier))
        if self.error:
            raise self.error
        return self.result


async def test_the_renderer_decodes_the_images_and_passes_kind_and_tier():
    result = IllustrationPreview(
        True,
        png=base64.b64encode(b"PNG").decode(),
        gif=base64.b64encode(b"GIF").decode(),
        warnings=[{"message": "S11", "line": 4}],
    )
    fake = FakePreview(result)
    out = await InProcessIllustrationRenderer(fake, asyncio.Semaphore(1)).render(
        phatsang_kit(), "Bus", "code", "backdrop", "hero"
    )
    assert out.ok and out.png == b"PNG" and out.gif == b"GIF" and out.warnings[0].line == 4
    assert fake.calls[0][1:] == ("Bus", "code", None, True, "backdrop", "hero")


async def test_the_renderer_reports_rejected_code_and_an_unavailable_engine():
    rejected = IllustrationPreview(False, diagnostics=[{"message": "TS2304", "line": 3}])
    out = await InProcessIllustrationRenderer(FakePreview(rejected), asyncio.Semaphore(1)).render(
        phatsang_kit(), "Bus", "code", "figure", "support"
    )
    assert not out.ok and out.diagnostics[0].line == 3
    with pytest.raises(RendererUnavailableError, match="dựng xem trước"):
        await InProcessIllustrationRenderer(
            FakePreview(error=IllustrationPreviewError("node chết")), asyncio.Semaphore(1)
        ).render(phatsang_kit(), "Bus", "code", "figure", "support")
