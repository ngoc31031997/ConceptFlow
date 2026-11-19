"""Bộ vẽ AI: mỗi hình một lượt gọi model, theo luật của phong cách tham chiếu, bộ dựng kiểm tra, và
nếu chưa qua thì gửi lại kèm lỗi có số dòng, tối đa `MAX_DRAW_ATTEMPTS` lượt.
"""

from __future__ import annotations

import logging
import re
from dataclasses import dataclass

from application.illustration_library import IllustrationLibrary, normalize_illustration
from application.illustration_ports import (
    ChatPort,
    PromptSourcePort,
    ReferenceSourcePort,
    RenderedIllustration,
)
from domain import drawer_prompts
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    MAX_EXEMPLARS,
    Illustration,
    IllustrationError,
    IllustrationFilter,
    IllustrationKind,
    IllustrationReadOnlyError,
    IllustrationStatus,
    InvalidIllustrationError,
    RefStyleRequiredError,
    detail_for_tier,
    kind_or_figure,
    tier_for_detail,
    tier_or_support,
    validate_illustration_name,
)

logger = logging.getLogger(__name__)

MAX_DRAW_ATTEMPTS = 3
#: Số hình đã duyệt của Creator cùng thư mục được nối sau Hình mẫu trong prompt.
MAX_OWN_REFERENCES = 2
DRAW_TEMPERATURE = 0.5
NO_REFERENCES_TEXT = "(Chưa có hình tham chiếu — dựng đúng theo luật style và linh kiện ở trên.)"

_REPLY_FIELD_RE = re.compile(r"^(TÊN|TÊN HIỂN THỊ|THẺ|MÔ TẢ|CÁCH GỌI):\s*(.+)$", re.M)
_REPLY_CODE_RE = re.compile(r"```(?:tsx|typescript|ts|jsx)?\s*\n(.*?)```", re.S)
_EXPORT_NAME_RE = re.compile(r"^export function ([A-Z][A-Za-z0-9]*)\s*\(", re.M)


@dataclass
class DrawRequest:
    description: str
    folder_id: str
    ref_id: str
    name: str = ""  # tuỳ chọn; để trống thì model tự đề xuất
    model: str = ""  # tuỳ chọn: model riêng cho lượt này
    kind: IllustrationKind = IllustrationKind.FIGURE
    #: Vai của hình trong video (hero, support, background); hình thư viện lấy từ mức chi tiết.
    tier: str = "support"


@dataclass
class DrawnReply:
    name: str
    title: str
    tags: str
    description: str
    usage: str
    code: str


def parse_drawn_reply(text: str) -> DrawnReply:
    fields = {"TÊN": "", "TÊN HIỂN THỊ": "", "THẺ": "", "MÔ TẢ": "", "CÁCH GỌI": ""}
    for m in _REPLY_FIELD_RE.finditer(text):
        fields[m.group(1)] = m.group(2).strip()
    code = _REPLY_CODE_RE.search(text)
    if code is None:
        raise IllustrationError("câu trả lời không có khối ```tsx")
    body = code.group(1).strip() + "\n"
    # Hàm được export mới là sự thật; dòng TÊN nói khác thì bỏ qua.
    exported = _EXPORT_NAME_RE.search(body)
    name = exported.group(1) if exported else fields["TÊN"]
    return DrawnReply(
        name=name, title=fields["TÊN HIỂN THỊ"], tags=fields["THẺ"], description=fields["MÔ TẢ"],
        usage=fields["CÁCH GỌI"], code=body)


