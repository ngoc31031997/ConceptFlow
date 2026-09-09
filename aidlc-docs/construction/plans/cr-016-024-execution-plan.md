# CR-016..024 — Kế hoạch thực hiện: kiểm soát style, thời lượng và chất lượng

## Date
2026-09-10

## Stage
Low-Level Design → Code Generation

## Phạm vi
8 CR đã duyệt (CR-016, 017, 018, 019, 020, 021, 023, 024). CR-022 hoãn.

Thứ tự dưới đây là thứ tự **phụ thuộc kỹ thuật**, không phải thứ tự ưu tiên
nghiệp vụ: mỗi bước chỉ bắt đầu khi thứ nó cần đã tồn tại.

---

## Hợp đồng liên service — chốt trước, vì nhiều bước cùng chạm

### H1 — `conceptflow` sống ở đâu
`services/rendering/conceptflow/`, **không** phải package top-level. Lý do:
build context của image là `./services/rendering` (docker-compose.yml), nên một
thư mục ngoài đó sẽ không vào được image mà không đổi context của mọi service.
`rendering` cũng là service duy nhất chạy Manim.

Hệ quả bắt buộc: script của Creator chạy trong **subprocess có env đã lược bỏ**
(`manim_renderer.py::_run_manim` chỉ truyền PATH/HOME/CF_MARKS_PATH), nên
`from conceptflow import *` sẽ không resolve. Phải thêm `PYTHONPATH` trỏ tới thư
mục chứa package vào `safe_env`. Đây là biến duy nhất được thêm; không mở thêm
đường nào khác ra khỏi subprocess.

### H2 — Giao kèo runtime giữa script và Rendering
Kênh liên lạc duy nhất từ script ra ngoài vẫn là **file JSONL trỏ bởi
`CF_MARKS_PATH`** (đang dùng cho `_cf_mark`). Sau CR-018 nó mang nhiều loại bản
ghi hơn, phân biệt bằng khoá `kind`:

| `kind` | Lượt ghi | Nội dung | CR |
|---|---|---|---|
| `narration` | dry | thứ tự, text | 018 |
| `beat` | dry | thứ tự, beat id | 019 |
| `chapter` | dry | thứ tự, tiêu đề | 018 |
| `mark` | thật | thứ tự, `renderer.time` | 002 (giữ nguyên) |
| `layout` | thật | bbox/màu/cỡ chữ các mobject | 021 |

Một định dạng, hai lượt, không thêm ống dẫn mới.

### H3 — Saga sau khi xong
```
parse_script -> validate_script -> [CHỜ DUYỆT] -> synthesize_speech
             -> render_scenes -> assemble_video -> qc_video
```
`classify_scenes` đổi vai thành `validate_script` (giữ vị trí, giữ tên bước
trong DB cho tới bước 4 của kế hoạch này). `qc_video` là bước thứ 7 **mới**.

### H4 — Bảng màu và font (CR-017 §Quyết định)
Nền `#080E1C` dùng chung intro/outro/thân. Nhấn: Catppuccin Mocha. Font:
Cormorant Garamond (H1/H2) + Be Vietnam Pro (còn lại) + JetBrains Mono (code).
Font phải cài vào image `rendering` và **kiểm dấu tiếng Việt** bằng chuỗi
`ỗ ự ằ ẩ ợ ữ ẵ ọ` trong test, không tin mô tả trên Google Fonts.

---

## Bước 1 — `conceptflow`: theme, base scene, components (CR-017)
Không phụ thuộc gì. Mọi bước sau đều cần.

**Thêm**
- `conceptflow/theme.py` — palette, thang cỡ chữ, font, safe margin, hằng số nhịp.
  Theme là dataclass chọn được theo tên (FR44.3), theme mới kế thừa rồi ghi đè.
- `conceptflow/scene.py` — `ConceptFlowScene(Scene)`, tự áp theme.
- `conceptflow/components/` — TitleCard, Callout, CodePanel, StepList,
  ComparisonSplit, Recap. Mỗi cái tự giữ trong safe margin (FR45.3).
- `conceptflow/transitions.py` — reveal/swap/emphasize/dismiss, run_time từ theme.
- `conceptflow/__init__.py` — bề mặt API công khai; đây là danh sách whitelist.

**Sửa**
- `Dockerfile` — cài 3 font, chạy `fc-cache`.
- `manim_renderer.py` — thêm `PYTHONPATH` vào `safe_env` (H1).
- `domain/script_lint.py` — whitelist thay blacklist (FR46.1/46.2).

