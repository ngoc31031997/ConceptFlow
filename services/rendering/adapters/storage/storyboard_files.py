"""Khung và bản chốt của Storyboard trên đĩa, dưới `{root}/{project_id}/storyboard/`.

Khung đang làm là `SH-xxx.png`; bản chốt `n` giữ bản sao khung ở `v{n}/SH-xxx.png`; `storyboard.json` là tài
liệu của bản chốt mới nhất. Mọi file ghi ra file tạm rồi đổi tên nên người đọc không thấy file dở. Đường dẫn
trả về hay nhận vào tương đối so với `{root}/{project_id}`.
"""

from __future__ import annotations

import json
import os
import shutil
import uuid
from pathlib import Path

from domain.project_asset import validate_project_id
from domain.storyboard import SHOT_ID_RE, StoryboardError, frame_relative_path, version_frame_path

SHARED_ROOT = "/shared"
DOCUMENT_FILE = "storyboard.json"


class StoryboardFiles:
    def __init__(self, root: str | Path = SHARED_ROOT) -> None:
        self._root = Path(root)

    def begin_frame(self, project_id: str, shot_id: str) -> str:
        """Đường dẫn file tạm (đuôi `.png`) để bộ dựng ghi khung của `shot_id`."""
        folder = self._folder(project_id)
        folder.mkdir(parents=True, exist_ok=True)
        self._check_shot(shot_id)
        return str(folder / f".{shot_id}.{uuid.uuid4().hex}.tmp.png")

    def commit_frame(self, project_id: str, shot_id: str, tmp_path: str) -> str:
        """Đổi tên file tạm thành khung chính thức, trả đường dẫn tương đối."""
        relative = frame_relative_path(shot_id)
        os.replace(tmp_path, self._root / project_id / relative)
        return relative

    def discard(self, tmp_path: str) -> None:
        Path(tmp_path).unlink(missing_ok=True)

    def exists(self, project_id: str, relative: str) -> bool:
        return self._resolve(project_id, relative).is_file()

    def read(self, project_id: str, relative: str) -> bytes:
        return self._resolve(project_id, relative).read_bytes()

    def copy_to_version(self, project_id: str, shot_id: str, version: int) -> str:
        """Sao khung đang làm của `shot_id` sang thư mục của bản chốt `version`, trả đường dẫn tương đối."""
        source = self._resolve(project_id, frame_relative_path(shot_id))
        target_relative = version_frame_path(version, shot_id)
        target = self._resolve(project_id, target_relative)
        target.parent.mkdir(parents=True, exist_ok=True)
        tmp = target.with_name(f".{target.name}.tmp")
        shutil.copyfile(source, tmp)
        os.replace(tmp, target)
        return target_relative

    def version_exists(self, project_id: str, version: int) -> bool:
        """Đã có thư mục ảnh của bản chốt `version` chưa (bản chốt bất biến, không ghi đè lên)."""
        return (self._folder(project_id) / f"v{version}").exists()

    def remove_version(self, project_id: str, version: int) -> None:
        """Xoá thư mục ảnh của một bản chốt chưa ghi xong; không có thì không phải lỗi."""
        shutil.rmtree(self._folder(project_id) / f"v{version}", ignore_errors=True)

    def write_document(self, project_id: str, document: dict) -> None:
        """Ghi `storyboard.json` (tài liệu của bản chốt mới nhất)."""
        folder = self._folder(project_id)
        folder.mkdir(parents=True, exist_ok=True)
        target = folder / DOCUMENT_FILE
        tmp = target.with_name(f".{DOCUMENT_FILE}.tmp")
        tmp.write_text(json.dumps(document, ensure_ascii=False, indent=2), encoding="utf-8")
        os.replace(tmp, target)

    def remove_project(self, project_id: str) -> None:
        """Xoá cả thư mục Storyboard của dự án (khung, bản chốt, ảnh tài sản); không có thì không phải lỗi."""
        shutil.rmtree(self._folder(project_id), ignore_errors=True)

    def _folder(self, project_id: str) -> Path:
        validate_project_id(project_id)
        return self._root / project_id / "storyboard"

    @staticmethod
    def _check_shot(shot_id: str) -> None:
        if not SHOT_ID_RE.match(shot_id):
            raise StoryboardError(f"mã shot không hợp lệ: {shot_id!r}")

    def _resolve(self, project_id: str, relative: str) -> Path:
        validate_project_id(project_id)
        base = (self._root / project_id).resolve()
        path = (base / relative).resolve()
        if not path.is_relative_to(base):
            raise StoryboardError(f"đường dẫn nằm ngoài thư mục dự án: {relative}")
        return path
