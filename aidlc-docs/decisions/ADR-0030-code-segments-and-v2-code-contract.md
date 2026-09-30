# ADR-0030: Lưu từng đoạn của bước Code trong authoring-service, llm-service chạy không trạng thái (contract `/v2/code/*`)

## Status
Accepted

## Date
2026-09-30

## Stage
Application Design (CR-050 Unit 2: FR-1..FR-9, FR-21, FR-22)

## Context
Bước Code (CR-039) do `llm-service` chạy trọn một lượt HTTP dài 1–2,5 giờ: khung (LAYOUT/cast), các đoạn N shot chạy song song, ghép, kiểm, sửa. Hiện trạng:
- Một đoạn lỗi làm cả bước thất bại. Các đoạn đã trả tiền chỉ nằm trong `ChunkCache` trong RAM của `llm-service` (`app/pipeline/run.py:157`), nên restart là mất.
- Khoá cache có cả prompt hệ thống. Prompt hệ thống lại chứa danh sách hình thư viện (`library_section`), nên duyệt thêm một hình là mất hết cache.
- `authoring-service` chỉ ghi `llm_usage` khi cả lượt kết thúc (`generate_authoring_code.go:92-96`). Lượt bị ngắt giữa chừng thì mất dấu các lượt đã tính tiền.
- Chẩn đoán kiểm biên dịch/bố cục chỉ nằm trong log.

CR-050 yêu cầu: lưu từng đoạn trong DB, đoạn lỗi không làm hỏng đoạn khác, chạy lại hoặc dán kết quả AI ngoài cho từng đoạn, và chỉ bỏ đoạn đã xong khi prompt hoặc đầu vào của đúng đoạn đó đổi (không bỏ khi đổi model).

## Các phương án
1. **`llm-service` vẫn điều phối cả bước, không giữ trạng thái; `authoring-service` là nơi lưu.** Mỗi lượt, `authoring-service` gửi kèm các đoạn đã xong. `llm-service` tính dấu vân tay, bỏ qua đoạn khớp, chạy đoạn còn thiếu và stream kết quả từng đoạn cùng từng lượt gọi. `authoring-service` ghi ngay khi nhận.
2. **`authoring-service` điều phối, `llm-service` chỉ cung cấp thao tác đơn lẻ** (sinh khung, sinh một đoạn, ghép-kiểm-sửa). Phải viết lại bằng Go phần song song, chia đôi khi hết budget, kiểm sớm từng đoạn và vòng sửa đang có bằng Python. Số request giữa hai service tăng, và hai nơi cùng phải hiểu định dạng shot.
3. Giữ `llm-service` có trạng thái (cache ra đĩa hoặc Redis). Vẫn là hộp đen với `authoring-service`, không có giao diện từng đoạn, dán AI ngoài phải đi vòng qua `llm-service`, và vẫn phải xoá cache bằng tay.

## Decision
Chọn **phương án 1**.

- **`authoring-service` sở hữu bảng `authoring_segments`**: mỗi đoạn có khoá, loại, danh sách shot, trạng thái, nội dung cuối, nguồn, dấu vân tay, lỗi và thời gian. Mọi thao tác của Creator (xem, chạy lại, sao chép prompt, dán) đi qua `authoring-service`.
- **`llm-service` không còn trạng thái.** Bỏ `ChunkCache`. Việc chia đoạn và tính dấu vân tay vẫn ở `llm-service`, vì chỉ nó hiểu storyboard, prompt đoạn và định dạng shot. `authoring-service` coi dấu vân tay là chuỗi mờ, chỉ so bằng nhau.
- **Dấu vân tay** không chứa model (C2) và không chứa danh sách hình thư viện:
  - khung = `sha256(engine, prompt, storyboard)`;
  - khung Remotion lấy từ storyboard = `sha256(layout)`;
  - đoạn = `sha256(engine, prompt, nội dung khung, JSON các shot của đoạn)`.
  - `prompt` là prompt `*_engineer_ai` đã render, chưa ghép phần hình thư viện.
- **Contract mới, có version**: `POST /v2/code/generate` (NDJSON), `POST /v2/code/segment-prompt`, `POST /v2/code/segment-parse`. Chi tiết ở `docs/contracts/authoring-llm-code-v2.md`. `authoring-service` là consumer duy nhất và hai service được build cùng nhau, nên `/v1/code/generate` được **xoá** trong cùng CR, không giữ song song.
- **Mọi lượt gọi đã tính tiền** được stream thành sự kiện `call` ngay khi xong. `authoring-service` ghi `llm_usage` ngay lúc đó.
- **Mỗi lần kiểm** được stream thành sự kiện `check`. `authoring-service` ghi vào bảng `code_check_diagnostics`. `rendering` thêm trường `rule` vào diagnostic (chỉ thêm trường, không phá contract cũ).

## Consequences
- Restart `llm-service` giữa lượt: các đoạn đã stream về đã nằm trong DB. `authoring-service` đánh dấu đoạn đang chạy là lỗi "bị ngắt"; Creator bấm chạy tiếp và chỉ trả tiền cho phần còn thiếu.
- Restart `authoring-service`: khi khởi động, mọi đoạn đang `running` được đánh dấu lỗi "bị ngắt".
- Mỗi lượt gửi lại toàn bộ đoạn đã xong (khoảng 130 KB code với video 52 shot). Chấp nhận được với HTTP nội bộ.
- Mất khả năng "nhớ đoạn này quá lớn, chia đôi luôn" giữa các lượt (trước đây nằm trong cache). Đoạn hết budget vẫn được chia đôi ngay trong lượt đó như hiện nay.
- Manim, hoặc Remotion không có `layout` trong storyboard: sửa storyboard sẽ đổi dấu vân tay của khung, nên mọi đoạn đều phải sinh lại. Đây là hệ quả đúng, vì mọi đoạn dựa trên khung.
- Đổi số shot mỗi đoạn làm đổi cách chia. Đoạn có khoá không còn trong cách chia mới sẽ bị bỏ.
