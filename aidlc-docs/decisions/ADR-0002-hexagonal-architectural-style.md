# ADR-0002: Hexagonal / Ports & Adapters as Architectural Style

## Status
Accepted

## Date
2026-08-04

## Stage
High-Level Design

## Context
requirements.md NFR1 yêu cầu kiến trúc phải hỗ trợ mở rộng đa lĩnh vực (content-type plugin) và TTS provider có thể thay thế, không hard-code vào chi tiết công nghệ cụ thể.

## Options Considered
### Option A: Hexagonal / Ports & Adapters
- What it is: Logic nghiệp vụ lõi tách biệt khỏi chi tiết công nghệ qua port (interface); mỗi công nghệ cụ thể là một adapter cắm vào.
- Strengths: Khớp tự nhiên với yêu cầu pluggable content-type/TTS; dễ test logic lõi độc lập với Manim/TTS thật.
- Trade-offs: Nhiều interface/abstraction hơn, cần kỷ luật giữ ranh giới port/adapter.

### Option B: Layered / N-tier truyền thống
- What it is: Presentation → Business/Service → Data Access.
- Strengths: Quen thuộc, dễ hiểu, ít abstraction hơn.
- Trade-offs: Không tự nhiên hỗ trợ pluggable content-type/TTS bằng Hexagonal.

### Option C: Domain-Driven Design với bounded context riêng
- What it is: Mỗi domain giáo dục (Programming, English...) là một bounded context riêng.
- Strengths: Rất rõ ràng khi có nhiều domain hoạt động song song.
- Trade-offs: Over-engineering khi hiện tại chỉ có 1 domain (lập trình) được implement.

## Decision
Chọn **Option A: Hexagonal / Ports & Adapters**.

## Rationale
Đây là lựa chọn tự nhiên nhất cho ràng buộc NFR1 — port cho content-type và TTS chính là cơ chế pluggable được yêu cầu. DDD (Option C) bị đánh giá là over-engineering ở giai đoạn hiện tại vì chỉ có 1 domain giáo dục thực sự được implement (lập trình); có thể cân nhắc lại khi có ≥2 domain thực tế.

## Consequences
- **Positive**: Thêm content-type plugin mới hoặc đổi TTS provider không cần sửa domain core; domain core dễ unit test độc lập với Manim/TTS thật.
- **Negative / Accepted Trade-offs**: Cần định nghĩa và duy trì nhiều interface (port) hơn cách viết code thẳng; cần kỷ luật kiến trúc để không rò rỉ chi tiết adapter vào domain core.
- **Follow-ups**: Application Design cần định nghĩa cụ thể các port/adapter cho từng service (đặc biệt Content Plugin Service, TTS Service, Rendering Service).

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/architectural-style.md`
- Related ADRs: ADR-0001
