# CR-103 · Tìm và thay một từ trong lời thoại (bước 6)

Ngày: 2026-10-26 · Nhánh: `feature/cr-103-script-find-replace` · Không chồng chéo với CR nào đang mở (CR-102 đã vào `main`).

## Yêu cầu của Creator
Khi đổi một từ trong lời thoại (ví dụ `TRNG` thành `TI A EN GI`), hệ thống đổi luôn ở các câu còn lại, thay vì phải sửa từng câu.

## Quyết định đã chốt (2026-10-26)
| Câu hỏi | Trả lời |
|---|---|
| Phạm vi | Chỉ bước 6 (lời thoại) |
| Cách bật | Hộp "Tìm và thay" riêng ở đầu bước: nhập từ cũ, từ mới, xem trước các câu sẽ đổi, rồi xác nhận |
| Câu của beat đã duyệt | Bỏ qua; beat đã duyệt giữ nguyên, hộp xem trước nói rõ số câu bị bỏ qua |
| Đổi vào đâu (đề xuất, Creator có thể đổi trong hộp) | Mặc định "Lời đọc" (chỉ đổi cách đọc, chữ hiển thị giữ nguyên); chọn "Chữ hiển thị" để đổi cả chữ trên màn hình |

## Yêu cầu chức năng
- **FR-1** `POST /v1/projects/{id}/script/replace` với `find`, `replace`, `field` (`spoken` hoặc `text`), `match_case`, `apply`.
- **FR-2** Khớp theo nguyên từ (ký tự liền kề không phải chữ hoặc số), phân biệt hoa thường theo `match_case` (mặc định có).
- **FR-3** `apply = false` chỉ trả danh sách câu sẽ đổi (trước, sau) và số câu của beat đã duyệt bị bỏ qua; không ghi gì.
- **FR-4** `apply = true` đổi mọi câu khớp ở beat chưa duyệt, mỗi beat đổi xong về chờ duyệt như một lần sửa tay, ghi một lần cho tất cả; có câu nào không hợp lệ sau khi đổi thì không đổi gì và báo câu đó.
- **FR-5** Với `field = spoken`, câu chưa có lời đọc riêng mà chữ hiển thị khớp thì lời đọc của câu đó được đặt bằng chữ hiển thị đã thay; chữ hiển thị không đổi.
- **FR-6** Hộp ở web-gui: hai ô, chọn đổi vào đâu, nút "Xem trước", danh sách trước/sau, nút "Đổi N câu"; sau khi đổi, trạng thái bước được tải lại.

## Kiểm thử
Go: khớp nguyên từ, hoa thường, `spoken` và `text`, bỏ qua beat đã duyệt, không ghi khi xem trước, không ghi gì khi một câu không hợp lệ. web-gui: hộp xem trước và xác nhận.
