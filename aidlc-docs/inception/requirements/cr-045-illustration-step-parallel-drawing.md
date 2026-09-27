# CR-045 — Bước "Hình minh hoạ" riêng, vẽ song song, tiến độ từng hình, cảnh báo style dễ đọc

## Date
2026-09-27

## Stage
Requirements Analysis — **Creator đã duyệt 2026-09-27** (qua trao đổi trong phiên). Đã code và test; **chờ Creator duyệt kết quả trước khi merge vào `main`**.

## Bối cảnh
CR-044 chạy việc lập danh sách và vẽ hình minh hoạ ở ĐẦU bước Code (Remotion). Hệ quả Creator thấy:
- Thẻ "Code" trong chuỗi AI đứng vài phút với dòng "Đang lập danh sách và vẽ hình minh hoạ…" và thanh chạy không xác định, trong khi chưa viết dòng code nào.
- Hình AI vẽ luôn ở trạng thái nháp, nên lần chạy chuỗi đầu tiên gần như luôn dừng ở Code với một thông báo màu đỏ như lỗi.
- Tối đa 10 hình; phần dư bị bỏ mà không báo.
- Vẽ lần lượt từng hình, chậm.
- "⚠ N cảnh báo style" chỉ hiện nội dung trong tooltip gốc của trình duyệt (phải rê chuột và chờ), không dùng được trên màn cảm ứng, không có tên luật.

## Quyết định của Creator
1. Không giới hạn số hình (bỏ con số 10, không đặt con số khác).
2. Mỗi hình đang vẽ có thanh tiến độ riêng.
3. Vẽ song song 4 hình.
4. "Bỏ qua" = bỏ qua hình đó cho video (hình vẫn ở thư viện). "Xoá" = xoá hình khỏi thư viện để thư viện không bị rác.
5. Tooltip cảnh báo phải hiện khi rê chuột (không phải tooltip gốc); câu chữ kỹ thuật chấp nhận được, nhưng phải sao chép được. Bỏ cảnh báo "màu ngoài bảng màu kênh" (S9) vì đây là hình minh hoạ.
6. Bấm "⚠ N cảnh báo style" mở danh sách ngay dưới ô (trượt ra theo luật UX), dùng được trên cảm ứng.
7. Mỗi dòng: tên luật trước, chi tiết sau — "S3 · Bo tròn — dòng 14: …". Tên luật đọc thẳng từ file luật server trả về.
8. Nút "Nhờ AI sửa các cảnh báo này": mở "Vẽ lại bằng AI" với ghi chú đã điền sẵn danh sách cảnh báo.
9. Làm tab riêng cho bước Hình minh hoạ.

## Yêu cầu chức năng
- **FR1 — Bước mới `illustrations`** (chỉ Remotion), giữa Visual và Code: tab `/create/script/illustrations`. Chạy bằng AI: lập danh sách từ storyboard nếu video chưa có, rồi vẽ mọi hình `planned`/`failed`. Chạy lại không lập lại danh sách (nút "Lập lại danh sách" làm việc đó).
- **FR2 — Chuỗi AI**: tab Kịch bản chạy `story → storyboard → illustrations → code` cho Remotion (3 bước như cũ cho Manim). Còn hình chờ duyệt → chuỗi dừng ở bước Hình minh hoạ với trạng thái "chờ duyệt" (không phải lỗi) và đưa Creator sang tab đó; không còn hình chờ → chạy tiếp Code.
- **FR3 — Bước Code chỉ còn là cổng**: không tự lập danh sách hay vẽ. Chưa từng lập danh sách → 409 "hãy chạy bước Hình minh hoạ trước"; còn hình chưa duyệt/bỏ qua → 409 như CR-044. Danh sách rỗng đã lập (video không cần hình) → qua cổng. Mốc "đã lập" lưu ở `project_authoring.illustrations_planned_at`; danh sách lập trước CR-045 (có dòng, không có mốc) vẫn tính là đã lập.
- **FR4 — Không giới hạn số hình**: bỏ `maxPlannedDrawings` và câu "Tối đa 10 mục" trong prompt lập danh sách. Bước Code nhận TẤT CẢ hình của video; trần 40 chỉ còn áp cho hình lấy thêm từ phần còn lại của thư viện (giới hạn độ dài prompt).
- **FR5 — Vẽ song song**: `ILLUSTRATION_DRAW_CONCURRENCY` (mặc định 4). Một hình hỏng chỉ đánh dấu hình đó `failed`; chỉ huỷ (Dừng) mới cắt cả lượt. Nút "Vẽ N hình còn thiếu" chạy đúng bước này trên server (không còn vẽ 2 hình một lượt từ trình duyệt). Dòng kẹt ở `drawing` do khởi động lại được vẽ lại.
- **FR6 — Tiến độ**: thẻ bước trong chuỗi hiện "Đang lập danh sách hình từ storyboard" rồi "Đang vẽ 2/5 hình · 1 lỗi · 3 hình dùng lại" với thanh phần trăm thật. Mỗi ô đang vẽ có thanh riêng: "Lần 2/3 · AI đang viết code · 3,2k ký tự · 42s" / "Đang kiểm tra và dựng ảnh". Hình đang xếp hàng hiện "Đang chờ tới lượt vẽ". Tiến độ từng hình chỉ giữ trong bộ nhớ authoring-service.
- **FR7 — Xoá hình nháp**: `DELETE /v1/projects/{id}/illustrations/{row}/drawing` xoá hình khỏi thư viện và chuyển dòng sang "Bỏ qua". Chỉ hình nháp do chính video này vẽ; hình đã duyệt hoặc dùng lại từ thư viện → 409 (có thể đang phục vụ video khác). "Dùng lại" sau khi xoá → dòng về `planned` để vẽ lại.
- **FR8 — Cảnh báo style**: bỏ kiểm tra S9 ở rendering, sửa câu luật S9 thành "ưu tiên bảng màu, vật có màu riêng được dùng đúng màu", xoá cảnh báo S9 đã lưu ở mỗi lần khởi động. Ô hình: tooltip tự vẽ hiện ngay khi rê chuột/focus; bấm → danh sách trượt ra dưới ô (ô rộng gấp đôi trên màn ≥ 720px), nút "Sao chép cảnh báo" và "Nhờ AI sửa các cảnh báo này" (không có với hình có sẵn/hình mẫu). Trình sửa hình dùng cùng danh sách và cùng nút.
- **FR9 — Nhật ký**: lượt chạy bước Hình minh hoạ ghi `source = "illustrations"` dưới số bước 5 (Code) — không thêm số bước mới vào luồng 1–13 dùng chung với orchestrator.

