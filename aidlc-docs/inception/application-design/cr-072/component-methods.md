# CR-072 — Phương thức và giao diện

Chữ ký ở mức cao: tên, đầu vào, đầu ra. Luật nghiệp vụ, schema chi tiết và mã lỗi để Functional Design của từng CR con. Route của trình duyệt đi qua api-gateway như hiện nay. Việc AI chạy lâu trả `operation_id` và theo dõi qua `/v1/operations/{id}` đã có.

Đổi hợp đồng giữa hai service theo ba bước (thêm bản mới, giữ bản cũ, bỏ bản cũ ở lần sau), theo `docs/contracts/README.md`.

## authoring-service

### VisualLanguageStore (U1)

```go
type VisualLanguageStore interface {
    List(ctx) ([]VisualLanguage, error)
    Get(ctx, id string) (VisualLanguage, error)
    Active(ctx) (VisualLanguage, error)                    // đúng một ngôn ngữ bật
    Activate(ctx, id string) error                          // tắt ngôn ngữ đang bật trong cùng giao dịch
    Copy(ctx, id, name string) (VisualLanguage, error)      // bản Creator sửa được, giữ kit_key của gốc
    UpdateRules(ctx, id string, r LanguageRules) (version int, error) // từ chối ngôn ngữ hệ thống
    PinForProject(ctx, projectID string) (LanguagePin, error)        // ghim id + phiên bản khi tạo project
}
// VisualLanguage: id, name, reference_note, description, kit_key, rules (illustration, backdrop), version, is_system, active
```

HTTP: `GET /v1/admin/visual-languages`, `GET /v1/admin/visual-languages/{id}`, `POST /{id}/activate`, `POST /{id}/copy`, `PUT /{id}`, `POST /{id}/samples` (dựng lại ảnh mẫu, trả `operation_id`).

### ReviewUnitStore, DependencyTracker, RedoService (U2)

```go
type UnitRef struct{ Kind, Key string }      // ví dụ {"board_scene", "3"}

type ReviewUnitStore interface {
    List(ctx, projectID string, kinds []string) ([]Unit, error)
    Approve(ctx, projectID string, ref UnitRef) error
    Unapprove(ctx, projectID string, ref UnitRef) error
    ApproveRemaining(ctx, projectID string, kinds []string) (int, error)
    SaveManualEdit(ctx, projectID string, ref UnitRef, content json.RawMessage) (Unit, error) // nguồn = creator
    AddNote(ctx, projectID string, ref UnitRef, text string) (Note, error)
    ClearNotes(ctx, projectID string, refs []UnitRef) error
}

type DependencyTracker interface {
    MarkBuilt(ctx, projectID string, ref UnitRef, inputs []UnitRef) error // ghi dấu đầu vào lúc dựng
    Stale(ctx, projectID string) ([]UnitRef, error)
    Impact(ctx, projectID string, ref UnitRef) ([]UnitRef, error)        // "sẽ làm lại" nếu ref đổi
}

type RedoService interface {
    Redo(ctx, projectID string, ref UnitRef, note string) (operationID string, error)
    RedoFromNotes(ctx, projectID string, scope []UnitRef) (operationID string, error) // gom ghi chú
}
// Mỗi loại đơn vị đăng ký một UnitGenerator: Generate(ctx, project, ref, note, neighbours) (content, inputs, error)
```

HTTP:
- `GET /v1/projects/{id}/units?kind=board_scene,voice_line`
- `POST /v1/projects/{id}/units/{kind}/{key}/approve`, `/unapprove`, `/notes`, `/redo` (`{note}`)
- `PUT /v1/projects/{id}/units/{kind}/{key}` (sửa tay)
- `POST /v1/projects/{id}/units/approve-remaining` (`{kinds}`)
- `POST /v1/projects/{id}/units/redo-notes` (`{scope}`)
- `GET /v1/projects/{id}/units/{kind}/{key}/impact`

### ScriptAuthoring, ArchetypeAdvisor (U3)

```go
GenerateLogline(ctx, projectID string) (operationID, error)          // bỏ qua nếu Creator đã viết
GenerateOutline(ctx, projectID string) (operationID, error)          // theo logline đã duyệt
GenerateSceneLines(ctx, projectID string, scenes []string) (operationID, error)
SuggestArchetypes(ctx, topic, formatID string) ([]ArchetypeSuggestion, error) // 3–5 gợi ý, có thể 1 kiểu mới
SaveProposedArchetype(ctx, s ArchetypeSuggestion) (VideoArchetype, error)
```

HTTP: `POST /v1/archetype-suggestions` (`{topic, format_id}`), `POST /v1/admin/video-archetypes` (đã có, dùng để lưu kiểu mới).

### ArtBible, CharacterLibrary, StoryboardStore (U4, U5)

