# CR-006 — Giữ chân người xem & metadata SEO (P2)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Điều kiện bật kiếm tiền của YouTube (1.000 subscriber + 4.000 giờ xem) phụ thuộc **watch-time** nhiều hơn chất lượng kỹ thuật. Pipeline hiện tại sản xuất được video đúng kỹ thuật nhưng thiếu toàn bộ các yếu tố kéo watch-time.

## Vấn đề (đã xác minh trong code)

1. **Không có chapters** — `services/orchestrator/internal/adapters/llm/ollama_client.go` đã sinh title/description/tags qua Ollama, nhưng không sinh timestamp chapters. Trong khi đó hệ thống **đã có sẵn** `scene_index` + duration của từng narration (và sau CR-002 là offset thật) — dựng chapters gần như miễn phí.
2. **Thumbnail hoàn toàn thủ công** — `services/web-gui/src/components/ThumbnailUpload.tsx` chỉ sinh *prompt* để Creator tự tạo ảnh ở công cụ khác rồi upload; `youtube_publisher.py` chỉ nhận `thumbnail_path` có sẵn.
3. **Không có hook / intro / end screen** — không có template nào trong pipeline. 70% người xem rời trong 15 giây đầu.
4. **Metadata sinh bởi model Ollama local nhỏ** — chất lượng tiêu đề/SEO hạn chế, không cấu hình được model mạnh hơn.
5. **Không có mô tả có cấu trúc** — không chapters, không CTA, không link.

## Functional Requirements

### FR15 — Chapters (mới)
- **FR15.1**: Hệ thống PHẢI sinh danh sách chapter (timestamp + tiêu đề ngắn) từ các mốc narration, gom nhóm hợp lý (không phải mỗi câu 1 chapter).
- **FR15.2**: Chapters PHẢI được chèn vào đầu phần mô tả YouTube đúng định dạng (`00:00 ...`, chapter đầu bắt buộc là `00:00`, tối thiểu 3 chapter, mỗi chapter ≥ 10s).
- **FR15.3**: Creator PHẢI sửa được chapters trước khi đăng.

### FR16 — Thumbnail tự động (mới)
- **FR16.1**: Hệ thống PHẢI đề xuất thumbnail tự động: trích frame đại diện từ video + overlay tiêu đề lớn.
- **FR16.2**: Creator vẫn upload được thumbnail riêng (giữ nguyên luồng hiện có).
- **FR16.3**: Thumbnail PHẢI đạt yêu cầu YouTube: 1280×720, < 2MB, JPG/PNG.

### FR17 — Hook & End screen (mới)
- **FR17.1**: PHẢI có template Manim cho intro hook (~5s) và end screen (~15–20s, chừa chỗ cho end-screen element của YouTube).
- **FR17.2**: Creator bật/tắt được từng phần; nội dung hook lấy từ tiêu đề/câu narration đầu.

### FR18 — Metadata (sửa đổi)
- **FR18.1**: Model sinh metadata PHẢI cấu hình được (`OLLAMA_MODEL` hoặc engine khác).
- **FR18.2**: Mô tả PHẢI ghép theo cấu trúc: tóm tắt → chapters → CTA → hashtag.

## Ràng buộc
- **C1**: FR15 phụ thuộc **CR-002** (cần offset thật thì timestamp chapter mới đúng).
- **C2**: FR17 chèn scene vào script của Creator ⇒ ảnh hưởng đếm `self.wait(AUTO)` trong `manim_renderer.py::_patch_auto_waits` (đang kiểm tra số lượng khớp tuyệt đối). Phải xử lý cẩn thận.
- **C3**: FR16 overlay text cần font hỗ trợ tiếng Việt trong image Rendering/Video Assembly.
- **C4**: Không có yêu cầu nào ở đây thay đổi kiến trúc — đều là bổ sung.

## Phạm vi tác động
- **Orchestrator** (Unit 8): sinh chapters, prompt Ollama, ghép mô tả.
- **Video Assembly** (Unit 6) hoặc **Rendering** (Unit 5): trích frame + overlay thumbnail.
- **Rendering** (Unit 5): template hook/end screen.
- **Web GUI** (Unit 10): sửa chapters, xem/chọn thumbnail đề xuất.
- **Publisher** (Unit 7): không đổi (đã nhận `description` + `thumbnail_path`).

## Tiêu chí nghiệm thu
1. Video đăng lên YouTube hiển thị đúng thanh chapters.
2. Thumbnail tự động sinh ra dùng được ngay (không bắt buộc phải sửa).
3. Mô tả có đủ 4 phần theo FR18.2.

## Câu hỏi còn mở
1. Cách gom narration thành chapter: theo LLM, theo số scene cố định, hay Creator tự đánh dấu trong script (vd. `# CHAPTER: "..."`)? → đề xuất marker trong script, nhất quán với `# NARRATION:`.
