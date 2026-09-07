# Execution Plan — CR-002 … CR-007: Nâng pipeline lên chuẩn video dài 5–10 phút, bật kiếm tiền

## Date
2026-09-07

## Bối cảnh
Đánh giá toàn bộ pipeline sản xuất video (ngày 2026-09-07) cho mục tiêu: xuất video 5–10 phút, đăng YouTube + TikTok, đủ chất lượng bật kiếm tiền. Kiến trúc (saga, hexagonal, outbox/inbox) không cần thay đổi. Vấn đề nằm ở **lớp adapter sản xuất media** — nơi mọi giới hạn đang được đặt cho video demo vài chục giây.

### Phạm vi đã chốt với Creator (2026-09-07)

**Chỉ thực hiện Pha 1 (CR-002 + CR-003) ngay bây giờ**, rồi đánh giá lại. CR-004 … CR-007 vẫn được ghi đầy đủ ở đây làm backlog đã phân tích, không triển khai trong đợt này.

Lý do: trước khi CR-002/CR-003 xong, không có cách nào kiểm chứng bất kỳ cải tiến chất lượng nào — mọi video dài đều lệch tiếng và timeout giữa chừng.

Hai quyết định khác đã chốt sẵn cho các pha sau (không chặn Pha 1, nhưng đã gỡ khỏi trạng thái "đang chặn"):
- **CR-005**: engine TTS = **Google Cloud Text-to-Speech**, dùng trong hạn mức free tier; Piper giữ làm fallback. Cần **ADR-0023** khi tới Pha 2B.
- **CR-007**: hỗ trợ **cả hai preset** — `short` (≤60s, YouTube Shorts) và `long` (60–180s, TikTok Creator Rewards).

Kết quả đánh giá được ghi thành 6 Change Request:

| CR | Tên | Ưu tiên | Trạng thái |
|---|---|---|---|
| [CR-002](../../inception/requirements/cr-002-narration-timeline-sync.md) | Đồng bộ narration/phụ đề theo timeline thật | **P0 — blocker** | Chờ duyệt |
| [CR-003](../../inception/requirements/cr-003-long-form-render-capacity.md) | Năng lực render video dài | **P0 — blocker** | Chờ duyệt |
| [CR-004](../../inception/requirements/cr-004-output-video-quality.md) | 1080p60 + profile encode chuẩn YouTube | P1 | Chờ duyệt |
| [CR-005](../../inception/requirements/cr-005-audio-quality.md) | Giọng đọc chất lượng cao, ducking, loudnorm | P1 | Chờ duyệt — engine **đã chốt: Google Cloud TTS (free tier)** |
| [CR-006](../../inception/requirements/cr-006-retention-and-seo.md) | Chapters, thumbnail, hook, metadata | P2 | Chờ duyệt |
| [CR-007](../../inception/requirements/cr-007-vertical-shorts-tiktok.md) | Clip dọc 9:16 cho Shorts/TikTok | P1 | Chờ duyệt — **đã chốt: 2 preset (short ≤60s + long 60–180s)** |

## Phụ thuộc giữa các CR

```
CR-003 (năng lực render)  ──┬─────────────────────────────┐
                            │                             │
CR-002 (sync timeline) ─────┼──> CR-005 (ducking cần      │
        │                   │      track narration đúng)   │
        │                   │                             │
        │                   └──> CR-004 (1080p60 cần       │
        │                          timeout/RAM đã nới)     │
        │                                    │             │
        ├──> CR-006 (chapters cần offset)    │             │
        │                                    │             │
        └──> CR-007 (cắt clip cần offset) <──┴─────────────┘
```

Quy tắc: **CR-002 và CR-003 phải xong trước mọi CR khác.** Trước khi hai CR này hoàn tất, pipeline không sản xuất được video dài đúng — mọi cải tiến chất lượng khác đều không kiểm chứng được.

---

## Pha 0 — Baseline & benchmark (bắt buộc, trước khi code)

Mục đích: có số liệu thật trên máy của Creator để chốt các giá trị timeout/RAM của CR-003, thay vì đoán.