```go
GenerateArtPass(ctx, projectID string, pass int) (operationID, error)  // 1: thế giới, bối cảnh, nhân vật, đạo cụ; 2: color key, khung mẫu
ArtItems(ctx, projectID string, pass int) ([]Unit, error)
AddMoodboardUpload(ctx, projectID, filePath string) (Unit, error)
ReplaceAssetSVG(ctx, projectID string, ref UnitRef, svgPath string) (Unit, error)
ReplaceAssetFromLibrary(ctx, projectID string, ref UnitRef, libraryID string) (Unit, error)
SaveCharacterToLibrary(ctx, projectID string, ref UnitRef) (Character, error)
ListLibraryCharacters(ctx) ([]Character, error)
GenerateStoryboard(ctx, projectID string, scenes []string) (operationID, error) // theo cảnh
```

HTTP:
- `POST /v1/projects/{id}/art/{pass}/generate`
- `GET /v1/projects/{id}/art/{pass}`
- `POST /v1/projects/{id}/art/moodboard` (file qua api-gateway)
- `PUT /v1/projects/{id}/art/assets/{kind}/{key}/svg`, `.../from-library`
- `GET /v1/admin/characters`, `POST /v1/projects/{id}/art/characters/{key}/save-to-library`
- `POST /v1/projects/{id}/authoring/storyboard/generate` (đổi: nhận danh sách cảnh)

authoring-service → llm-service: `POST /v1/storyboard/finalize` mở rộng schema (chỉ thêm trường; bên nhận cũ bỏ qua được). authoring-service → rendering: `POST /v1/stills` (dưới).

### PronunciationDictionary, VoiceLines, AnimaticPlan (U6)

```go
type PronunciationDictionary interface {
    List(ctx) ([]Pronunciation, error)
    Upsert(ctx, word, sayAs string) error
    Delete(ctx, word string) error
    Resolve(ctx, text string) (sayAs string, error) // thay từ theo từ điển, trả cách đọc cho SSML
}
VoiceRequestFor(ctx, projectID string, lines []UnitRef) ([]LineToSynthesize, error) // đã tra từ điển, kèm tốc độ, nghỉ
RecordSynthesized(ctx, projectID string, results []LineResult) error              // do orchestrator gọi
UpdateLine(ctx, projectID string, ref UnitRef, text, sayAs string, speed float64, pauseBefore, pauseAfter time.Duration) (Unit, error)
AnimaticPlanFor(ctx, projectID string) (AnimaticPlan, error) // shot: still_url, audio_url, start, duration, line
SetShotDuration(ctx, projectID string, ref UnitRef, delta time.Duration) error
ProposePacing(ctx, projectID, sceneKey, note string) (operationID, error)
ApplyPacing(ctx, projectID, proposalID string) error
```

HTTP: `GET/PUT/DELETE /v1/admin/pronunciations`, `PUT /v1/projects/{id}/units/voice_line/{key}`, `GET /v1/projects/{id}/animatic`, `PUT /v1/projects/{id}/animatic/shots/{key}/duration`, `POST /v1/projects/{id}/animatic/scenes/{key}/pacing`, `POST /v1/projects/{id}/animatic/pacing/{proposal}/apply`.

Nội bộ, từ orchestrator: `GET /internal/v1/authoring/{id}/voice-lines?pending=1`, `POST /internal/v1/authoring/{id}/voice-lines/results`.

### CodeSegments, ValidationFindings, ReviewSummary (U7)

```go
GenerateScene(ctx, projectID, sceneKey string) (operationID, error)      // đoạn = cảnh
RerunShot(ctx, projectID string, ref UnitRef, note string) (operationID, error) // không ghi chú = chạy lại như cũ
ShotPreview(ctx, projectID string, ref UnitRef) (operationID, error)     // gọi rendering ShotPreview
Findings(ctx, projectID string) ([]Finding, error)                       // mỗi finding có UnitRef, rule, severity
FixFinding(ctx, projectID, findingID string) (operationID, error)        // AI sửa đúng đơn vị đó
IgnoreWarning(ctx, projectID, findingID string) error
Summary(ctx, projectID string) (ReviewSummary, error)
```

authoring-service → llm-service: `POST /v2/code/generate` nhận `segment_kind = "scene"`; trả chẩn đoán theo shot; không còn vòng sửa bằng AI (thêm trường, giữ hành vi cũ cho bên gọi cũ tới khi bỏ).

### EditPlanner (U8), ThumbnailComposer (U9)

```go
PlanFromNotes(ctx, projectID, sceneKey string, notes []TimedNote) (EditPlan, error) // mỗi ghi chú: tầng sửa + đơn vị bị làm lại
ApplyEditPlan(ctx, projectID, planID string) (operationID, error)
ComposeThumbnails(ctx, projectID, title string) ([]ThumbnailOption, error) // 3 phương án, ảnh từ StillRenderService
ExternalThumbnailPrompt(ctx, projectID string) (string, error)             // theo ngôn ngữ hình, thế giới, bảng màu
```

Nội bộ, từ orchestrator: `POST /internal/v1/authoring/{id}/edit-plan`, `POST /internal/v1/authoring/{id}/edit-plan/{plan}/apply`. Từ trình duyệt: `POST /v1/projects/{id}/thumbnails` (`{title}`), `GET /v1/projects/{id}/thumbnails/external-prompt`.

## orchestrator

