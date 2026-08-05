# Domain Entities — Unit 3: TTS Service

## `SpeechRequest` (value object)
| Field | Type | Description |
|---|---|---|
| `project_id` | `str` | ID dự án video, dùng để tính đường dẫn shared volume |
| `scene_index` | `int` | Vị trí scene trong project, dùng để tính đường dẫn shared volume |
| `text` | `str` | Nội dung lời thoại (`narration_text` từ script) cần synthesize |
| `language` | `Literal["vi", "en"]` | Ngôn ngữ giọng đọc |

Không bổ sung field ngoài Low-Level Design (Functional Design Question 5) — giữ entity gọn, tránh rò rỉ chi tiết implementation (voice model cụ thể) ra ngoài contract.

## `SpeechResult` (value object)
| Field | Type | Description |
|---|---|---|
| `audio_path` | `str` | Đường dẫn file `.wav` trong shared volume |
| `duration_seconds` | `float` | Thời lượng audio, đo trực tiếp từ file `.wav` (làm tròn 2 chữ số thập phân) |

## Relationships
`SpeechRequest` → (qua `SynthesizeSpeechUseCase` + `TTSEnginePort`) → `SpeechResult`. Không có entity nào khác — TTS Service là stateless, không có domain model phức tạp (không có aggregate, không có entity có identity/lifecycle riêng ngoài 2 value object trên).
