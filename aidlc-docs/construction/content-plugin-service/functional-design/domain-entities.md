# Domain Entities — Unit 2: Content Plugin Service

## Scene (Value Object)
| Field | Type | Constraint |
|---|---|---|
| `scene_index` | int | >= 0 |
| `narration_text` | str | **bắt buộc, không rỗng** |
| `illustration_hint` | str | optional |
| `code_snippet` | str \| None | optional (chỉ khi scene minh họa code — Story B3) |
| `category_hint` | str | **bắt buộc, không rỗng, phải khớp 1 trong `supported_categories` của plugin** |

## ClassificationResult (Value Object)
| Field | Type | Constraint |
|---|---|---|
| `category` | str | = `category_hint` đã validate (theo Question 1: A) |
| `animation_template_id` | str | suy ra từ mapping tĩnh (xem `business-rules.md`) |

## Plugin (Entity, qua ContentPluginPort)
| Field | Type | Constraint |
|---|---|---|
| `plugin_id` | str | unique trong registry |
| `name` | str | tên hiển thị (vd. "Lập trình") |
| `supported_categories` | list[str] | vd. `["algorithm", "concept"]` cho ProgrammingPlugin |

## Relationships
`Plugin` (1) --- classifies ---> `Scene` (n) --- produces ---> `ClassificationResult` (1 per scene). Không có quan hệ persistence (mọi entity là value object trong bộ nhớ, không lưu database — theo Question 6, Low-Level Design).