```go
// FlowDefinition (U2)
FlowStateFor(status ProjectStatus, wizardStep int, content AuthoredContent) FlowState // 17 bước; AuthoredContent thêm cờ cho từng chặng mới
PhaseOf(step int) Phase                                                             // Tiền kỳ, Sản xuất, Hậu kỳ

// DraftVoiceDispatcher (U6)
DispatchLines(ctx, projectID string) error           // lấy câu chờ đọc từ authoring-service, gửi lệnh synthesize_lines
OnLinesSynthesized(ctx, evt LinesSynthesized) error  // chuyển kết quả về authoring-service

// RenderSaga (U8)
StartRender(ctx, projectID string) error                       // một lệnh render_scene cho mỗi cảnh
OnSceneRendered(ctx, evt SceneRendered) error
OnSceneRenderFailed(ctx, evt SceneRenderFailed) error          // transient → thử lại một lần; còn lại → chờ Creator
RetryScene(ctx, projectID, sceneKey string) error
CancelScene(ctx, projectID, sceneKey string) error

// EditSession (U8)
AddTimedNote(ctx, projectID, sceneKey string, at time.Duration, text string) (TimedNote, error)
RequestPlan(ctx, projectID, sceneKey string) (EditPlan, error)  // gọi authoring-service
RunPlan(ctx, projectID, planID string) error                    // authoring làm lại đơn vị, rồi RetryScene
RenderOnly(ctx, projectID, sceneKey string) error
```

HTTP nội bộ, từ authoring-service: `POST /internal/v1/projects/{id}/draft-voice` (gọi `DispatchLines`).

HTTP (qua api-gateway): `GET /v1/projects/{id}/render/scenes`, `POST /v1/projects/{id}/render/scenes/{key}/retry`, `/cancel`, `/render-only`; `GET/POST /v1/projects/{id}/edit/scenes/{key}/notes`, `POST /v1/projects/{id}/edit/scenes/{key}/plan`, `POST /v1/projects/{id}/edit/plans/{plan}/run`; `PUT /v1/projects/{id}/edit/joins/{key}` (chuyển cảnh ở chỗ cắt).

## Lệnh và sự kiện RabbitMQ

| Lệnh / sự kiện | Gửi → nhận | Nội dung chính | CR con |
|---|---|---|---|
| `synthesize_lines` (mới) | orchestrator → tts | `project_id`, `voice`, `lines[{line_id, text, say_as, speed, pause_before, pause_after}]` | U6 |
| `lines_synthesized` (mới) | tts → orchestrator | `results[{line_id, path, duration, cache_hit}]`, lỗi theo câu | U6 |
| `synthesize_speech` (giữ) | orchestrator → tts | Bước TTS của saga; dùng lại file theo nội dung và cách đọc | U6 |
| `render_scene` (mới) | orchestrator → rendering | `project_id`, `scene_key`, `code_ref`, `quality`, `handles{head, tail}`, `grade{color_key}` | U8 |
| `scene_rendered` / `scene_render_failed` (mới) | rendering → orchestrator | đường dẫn, số khung; hoặc lỗi kèm `transient`, `shot_key`, `frame` | U8 |
| `render_scenes` (giữ tới khi bỏ) | orchestrator → rendering | Lệnh cũ cả video; bỏ sau khi saga mới chạy ổn | U8 |
| `assemble_video` (đổi, thêm trường) | orchestrator → video-assembly | `scenes[{path, head, tail, join, transition}]` | U8 |
| tiến độ (`progress.fanout`) | mọi service → api-gateway | thêm `scene_key` cho render; `stage` cho merge | U8 |

## rendering

```
POST /v1/stills            {composition, props, width, height, frame?}         → {path, cache_hit}
POST /v1/previews/shot     {project_id, scene_key, shot_key, code_ref}         → {path, duration}  (480p)
GET  /v1/kits              → [{kit_key, version}]                                (để authoring-service kiểm kit_key)
```

```ts
// Remotion
getKit(kitKey: string): LanguageKit            // rig cơ sở, hình nền tảng, quy ước ánh sáng
<LanguageSample kitKey scene="fig"|"space" />
<ArtSheet kind="location"|"character"|"prop" … />
<ColorKeyFrame colorKey … /> <StyleFrame … /> <AnimaticStill … /> <Thumbnail layout title … />
<Rig keys={[…]} hold ease overshoot … />       // key pose
<Fx name="pollen-glow"|"scatter"|… color={PALETTE.x} />
```

## tts

```python
build_ssml(text: str, voice: str, say_as: str | None, speed: float, pause_before: float, pause_after: float) -> str
cache_key(text, voice, say_as, speed, pause_before, pause_after) -> str   # cùng nội dung và cách đọc thì dùng lại
```

## video-assembly

```python
assemble(scenes: list[SceneClip], narration: list[LineAudio], music: MusicSpec, out: Path, progress: StageProgress) -> AssemblyResult
check_joins(scenes: list[SceneClip]) -> list[QCFinding]      # khung cuối và khung đầu ở chỗ nối liền mạch
check_black_frames(video: Path) -> list[QCFinding]
```
