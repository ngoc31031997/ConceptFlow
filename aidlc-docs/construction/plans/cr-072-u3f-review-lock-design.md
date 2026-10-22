# CR-072 U3f · Bước 7 Rà soát và chốt kịch bản — thiết kế

Nhánh `feature/cr-072-u3f-review-lock` (worktree riêng, từ main `016c941`). Nguồn: FR-32, tiêu chí chấp nhận 12 và 14, ADR-0035, ADR-0044, `unit-of-work.md` mục U3f, mockup "Giao diện đã chốt" (màn `lock`). Khuôn: Viết kịch bản (U3e).

## Creator đã chốt (2026-10-22)

1. **Không đụng `flow.go`**: chốt phiên bản lưu vào DB, có `HandoffFor(role)` nhưng Storyboard, Hình minh hoạ, Code cũ chưa đọc bản chốt. U4a, U5, U4c, U7a sẽ nối dần. Nút "Sang Mỹ thuật lượt 1" mở trang "chưa phát triển".
2. **AI rà soát theo sáu vòng** của Creator, chia việc giữa máy và AI (bảng dưới). AI viết nháp ghi chú bàn giao.
3. **Rà soát chạy cả kịch bản trong một lượt; ghi chú bàn giao viết theo từng beat.**
4. **Vòng 6 "Final consistency"** là nút "Rà soát lại" ở tab Rà soát. Đây là thay đổi duy nhất so với mockup.

| Vòng | Nhóm trên màn | Máy kiểm (tính lại mỗi lần tải) | AI rà soát |
|---|---|---|---|
| 1 Fact-check | Độ chính xác · Nhất quán với nghiên cứu | dữ kiện bắt buộc chưa có trong lời thoại; câu dùng dữ kiện có đoạn trích không khớp; câu có con số/mốc thời gian mà chưa gắn dữ kiện; khi bỏ qua Nghiên cứu: đúng một cảnh báo "chưa có nguồn" | giả thuyết bị nói như sự thật; khẳng định chưa có bằng chứng trong bảng dữ kiện |
| 2 Logic & narrative | Logic và nhịp | chỗ nối có cờ (luật của bước 6) | lỗ hổng lập luận, ý lặp, chuyển đột ngột, nhân quả chưa rõ |
| 3 Audience & clarity | Dễ hiểu sai | câu có cờ "Khó nghe" (bước 6) | thuật ngữ khó, giải thích mơ hồ, kiến thức bị giả định là đã biết |
| 4 Timing & narration | Thời lượng | tổng thời lượng so với khung; câu có cờ "Câu dài" (bước 6) | đoạn giải thích quá dày, nhịp quá nhanh |
| 5 Production feasibility | Khả năng sản xuất | — | nội dung khó diễn đạt bằng hình, quá nhiều khái niệm cùng lúc, cảnh cần bối cảnh chưa có |
| 6 Final consistency | (nút "Rà soát lại") | mục "AI chưa rà soát bản này" khi lời thoại đổi sau lần rà soát gần nhất | so bản hiện tại với các mục cũ: mục đã hết thì "✓ đã sửa", mục Creator báo đã sửa mà vẫn còn thì mở lại, mục mới thì thêm vào |

## Phụ thuộc và trùng lặp

- Cần U3a (khung đơn vị), U3b (dữ kiện), U3d (beat), U3e (`script_beat`): tất cả đã vào main.
- Nhánh mở `fix/beat-sheet-from-story` chỉ đụng prompt và luồng beat sheet, không đụng tệp của bước này.
- U3f chặn U4a, U5, U4c, U7a, U7c (họ đọc `script_version` và `HandoffFor`).

## Thực thể

| Lưu ở | Kind / bảng | Key | Nội dung |
|---|---|---|---|
| `review_units` | `handoff_beat` (đơn vị duyệt, có trong API đơn vị chung) | khoá beat (`b1`…) | `must[]`, `terms[]`, `exact[]`, `free[]`, `unsure[]`. Dựng từ `script_beat:{key}`, nên lời thoại đổi thì ghi chú `stale` |
| `review_units` | `script_review` (không qua API đơn vị chung) | `main` | các mục AI rà soát (mã `r1`, `r2`… không dùng lại), ghi chú rà soát của AI, ghi chú của Creator, quyết định từng mục (đã sửa / chấp nhận kèm lý do), hash kịch bản lúc rà soát |
| `review_units` | `glossary` (không qua API đơn vị chung) | `main` | các dòng thuật ngữ, nói là, vẽ là, tránh; dòng Creator thêm có dấu |
| bảng mới `script_versions` | — | `(project_id, version)` | ảnh chụp bất biến: brief, dữ kiện dùng, câu chuyện, beat, lời thoại, ghi chú bàn giao, thuật ngữ, mục đã chấp nhận và ghi chú rà soát; hash từng phần; ghi chú phiên bản; lúc chốt |