## Phạm vi kỹ thuật
- `authoring-service`: `StepIllustrations`, `Prepare`/`DrawMissing`/`Gate`/`DeleteDrawing`, tiến độ từng hình (`DrawProgress`), trường tiến độ `drawings_*`, `ChainState.waiting`, cột `illustrations_planned_at`, `ILLUSTRATION_DRAW_CONCURRENCY`.
- `rendering`: bỏ S9.
- `api-gateway`: route DELETE mới.
- `web-gui`: trang `IllustrationsStepPage`, `StyleWarnings`, `IllustrationsGateNote`, stepper 4 bước, tiến độ từng ô.
- Không thêm service; không đổi số bước 1–13.

## Kiểm thử
- Go (`-race`): lập danh sách không bị cắt (15 hình), vẽ song song đúng 3/7 cùng lúc, tiến độ sống của một hình đang vẽ, từ chối vẽ trùng, vẽ lại dòng kẹt, cổng phân biệt "chưa lập" với "không cần hình", xoá hình nháp / từ chối hình đã duyệt và hình thư viện, bước Hình minh hoạ báo tiến độ và ghi nhật ký, chuỗi dừng chờ duyệt / chạy tiếp Code, route DELETE, mã 409.
- Postgres thật (Postgres 16 cục bộ): mốc "đã lập", xoá cảnh báo S9 đã lưu khi schema chạy lại.
- rendering: màu ngoài bảng không còn cảnh báo; các luật khác giữ nguyên.
- web-gui (vitest, 372 test): tên luật đọc từ file luật, tooltip/danh sách/sao chép/nhờ AI sửa, thanh tiến độ từng ô, nút vẽ gọi bước trên server, xoá hình nháp, trang mới (Remotion/Manim), stepper 4 bước với phần trăm, chuỗi chờ duyệt không hiện như lỗi, chuỗi Remotion 4 bước, ghi chú ở tab Code. Chụp màn hình thật (Chromium, API giả) ở 1280px và 390px.

## Chưa kiểm chứng / chưa làm
- Chưa chạy với model thật (môi trường này không có HIVE key) — thời gian vẽ song song thực tế và việc Hive có chặn vì gọi dồn (rate limit) ở 4 luồng hay không chưa đo. Rendering dựng ảnh xem trước từng ảnh một (`preview_gate`), nên phần dựng ảnh vẫn tuần tự.
- Chưa rebuild/khởi động lại Docker (môi trường này không có Docker daemon).
- Thanh 13 bước bên trái không có mục riêng cho Hình minh hoạ (tab nằm dưới bước 5 — Code); vào tab qua nút Tiếp tục ở Visual, nút Quay lại ở Code, chuỗi AI, hoặc ghi chú trên tab Code.

---

## Bản sửa đổi (CR-046, 2026-09-27) — FR9 đã bị đảo ngược

**FR9 của CR-045 (nhật ký chạy dưới số bước của Code, không thêm số bước riêng vào luồng 1–13) đã bị đảo ngược bởi CR-046, theo yêu cầu tường minh của Creator (chủ dự án).**

Lý do đảo ngược: sau khi dùng thử, Creator thấy bước Hình minh hoạ "vô hình" trên thanh bước bên trái — nằm lọt dưới bước Code khiến khó theo dõi tiến độ tổng thể và dễ nhầm là một phần của bước Code. Creator xác nhận muốn có mục riêng, có số, trên thanh bước, dù biết điều này đảo ngược thiết kế đã duyệt của CR-045.

Thay đổi cụ thể:
- Hình minh hoạ giờ là **bước 6** có số riêng trong luồng dùng chung giữa orchestrator, authoring-service và web-gui — luồng đổi từ 13 bước thành **14 bước**.
- Mọi bước từ Validate trở đi lùi lại một số: Validate 6→7, Review 7→8, TTS 8→9, Render 9→10, Merge 10→11, Cắt short 11→12, Kết quả 12→13, Publish 13→14.
- Nhật ký (`project_events`) của lượt chạy Hình minh hoạ giờ ghi thẳng `flow_step = 6`, thay vì workaround cũ (`flow_step = 5` tức số của Code, phân biệt bằng `source = "illustrations"`).
- Trên thanh bước, bước 6 hiện trạng thái "Không dùng" (bị vô hiệu hoá, không bấm được) khi `renderEngine !== "remotion"` — vì bước này chỉ áp dụng cho video Remotion.
- Xem chi tiết đầy đủ ở `aidlc-docs/inception/requirements/cr-046-illustration-step-promoted.md`.

Các phần khác của CR-045 (FR1–FR8: bước riêng trong wizard, không giới hạn số hình, vẽ song song, tiến độ từng hình, xoá hình nháp, cảnh báo style) **không đổi** — chỉ FR9 bị đảo ngược.
