# Low-Level Design Plan — Unit 9: API Gateway

## Unit Context
- **Stories**: A1 (soạn script), B1 (chọn plugin), C1 (khởi chạy render), C6 (theo dõi tiến trình — SSE), E1 (xác thực YouTube OAuth) — Gateway là lớp routing chung cho GUI (Unit 10), không tự chứa business logic
- **Scope**: Routing REST/SSE cho GUI, proxy tới Orchestrator Service (Unit 8), Content Plugin Service (Unit 2, `GET /plugins`), Publisher Service (Unit 7, OAuth flow)
- **Depends on**: Unit 8, Unit 2, Unit 7 (đều đã Code Generation xong)
- **Tech stack (đã chốt ở ADR-0009)**: Node.js (Express hoặc Fastify — quyết định cụ thể ở NFR Requirements stage của unit này)

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `module-structure.md`
- [x] Tạo `dependency-injection.md`
- [x] Tạo `interface-contracts.md`
- [x] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Dependency Direction (BẮT BUỘC)
Gateway không có business logic phức tạp (chỉ routing/proxy/SSE fan-out) — khác các unit nghiệp vụ trước dùng Hexagonal đầy đủ.

A) 💡 Suggested: **Layered đơn giản** (không phải Hexagonal/Clean đầy đủ — không tương xứng với 1 service chỉ làm reverse-proxy): `routes/` (định nghĩa endpoint, map tới handler) → `handlers/` (logic proxy: forward request, transform response nếu cần) → `clients/` (HTTP client tới từng service downstream + AMQP client cho SSE). Dependency direction: `routes` → `handlers` → `clients`, không ngược lại
   - ✅ Strengths: đúng bản chất — Gateway là "dumb pipe" có chủ đích, không cần layer trừu tượng hóa domain không tồn tại
   - ⚠️ Trade-offs: khác kiến trúc Hexagonal của Unit 2–8 — nhưng phù hợp vì Gateway không có domain logic để bảo vệ bằng ports/adapters

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Dependency Injection (BẮT BUỘC)
A) 💡 Suggested: Constructor injection thủ công qua factory function đơn giản (Node.js idiom, không dùng DI container như `InversifyJS`/`tsyringe` — quy mô nhỏ không cần). Mỗi client (Orchestrator/ContentPlugin/Publisher HTTP client, RabbitMQ SSE consumer) được construct 1 lần ở composition root (`app.js`/`server.js`), truyền vào handler qua closure hoặc `req.app.locals`
   - ✅ Strengths: đơn giản, đúng idiom Node.js/Express phổ biến, nhất quán tinh thần "constructor injection thủ công" đã dùng ở Unit 8
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Routing Table — Xác nhận đầy đủ endpoint cần proxy
A) 💡 Suggested:
| Path (GUI-facing) | Method | Proxy tới | Ghi chú |
|---|---|---|---|
| `/v1/plugins` | GET | Content Plugin Service | passthrough |
| `/v1/sagas/render` | POST | Orchestrator Service | passthrough |
| `/v1/sagas/publish` | POST | Orchestrator Service | passthrough |
| `/v1/projects/{project_id}` | GET | Orchestrator Service | passthrough |
| `/v1/projects/{project_id}/retry` | POST | Orchestrator Service | passthrough |
| `/v1/auth/youtube/start` | GET | Publisher Service | passthrough (302 redirect) |
| `/v1/auth/youtube/callback` | GET | Publisher Service | passthrough |
| `/v1/progress/{project_id}` | GET (SSE) | Gateway tự xử lý (không proxy) | Gateway là AMQP consumer của `progress.fanout`, fan-out qua SSE theo `project_id` — KHÔNG phải HTTP proxy vì downstream không có REST progress endpoint nào (progress chỉ đến qua AMQP) |
   - ✅ Strengths: khớp đầy đủ `integration-boundaries.md` + interface-contracts.md của Unit 2/7/8, không thiếu endpoint nào GUI (Unit 10) cần
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: SSE Fan-out Mechanism (đặc thù unit này)
Gateway phải là AMQP consumer của `progress.fanout` (Unit 1's topology) và fan-out tới đúng SSE client đang theo dõi `project_id` tương ứng.

A) 💡 Suggested: Gateway declare 1 exclusive queue riêng bind vào `progress.fanout` lúc start (mirror cách fanout exchange hoạt động — mọi Gateway instance nhận toàn bộ progress message). In-memory `Map<project_id, Response[]>` giữ danh sách SSE connection đang mở theo `project_id`; khi nhận AMQP message, lookup map, ghi `data: <json>\n\n` vào từng response tương ứng. Khi client đóng kết nối (`req.on('close')`), xóa khỏi map
   - ✅ Strengths: đơn giản, đủ cho 1 instance Gateway (Scaling: Fixed 1, nhất quán các unit khác), không cần Redis pub/sub hay cơ chế phức tạp
   - ⚠️ Trade-offs: state SSE connection chỉ tồn tại trong memory 1 instance — nếu Gateway restart, client phải tự reconnect (hành vi chuẩn của SSE, browser tự retry)

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Interface Contracts — API Versioning
A) 💡 Suggested: Gateway không tự định nghĩa version riêng — chỉ forward nguyên trạng prefix `/v1/` đã có ở từng service downstream (URI versioning, nhất quán toàn hệ thống, `interface-contracts.md` của Unit 2/7/8 đều dùng `/v1/`). SSE endpoint `/v1/progress/{project_id}` cũng theo cùng convention
   - ✅ Strengths: nhất quán, không thêm tầng versioning phức tạp không cần thiết ở Gateway
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Distributed Tracing & Correlation (BẮT BUỘC — Gateway là entry point của mọi request)
A) 💡 Suggested: Gateway SINH `X-Request-ID` mới (UUID) cho MỌI request đến từ GUI nếu chưa có header này (Gateway là entry point đầu tiên của hệ thống REST — khác Unit 7/8 chỉ nhận lại header đã có), forward header này nguyên trạng tới service downstream. Log mỗi request/response ở Gateway kèm `X-Request-ID`. Với SSE: `project_id` đóng vai trò correlation chính (không có khái niệm request/response 1-lần)
   - ✅ Strengths: đúng vai trò entry point — Gateway là nơi correlation ID bắt đầu, nhất quán các unit downstream đã "mirror Gateway's convention" chờ sẵn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Sequence Flows cần thiết kế
A) 💡 Suggested: 4 flow chính — (1) REST proxy đơn giản (GET /plugins ví dụ điển hình, áp dụng chung cho mọi passthrough endpoint), (2) OAuth redirect flow (GET /v1/auth/youtube/start → 302 → Google → callback), (3) SSE subscribe + AMQP fan-out (client mở `/v1/progress/{id}`, Gateway consume `progress.fanout`, forward theo `project_id`), (4) Lỗi downstream (service nghiệp vụ trả lỗi/timeout → Gateway forward status code + error body nguyên trạng, không transform)
   - ✅ Strengths: bao phủ đủ pattern (không cần vẽ riêng từng endpoint passthrough vì logic giống hệt nhau)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: State Management
A) 💡 Suggested: Gateway KHÔNG có database riêng — hoàn toàn stateless ngoại trừ in-memory SSE connection map (Question 4, tồn tại tạm thời trong process, không persist). Không cần Postgres/Redis cho Gateway
   - ✅ Strengths: đơn giản, đúng bản chất "dumb pipe", giảm hạ tầng cần quản lý
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
