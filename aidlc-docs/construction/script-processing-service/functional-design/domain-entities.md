# Domain Entities — Unit 4: Script Processing Service

## `Scene` (value object)
| Field | Type | Description |
|---|---|---|
| `scene_index` | `int` | 0-based, theo thứ tự xuất hiện thực tế của heading `## Scene N` trong script |
| `narration_text` | `str` | Lời thoại — bắt buộc không rỗng (Business Rule) |
| `illustration_hint` | `str \| None` | Gợi ý minh họa (từ dòng blockquote `>`) — optional (Functional Design Question 2) |
| `code_snippet` | `str \| None` | Nội dung code fence (nếu có) — tối đa 1 code fence/scene (Question 3) |

## `ParsedScript` (value object)
| Field | Type | Description |
|---|---|---|
| `scenes` | `list[Scene]` | Danh sách scene theo thứ tự xuất hiện |

Không lưu `raw_script` gốc — script gốc thuộc trách nhiệm GUI/Orchestrator (Question 5), Script Processing Service hoàn toàn stateless.

## Relationships
`raw_script: str` → (qua `MarkdownScriptParser` implement `ScriptParserPort`) → `ParsedScript(scenes: list[Scene])`. Không có entity nào khác — không có aggregate, không có entity với lifecycle riêng.
