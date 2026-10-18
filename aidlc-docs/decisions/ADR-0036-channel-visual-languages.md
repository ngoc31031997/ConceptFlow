# ADR-0036: Ngôn ngữ hình của kênh: luật ở DB có phiên bản, rig là kit Remotion, một ngôn ngữ bật

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Luật style hình hiện là một bộ duy nhất trong DB (CR-067). Creator muốn nhiều ngôn ngữ hình, chỉ một ngôn ngữ bật tại một thời điểm, đổi được sau này, và muốn so trước khi chốt (P-3, P-4). Một ngôn ngữ gồm cả luật (chữ) lẫn cách vẽ người và con vật (code rig). AD-3 hỏi cách lưu.

## Options Considered
### Option A: luật ở DB, rig là kit Remotion build sẵn
- What it is: authoring-service lưu luật, mô tả, `kit_key`, phiên bản; rendering có một kit cho mỗi ngôn ngữ hệ thống; ngôn ngữ Creator sao chép dùng lại kit của gốc.
- Strengths: code rig qua kiểm kiểu và test lúc build; luật sửa được trên giao diện.
- Trade-offs: ngôn ngữ có cách vẽ rig khác hẳn cần một CR để viết kit mới.

### Option B: lưu cả code rig dạng chữ trong DB
- What it is: rendering nạp code rig lúc chạy.
- Strengths: thêm ngôn ngữ không cần build lại.
- Trade-offs: code không được kiểm lúc build; lỗi chỉ lộ khi render.

## Decision
A (AD-3 A). Bốn ngôn ngữ hệ thống: Sách tranh tươi, Khoa học phát sáng, Giấy cắt lớp, Đường nét tối giản. Tên và luật không nhắc tên kênh (giữ quyết định của CR-066/CR-067). Project ghim id và phiên bản ngôn ngữ khi tạo.

## Rationale
Rig là code chạy trong mọi khung hình của video; lỗi ở đây làm hỏng render. Giữ nó trong project Remotion cho nó đi qua cùng tsc và test với các kit hiện có. Luật thì cần Creator sửa được thường xuyên, nên ở DB như prompt.

## Consequences
- **Positive**: đổi ngôn ngữ không làm hỏng video đang dở; so được bốn ngôn ngữ trên cùng hai cảnh thử.
- **Negative / Accepted Trade-offs**: bốn kit phải bảo trì song song; thư viện hình mới phải đọc rõ theo ngôn ngữ đang bật.

## Implementation note (U1)
U1 giao một ngôn ngữ hệ thống (Khoa học phát sáng, kit `phatsang`); ba ngôn ngữ còn lại chưa có kit. Màu vào qua `Scene.light`, ảnh mẫu qua `StillRenderService` (`/shared/stills`), hạng hình (chính/phụ/phông) và mức chi tiết (Kỹ/Vừa/Bóng) lưu theo project và thư viện.