- [ ] 0.1 — Viết 1 script Manim tham chiếu ~5 phút (≥15 narration, xen kẽ `self.play` dài) làm **fixture test chung cho tất cả CR**.
- [ ] 0.2 — Chạy render script này với cấu hình hiện tại, đo: thời gian render, RAM đỉnh, dung lượng đĩa, ở cả `-qm` và `-qh`.
- [ ] 0.3 — Ghi lại độ lệch audio/hình thực tế ở narration cuối (bằng chứng số cho CR-002).
- [ ] 0.4 — Chốt giá trị `RENDER_TIMEOUT_SECONDS`, `RENDER_MEMORY_LIMIT_GB`, `ASSEMBLY_TIMEOUT_SECONDS` từ số đo (× hệ số an toàn 2).

**Đầu ra**: `aidlc-docs/construction/build-and-test/long-form-baseline.md` + fixture script.

---

## Pha 1 — Sửa blocker (CR-003 rồi CR-002)

Làm CR-003 trước vì nếu không nới giới hạn thì **không thể test được CR-002** trên video dài.

### 1A — CR-003 phần "nới giới hạn" (nhanh, ít rủi ro)
- [ ] 1A.1 — `docker-compose.yml`: `RENDER_TIMEOUT_SECONDS`, `ASSEMBLY_TIMEOUT_SECONDS`, `RENDER_MEMORY_LIMIT_GB` theo Pha 0.4.
- [ ] 1A.2 — `services/rendering/adapters/rendering/manim_renderer.py`: đọc memory limit từ env; sửa `_limit_child_resources` (bỏ `RLIMIT_CPU` hoặc tách khỏi wall-clock) — FR11.2/FR11.3.
- [ ] 1A.3 — `infra/rabbitmq/rabbitmq.conf`: đặt `consumer_timeout` đủ lớn (mặc định RabbitMQ là 30 phút và **sẽ giết consumer giữa lúc render** — CR-003 §C2). Đây là bug tiềm ẩn chưa từng lộ ra.
- [ ] 1A.4 — `docker-compose.yml`: `deploy.resources.limits` cho `rendering` + `video-assembly` — FR11.6.
- [ ] 1A.5 — **Verify**: render fixture Pha 0 ở `-qm` chạy hết không timeout, RabbitMQ không redeliver.

### 1B — CR-002 (lõi, rủi ro cao nhất)
- [ ] 1B.1 — **Spike kỹ thuật trước**: xác nhận `scene.renderer.time` cho đúng offset trên đúng version Manim đang pin (CR-002 §C2). Nếu không, tìm cơ chế khác trước khi thiết kế tiếp.
- [ ] 1B.2 — `manim_renderer.py`: chèn preamble `_cf_mark` + đổi substitution thành `(_cf_mark(self, i), self.wait(D))`; đọc `marks.jsonl`; `ffprobe` lấy thời lượng thật — FR10.1.
- [ ] 1B.3 — `rendering/domain/models.py` + `application/render_script.py`: `ScriptRenderResult` mang `wait_offsets` + `video_duration_seconds` — FR3.5.
- [ ] 1B.4 — `rendering/adapters/messaging/producer.py`: `rendering_completed_envelope` mang 2 trường mới — FR10.2. **Cập nhật `interface-contracts.md` của Unit 5** (bài học từ đợt sửa bug 2026-09-05: contract drift giữa unit là nguồn bug nghiêm trọng nhất của dự án này).
- [ ] 1B.5 — Orchestrator: lưu `WaitOffsets` trên `domain.Project` + cột Postgres mới; validate số offset = số scene (FR10.5); `subtitleCues()` dùng offset (FR10.4); payload `assemble_video` đổi sang `narration_segments: [{audio_path, start_time}]` (FR5.5). **Lưu ý migration**: `db.go` chỉ `CREATE TABLE IF NOT EXISTS`, volume dev hiện có cần `ALTER TABLE` thủ công (đã gặp với `category_hint`).
- [ ] 1B.6 — `video-assembly`: `domain/models.py` + `ffmpeg_assembler.py` — thay `concat` bằng `adelay=<ms>|<ms>` mỗi input rồi `amix=inputs=N:normalize=0` (bắt buộc `normalize=0`, CR-002 §C5); `tpad` cho FR10.6; bỏ `-shortest` khỏi nhánh này.
- [ ] 1B.7 — **Verify (tiêu chí nghiệm thu CR-002)**: fixture Pha 0 → lệch < 200ms ở narration cuối, cả 2 chế độ bật/tắt TTS; phụ đề khớp; câu cuối không cụt.

