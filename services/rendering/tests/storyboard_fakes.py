"""Bản giả trong bộ nhớ của kho Storyboard và bộ dựng khung, dùng cho test use case và route."""

from __future__ import annotations

from dataclasses import replace

from adapters.rendering.illustration_previewer import IllustrationPreviewError
from domain.storyboard import ShotFrame, StoryboardVersion, VersionConflictError

PNG = b"\x89PNG\r\n\x1a\n" + b"fake-frame"


class FakeStoryboardRepo:
    def __init__(self) -> None:
        self.shots: dict[tuple[str, str], ShotFrame] = {}
        self.versions: dict[tuple[str, int], StoryboardVersion] = {}
        self.deleted: list[str] = []

    async def get_shot(self, project_id, shot_id):
        return self.shots.get((project_id, shot_id))

    async def upsert_shot(self, shot):
        row = replace(shot, drawn_at="2026-11-19T00:00:00Z")
        self.shots[(shot.project_id, shot.shot_id)] = row
        return row

    async def next_version(self, project_id):
        return max([v for (p, v) in self.versions if p == project_id], default=0) + 1

    async def insert_version(self, version):
        if (version.project_id, version.version) in self.versions:
            raise VersionConflictError(version.version)
        row = replace(version, locked_at="2026-11-19T00:00:00Z")
        self.versions[(version.project_id, version.version)] = row
        return row

    async def latest_version(self, project_id):
        numbers = [v for (p, v) in self.versions if p == project_id]
        return self.versions[(project_id, max(numbers))] if numbers else None

    async def get_version(self, project_id, version):
        return self.versions.get((project_id, version))

    async def delete_project(self, project_id):
        self.deleted.append(project_id)
        self.shots = {k: v for k, v in self.shots.items() if k[0] != project_id}
        self.versions = {k: v for k, v in self.versions.items() if k[0] != project_id}


class FakeFrameRenderer:
    """Ghi một PNG giả ra `output`, nhớ các spec đã nhận; `error` đặt thì như bộ dựng hỏng."""

    def __init__(self) -> None:
        self.specs: list[dict] = []
        self.error: str = ""

    def frame(self, spec, output):
        self.specs.append(spec)
        if self.error:
            raise IllustrationPreviewError(self.error)
        with open(output, "wb") as handle:
            handle.write(PNG)
