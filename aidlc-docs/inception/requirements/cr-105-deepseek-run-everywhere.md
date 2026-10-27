# CR-105 · Nút "Chạy bằng DeepSeek" ngay cạnh mọi nút AI ngoài (bước 1–7)

## Yêu cầu
Ở mọi chỗ gọi AI ngoài từ bước 1 đến bước 7 có một nút "Chạy bằng DeepSeek" đặt **ngay cạnh** nút "AI ngoài" (giống bước 3). Bấm một lần là chạy: không mở hộp thoại, extension AI Bridge chạy prompt trên DeepSeek, hệ thống tự đọc câu trả lời rồi cập nhật màn hình.

## Phạm vi (mọi nơi đang mở `ExternalAiDialog`)
| Nơi | Cách gắn nút |
|---|---|
| `ExternalAiButton` (thẻ Ý tưởng, Brief, Câu chuyện, Beat, Kịch bản, Lock Review, Lock Handoff) | Nút DeepSeek nằm trong chính component này, nên mọi thẻ đều có |
| Bước 1 · tạo dự án (`ScriptStepPage`, "Tạo dự án + AI ngoài") | Thêm "Tạo dự án + DeepSeek": tạo dự án xong thì chạy DeepSeek luôn |
| Bước 3 · sinh câu hỏi (`QuestionsCard`) | Thêm nút DeepSeek cạnh nút AI ngoài; vẫn hỏi "giữ hay thay câu của tôi" như hai đường còn lại |
| Form làm lại kèm ghi chú (Ý tưởng, Brief, Unit) | Thêm nút DeepSeek ở chân form, cạnh "AI ngoài", gửi kèm ghi chú |

Màn nghiên cứu chính của bước 3 (`ResearchStepPage`) đã có sẵn nút nên giữ nguyên. Nút DeepSeek trong hộp thoại AI ngoài cũng giữ lại.

## Thiết kế
- **Hook `useDeepSeekRun(run, onDone, modes)`**: chạy `run({mode:"prompt"})` để lấy prompt → `useAiBridge().run("deepseek", …)` → `run({mode:"paste", reply})` → `onDone`. Hook trả về `available`, `stage`, `start`, `cancel`, `error` và `failedReply` (câu trả lời mà hệ thống không đọc được).
- **Component `DeepSeekRunButton`**: là nút `run` "Chạy bằng DeepSeek". Khi đang chạy thì đổi thành `AiBridgeRunStatus` (trạng thái kèm nút Huỷ). Nếu không có extension thì không hiện nút, để các thẻ không bị rối; hộp thoại AI ngoài sẽ có thêm dòng hướng dẫn cài extension như bước 3.
- **Không giấu lỗi**:
  - Lỗi từ extension (chưa đăng nhập, hết giờ…) hiện ngay dưới nút.
  - DeepSeek trả lời nhưng hệ thống không đọc được: mở hộp thoại AI ngoài, dán sẵn câu trả lời và báo lỗi, để bạn sửa rồi bấm "Đọc kết quả". Muốn vậy `ExternalAiDialog` cần thêm `initialReply` và `initialError`.
- **Chế độ DeepSeek**: mặc định bật suy nghĩ sâu, không tra web (giống `ExternalAiDialog`). Mỗi lúc chỉ chạy một lượt; trong lúc chạy thì nút AI ngoài và nút của hệ thống trên cùng thẻ bị khoá.
- **Không đổi backend hay contract**: vẫn dùng chế độ `prompt` và `paste` sẵn có (ai-bridge-v1).

## Bỏ tự động review ở bước 1 và 2
Hiện `useProjectIdea.generate` và `useProjectBrief.generate` tự gọi review bằng AI hệ thống sau khi AI hệ thống viết hoặc làm lại. Bỏ bước này: AI viết xong là dừng. Creator tự chọn cách đánh giá ở ô đánh giá: "Nhờ AI đánh giá", "AI ngoài" hoặc "Chạy bằng DeepSeek". Đánh giá cũ (nếu có) vẫn hiện kèm dấu đã cũ (`reviewStale`), và không chặn bước duyệt. Sửa lại các test đang chờ review tự chạy.

## Extension AI Bridge (1.3.0)
Sửa sau lần thử thật, đổi một phần quyết định của CR-104 (tab DeepSeek mở ở nền và được giữ lại):
- Tab DeepSeek mở ở chế độ đang xem (`active: true`) vì Chrome làm chậm tab ẩn, khiến câu trả lời xong rồi mà ConceptFlow đọc muộn. Lấy được kết quả thì extension đưa Creator về tab ConceptFlow và đóng tab DeepSeek. Khi có lỗi hoặc Creator bấm Huỷ thì tab được giữ lại để xem.
- Chờ câu trả lời: kiểm tra ngay khi trang đổi (`MutationObserver`, tối đa mỗi 300ms), ngoài nhịp 1 giây. Khi không thấy hàng nút dưới câu trả lời, chờ nội dung đứng yên 20 giây thay vì 60 giây.
- Nút gửi: luôn lấy nút thật bao quanh phần tử khớp (`button` hoặc `role="button"`), không lấy div con trang trí như `ds-button__background`.

## Kiểm thử
Unit test với `fakeAiBridge` cho các trường hợp:
- có extension và không có extension
- chạy thành công thì gọi `onDone`
- extension báo lỗi
- câu trả lời không đọc được thì mở hộp thoại kèm câu trả lời
- huỷ giữa chừng

Mỗi nhóm nơi gắn nút có một test cho thấy nút hiện ra. Không gọi AI thật.