### 1C — CR-003 phần "trải nghiệm render dài"
- [ ] 1C.1 — Progress định kỳ khi render: đọc stdout Manim, publish progress ≥ mỗi 15s — FR11.4.
- [ ] 1C.2 — Bật Manim cache tuỳ chọn + cache dir bền trên volume — FR11.5.
- [ ] 1C.3 — Web GUI hiển thị % render.
- [ ] 1C.4 — Dọn artifact tạm (CR-003 §C3).
- [ ] 1C.5 — (Hoãn) FR11.7 render per-scene — đánh giá lại sau Pha 1; chỉ làm nếu thời gian render vẫn là nút thắt.

**Mốc hoàn thành Pha 1**: render được video 8 phút, tiếng/hình khớp, không timeout. Đây là điểm đầu tiên hệ thống thực sự dùng được cho mục tiêu đề ra.

---

## Pha 2 — Chất lượng nghe/nhìn (CR-004 + CR-005)

Làm song song được vì chạm hai khâu khác nhau (hình vs tiếng), trừ phần trộn âm của CR-005 phụ thuộc CR-002 đã xong ở Pha 1.

### 2A — CR-004 (hình)
- [ ] 2A.1 — `RENDER_QUALITY` env → cờ `-qm`/`-qh`/`-qk` trong `manim_renderer.py::_run_manim` — FR12.1.
- [ ] 2A.2 — Profile encode chuẩn upload trong `ffmpeg_assembler.py` — FR12.2.
- [ ] 2A.3 — Giữ `-c:v copy` khi không cần vẽ đè — FR12.3.
- [ ] 2A.4 — `subtitle_file.py` lấy PlayRes từ độ phân giải thật — FR12.5.
- [ ] 2A.5 — Preset chất lượng ở GUI (Nháp/Chuẩn/Cao) truyền qua saga — FR12.6.
- [ ] 2A.6 — **Verify**: `ffprobe` xác nhận 1080p60/yuv420p/High/aac 48k + faststart; upload thử private.
- [ ] (Chốt sau) FR12.4 — nơi burn phụ đề (Manim / ffmpeg / soft-sub): quyết ở Low-Level Design.

### 2B — CR-005 (tiếng)
- [ ] 2B.0 — **CHẶN**: Creator chốt engine TTS (xem "Quyết định cần Creator" bên dưới) → viết **ADR mới** nếu chọn engine cloud (phá vỡ nguyên tắc local-only ở README).
- [ ] 2B.1 — Adapter TTS mới qua `TTSEnginePort`; `voice_registry` thêm trường `engine`; sinh file nghe thử — FR13.1/13.2/13.4.
- [ ] 2B.2 — Fallback về Piper khi thiếu key/lỗi mạng, có cảnh báo ở GUI — FR13.3.
- [ ] 2B.3 — `sidechaincompress` ducking + volume nhạc chỉnh được — FR14.1/14.2.
- [ ] 2B.4 — `loudnorm=I=-14:TP=-1.5:LRA=11` — FR14.3.
- [ ] 2B.5 — Padding đầu/cuối — FR14.5.
- [ ] 2B.6 — **Verify**: `ffmpeg -af ebur128` cho −14 ±1 LUFS; nghe kiểm ducking.

---

## Pha 3 — Phân phối & giữ chân (CR-006 + CR-007)

### 3A — CR-006
- [ ] 3A.1 — Marker `# CHAPTER: "..."` trong Script Processing (đề xuất, thay vì để LLM tự gom) — CR-006 câu hỏi mở 1.
- [ ] 3A.2 — Sinh chapters từ offset (CR-002) + ghép mô tả 4 phần — FR15/FR18.2.
- [ ] 3A.3 — Thumbnail tự động: trích frame + overlay tiêu đề (cần font tiếng Việt trong image) — FR16.
- [ ] 3A.4 — Template hook/end screen — FR17. **Cẩn thận**: chèn scene làm lệch số đếm `self.wait(AUTO)` trong `_patch_auto_waits` (CR-006 §C2).
- [ ] 3A.5 — `OLLAMA_MODEL` cấu hình được — FR18.1.

