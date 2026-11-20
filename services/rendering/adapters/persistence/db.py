"""PostgreSQL connection pool + schema bootstrap (ADR-0013).

Bootstraps the Outbox/Inbox tables with a plain CREATE TABLE IF NOT
EXISTS at startup rather than a migration tool — appropriate at this
project's MVP scale (see ADR-0013's "Follow-ups").
"""

from __future__ import annotations

import os

import asyncpg

DATABASE_URL = os.environ["DATABASE_URL"]

SCHEMA = """
CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_events_unpublished
    ON outbox_events (created_at) WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS illustration_folders (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0,
    is_system BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS illustrations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    folder_id TEXT NOT NULL REFERENCES illustration_folders (id),
    tags TEXT[] NOT NULL DEFAULT '{}',
    description TEXT NOT NULL DEFAULT '',
    usage TEXT NOT NULL DEFAULT '',
    code TEXT NOT NULL DEFAULT '',
    builtin BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'draft',
    version INTEGER NOT NULL DEFAULT 1,
    kind TEXT NOT NULL DEFAULT 'figure',
    detail_level TEXT NOT NULL DEFAULT 'medium',
    ref_id TEXT NOT NULL DEFAULT '',
    exemplar BOOLEAN NOT NULL DEFAULT false,
    source_id TEXT REFERENCES illustrations (id) ON DELETE SET NULL,
    home_folder_id TEXT REFERENCES illustration_folders (id) ON DELETE SET NULL,
    warnings JSONB NOT NULL DEFAULT '[]',
    preview_version INTEGER NOT NULL DEFAULT 0,
    preview_png_path TEXT NOT NULL DEFAULT '',
    preview_gif_path TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT illustrations_name_key UNIQUE (name)
);

CREATE INDEX IF NOT EXISTS idx_illustrations_folder ON illustrations (folder_id);

-- Tài sản của một dự án ở bước Storyboard; base_illustration_id và library_illustration_id trỏ tới
-- illustrations.id (chuỗi rỗng = không có).
CREATE TABLE IF NOT EXISTS project_assets (
    project_id TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    concept_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'figure',
    tier TEXT NOT NULL DEFAULT 'support',
    portrayal TEXT NOT NULL DEFAULT '',
    must_not_imply TEXT[] NOT NULL DEFAULT '{}',
    palette_tokens TEXT[] NOT NULL DEFAULT '{}',
    expressions TEXT[] NOT NULL DEFAULT '{}',
    base_illustration_id TEXT NOT NULL DEFAULT '',
    code TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 0,
    preview_path TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'planned',
    to_library BOOLEAN NOT NULL DEFAULT true,
    library_folder_id TEXT NOT NULL DEFAULT '',
    library_illustration_id TEXT NOT NULL DEFAULT '',
    warnings JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, asset_id)
);

-- Các unit kế hoạch hình (VP-…) mà tài sản xuất hiện; thêm cho bảng đã có từ trước.
ALTER TABLE project_assets ADD COLUMN IF NOT EXISTS units TEXT[] NOT NULL DEFAULT '{}';

-- Bộ đếm mã tài sản theo (dự án, tên viết hoa): chỉ tăng, nên mã đã cấp không bao giờ được dùng lại.
CREATE TABLE IF NOT EXISTS project_asset_counters (
    project_id TEXT NOT NULL,
    name_key TEXT NOT NULL,
    last_number INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, name_key)
);

-- Khung đã vẽ gần nhất của mỗi shot Storyboard: frame_path tương đối so với /shared/{project_id}, rỗng (NULL)
-- khi lần vẽ gần nhất lỗi (error giữ thông báo); spec_hash do người gọi tính, frame_hash là SHA-256 của PNG.
CREATE TABLE IF NOT EXISTS storyboard_shots (
    project_id TEXT NOT NULL,
    shot_id TEXT NOT NULL,
    unit_id TEXT NOT NULL,
    beat_id TEXT NOT NULL,
    frame_path TEXT,
    frame_hash TEXT NOT NULL DEFAULT '',
    spec_hash TEXT NOT NULL,
    drawn_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (project_id, shot_id)
);

-- Các bản chốt Storyboard: chỉ thêm, không sửa; sequence_version là số bản chốt bước 10 mà bản này dựa vào.
CREATE TABLE IF NOT EXISTS storyboard_versions (
    project_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    document JSONB NOT NULL,
    sequence_version INTEGER NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    locked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, version)
);
"""


async def create_pool() -> asyncpg.Pool:
    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(SCHEMA)
    return pool
