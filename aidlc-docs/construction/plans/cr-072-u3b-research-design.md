# CR-072-U3b — Nghiên cứu và kiểm chứng (bước 3, tuỳ chọn)

Trạng thái: **đề xuất, chờ Creator duyệt thiết kế.** Nhánh `feature/cr-072-u3b-research`, từ main `57642e8`. Cỡ L. Phụ thuộc U3a (đã vào main).

## Yêu cầu gốc (nguyên văn)

> "Cho tôi làm U3b (Nghiên cứu) khi bạn sẵn sàng." (Creator, 2026-10-18)
>
> "bạn chỉ làm bước nghiên cứu theo file mockup thôi mà" (Creator, 2026-10-18, sau bản thiết kế đầu)

**Phạm vi sau câu trả lời thứ hai**: chỉ màn bước 3 của mockup bản 9 và backend nó cần. Bỏ khỏi bản đầu: dời số bước của luồng chạy (13 → 14) và migration `project_events` ở orchestrator; cho bước Kịch bản hiện có đọc dữ kiện (`{{research_facts}}`, `story_facts_sha`, lời nhắc "viết theo dữ kiện cũ"); lệch mockup ở ô câu hỏi. Các mục bị bỏ dưới đây được thay bằng mục "Phạm vi đã thu hẹp"; phần nào mâu thuẫn thì mục đó thắng.

## Phạm vi đã thu hẹp (thay các mục tương ứng bên dưới)

- **Luồng**: bước 3 chỉ là một màn của luồng 21 bước trên web (`FLOW_VIEW` bước 3 trỏ tới màn mới, như mockup), không thêm bước vào `flow.go`, không đổi `FlowStateFor`, không đổi orchestrator, không migration. Thanh bước hiện nhãn "tuỳ chọn" / "bỏ qua". "Sang bước tiếp" ở Cấu hình mở Nghiên cứu; "Sang xây dựng câu chuyện →" (bật khi bảng đã duyệt hoặc đã bỏ qua) mở bước 4 Câu chuyện như mockup; bước này chưa có nên màn đó hiện "Bước này chưa phát triển" cho tới U3c (Creator, 2026-10-18: "là mở ra Câu chuyện chưa có thì hiện chưa phát triển thôi").
- **Câu hỏi**: làm đúng mockup: mỗi câu hỏi là một ô nhập trên trang, lưu khi rời ô; "× " xoá có xác nhận; "+ Thêm câu hỏi"; "Sinh lại từ brief" (cam khi có câu đã sửa, có xác nhận; tím khi không). Lần đầu chưa có câu hỏi thì nút này sinh câu hỏi.
- **Brief đổi**: như mockup, bước 3 hiện dải "Brief vừa đổi: câu hỏi nghiên cứu cần sinh lại theo brief mới. Dữ kiện đã có giữ nguyên." và ↻ trên thanh bước; dùng cột `input_ref/input_sha` để biết.
- **Bảng đã duyệt** về chờ duyệt khi dán thêm (như mockup), thêm, sửa, bỏ, đổi loại hay bắt buộc nói.
- **Bước Kịch bản hiện có không đổi** trong U3b.
- Bỏ khỏi phạm vi file: `orchestrator/*`, `domain/flow.go` hai bản, `tests/contracts/test_flow_steps_contract.py`, `render_prompt*.go`, cột `story_facts_sha`, trường `research_resolved` của summaries.

