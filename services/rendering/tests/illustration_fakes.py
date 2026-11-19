"""Bản giả trong bộ nhớ của các cổng thư viện hình, dùng chung cho test use case và test HTTP.

Không bản nào gọi model, visual-service, trình duyệt hay database thật.
"""

from __future__ import annotations

from application.illustration_ports import ChatResult, RenderedIllustration
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    AlreadyExemplarError,
    CodeFinding,
    ExemplarLimitError,
    FolderNotFoundError,
    Illustration,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationNameTakenError,
    IllustrationNotFoundError,
    ReferenceStyleNotFoundError,
    RendererUnavailableError,
    builtin_illustrations,
    system_folders,
)
from tests.kit_fixtures import phatsang_kit

REF = "R12"
BUS_CODE = "export function Bus() { return null; }"


class FakeRepo:
    def __init__(self) -> None:
        self.folders: list[IllustrationFolder] = system_folders()
        self.rows: dict[str, Illustration] = {b.id: b for b in builtin_illustrations()}
        self.previews: dict[str, tuple[int, str, str]] = {}
        self.next = 0

    def _stamp(self) -> str:
        self.next += 1
        return f"2026-09-29T00:00:{self.next:02d}Z"

    async def list_folders(self):
        return list(self.folders)

    async def create_folder(self, folder):
        if any(f.id == folder.id for f in self.folders):
            from domain.illustration import FolderTakenError

            raise FolderTakenError()
        self.folders.append(folder)
        return folder

    async def delete_folder(self, folder_id):
        self.folders = [f for f in self.folders if f.id != folder_id]

    async def list_illustrations(self, flt: IllustrationFilter):
        return [
            r
            for r in self.rows.values()
            if (not flt.folder_id or r.folder_id == flt.folder_id)
            and (not flt.status or r.status == flt.status)
        ]

    async def get_illustration(self, illustration_id):
        if illustration_id not in self.rows:
            raise IllustrationNotFoundError()
        row = self.rows[illustration_id]
        return row.with_(has_preview=self._has_preview(row))

    def _has_preview(self, row: Illustration) -> bool:
        pv = self.previews.get(row.id)
        return bool(pv and pv[0] == row.version and pv[1])

    async def create_illustration(self, illustration):
        if any(r.name == illustration.name for r in self.rows.values()):
            raise IllustrationNameTakenError()
        if not any(f.id == illustration.folder_id for f in self.folders):
            raise FolderNotFoundError()
        self.next += 1
        row = illustration.with_(id=f"i{self.next}", created_at=self._stamp(), updated_at=self._stamp())
        self.rows[row.id] = row
        return row

    async def update_illustration(self, illustration):
        if illustration.id not in self.rows or self.rows[illustration.id].builtin:
            raise IllustrationNotFoundError()
        row = illustration.with_(created_at=self.rows[illustration.id].created_at)
        self.rows[row.id] = row
        return row

    async def delete_illustration(self, illustration_id):
        if illustration_id in self.rows and not self.rows[illustration_id].builtin:
            del self.rows[illustration_id]
            for r in list(self.rows.values()):
                if r.source_id == illustration_id:
                    self.rows[r.id] = r.with_(source_id="")

    async def create_exemplar(self, illustration, limit):
        count = 0
        for e in self.rows.values():
            if e.exemplar:
                count += 1
                if illustration.source_id and e.source_id == illustration.source_id:
                    raise AlreadyExemplarError()
            if e.name == illustration.name:
                raise IllustrationNameTakenError()
        if count >= limit:
            raise ExemplarLimitError()
        self.next += 1
        row = illustration.with_(
            id=f"i{self.next}", folder_id=EXEMPLAR_FOLDER_ID, exemplar=True, created_at=self._stamp()
        )
        self.rows[row.id] = row
        return row

    async def release_exemplar(self, illustration_id, folder_id):
        row = self.rows.get(illustration_id)
        if row is None or not row.exemplar:
            raise IllustrationNotFoundError()
        row = row.with_(exemplar=False, home_folder_id="", folder_id=folder_id)
        self.rows[illustration_id] = row
        return row

    async def set_preview(self, illustration_id, version, png_path, gif_path):
        self.previews[illustration_id] = (version, png_path, gif_path)

    async def get_preview(self, illustration_id):
        if illustration_id not in self.rows:
            raise IllustrationNotFoundError()
        pv = self.previews.get(illustration_id)
        if pv is None or pv[0] != self.rows[illustration_id].version:
            return "", ""
        return pv[1], pv[2]


