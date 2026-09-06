# Low-Level Design — CR-001: TTS/Subtitle Toggles, Voice Selection, Subtitle Style

Cross-cutting design cho CR-001 (đã duyệt: `aidlc-docs/inception/requirements/cr-001-audio-subtitle-options.md`). Ghi chung 1 file thay vì 1 LLD riêng/unit vì thay đổi chủ yếu là mở rộng field trên payload đã có + 2 module mới nhỏ (duration estimator, subtitle burn-in) — theo tinh thần Adaptive Workflow (không tạo tài liệu nặng hơn mức cần thiết).

## 1. Data Contract mới (xuyên suốt saga)

```
RenderInput (mở rộng, mọi field mới có default để tương thích ngược — C4):
  project_id: string
  script_content: string
  voice_language: "vi" | "en"
  tts_enabled: bool = true
  voice_id: string | null            # null hợp lệ khi tts_enabled=false; bắt buộc khi true
  subtitles_enabled: bool = false
  subtitle_style: SubtitleStyle | null  # null -> dùng default ở Video Assembly khi subtitles_enabled=true
  background_music_path: string | null

SubtitleStyle:
  font_size: "small" | "medium" | "large" = "medium"
  text_color: string (hex, vd "#FFFFFF") = "#FFFFFF"
  background_opacity: float 0.0-1.0 = 0.6   # 0 = không nền
  position: "bottom" | "top" = "bottom"

Voice (danh mục tĩnh, TTS Service /v1/voices):
  voice_id: string        # vd "vi_VN-vais1000-medium"
  language: "vi" | "en"
  gender: "female" | "male"
  quality: "x_low" | "low" | "medium" | "high"
  sample_audio_url: string   # file mẫu tĩnh sinh sẵn lúc build
```

**4 giọng chốt** (CR-001 §"Danh sách giọng CHỐT"): `vi_VN-vais1000-medium` (vi/nữ), `vi_VN-vivos-x_low` (vi/nam), `en_US-lessac-medium` (en/nữ), `en_US-ryan-high` (en/nam).

## 2. Web GUI (Unit 10)

- `ProjectDraftContext`: thêm `ttsEnabled`, `voiceId`, `subtitlesEnabled`, `subtitleStyle` + action tương ứng (`SET_TTS_ENABLED`, `SET_VOICE_ID`, `SET_SUBTITLES_ENABLED`, `SET_SUBTITLE_STYLE`).
- `VoiceLanguageSelector` → đổi tên/mở rộng thành `VoiceSelector`: khi `ttsEnabled=false`, ẩn hoàn toàn phần chọn giọng (chỉ hiện toggle). Khi `true`: hiện 2 nút toggle bật/tắt TTS + subtitle ở đầu, danh sách giọng lọc theo `voiceLanguage` (2 giọng/ngôn ngữ), mỗi giọng có nút ▶ Nghe thử (phát `sample_audio_url`, dùng `<audio>` HTML, không autoplay), card giọng đang chọn có viền nhấn.
- `SubtitleStylePanel` (component mới): hiện khi `subtitlesEnabled=true` — 4 control (cỡ chữ, màu chữ color picker, độ mờ nền slider, vị trí) + khung preview tĩnh (div mô phỏng khung hình 16:9 với text mẫu áp style hiện tại, cập nhật realtime — không cần render thật).
- `client.ts`: `RenderInput` type mở rộng theo mục 1; `startRenderSaga` gửi nguyên object (API Gateway/Orchestrator forward xuống Postgres).
- Danh sách voice: gọi `GET /v1/voices` qua Gateway lúc mount `NewProjectPage` (không hardcode ở FE — TTS Service là nguồn sự thật).

## 3. API Gateway (Unit 9)

- `POST /v1/sagas/render`: forward thêm 4 field mới trong body xuống Orchestrator, không validate business logic (giữ nguyên vai trò pass-through hiện có).
- Thêm route mới `GET /v1/voices` → proxy `GET {TTS_SERVICE_URL}/v1/voices` (TTS Service cần lộ REST endpoint này ngoài AMQP — xem mục 4).

## 4. TTS Service (Unit 3)

- `voice_registry.py`: đổi từ map `language -> path` sang map `voice_id -> VoiceMetadata{path, language, gender, quality, sample_path}`. Hàm `get_voice_model_path(voice_id)` thay `get_voice_model_path(language)`.
- **REST mới** (ngoài AMQP consumer hiện có — ADR-0014 không đổi, chỉ thêm 1 endpoint đọc-only cho catalog): `GET /v1/voices` trả `Voice[]` tĩnh từ registry, không chạm domain/application.
- **Sample audio**: script `scripts/generate_voice_samples.py` chạy 1 lần ở build/Docker image (không phải mỗi request) — với mỗi voice, gọi `PiperTTSAdapter.synthesize("Xin chào, đây là giọng đọc mẫu." | "Hello, this is a sample voice.", voice_id, /app/voice_samples/{voice_id}.wav)`. `sample_audio_url` trỏ tới static file này, phục vụ qua Nginx/static mount (nhất quán cách Web GUI phục vụ asset tĩnh khác — kiểm tra `infrastructure-design.md` của Web GUI khi implement).
- `synthesize_speech` (AMQP consumer): payload `scenes[]` thêm field `voice_id` (thay `language` dùng để chọn model — `language` vẫn giữ để log/hiển thị). Khi `tts_enabled=false`, **Orchestrator không gửi command này** (mục 6) — TTS Service không cần biết khái niệm tắt/bật.
- Docker image: tải 4 file `.onnx` (+ `.onnx.json`) tương ứng 4 voice_id đã chốt vào `/app/voices/`, thay vì 2 file hiện tại.

