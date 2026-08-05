# Tech Stack Decisions — Unit 3: TTS Service

## Language/Runtime: Python 3.12
- **Rationale**: Ràng buộc kỹ thuật cứng — Piper (và Coqui TTS nếu thêm sau, ADR-0010) là thư viện Python-first, không có binding chính thức chất lượng cho ngôn ngữ khác. Khớp ADR-0009 (polyglot có chọn lọc).

## Framework: FastAPI
- **Ecosystem**: Mature, hỗ trợ async tốt (kết hợp threadpool cho CPU-bound work — xem `nfr-requirements.md` Performance), tích hợp Pydantic cho validation, OpenAPI docs tự động.
- **Performance**: Route handler CPU-bound (Piper synthesis) chạy qua threadpool để không block event loop — pattern chuẩn của FastAPI cho blocking I/O/CPU work.
- **Team familiarity**: Đã dùng cho Unit 2 (Content Plugin Service), tái sử dụng kinh nghiệm.
- **Maintenance**: Cộng đồng lớn, release cadence ổn định.
- **Licensing**: MIT, không có chi phí.

## TTS Engine: Piper (ADR-0010)
- Xem ADR-0010 cho phân tích lựa chọn engine (Piper vs Coqui vs cả hai).

## Testing
- `pytest` cho unit test (domain/application layer, dùng fake `TTSEnginePort` implementation theo `dependency-injection.md`).
- `pytest-asyncio` không bắt buộc (route handler CPU-bound chạy đồng bộ qua threadpool) — dùng `TestClient` (đồng bộ) của FastAPI cho test API layer.

Không cần ADR riêng cho framework/testing — hệ quả trực tiếp của ADR-0003/ADR-0009, không có trade-off cạnh tranh đáng kể ở mức chi tiết này (đã có ADR-0010 cho quyết định TTS engine, là quyết định trade-off thực sự của unit này).