**Rủi ro đã biết**: bộ component chọn sai sẽ bóp nghẹt nội dung. Bước này rút
component từ `tests/fixtures/long_form_reference.py` và `scriptTemplates.ts` —
khuôn hình **đã thực sự dùng**, không phải tưởng tượng (CR-017 §Rủi ro).

---

## Bước 2 — `narrate()` + render hai lượt + Quality Service (CR-018 + CR-020)
Làm chung một đợt: cả hai cần **cùng một lượt dry**, tách ra là viết hai lần
cùng một thứ.

**Thêm**
- `conceptflow`: `narrate()`, `beat()`, `chapter()` trên `ConceptFlowScene`.
- `rendering`: lượt dry (không encode, timeout riêng, cùng mức cách ly — FR49.3).
- `services/quality/` — đổi vai từ `content-plugin`, giữ queue/DB/Inbox/Outbox.

**Sửa**
- `manim_renderer.py` — bỏ `_patch_auto_waits` và toàn bộ nhánh thay thế chuỗi;
  duration làm tròn về bội số `1/fps` (FR49.5, cứu cache).
- `script-processing` — bỏ regex `# NARRATION`, đọc kết quả lượt dry.
- `orchestrator` — `classify_scenes` → `validate_script`, đặt trước TTS.
- Gỡ `/v1/plugins` khỏi api-gateway + web-gui.

**Không làm**: công cụ chuyển đổi script cũ (FR50 đã gỡ — không có script cũ).

---

## Bước 3 — Ước lượng thời lượng lúc soạn (CR-016)
Độc lập về kỹ thuật, nhưng đặt sau bước 2 vì editor phải đọc `self.narrate("…")`
chứ không phải `# NARRATION`.

- `web-gui/src/utils/` — ước lượng khớp `EstimateNarrationDuration` (test dùng
  chung bảng dữ liệu với Go — FR42.2).
- `tts` — bảng `voice_calibration`, ghi (số từ, duration) sau mỗi lần tổng hợp.

---

## Bước 4 — Beat sheet (CR-019)
Cần `beat()` ở bước 2.

- Format là dữ liệu, có phiên bản, nhân bản/sửa được trên GUI (FR51.5/51.6).
- Validate beat trong `quality-service`, dùng chung lượt dry.
- Component `hook`/`cta`/`recap` — khả thi vì narration đã nằm được trong hàm.
- Chuyển `# CHAPTER:` → `beat()`.

---

## Bước 5 — Cổng duyệt dàn ý (CR-024)
Cần dữ liệu lượt dry (bước 2) và beat (bước 4).

- `orchestrator` — trạng thái chờ + endpoint duyệt/từ chối, idempotent qua
  `SagaStep` sẵn có (FR69.4). **Đây là lần đầu Saga dừng chờ người** — phần rủi
  ro nhất của bước này là khôi phục sau restart, không phải giao diện.
- `web-gui` — màn dàn ý, sửa lời thoại tại chỗ (FR70), ghi ngược vào literal.

---

## Bước 6 — Intro/outro (CR-023)
Độc lập với bước 2–5. Có thể chen vào bất cứ lúc nào sau bước 1.

- Khe upload video + nhạc, chuẩn hoá theo `RENDER_QUALITY`, cache.
- Intro Manim mặc định; outro Manim (logo + ô video đề xuất + đăng ký).
- **Dịch timeline**: dùng lại đúng đường `lead_in` đang có, không mở đường thứ
  hai (bài học ADR-0027). Sửa `BuildChapterTimestamps` (`times[0] = 0` hiện sai
  khi có intro).

---

## Bước 7 — QC tự động (CR-021)
Cần layout data (H2) và `quality-service` (bước 2).

- Preamble ghi thêm bản ghi `layout`.
- Luật chấm: tràn khung, chồng lấn, chữ nhỏ, tương phản, hình chết.
- Âm thanh: đo LUFS thật, chồng lấn narration, clipping, cue chồng.
- Chạy **chế độ chỉ-báo trước**, chỉ bật cổng chặn sau khi hiệu chỉnh ngưỡng.

---

## Ghi chú về commit
Code một lượt theo thứ tự trên, chia commit theo CR khi hoàn tất (yêu cầu của
Creator, 2026-09-10). Mỗi commit mang mã CR tương ứng ở dòng đầu.