### 3B — CR-007
- [ ] 3B.0 — **CHẶN**: Creator chốt ưu tiên Shorts (≤60s) hay TikTok Rewards (>60s).
- [ ] 3B.1 — Marker `# CLIP:` / `# ENDCLIP` trong Script Processing — FR19.2.
- [ ] 3B.2 — Bước sinh clip dọc trong Video Assembly (`scale`+`boxblur`+`overlay`) — FR19.3.
- [ ] 3B.3 — Phụ đề khổ dọc, chữ to, căn giữa — FR19.4.
- [ ] 3B.4 — Bước saga `generate_clips` + tải clip từ GUI — FR20.1.
- [ ] 3B.5 — (Hoãn) Adapter TikTok API — FR20.2, chỉ làm nếu Creator muốn qua quy trình duyệt app của TikTok.

---

## Quy trình AI-DLC áp dụng

Theo tinh thần Adaptive Workflow và tiền lệ CR-001 (1 file LLD chung thay vì 1 LLD/unit):

| CR | Cần stage nào |
|---|---|
| CR-002 | Low-Level Design chung (1 file) — thay đổi contract xuyên 3 unit, **bắt buộc** |
| CR-003 | Không cần LLD — chỉnh config + 1 hàm; đi thẳng Code Generation |
| CR-004 | LLD ngắn (chốt FR12.4) |
| CR-005 | **ADR mới** (engine TTS) + LLD |
| CR-006 | LLD ngắn |
| CR-007 | LLD (bước saga mới) |

**Bắt buộc cho mọi CR**: cập nhật `interface-contracts.md` của từng unit bị đổi payload, và chạy integration test xuyên unit — 4 trong 5 bug nghiêm trọng của dự án tính tới nay đều là contract drift giữa các unit đã được duyệt riêng lẻ (xem `audit.md`, 2026-09-05).

## Quyết định

### Đã chốt (2026-09-07)
| # | Quyết định | Ảnh hưởng |
|---|---|---|
| 1 | Phạm vi đợt này: **chỉ Pha 1 (CR-002 + CR-003)** | Pha 2/3 thành backlog đã phân tích |
| 2 | Engine TTS: **Google Cloud TTS**, trong hạn mức free tier, Piper làm fallback | Pha 2B; cần ADR-0023 |
| 3 | Clip dọc: **2 preset** `short` (≤60s) + `long` (60–180s) | Pha 3B |

### Còn mở — quyết khi tới pha tương ứng, không chặn Pha 1
| # | Câu hỏi | Quyết ở đâu |
|---|---|---|
| 4 | CR-003 FR11.7 — render per-scene: làm hay hoãn? | Đánh giá lại ở bước 1C.5, sau khi có số đo thật của Pha 0 |
| 5 | CR-004 FR12.4 — burn phụ đề ở Manim / ffmpeg / soft-sub? | Low-Level Design của CR-004 |
| 6 | CR-005 — hạng giọng Google (Standard/WaveNet/Neural2) và hành vi khi vượt quota | Low-Level Design của CR-005 |
| 7 | CR-006 — cách gom chapter (marker `# CHAPTER:` vs LLM tự gom) | Low-Level Design của CR-006 |
| 8 | CR-007 — có làm adapter TikTok API không? | Low-Level Design của CR-007 |

## Thứ tự thực hiện

**Đợt này**: Pha 0 → 1A → 1B → 1C.
**Sau đó, đánh giá lại rồi mới quyết**: 2A ∥ 2B → 3A → 3B.

Trong Pha 1, nếu chỉ làm được một việc: **1B (CR-002)** — không sửa nó thì mọi thứ khác vô nghĩa vì video dài luôn lệch tiếng.

## Bước tiếp theo ngay
1. Chạy **Pha 0** (benchmark) để có số đo thật → chốt giá trị timeout/RAM cho 1A.
2. Viết **Low-Level Design cho CR-002** (bắt buộc — đổi contract xuyên 3 unit), bắt đầu bằng spike 1B.1 xác minh `scene.renderer.time`.
3. CR-003 không cần LLD, đi thẳng Code Generation sau khi có số Pha 0.
