# Domain Entities — Unit 6: Video Assembly Service

## `VideoAssemblyRequest`
| Field | Type | Notes |
|---|---|---|
| `project_id` | `str` | Không rỗng (Rule 1) |
| `scenes` | `list[SceneAssemblyInput]` | Không rỗng; sort theo `scene_index` trước xử lý (Rule 2) |
| `background_music_path` | `str \| None` | Nếu có, file phải tồn tại (Rule 1) |

## `SceneAssemblyInput` (value object, **mới ở Functional Design — Revision LLD Question 2**)
| Field | Type | Notes |
|---|---|---|
| `scene_index` | `int` | ≥ 0; dãy liên tục không thiếu/trùng trên toàn `scenes` (Rule 2) |
| `clip_path` | `str` | Animation clip câm (video-only), không rỗng, phải tồn tại trên shared volume |
| `audio_path` | `str` | Audio giọng đọc, không rỗng, phải tồn tại trên shared volume |

## `VideoAssemblyResult`
| Field | Type | Notes |
|---|---|---|
| `video_path` | `str` | Đường dẫn video .mp4 cuối cùng trên shared volume — KHÔNG bổ sung field khác (Rule 6) |

## `MediaFormat` (value object nội bộ, dùng bởi `MediaFormatInspector`)
| Field | Type | Notes |
|---|---|---|
| `codec` | `str` | Từ ffprobe |
| `resolution` | `tuple[int, int]` | Từ ffprobe (width, height) |
| `framerate` | `float` | Từ ffprobe |

Không phải 1 phần của event contract — chỉ dùng nội bộ trong `adapters/assembly/ffprobe_inspector.py` để so sánh giữa các scene (Rule 3).

## Errors (Domain)
| Error | Trigger | Event kết quả |
|---|---|---|
| `MissingArtifactError` | 1 trong `clip_path`/`audio_path`/`background_music_path` không tồn tại | `assembly_failed` |
| `InvalidSceneIndexError` | `scene_index` thiếu/trùng trong dãy liên tục từ 0 | `assembly_failed` |
| `InconsistentMediaFormatError` | ffprobe pre-check phát hiện codec/resolution/framerate lệch giữa các scene | `assembly_failed` |
| `AssemblyEngineError` | ffmpeg exit code khác 0, hoặc timeout (`ASSEMBLY_TIMEOUT_SECONDS`) | `assembly_failed` |

## Relationships
```
VideoAssemblyRequest
  ├── 1..* SceneAssemblyInput (sorted by scene_index)
  └── 0..1 background_music_path

AssembleVideoUseCase
  └── produces → VideoAssemblyResult (on success)
      or raises → MissingArtifactError | InvalidSceneIndexError | InconsistentMediaFormatError | AssemblyEngineError (on failure)
```
