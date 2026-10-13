# ADR-0039: Một API ảnh tĩnh chung ở rendering; animatic phát trong trình duyệt

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Nhiều tính năng mới cần ảnh tĩnh render bằng Remotion: ảnh mẫu ngôn ngữ hình, bộ mỹ thuật, khung mẫu, khung animatic, thumbnail. Rendering đã có `renderStill` cho xem trước hình (`/v1/illustrations/preview`). Animatic (FR-10) cần phát khung tĩnh theo giọng và thời lượng câu; Creator sửa nó theo từng shot (P-7 B). AD-4 và AD-5 hỏi cách làm.

## Options Considered
### Option A: một API ảnh tĩnh chung có cache
- What it is: `POST /v1/stills` nhận composition và props, trả PNG; hàng đợi và cache theo nội dung.
- Strengths: một chỗ để tối ưu; thêm loại ảnh chỉ là thêm composition.
- Trade-offs: API chung phải đủ tổng quát.

### Option B: mỗi tính năng một endpoint
- Trade-offs: lặp code hàng đợi và cache.

### Option C: animatic dựng thành video 480p bằng Remotion
- Strengths: có file video; phụ đề giống bản cuối.
- Trade-offs: mỗi lần sửa phải dựng lại; tốn CPU.

### Option D: animatic ghép bằng ffmpeg ở video-assembly
- Strengths: nhẹ.
- Trade-offs: phụ đề khác bản cuối; vẫn phải ghép lại khi sửa.

### Option E: animatic phát trong trình duyệt
- What it is: web-gui phát ảnh tĩnh từng shot, file giọng từng câu, phụ đề theo mốc thời gian.
- Strengths: không tốn lượt render video; sửa là thấy ngay.
- Trade-offs: không có file video animatic để tải về.

## Decision
A cho ảnh tĩnh (AD-4 A), E cho animatic (AD-5 C).

## Rationale
Animatic chỉ để Creator chốt nhịp, và được sửa nhiều lần. Phát trong trình duyệt bỏ hẳn bước dựng video, nên chỉ ảnh tĩnh của shot bị sửa phải render lại, và ảnh đó đi qua cùng cache chung.

## Consequences
- **Positive**: sửa thời lượng không tốn gì; ảnh tĩnh dùng chung cache giữa các tính năng.
- **Negative / Accepted Trade-offs**: api-gateway phải phát thêm ảnh và file giọng từ `/shared`; không có file animatic để chia sẻ.