Mục máy kiểm không lưu. Quyết định của Creator về một mục máy kiểm lưu theo mã mục kèm dấu vân tay của nội dung mục, nên mục đổi nội dung thì quyết định cũ tự hết hiệu lực.

## Luồng nghiệp vụ

1. **Điều kiện vào**: lời thoại mọi beat đã duyệt (bước 6 xong). Chưa xong thì báo "Duyệt lời thoại ở bước Viết kịch bản trước".
2. **Tab Rà soát**: nút "AI rà soát" (lần đầu) hoặc "Rà soát lại" (vòng 6) gọi một lượt AI (vai `script_review`) cho cả kịch bản. Mỗi mục có các nút như mockup:
   - **Mở câu đó**: mở bước Viết kịch bản tại câu đó.
   - **Mở ở Nghiên cứu**: cho mục dữ kiện.
   - **Áp dụng gợi ý**: sửa chữ câu bằng câu AI gợi ý. Đây là sửa tay qua đơn vị `script_beat`, nên beat đó về "chờ duyệt" và ghi chú bàn giao của beat thành "cần làm lại".
   - **Thêm vào ghi chú bàn giao**: thêm dòng AI đề xuất vào mục bàn giao của beat. Ghi chú đó về "chờ duyệt".
   - **Chấp nhận, ghi lý do**: hộp thoại.
   Có thêm ô "Ghi chú của bạn".
3. **Tab Ghi chú bàn giao**: bảng thuật ngữ ("+ Thêm thuật ngữ") và một thẻ cho mỗi beat, mỗi thẻ năm mục kèm vai đọc. Nút "AI viết ghi chú bàn giao" viết lần lượt các beat chưa có ghi chú; thuật ngữ AI đề xuất gộp vào bảng, dòng Creator viết giữ nguyên. Mỗi beat có Sửa tay (mỗi dòng một mục), Làm lại kèm ghi chú, Duyệt; có "Duyệt tất cả ghi chú còn lại".
4. **Tab Bộ bàn giao và phiên bản**: điều kiện chốt, bộ A và bộ B, bảng "Các bước sau đọc gì", lịch sử phiên bản. Bản nháp so hash từng phần với bản chốt gần nhất và liệt kê thay đổi ("Lời thoại · B3"). Nút "Chốt phiên bản vN": bản đầu chốt ngay; bản sau qua hộp xác nhận nêu thay đổi và phần sẽ làm lại.

### Điều kiện chốt (đúng mockup)

- Không còn mục rà soát cần xử lý, tính cả mục máy kiểm và mục AI. Lời thoại đổi sau lần rà soát gần nhất, hoặc chưa rà soát lần nào, là một mục cần xử lý: Creator bấm "Rà soát lại" hoặc chấp nhận kèm lý do (khi chưa cấu hình AI).
- Lời thoại mọi beat đã duyệt.
- Ghi chú bàn giao mọi beat đã duyệt và không "cần làm lại".

### Các bước sau đọc gì (`HandoffFor(role)`, FR-32 mục 4)

| Vai | Đọc từ phiên bản đã chốt |
|---|---|
| `art_world` Mỹ thuật lượt 1 | giọng điệu và ngoài phạm vi của brief, thuật ngữ, phải giữ chính xác |
| `storyboard` | lời thoại theo beat, mục tiêu và chuyển tiếp của beat, bắt buộc truyền đạt, được phép sáng tạo, điểm chưa chắc |
| `art_color` Mỹ thuật lượt 2 | chức năng từng beat |
| `voice` Giọng | lời thoại, thuật ngữ |
| `illustration` | phải giữ chính xác, thuật ngữ, điểm chưa chắc |
| `code` | được phép sáng tạo |
| `validate` | phải giữ chính xác |

Route nội bộ `GET /internal/v1/authoring/{id}/handoff?role=…` trả phần của vai đó từ bản chốt mới nhất (404 khi chưa chốt). CR này chưa có bên nào gọi route; các CR con sau sẽ dùng.

## Thiết kế mức thấp

### authoring-service (Go, hexagonal như U3e)