## 5. Script Processing Service (Unit 4)

- Module mới `application/estimate_duration.py`: `estimate_narration_duration(text: str, language: str) -> float` — tốc độ mặc định 150 từ/phút (en), 140 từ/phút (vi) (CR-001, không cho chỉnh — giữ nội bộ, không lộ ra API/GUI).
- `parse_script` output (`script_parsed` event) không đổi cấu trúc — vẫn `scenes[]` với `narration_text`. Việc estimate duration chỉ dùng ở Rendering (mục 6), không cần Script Processing tính trước — đặt hàm này ở Script Processing vì đây là service sở hữu logic xử lý text narration (cùng chỗ với `manim_script_parser.py`), Rendering import qua shared lib **hoặc** duplicate hàm nhỏ (10 dòng) nếu 2 service không share package — quyết định cụ thể khi Code Generation, tuỳ cấu trúc monorepo hiện có (kiểm tra có shared Python package chung chưa trước khi quyết).

## 6. Orchestrator Service (Unit 8)

- `Project` domain struct: thêm `TTSEnabled bool`, `VoiceID string`, `SubtitlesEnabled bool`, `SubtitleStyle SubtitleStyle` (JSON column, giống cách `scenes` đã lưu JSON).
- `StartRenderSagaUseCase`: đọc `tts_enabled` từ request.
  - Nếu **true**: giữ nguyên flow hiện tại (`parse_script` → `classify_scenes` → `synthesize_speech` → `render_scenes` → `assemble_video`), command `synthesize_speech` mang thêm `voice_id`.
  - Nếu **false**: **bỏ qua bước `synthesize_speech`** — sau `scenes_classified`, dispatch thẳng `render_scenes` với mỗi scene có `audio_path: null, duration_seconds: <estimate_narration_duration(narration_text, language)>`. `HandleStepEventUseCase` cần nhánh rẽ theo `Project.TTSEnabled` khi quyết định command kế tiếp sau `scenes_classified` (state machine hiện có nhánh tuyến tính — đây là điểm thay đổi lớn nhất, cần review kỹ khi code).
- `assemble_video` command: thêm `subtitles_enabled`, `subtitle_style`, và mỗi scene thêm `narration_text` (để Video Assembly burn-in) — hiện `assemble_video` chỉ nhận `clip_path`/`audio_path`.
- `ProjectStatus` enum: không cần thêm state mới — nhánh rẽ là trong cùng 1 transition, không phải trạng thái riêng.

## 7. Rendering Service (Unit 6)

- `manim_renderer.py`: `AUTO_WAIT_RE` substitution hiện dùng `duration_seconds` từ mỗi scene bất kể nguồn gốc (TTS thật hay estimate) — **không cần đổi code**, vì Orchestrator đã chuẩn hoá `duration_seconds` ở cả 2 nhánh (mục 6). Đây là điểm thiết kế quan trọng: Rendering Service không cần biết TTS bật/tắt — chỉ cần `duration_seconds` luôn có giá trị hợp lệ.
- `render_scenes` input: `audio_path` optional (null khi tắt TTS) — Rendering không dùng field này (chỉ Video Assembly dùng), nên không ảnh hưởng.

## 8. Video Assembly Service (Unit 7)

- `ffmpeg_assembler.py`: 2 thay đổi độc lập:
  1. **Audio track optional**: khi mọi scene có `audio_path=null`, ghép chỉ video+nhạc nền (nếu có) hoặc video câm hoàn toàn — nhánh ffmpeg command hiện có cho "no background music" có thể tái dùng làm mẫu cho "no narration audio" (cùng dạng optional-input).
  2. **Subtitle burn-in** (mới): khi `subtitles_enabled=true`, sinh file `.srt`/`.ass` từ `scenes[].narration_text` + `duration_seconds` (cumulative timing), dùng ffmpeg filter `subtitles=` hoặc `drawtext` áp `subtitle_style` (màu, cỡ, nền, vị trí) — cần chọn `.ass` nếu muốn style linh hoạt (font/màu/nền per-file) thay vì `.srt` (không style được qua filter `subtitles=` một cách đơn giản). Quyết định cụ thể format (`.ass` khuyến nghị) khi Code Generation.
- `domain/models.py`: `AssemblyRequest` thêm `subtitles_enabled`, `subtitle_style`, mỗi scene thêm `narration_text`.

## 9. Rủi ro / cần xác nhận khi code

1. **Orchestrator state machine rẽ nhánh**: đây là thay đổi rủi ro nhất (Go, saga hiện tuyến tính 5 bước cố định). Cần viết test cho cả 2 nhánh (`tts_enabled=true/false`) ở `handle_step_event_test.go`.
2. **Subtitle timing khi tắt TTS**: ước lượng theo tốc độ đọc trung bình có thể lệch với animation thực tế nếu Manim scene có khoảng dừng dài — chấp nhận cho MVP (đã ghi trong CR-001), không block.
3. **`.ass` subtitle style**: cần xác nhận ffmpeg trong Video Assembly Docker image đã có `libass` (thường có sẵn trong ffmpeg full build) trước khi code — kiểm tra `Dockerfile` hiện tại của Unit 7.

## Phạm vi KHÔNG làm trong CR-001
- Không thêm engine TTS khác (viXTTS/Coqui) — C2 trong CR-001.
- Không cho chỉnh tốc độ đọc ước lượng.
- Không hỗ trợ import phụ đề từ file ngoài.
