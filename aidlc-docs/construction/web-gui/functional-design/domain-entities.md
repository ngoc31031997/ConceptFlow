# Domain Entities — Unit 10: Web GUI

## Entity: ProjectDraft (Question 4, local state trước khi submit)
Không phải entity backend — chỉ tồn tại trong `ProjectDraftContext` (React state), reset sau khi submit thành công.

| Field | Type | Ghi chú |
|---|---|---|
| `scriptContent` | `string` | Bắt buộc (Rule 1) |
| `pluginId` | `string \| null` | Bắt buộc (Rule 1) |
| `voiceLanguage` | `"vi" \| "en"` | Mặc định `"vi"` |
| `backgroundMusicPath` | `string \| null` | Optional |

### Reducer Actions
```typescript
type ProjectDraftAction =
  | { type: "SET_SCRIPT"; payload: string }
  | { type: "SET_PLUGIN"; payload: string }
  | { type: "SET_VOICE_LANGUAGE"; payload: "vi" | "en" }
  | { type: "SET_BACKGROUND_MUSIC"; payload: string | null }
  | { type: "RESET" };
```

## Entity: Project (đã định nghĩa ở `interface-contracts.md`, nhắc lại ở đây với vai trò domain entity GUI)
Nhận nguyên trạng từ Orchestrator (qua Gateway) — GUI KHÔNG sở hữu, chỉ đọc/hiển thị. Xem `interface-contracts.md` cho type đầy đủ.

## Value Object: PublishMetadata (Question 4, form E2)
| Field | Type | Ghi chú |
|---|---|---|
| `youtubeTitle` | `string` | Bắt buộc, tối đa 100 ký tự (Rule 2) |
| `description` | `string?` | Optional |
| `tags` | `string[]?` | Optional |
| `visibility` | `"public" \| "unlisted" \| "private"` | Bắt buộc chọn, mặc định `"private"` (Rule 2) |

## Value Object: ProgressState (local UI state, tổng hợp từ ProgressMessage qua thời gian)
| Field | Type | Ghi chú |
|---|---|---|
| `currentStep` | `string \| null` | Step đang xử lý gần nhất |
| `sceneIndex` | `number \| null` | Từ `ProgressMessage.scene_index` nếu có |
| `sceneTotal` | `number \| null` | Từ `ProgressMessage.scene_total` nếu có |
| `status` | `"in_progress" \| "completed" \| "failed"` | Trạng thái message gần nhất |
| `errorMessage` | `string \| null` | Set khi `status = "failed"` |

## Quan hệ
```
ProjectDraft (local, trước submit) --[POST /v1/sagas/render]--> Project (backend, sau khi có project_id)
Project --[SSE progress.fanout]--> ProgressState (local, tổng hợp real-time)
Project + PublishMetadata (local form) --[POST /v1/sagas/publish]--> Project (cập nhật youtube_video_url)
```