| Lớp | Tệp mới |
|---|---|
| domain | `script_review.go` (mục rà soát, nhóm, máy kiểm, đọc câu trả lời AI rà soát, gộp vòng 6), `handoff.go` (năm mục, thuật ngữ, đọc câu trả lời AI, vai đọc từng mục, `HandoffFor`), `script_version.go` (ảnh chụp, hash từng phần, so bản nháp), `prompt_seed_script_review.go` (vai `script_review`, `script_handoff`) |
| application | `script_lock_desk.go` (trạng thái màn, xử lý mục, ghi chú, thuật ngữ, chốt), `script_reviewer.go` (lời gọi AI rà soát), `handoff_units.go` (UnitKind `handoff_beat` và AI viết ghi chú) |
| adapters/http | `script_lock.go` |
| adapters/postgres | `script_version_repository.go`, bảng `script_versions`, xoá kèm project |
| cmd | nối dây trong `main.go` |

Model AI: dùng lại khoá `story` như bước 6, vì không có cột model riêng cho bước này.

### Hợp đồng HTTP công khai (tiền tố `/v1/projects/{id}`)

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/script-lock` | toàn màn: mục rà soát, ghi chú, thuật ngữ, ghi chú bàn giao theo beat, bộ A và B, điều kiện chốt, phiên bản, thay đổi của bản nháp |
| POST | `/script-lock/review` | AI rà soát hoặc rà soát lại (`X-Operation-Id`) |
| POST | `/script-lock/findings/{id}/apply` · `/handoff` · `/accept` `{why}` | xử lý một mục |
| POST | `/script-lock/notes` `{text}` | thêm ghi chú của Creator |
| POST | `/script-lock/glossary` `{term,say,draw,avoid}` | thêm thuật ngữ |
| POST | `/script-lock/handoff/generate` `{beat?}` | AI viết ghi chú bàn giao cho các beat chưa có (`X-Operation-Id`) |
| POST | `/script-lock/lock` `{note}` | chốt phiên bản mới |
| * | `/units/handoff_beat/{key}` (+ `/redo`, `/impact`, `/approve`, `/unapprove`) | API đơn vị chung |

Tài liệu `docs/contracts/script-lock-v1.md` và thêm một dòng vào README hợp đồng.

### api-gateway

Proxy `/v1/projects/:id/script-lock…`; route AI qua `authoringAi` (timeout dài) như `/script/generate`.

### web-gui

Trang `/projects/:id/lock` (`screen: "lock"`), bám bố cục mockup: dải trạng thái và ba tab. Thành phần mới nằm trong `components/lock/`: `LockReviewTab`, `LockHandoffTab`, `LockPackageTab`, hộp thoại Chấp nhận, Thêm thuật ngữ, xác nhận chốt. Dùng lại `UnitEditModal`, `UnitRedoModal`, `UnitReviewBar`, `UnitStatusChip`, `ImpactConfirm`, `AiOperationCard`, `AiModelSelect`, `FactTags`. Bước Viết kịch bản: "Sang rà soát và chốt" mở trang này, và mở được tại một câu (`?line=b5.1`, câu đó được cuộn tới và tô sáng). `ProjectFlowContext` đọc thêm `locked_version`; thanh bước đánh dấu bước 7 xong khi đã chốt.

### Kiểm thử

- Go: máy kiểm (dữ kiện bắt buộc, link không khớp, số chưa gắn dữ kiện, bỏ qua Nghiên cứu cho đúng một cảnh báo, thời lượng); đọc câu trả lời AI (đúng, hỏng, nhóm lạ, mã câu lạ); gộp vòng 6; ghi chú bàn giao và gộp thuật ngữ; hash, so bản nháp, chốt v1 rồi v2 có lịch sử (tiêu chí 14); điều kiện chốt; use case với AI giả; HTTP; kho `script_versions` (bỏ qua khi thiếu `TEST_DATABASE_URL`).
- Web-gui: vitest cho trang và hook.
- Không gọi Hive hay Ollama thật. Dọn dữ liệu test sau khi chạy.

## Khác mockup (nói rõ)

- Nút "Rà soát lại" và mục "AI chưa rà soát bản này" (vòng 6, Creator chọn).
- Áp dụng gợi ý đưa beat về "chờ duyệt", đúng luật đơn vị (FR-17). Mockup không vẽ việc này.
- Khi chốt v2, hộp xác nhận nói rõ chưa bước nào phía sau đọc phiên bản kịch bản (vì không đụng `flow.go`). Mockup liệt kê storyboard, giọng… sẽ làm lại.

## Chưa làm (nói rõ)

- Storyboard, Hình minh hoạ, Code cũ đọc bản chốt (Creator chọn không đụng `flow.go`).
- Sửa, xoá một dòng thuật ngữ: mockup chỉ có thêm.
- Chưa chạy AI thật nếu local thiếu `HIVE_API_KEY`; phần đó chỉ kiểm bằng AI giả.