class IllustrationDrawer:
    def __init__(
        self, library: IllustrationLibrary, chat: ChatPort, prompts: PromptSourcePort,
        references: ReferenceSourcePort, max_tokens: int,
    ) -> None:
        self._library = library
        self._chat = chat
        self._prompts = prompts
        self._references = references
        self._max_tokens = max_tokens

    async def _reference_text(self, folder_id: str, skip_id: str, kind: IllustrationKind) -> str:
        """Hình tham chiếu: Hình mẫu Creator đã chọn (cho hình), rồi các hình đã duyệt cùng loại và cùng
        thư mục của Creator."""
        parts: list[str] = []

        def add(i: Illustration, why: str) -> bool:
            if not i.code or i.id == skip_id:
                return False
            parts.append(f"### {i.name} — {i.title} ({why})\n```tsx\n{i.code}```\n\n")
            return True

        exemplars = [] if kind == IllustrationKind.BACKDROP else await self._library.exemplars()
        for exemplar in exemplars[:MAX_EXEMPLARS]:
            add(exemplar, "hình mẫu chuẩn")
        if folder_id and folder_id != EXEMPLAR_FOLDER_ID:
            mine = await self._library.repo.list_illustrations(
                IllustrationFilter(folder_id=folder_id, status=IllustrationStatus.APPROVED.value))
            own = 0
            for i in mine:
                if own < MAX_OWN_REFERENCES and not i.read_only and kind_or_figure(i.kind) == kind \
                        and add(i, "hình Creator đã duyệt, cùng thư mục"):
                    own += 1
        return "".join(parts).strip()

    async def _system_prompt(
        self, folder_id: str, skip_id: str, kind: IllustrationKind, style: dict, tier: str,
    ) -> tuple[str, str]:
        """Prompt hệ thống của bộ vẽ, đã điền biến, và role của nó. Template đọc từ llm-service mỗi lần vẽ."""
        role = drawer_prompts.role_for(kind)
        template = await self._prompts.get_template(role)
        allowed = set(drawer_prompts.variables_for(kind))
        unknown = drawer_prompts.placeholders(template) - allowed
        if unknown:
            raise IllustrationError(
                f"prompt {role} có biến code không biết điền: {', '.join(sorted(unknown))}")
        values = drawer_prompts.style_values(style)
        values["tier"] = drawer_prompts.tier_instruction(tier)
        values["helpers"] = drawer_prompts.helpers_text(kind)
        values["references"] = await self._reference_text(folder_id, skip_id, kind) or NO_REFERENCES_TEXT
        text = drawer_prompts.fill(template, values)
        return text, role

    async def _draw(
        self, system: str, role: str, user: str, model: str, ref_id: str, kit: dict,
        kind: IllustrationKind, tier: str,
    ) -> tuple[DrawnReply, RenderedIllustration]:
        """Chạy model tới khi bộ dựng nhận hình. `user` là lượt đầu; mỗi lượt hỏng nối thêm code vừa
        vẽ và lỗi."""
        model = model or await self._prompts.step_default_model(drawer_prompts.STEP_ILLUSTRATIONS)
        turn = user
        last_err: IllustrationError | None = None
        for attempt in range(1, MAX_DRAW_ATTEMPTS + 1):
            res = await self._chat.chat(
                system=system, user=turn, model=model, max_tokens=self._max_tokens,
                temperature=DRAW_TEMPERATURE, role=role, step=drawer_prompts.STEP_ILLUSTRATIONS,
                phase=f"attempt_{attempt}")
            try:
                reply = parse_drawn_reply(res.content)
                validate_illustration_name(reply.name)
            except IllustrationError as exc:
                last_err = exc
                turn = f"{user}\n\nLần trước câu trả lời sai khuôn: {exc}. Trả lời lại đúng khuôn."
                continue
            try:
                out = await self._library.render(ref_id, reply.name, reply.code, kind, tier, kit=kit)
            except InvalidIllustrationError as exc:
                last_err = exc
                turn = (
                    f"{user}\n\nCode lần trước của bạn:\n```tsx\n{reply.code}```\n\n"
                    f"Chưa qua kiểm tra: {exc}\nSửa đúng các lỗi đó (số dòng tính trong code trên), "
                    "giữ nguyên ý hình, trả lời lại đầy đủ theo khuôn.")
                continue
            return reply, out
        prefix = f"AI vẽ {MAX_DRAW_ATTEMPTS} lần vẫn chưa qua kiểm tra: "
        if isinstance(last_err, InvalidIllustrationError):
            raise InvalidIllustrationError(last_err.diagnostics, prefix=prefix)
        raise IllustrationError(prefix + str(last_err))

    async def draw(self, req: DrawRequest) -> Illustration:
        """Nhờ model vẽ một hình mới và lưu thành bản nháp."""
        description = req.description.strip()
        if not description:
            raise IllustrationError("mô tả hình cần vẽ là bắt buộc")
        await self._library.writable_folder(req.folder_id)
        user = "Vẽ hình: " + description
        if req.name.strip():
            user += "\nTên component bắt buộc: " + req.name.strip()
        kind = kind_or_figure(req.kind)
        if not req.ref_id:
            raise RefStyleRequiredError()
        tier = tier_or_support(req.tier)
        kit = await self._library.kit_for(req.ref_id)
        style = await self._references.get_style(req.ref_id)
        system, role = await self._system_prompt(req.folder_id, "", kind, style, tier)
        reply, out = await self._draw(system, role, user, req.model, req.ref_id, kit, kind, tier)
        row = normalize_illustration(Illustration(
            name=reply.name, title=reply.title, folder_id=req.folder_id,
            tags=tuple(reply.tags.split(",")), description=reply.description, usage=reply.usage,
            code=reply.code, kind=kind, ref_id=req.ref_id, detail_level=detail_for_tier(tier)))
        saved = await self._library.repo.create_illustration(row.with_(
            status=IllustrationStatus.DRAFT, version=1, warnings=tuple(out.warnings)))
        await self._library.save_preview(saved.id, saved.version, out)
        return saved.with_(has_preview=True)

    async def redraw(self, illustration_id: str, note: str, model: str, ref_id: str = "") -> Illustration:
        """Nhờ model vẽ lại một hình của Creator theo ghi chú ("bánh xe to hơn"). Tên giữ nguyên; kết quả
        là một phiên bản nháp mới."""
        existing = await self._library.repo.get_illustration(illustration_id)
        if existing.read_only:
            raise IllustrationReadOnlyError()
        user = (
            f"Vẽ lại hình {existing.name} ({existing.title}): {existing.description}\n"
            f"Giữ nguyên tên component {existing.name} và các prop đang có.\n"
            f"Code hiện tại:\n```tsx\n{existing.code}```")
        if note.strip():
            user += "\nYêu cầu sửa của Creator: " + note.strip()
        else:
            user += "\nCreator chưa ưng hình này: vẽ một phiên bản khác, đúng luật style hơn."
        style_ref = existing.ref_id or ref_id
        if not style_ref:
            raise RefStyleRequiredError()
        tier = tier_for_detail(existing.detail_level)
        kit = await self._library.kit_for(style_ref)
        style = await self._references.get_style(style_ref)
        system, role = await self._system_prompt(existing.folder_id, existing.id, existing.kind, style, tier)
        reply, out = await self._draw(system, role, user, model, style_ref, kit, existing.kind, tier)
        if reply.name != existing.name:
            raise IllustrationError(
                f"AI đổi tên component thành {reply.name} — vẽ lại phải giữ tên {existing.name}")
        saved = await self._library.repo.update_illustration(existing.with_(
            code=reply.code, usage=reply.usage or existing.usage, ref_id=style_ref,
            status=IllustrationStatus.DRAFT, version=existing.version + 1, warnings=tuple(out.warnings)))
        await self._library.save_preview(illustration_id, saved.version, out)
        return saved.with_(has_preview=True)
