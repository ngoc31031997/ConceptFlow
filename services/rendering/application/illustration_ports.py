"""Cổng của thư viện hình minh hoạ: nơi lưu hàng, nơi lưu ảnh, bộ dựng và nguồn bộ thông số vẽ."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Protocol

from domain.illustration import (
    CodeFinding,
    Illustration,
    IllustrationFilter,
    IllustrationFolder,
)


@dataclass
class RenderedIllustration:
    """Kết quả dựng một hình: đạt thì có PNG và GIF, không đạt thì có chẩn đoán kèm số dòng."""

    ok: bool
    diagnostics: list[CodeFinding] = field(default_factory=list)
    #: Cảnh báo style, không chặn lưu.
    warnings: list[CodeFinding] = field(default_factory=list)
    png: bytes = b""
    gif: bytes = b""


class IllustrationRepoPort(Protocol):
    async def list_folders(self) -> list[IllustrationFolder]: ...

    async def create_folder(self, folder: IllustrationFolder) -> IllustrationFolder: ...

    async def delete_folder(self, folder_id: str) -> None: ...

    async def list_illustrations(self, flt: IllustrationFilter) -> list[Illustration]: ...

    async def get_illustration(self, illustration_id: str) -> Illustration: ...

    async def create_illustration(self, illustration: Illustration) -> Illustration: ...

    async def update_illustration(self, illustration: Illustration) -> Illustration: ...

    async def delete_illustration(self, illustration_id: str) -> None:
        """Xoá một hàng không phải hình có sẵn; hàng không có thì không phải lỗi."""

    async def create_exemplar(self, illustration: Illustration, limit: int) -> Illustration:
        """Lưu một Hình mẫu, từ chối (ExemplarLimitError, AlreadyExemplarError) dưới khoá để hai lần
        "Đặt làm mẫu" cùng lúc không cùng vượt giới hạn."""

    async def release_exemplar(self, illustration_id: str, folder_id: str) -> Illustration:
        """Đưa một Hình mẫu gốc về thành hình đã duyệt thường, xếp vào `folder_id`."""

    async def set_preview(self, illustration_id: str, version: int, png_path: str, gif_path: str) -> None: ...

    async def get_preview(self, illustration_id: str) -> tuple[str, str]:
        """Đường dẫn PNG và GIF của bản hiện tại, hoặc hai chuỗi rỗng khi chưa dựng cho phiên bản này."""


class PreviewFilesPort(Protocol):
    def save(self, illustration_id: str, version: int, png: bytes, gif: bytes) -> tuple[str, str]:
        """Ghi ảnh của một phiên bản, trả đường dẫn PNG và GIF (GIF rỗng thì đường dẫn rỗng)."""

    def read(self, path: str) -> bytes: ...

    def remove(self, illustration_id: str) -> None:
        """Xoá cả thư mục ảnh của một hình; không có thì không phải lỗi."""


class IllustrationRendererPort(Protocol):
    async def render(self, kit: dict, name: str, code: str, kind: str, tier: str) -> RenderedIllustration:
        """Kiểm tra rồi dựng `code` (rỗng = hình dựng sẵn của kit theo tên). Code sai là kết quả
        `ok=False`; bộ dựng không chạy được ném RendererUnavailableError."""


class ReferenceSourcePort(Protocol):
    """Phong cách tham chiếu ở visual-service."""

    async def get_kit(self, ref_id: str) -> dict:
        """Bộ thông số vẽ của phong cách. Ném ReferenceStyleNotFoundError, ReferenceKitMissingError
        hoặc ReferenceServiceError."""

    async def get_style(self, ref_id: str) -> dict:
        """Mô tả của phong cách: `{id, name, summary, drawing{…}, layout{…}}`."""


@dataclass
class ChatResult:
    content: str
    call_id: str = ""


class ChatPort(Protocol):
    async def chat(
        self, *, system: str, user: str, model: str, max_tokens: int, temperature: float,
        role: str, step: str, phase: str,
    ) -> ChatResult: ...


class PromptSourcePort(Protocol):
    async def get_template(self, role: str) -> str:
        """Template đang chạy của role ở llm-service (bản đang dùng, không có thì bản hệ thống)."""

    async def step_default_model(self, step: str) -> str:
        """Model mặc định của bước, rỗng khi không đọc được."""
