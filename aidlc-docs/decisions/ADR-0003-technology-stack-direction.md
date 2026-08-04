# ADR-0003: Technology Stack Direction (Python/FastAPI Backend + React Frontend)

## Status
Accepted

## Date
2026-08-04

## Stage
High-Level Design

## Context
Manim (thư viện animation lõi) là Python. Cần chọn hướng công nghệ cho GUI và giao tiếp GUI-backend.

## Options Considered
### Option A: Backend Python (FastAPI) + Frontend Web (React), giao tiếp REST/SSE
- What it is: Backend expose local HTTP API bằng FastAPI, frontend React chạy trong container riêng, giao tiếp qua REST cho cấu hình và SSE cho tiến trình render.
- Strengths: FastAPI nhẹ, phù hợp expose API cục bộ; SSE hỗ trợ tốt hiển thị tiến trình render real-time; hệ sinh thái React mature cho GUI phức tạp (soạn script, preview video).
- Trade-offs: Cần đóng gói cả Node.js build tooling cho frontend trong Docker image, tăng độ phức tạp build.

### Option B: Desktop app Python thuần (PyQt/PySide hoặc Tkinter)
- What it is: Không cần web server, không cần frontend riêng, toàn bộ là 1 ứng dụng desktop Python.
- Strengths: Đơn giản hóa Docker packaging (không cần Node.js/build frontend), toàn bộ là Python.
- Trade-offs: Hiển thị GUI từ trong Docker container ra máy host phức tạp hơn nhiều (cần X11 forwarding/VNC trên macOS/Linux), trải nghiệm preview video kém hơn web.

## Decision
Chọn **Option A**: Backend Python/FastAPI (cho mỗi microservice) + Frontend React, giao tiếp REST (cấu hình) + Server-Sent Events (tiến trình render — điều chỉnh từ đề xuất ban đầu WebSocket sang SSE theo yêu cầu người dùng, vì luồng tiến trình render là một chiều server→client).

## Rationale
Web GUI tránh được vấn đề hiển thị GUI desktop từ trong Docker container ra máy host (X11/VNC phức tạp trên macOS — môi trường làm việc của người dùng), và cho trải nghiệm preview video tốt hơn qua trình duyệt. SSE được chọn thay WebSocket vì luồng tiến trình render chỉ cần server đẩy dữ liệu một chiều, không cần kênh hai chiều phức tạp của WebSocket.

## Consequences
- **Positive**: Trải nghiệm GUI phong phú qua trình duyệt; SSE đơn giản hơn WebSocket để implement và debug cho use case một chiều.
- **Negative / Accepted Trade-offs**: Docker image cần cả Python runtime và Node.js build tooling cho frontend, tăng kích thước/độ phức tạp build so với thuần Python.
- **Follow-ups**: Application Design cần định nghĩa cụ thể REST endpoint và SSE event schema giữa GUI và API Gateway.

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/technology-direction.md`
- Related ADRs: ADR-0001, ADR-0005