Định nghĩa của bước nằm ở các tài liệu đã duyệt: FR-28 và tiêu chí chấp nhận 11–13 (`aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md`), ADR-0043, mục U3b của `unit-of-work.md`, màn "Nghiên cứu và kiểm chứng" ở mockup bản 9 (https://claude.ai/artifact/HibBJUisUocqKXDZD57cTH). Tài liệu này chốt phần Functional Design mà ADR-0043 và `unit-of-work.md` để lại cho U3b: mẫu JSON, cách chuẩn hoá khi so đoạn trích, số liệu của `SourceFetcher`, và cách bước 3 chen vào luồng đang chạy.

## Hiện trạng

- Khung đơn vị (U3a): bảng `review_units` (`services/authoring-service/internal/adapters/postgres/db.go:211`), `ReviewUnitsUseCase` với registry loại đơn vị (`application/review_units.go:31`), HTTP chung `/v1/projects/{id}/units/{kind}/{key}` (`adapters/http/review_units.go`, route gateway `api-gateway/src/routes/projects.js:133-138`). Loại duy nhất là `brief` (`application/creative_brief.go`). `status` chỉ có `pending|approved`; chưa có `stale` và chưa có `DependencyTracker` (U3a để lại cho U3b, `cr-072-u3a-idea-brief-design.md` mục 1).
- Luồng 13 bước định nghĩa ba lần giống nhau: `authoring-service/internal/domain/flow.go`, `orchestrator/internal/domain/flow.go` (hai file phải trùng byte, `tests/contracts/test_flow_steps_contract.py`) và `web-gui/src/utils/flow.ts`. `FlowStateFor` (`flow.go:79`) suy bước của bản nháp từ `AuthoredContent` (Story, Storyboard, Code, Illustrations) mà orchestrator lấy từ `GET /internal/v1/authoring/summaries` (`orchestrator/internal/application/authoring_port.go:12-29`; `IllustrationsReady == nil` đọc là xong). Bước 3 của mockup ("Nghiên cứu") chưa có bước chạy tương ứng: `FLOW_VIEW` ghi `server: []` (`flow.ts:106`).
- Số bước được lưu ở `project_events.flow_step` / `from_flow_step`; lần chèn bước Hình minh hoạ đã dời số bằng migration có khoá `schema_migrations` (`orchestrator/internal/adapters/postgres/db.go:452-482`).
- Bước Kịch bản hiện có (`story_architect`) đọc `{{creative_brief}}` (`render_prompt.go`, `render_prompt_input.go:124`) và dấu `story_brief_sha` báo dàn ý viết theo brief cũ (`review_unit_repository.go:91`). Chưa có chỗ nào đọc dữ kiện.
- authoring-service chưa gọi ra Internet; chưa có HTTP client tải trang ngoài. Compose đặt nó trên mạng `backend` (bridge).
- Tạo bản mới (`orchestrator/internal/application/fork_project.go:74`) không sao chép dữ liệu soạn nào của authoring-service ngoài những gì `ForkAuthoringPort` có.

## Yêu cầu

Theo FR-28 (chín mục), thêm các điều kiện cụ thể dưới đây.

1. **Bước 3 là một bước thật của luồng**: luồng chạy thành 14 bước, Nghiên cứu ở số 3, các bước sau dời một số (Kịch bản 4 … Publish 14). Thanh bước, Bản đồ luồng, danh sách video, nhật ký sự kiện và `Bản nháp mở lại` đều dùng số mới.
2. **Câu hỏi cần tra**: đơn vị `research_question` (mã Q1, Q2…, không bao giờ dùng lại). AI sinh từ brief bằng nút "Sinh câu hỏi từ brief" (4–8 câu); sửa, thêm tay, xoá có xác nhận; "Sinh lại từ brief" thay câu AI viết, giữ câu Creator viết/sửa, trừ khi Creator đồng ý thay (xác nhận). Không cần duyệt.
3. **Prompt nghiên cứu** (`GET …/research/prompt`) và **prompt bổ sung** (`GET …/research/follow-up-prompt`) dựng từ brief, câu hỏi, luật nguồn và mẫu `conceptflow.research.v1` (mục "Mẫu JSON"). Modal chép prompt (luật giao diện 4).
4. **Dán kết quả** (`POST …/research/paste`): tìm khối JSON, gộp theo mã, bỏ dòng lỗi kèm lý do, một dòng nhật ký mỗi lần dán; không có khối JSON thì báo rõ và không đọc gì.
5. **Kiểm link** chạy nền theo ADR-0043; kết quả `khớp / không khớp (câu gần nhất) / không tải được (lý do) / không có nguồn / đang kiểm`; "Kiểm lại" từng dữ kiện; lưu sửa link hoặc đoạn trích thì kiểm lại.
6. **Bảng dữ kiện**: đổi loại, "bắt buộc nói", Sửa (modal), Bỏ / Giữ lại (xác nhận khi có chỗ đang dùng), thêm tay (modal), lọc theo kết quả kiểm link và "Đã bỏ".
7. **Độ phủ câu hỏi** và nút xem prompt bổ sung cho các câu hỏi chưa đủ hai nguồn độc lập khớp hoặc có dữ kiện không khớp.
8. **Duyệt bảng dữ kiện** (đơn vị `research:main`): cần ít nhất một dữ kiện đang giữ; không cần mọi link khớp. **Bỏ qua bước** (xác nhận) và **Làm nghiên cứu** để quay lại; không bước nào bị chặn.
9. **Câu hỏi cũ khi brief đổi**: brief đổi thì câu hỏi do AI sinh hiện nhãn "cần sinh lại" (dữ kiện giữ nguyên); danh sách ảnh hưởng của brief (U3a) thêm dòng này.
10. **Bước Kịch bản hiện có đọc dữ kiện** qua biến mới `{{research_facts}}` (mã, loại, nhãn nguồn, bắt buộc nói); bỏ qua bước hoặc chưa có thì câu thay thế rõ ràng; dữ kiện đổi sau khi lưu dàn ý thì màn Kịch bản báo "viết theo dữ kiện cũ".
11. Cổng `ResearchSource` (một bản "paste") để API tìm kiếm cắm sau mà không đổi bảng dữ kiện.

**Tiêu chí xong** (từ `unit-of-work.md`): dán câu trả lời có 12 dữ kiện và một dòng lỗi thì bảng có 12 dữ kiện và nhật ký ghi dòng bị bỏ kèm lý do; đoạn trích không có trong trang hiện "không khớp"; link tới `localhost`, `10.x`, `169.254.169.254`, tên service trong mạng Docker bị từ chối, có test; dán lần hai cập nhật đúng mã đã có; bỏ qua bước thì luồng vẫn đi tiếp; luồng 14 bước.

**Không làm**: tra nguồn tự động bằng API (backlog); dữ kiện "chưa có nguồn" do AI tự liệt kê (U3c); gắn dữ kiện vào beat và lời thoại (U3d, U3e); kiểm lại sau khi chốt (backlog); sao chép bảng dữ kiện sang bản mới khi "tạo bản mới" (xem Rủi ro); đọc PDF.

## Giải pháp đề xuất

### 1. Chen bước 3 vào luồng (quyết định đã có ở ADR-0042; đây là cách làm)

- `flow.go` (hai bản): thêm `FlowResearch = 3`, dời `FlowStory…FlowPublish` thành 4…14, `FlowStepsTotal = 14`, nhãn "Nghiên cứu". `AuthoredContent` thêm `Research bool` ("bảng dữ kiện đã duyệt hoặc đã bỏ qua"). `AuthoringSummary` thêm `ResearchResolved *bool`; `nil` (authoring-service cũ không gửi) đọc là xong, đúng kiểu `IllustrationsReady`.
- `FlowStateFor` cho bản nháp: bước Cấu hình xong mà chưa có dàn ý (`!content.Story`) và `!content.Research` thì đứng ở **Nghiên cứu**; đã có dàn ý, storyboard hay code thì giữ như hiện nay (project làm dở không bị kéo lùi). `FlowStepForAuthoring` không đổi tên, chỉ đổi số theo hằng.
- Orchestrator: migration một lần có khoá `schema_migrations` dời `project_events.flow_step` và `from_flow_step` `>= 3` lên 1, theo đúng mẫu đã có ở `db.go:452`. Hằng và nhãn đổi theo `flow.go`; chỗ nào dùng số trần được tìm bằng `graphify affected`.
- web-gui `flow.ts`: `FLOW_LABELS`, `FLOW_STEP_PURPOSE`, các hằng `FLOW_*`, `AUTHORING_STEP_FLOW`, `authoringRoute`, `FORK_STEPS`, `FLOW_VIEW[].server` (bước 3 `server: [3]`, các bước sau +1), `AUTO_STEPS`, `isCancellableStep`, `useStepNav` (`step <= 6` → `<= 7`). Test hợp đồng luồng cập nhật mô tả "14 bước".
- Bước 3 có nhãn "tuỳ chọn" / "bỏ qua" trên thanh bước (mockup).

### 2. Dữ liệu (authoring-service)

Dùng lại `review_units`, thêm ba loại đơn vị và hai cột:

| Loại : key | Nội dung `content` | Ghi chú |
|---|---|---|
| `research:main` | `{skipped, next_question}` | `status=approved` là "đã duyệt bảng"; `skipped` là "đã bỏ qua"; đã xong khi một trong hai |
| `research_question:Qn` | `{text}` | `creator_fields=["text"]` khi Creator viết/sửa; AI viết thì để trống |
| `fact:Fnn` | `{question, claim_vi, type, source_title, publisher, year, url, quote, confidence, note, must, keep, link}` | `type` ∈ `fact|hypothesis|unknown|misconception`; `link = {state: ok|bad|na|none|run, why, nearest, checked_at}` |

- `review_units` thêm cột `input_ref TEXT` và `input_sha TEXT` (đầu vào lúc dựng, ADR-0035): đơn vị `stale` khi `md5` của đơn vị `input_ref` hiện nay khác `input_sha`. Câu hỏi AI sinh ghi `input_ref='brief:main'`; câu hỏi tay để trống. JSON của đơn vị thêm `stale`. Đây là phần `DependencyTracker` tối thiểu mà U3b cần; các CR sau dùng lại cho story/beat/line.
- Bảng mới `research_pastes (project_id, n, at, added TEXT[], updated TEXT[], rejected JSONB)` cho nhật ký dán (không lưu nguyên văn câu trả lời).
- `project_authoring` thêm `research_model` (model chọn ở nút AI của bước, như `brief_model`) và `story_facts_sha` (md5 của các dữ kiện đang giữ lúc lưu dàn ý, để báo "viết theo dữ kiện cũ").
- Mã dữ kiện `F01…`: dán vào giữ nguyên mã của câu trả lời (chuẩn hoá `F1` → `F01`); thêm tay lấy mã lớn nhất + 1. Mã câu hỏi tăng dần theo `next_question`, không dùng lại, nên dữ kiện của câu hỏi đã xoá vẫn giữ `question` cũ (hiện "câu hỏi đã bỏ", không tính độ phủ).

### 3. Mẫu JSON `conceptflow.research.v1`

```json
{"schema":"conceptflow.research.v1","claims":[{"id":"F01","question":"Q1","claim_vi":"...","type":"fact","source_title":"...","publisher":"...","year":2020,"url":"https://...","quote":"...","confidence":"high","note":""}]}
```

- Tìm khối: ưu tiên các khối ```json; không có thì lấy mọi đoạn `{…}` cân ngoặc có chuỗi `conceptflow.research.v1`. Nhiều khối thì đọc hết.
- Mỗi dòng bị bỏ kèm lý do khi: thiếu `id`/`claim_vi`/`type`; `type` ngoài bốn giá trị; `question` không có trong danh sách câu hỏi; `url` không phải `http(s)`, dài quá 2048 ký tự hoặc có thông tin đăng nhập; `type` ≠ `unknown` mà có `url` rỗng thì vẫn nhận nhưng `link=none`; trùng mã trong cùng lần dán thì dòng sau thắng và dòng trước ghi vào nhật ký.
- Giới hạn: câu trả lời ≤ 1 MB, ≤ 300 dòng.
- Gộp theo mã: dữ kiện chưa có thì thêm; đã có thì cập nhật các trường chữ và nguồn, **giữ** các trường Creator đã sửa tay, `must` và `keep`; `url` hoặc `quote` đổi thì kiểm lại link.
- Ghi vào `docs/contracts/research-v1.md` và mục lục `docs/contracts/README.md` (mẫu JSON, luật dán, và đường ra Internet của authoring-service).

### 4. `SourceFetcher` và `SourceVerifier` (ADR-0043; số liệu chốt)

| Hạng mục | Giá trị |
|---|---|
| Lược đồ, cổng | `http`, `https`; cổng 80, 443 |
| Tên chặn trước khi phân giải | `localhost`, tên không có dấu chấm (tên service Docker như `rabbitmq`), đuôi `.local`, `.internal`, `.localdomain`, `.lan` |
| Địa chỉ chặn sau khi phân giải | loopback, riêng (10/8, 172.16/12, 192.168/16, fc00::/7), link-local (169.254/16 gồm metadata đám mây, fe80::/10), `0.0.0.0/8`, CGNAT 100.64/10, 198.18/15, multicast, broadcast, 240/4, IPv4-mapped IPv6 được bóc rồi kiểm; **mọi** địa chỉ trả về phải công khai |
| Chống đổi địa chỉ giữa lúc kiểm và lúc nối | kiểm lại địa chỉ thực sự được nối (`net.Dialer.Control`), không phân giải lần hai |
| Chuyển hướng | tối đa 5; mỗi lần kiểm như link mới |
| Thời gian | nối 4 giây, cả lần tải 12 giây |
| Dung lượng | tối đa 5 MiB; vượt thì "không tải được: trang quá lớn" |
| Kiểu nội dung | `text/html`, `application/xhtml+xml`, `text/plain`; kiểu khác (PDF, ảnh…) là "không tải được: loại tệp chưa đọc được" |
| Đồng thời | 4 trang cùng lúc cho cả service |
| Khác | không cookie, không header đăng nhập, không chạy script; `User-Agent` nêu tên ứng dụng; trạng thái 401/402/403/429/5xx → "không tải được" kèm mã |

Chuẩn hoá khi so (cả trang và đoạn trích): bỏ `script/style/noscript` và thẻ, giải mã thực thể HTML, chuẩn Unicode NFKC, chữ thường, nháy cong → nháy thẳng, mọi loại gạch → `-`, bỏ ký tự rộng bằng 0 và gạch nối mềm, bỏ dấu chú thích như `[1]` / `[citation needed]`, gộp khoảng trắng. Đoạn trích có `…` hoặc `...` thì tách thành các đoạn và yêu cầu có đủ theo thứ tự. Đoạn trích chuẩn hoá ngắn hơn 20 ký tự thì không kiểm được: "không khớp: đoạn trích quá ngắn". Không thấy đoạn trích: tách trang thành câu, chọn câu có độ giống từ (Jaccard theo từ) cao nhất ≥ 0,5 làm "câu gần nhất" (cắt 240 ký tự).

Kiểm chạy nền trong tiến trình authoring-service (không dùng context của request): đặt `link.state=run`, rồi mỗi dữ kiện một tác vụ trong hàng đợi 4 luồng; kết quả ghi bằng cập nhật có điều kiện (`url` và `quote` hiện tại còn giống lúc bắt đầu, nếu không thì bỏ kết quả) để không đè bản Creator vừa sửa. Lúc khởi động, mọi `run` còn sót chuyển thành "không tải được: bị gián đoạn, bấm Kiểm lại". Màn hình hỏi lại `GET …/research` mỗi 2 giây khi còn dữ kiện `run`.

`ResearchSource` là interface `Name() string` kèm bản `paste` (như `component-methods.md`); không thêm phương thức chưa có người dùng.

### 5. Độ phủ và prompt bổ sung

Độ phủ tính khi đọc, không lưu: số **tên miền độc lập** (bỏ `www.`) trong các dữ kiện đang giữ của câu hỏi có `link=ok`. Đủ khi ≥ 2. Câu hỏi yếu (< 2 hoặc có dữ kiện `bad`) vào prompt bổ sung cùng danh sách dữ kiện cần nguồn khác; mã đánh tiếp từ mã lớn nhất. (Mockup đếm theo tên nguồn; đổi sang tên miền vì tên nguồn do AI viết nên không đối chiếu được.)

### 6. Prompt và model AI

Hai vai prompt mới, seed trong authoring-service (sửa được ở Cài đặt prompt): `research_questions` (trả `{"questions":["…"]}`, 4–8 câu, bằng ngôn ngữ nội dung) và `research_prompt` (văn bản chép ra ngoài; biến `topic`, `central_question`, `audience`, `target_duration`, `questions`, `extra`, `next_fact_id`; nội dung theo luật của mockup: chỉ dẫn nguồn đã mở, đoạn trích nguyên văn ngôn ngữ gốc, ưu tiên nguồn bình duyệt/bách khoa/viện, ≥ 2 nguồn độc lập, `unknown` thì để trống nguồn, không bịa). Việc gọi AI sinh câu hỏi dùng model chọn ở nút (khoá `research`), mặc định theo Cài đặt như các nút AI khác. `story_architect` thêm biến `{{research_facts}}` (bản prompt mới); prompt Creator tự viết không có biến thì dữ kiện không vào prompt và màn Kịch bản báo như với `{{creative_brief}}`.

### 7. API (authoring-service, qua api-gateway; route cụ thể)

- `GET /v1/projects/{id}/research` → `{gate, questions[], facts[], pastes[], coverage[], weak_questions[], story_prompt_reads_facts, story_stale}` (một lần đủ cho cả màn).
- `POST …/research/questions/generate` `{replace_edited, note?}` (theo dõi bằng `X-Operation-Id`); `POST …/research/questions` `{text}`; `DELETE …/research/questions/{qid}`; sửa chữ qua `PUT /units/research_question/{qid}`.
- `GET …/research/prompt`, `GET …/research/follow-up-prompt` → `{prompt}`.
- `POST …/research/paste` `{raw}` → `{added[], updated[], rejected[{row, reason}], log}`.
- `POST …/research/facts` (thêm tay); `PUT …/research/facts/{fid}` `{type?, must?, keep?}`; sửa chữ, link, đoạn trích qua `PUT /units/fact/{fid}`; `GET /units/fact/{fid}/impact`.
- `POST …/research/verify` `{fact_ids}`; `PUT …/research/skip` `{skipped}`; duyệt bảng qua `POST /units/research/main/approve`.
- Nội bộ: `GET /internal/v1/authoring/summaries` thêm `research_resolved` (chỉ thêm trường, cách cũ vẫn chạy).

### 8. Màn hình (web-gui)

Trang mới `ResearchStepPage.tsx` (`/create/script/research`) theo mockup: dải "Tuỳ chọn · Bỏ qua bước này", ① Câu hỏi cần tra, ② Prompt nghiên cứu, ③ Dán kết quả (nhật ký + ô dán + "Đọc kết quả"), ④ Bảng dữ kiện (lọc, loại, nguồn, kiểm link, bắt buộc nói, Sửa/Kiểm lại/Bỏ), ⑤ Độ phủ + prompt bổ sung, Duyệt bảng dữ kiện, "Sang xây dựng câu chuyện →". Theo luật giao diện 4–6: Sửa dữ kiện, Thêm dữ kiện tay, xem/chép prompt, và mọi xác nhận (bỏ qua bước, xoá câu hỏi, sinh lại câu hỏi đã sửa, bỏ dữ kiện đang dùng) là modal dùng `Dialog`; nút "Bỏ qua bước" và "Bỏ dữ kiện đang dùng" màu cam, "Xoá câu hỏi" đỏ, "Sinh câu hỏi" tím, "Đọc kết quả" và "Kiểm lại" vàng. Mockup vẽ mỗi câu hỏi là một ô nhập sửa thẳng trên trang; theo luật 4, câu hỏi hiện dạng chữ và sửa trong modal "Sửa câu hỏi", như U3a làm với brief (lệch mockup có chủ ý ở điểm này). Thanh bước 3 hiện "tuỳ chọn" / "bỏ qua". Bước 2 "Sang bước tiếp" dẫn sang Nghiên cứu.

### Điểm AI đề xuất, Creator có thể đổi

1. **Sinh câu hỏi bằng nút**, không tự chạy khi mở bước (tránh tốn token khi Creator định bỏ qua). Khác: tự chạy khi brief vừa duyệt.
2. **Brief đổi thì câu hỏi chỉ hiện nhãn "cần sinh lại"**, không tự sinh lại (Creator bấm, có xác nhận nếu sẽ thay câu đã sửa). Mockup ghi "sinh lại theo brief mới"; làm tự động sẽ ghi đè câu Creator đã sửa mà không hỏi.
3. **Bảng đã duyệt mà dán thêm, thêm tay, bỏ hay đổi loại / bắt buộc nói dữ kiện thì về chờ duyệt** (duyệt là duyệt bảng hiện tại). Kết quả kiểm link không đổi trạng thái duyệt.

## Phụ thuộc

Đã kiểm `git branch --no-merged origin/main`, mọi worktree (`scripts/worktree.sh list`) và file đang đổi:
- `feature/cr-072-u1-visual-language` (U1, kit vẽ Remotion): nhánh chưa có commit nào ngoài main, worktree chỉ giữ tài liệu thiết kế chưa commit. **Độc lập** (không chạm luồng, review unit, prompt).
- `feature/cr-072-u2a-three-phase-flow` (U2a, tạm dừng): không có commit khác main; phần khung nhóm bước, thanh bước, Bản đồ luồng của nó đã vào main cùng U3a. **Độc lập**; phần còn lại của U2a (xoá dữ liệu cũ, trang luồng) không đụng bước 3. Khi U2a quay lại phải dùng số bước 14.
- `chore/worktree-per-cr`, `chore/local-ollama-host`: hạ tầng làm việc / cấu hình Ollama. **Độc lập**.
- Không CR/fix mở nào khác.
- Thứ tự trong CR-072: U3b chặn U3c (cần `fact`, `stale`, số bước 14) → làm trước U3c.

## Phạm vi

- **authoring-service**: `domain/flow.go`, `domain/research.go` (mới: fact, question, mã, parse JSON, chuẩn hoá, độ phủ), `domain/review_unit.go` (`Stale`, `InputRef`), `domain/prompt_template.go` (hai vai mới), `application/research_desk.go`, `research_paste.go`, `source_verifier.go` (mới), `application/source_fetcher.go` (cổng) và `adapters/sourcefetch/` (mới: HTTP client có chặn địa chỉ), `application/creative_brief.go` (Impact thêm dòng câu hỏi), `application/render_prompt*.go` (`{{research_facts}}`), `adapters/postgres/db.go` (cột, bảng), `review_unit_repository.go`, repository nhật ký dán, `adapters/http/research.go` (mới), `router.go`, `cmd/authoring/main.go` (nối dây, quét `run` còn sót), seed prompt.
- **orchestrator**: `domain/flow.go` (bản giống hệt), `application/authoring_port.go`, `adapters/postgres/db.go` (migration dời số bước), chỗ dùng số bước trần (tìm bằng `graphify affected FlowStateFor` ở bước làm task), test luồng.
- **api-gateway**: `routes/projects.js` (route mới, tải trang là việc nền nên dùng client thường; sinh câu hỏi dùng client AI dài).
- **web-gui**: `utils/flow.ts`, `hooks/useStepNav.ts`, `pages/ResearchStepPage.tsx` + hook + `api/client.ts` + các thẻ/modal mới, `App` route, màn Kịch bản (nhắc dữ kiện).
- **docs**: `docs/contracts/research-v1.md`, `docs/contracts/README.md`, `tests/contracts/test_flow_steps_contract.py` (14 bước), `aidlc-docs/aidlc-state.md`.
- **DB**: thêm cột/bảng như mục 2; migration số bước ở orchestrator; không xoá dữ liệu.
- `graphify affected` sẽ chạy cho `FlowStateFor`, `AuthoredContent`, `AuthoringSummary`, `ReviewUnitsUseCase`, `StoryBriefLink` ở bước chia task để liệt kê hết người gọi.

## Kế hoạch thực hiện

1. Luồng 14 bước: `flow.go` ×2, `AuthoredContent.Research`, summaries, migration `project_events`, `flow.ts` và các chỗ dùng số bước, test hợp đồng luồng.
2. Domain nghiên cứu: kiểu, mã, parse khối JSON, chuẩn hoá, so đoạn trích, câu gần nhất, độ phủ (thuần Go, test bảng).
3. `SourceFetcher`: bộ chặn địa chỉ, client, giới hạn (test với máy chủ giả trên cổng loopback bằng hàm khởi tạo nội bộ cho phép loopback; test bộ chặn thật cho localhost, `10.x`, `169.254.169.254`, tên Docker, chuyển hướng vào mạng nội bộ).
4. Lưu trữ: cột `input_ref/input_sha`, `stale`, bảng nhật ký dán, cập nhật có điều kiện kết quả link, quét `run` còn sót, các cột `project_authoring`.
5. Use case: `ResearchDesk` (câu hỏi, prompt, dán, thêm/sửa/bỏ dữ kiện, bỏ qua, duyệt), `SourceVerifier` (hàng đợi nền), đăng ký loại `research_question`, `fact`, `research` vào `ReviewUnitsUseCase`; Impact của brief và của dữ kiện.
6. Prompt: hai vai seed, `{{research_facts}}` cho `story_architect`, `story_facts_sha`.
7. HTTP authoring-service + route gateway.
8. web-gui: client, hook, trang, modal, thanh bước, nhắc ở màn Kịch bản.
9. Tài liệu hợp đồng, `aidlc-state.md`, mockup/dashboard nếu lệch (ghi vào báo cáo).

## Kiểm tra

- Go `go test ./...` ở authoring-service và orchestrator; `npm test` api-gateway; web-gui `npx vitest run`, `npx tsc --noEmit`, `npx eslint`; `pytest tests/contracts`.
- Test bắt buộc theo tiêu chí: dán 12 dữ kiện + 1 dòng lỗi; dán lần hai cập nhật đúng mã; đoạn trích không có trong trang → "không khớp" kèm câu gần nhất; bộ chặn địa chỉ; bỏ qua bước thì `FlowStateFor` đi tiếp sang Kịch bản; migration dời số bước chạy một lần. Test không gọi Hive hay Ollama (AI giả); tải trang dùng máy chủ giả, không ra Internet.
- Build lại: authoring-service, orchestrator, api-gateway, web-gui.
- Trên app: tạo project, duyệt brief, sang Nghiên cứu; thêm câu hỏi tay; copy prompt; dán một khối JSON mẫu có link Wikipedia thật (cần Internet từ container) và xem link chuyển "khớp"/"không khớp"; thử link `http://localhost` bị từ chối; duyệt bảng rồi sang Kịch bản; bỏ qua bước rồi quay lại bằng "Làm nghiên cứu"; mở một project cũ đang ở Kịch bản/Storyboard vẫn đúng bước.

## Rủi ro

- **Dời số bước** ảnh hưởng nhật ký và danh sách video đã có: migration có khoá chạy một lần; project đang chạy giữ đúng bước nhờ ánh xạ theo trạng thái, nhưng liên kết sâu cũ chứa `?step=` có thể trỏ lệch một bước.
- **Lần đầu authoring-service ra Internet**: SSRF là rủi ro chính; chặn ở cả tên, địa chỉ phân giải và địa chỉ thực nối, kiểm lại ở mỗi lần chuyển hướng. Nếu mạng Docker của máy Creator chặn ra ngoài, mọi link hiện "không tải được" (báo lý do).
- Trang trả phí, trang dựng bằng JavaScript, hoặc chặn bot cho kết quả "không tải được"; PDF chưa đọc được. Chấp nhận (ADR-0043).
- "Tạo bản mới" không sao chép bảng dữ kiện (`ForkAuthoringPort` chưa có phương thức); bản mới bắt đầu nghiên cứu lại. Ghi rõ trên màn tạo bản mới khi làm task.
- Kiểm link nền chạy trong tiến trình: khởi động lại giữa chừng để lại `run`, đã có quét lúc khởi động.
- Chất lượng sinh câu hỏi phụ thuộc model; không có LLM thì Creator tự thêm câu hỏi.
