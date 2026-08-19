# Domain Entities — Unit 5: Rendering Service

## `SceneRenderRequest` (value object)
| Field | Type | Description |
|---|---|---|
| `project_id` | `str` | Zero-trust validated: không rỗng |
| `scene_index` | `int` | Zero-trust validated: không âm |
| `narration_text` | `str` | Zero-trust validated: không rỗng (dù đã validate ở Script Processing Service) |
| `illustration_hint` | `str \| None` | Không validate thêm (thông tin tham khảo, không quyết định logic render) |
| `code_snippet` | `str \| None` | Không validate nội dung (chỉ hiển thị nguyên văn) |
| `code_language` | `str \| None` | Không validate — nếu không xác định được lexer, fallback plain text (Business Rule 4) |
| `animation_template_id` | `str` | Zero-trust validated: phải có trong `AnimationTemplateRegistry` |
| `audio_path` | `str` | Zero-trust validated: không rỗng |
| `duration_seconds` | `float` | Zero-trust validated: phải > 0 |

## `SceneRenderResult` (value object)
| Field | Type | Description |
|---|---|---|
| `animation_path` | `str` | Đường dẫn file `.mp4` trong shared volume |
| `duration_seconds` | `float` | Thời lượng thực tế của animation (có thể lớn hơn `duration_seconds` input — Business Rule 2) |

Giữ nguyên như Low-Level Design — không bổ sung field debug/log (Question 5).

## Relationships
`SceneRenderRequest` → (qua `RenderSceneUseCase` + `AnimationRendererPort` + `AnimationTemplateRegistry`) → `SceneRenderResult`. Không có aggregate hay entity có lifecycle riêng — hoàn toàn stateless ngoài file animation trên shared volume.