class FakeFiles:
    def __init__(self) -> None:
        self.files: dict[str, bytes] = {}

    def save(self, illustration_id, version, png, gif):
        for path in [p for p in self.files if p.startswith(f"/shared/illustrations/{illustration_id}/")]:
            del self.files[path]
        base = f"/shared/illustrations/{illustration_id}/v{version}"
        self.files[base + ".png"] = png
        if gif:
            self.files[base + ".gif"] = gif
        return base + ".png", base + ".gif" if gif else ""

    def read(self, path):
        return self.files[path]

    def remove(self, illustration_id):
        for path in [p for p in self.files if p.startswith(f"/shared/illustrations/{illustration_id}/")]:
            del self.files[path]


class FakeRenderer:
    def __init__(self) -> None:
        self.calls: list[str] = []
        self.last: dict = {}
        self.fail = False
        self.unreachable = False

    async def render(self, kit, name, code, kind, tier):
        self.calls.append(f"{name}|{code}")
        self.last = {"kit": kit, "kind": kind, "tier": tier}
        if self.unreachable:
            raise RendererUnavailableError("dựng xem trước: rendering unreachable")
        if self.fail or "BROKEN" in code:
            return RenderedIllustration(False, [CodeFinding("TS2304: Cannot find name", 3)])
        return RenderedIllustration(True, png=f"PNG:{name}{code}".encode(), gif=b"GIF")


STYLE = {
    "tags": ["khoa học"], "source": {"url": ""}, "created_at": "2026-01-01T00:00:00Z",
    "id": REF,
    "name": "Khoa học phát sáng",
    "summary": "Khối bo tròn, ánh sáng kể chuyện.",
    "drawing": {
        "medium": "vector 2D",
        "shape": "khối bo tròn",
        "outline": "không viền",
        "fill": "phẳng",
        "shading": "theo nguồn sáng",
        "light": "một nguồn",
        "depth": "lớp",
        "perspective": "phẳng",
        "texture": "vân nhẹ",
    },
    "layout": {"structure": "trung tâm", "layering": "bốn lớp", "persistent_elements": ""},
}


class FakeReferences:
    def __init__(self) -> None:
        self.kit_calls: list[str] = []

    async def get_kit(self, ref_id):
        self.kit_calls.append(ref_id)
        if ref_id != REF:
            raise ReferenceStyleNotFoundError(ref_id)
        return phatsang_kit()

    async def get_style(self, ref_id):
        if ref_id != REF:
            raise ReferenceStyleNotFoundError(ref_id)
        return STYLE


class ScriptedChat:
    def __init__(self, replies: list[str]) -> None:
        self.replies = replies
        self.calls: list[dict] = []

    async def chat(self, **kwargs):
        self.calls.append(kwargs)
        reply = self.replies[0]
        if len(self.replies) > 1:
            self.replies = self.replies[1:]
        return ChatResult(content=reply)


FIGURE_TEMPLATE = (
    "PHONG CÁCH {{style_name}}: {{style_summary}}\n"
    "VẼ: {{drawing_medium}} / {{drawing_shape}} / {{drawing_outline}} / {{drawing_fill}} / "
    "{{drawing_shading}} / {{drawing_light}} / {{drawing_depth}} / {{drawing_perspective}} / "
    "{{drawing_texture}}\n"
    "BỐ CỤC: {{layout_structure}} / {{layout_layering}} / {{layout_persistent_elements}}\n"
    "HẠNG: {{tier}}\nLINH KIỆN:\n{{helpers}}\nTHAM CHIẾU:\n{{references}}\n"
    "CÁCH GỌI: backdropProps={{color, ground}}"
)
BACKDROP_TEMPLATE = (
    "NỀN {{style_name}} {{drawing_light}} {{layout_layering}}\n"
    "LINH KIỆN:\n{{helpers}}\nTHAM CHIẾU:\n{{references}}"
)


class FakePrompts:
    def __init__(self) -> None:
        self.templates = {
            "illustration_drawer": FIGURE_TEMPLATE,
            "illustration_backdrop_drawer": BACKDROP_TEMPLATE,
        }
        self.model = "model-mac-dinh"
        self.asked: list[str] = []

    async def get_template(self, role):
        from domain.illustration import PromptUnavailableError

        self.asked.append(role)
        if role not in self.templates:
            raise PromptUnavailableError(f'llm-service chưa có prompt "{role}"')
        return self.templates[role]

    async def step_default_model(self, step):
        return self.model


def reply(name: str, code: str) -> str:
    return (
        f"TÊN: {name}\nTÊN HIỂN THỊ: Xe máy\nTHẺ: xe, xe máy, Xe\nMÔ TẢ: Xe máy đỏ nhìn ngang.\n"
        f"CÁCH GỌI: <{name} color /> — 300×200\n```tsx\n{code}\n```\n"
    )
