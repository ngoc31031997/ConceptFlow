# NFR Requirements Plan — Unit 7: Publisher Service

## Unit Context
Hybrid REST (OAuth flow) + message-driven (publish_video), network-I/O-bound (YouTube upload), Postgres Inbox/Outbox + `oauth_credentials` (ADR-0013, ADR-0016).

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `nfr-requirements.md`
- [ ] Tạo `tech-stack-decisions.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack Consistency (BẮT BUỘC)
A) 💡 Suggested: Python 3.12 (ADR-0009) + FastAPI (ADR-0003, mirror Unit 2 — unit đầu tiên từ Unit 3 có lại REST endpoint) + `google-api-python-client` + `google-auth-oauthlib` (YouTube Data API, `technology-direction.md`)
   - ✅ Strengths: nhất quán hệ thống, đúng thư viện chính thức Google
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 2: Performance — Đã xác định ở Low-Level Design
A) 💡 Suggested: Xác nhận lại: upload chạy trong `ThreadPoolExecutor` (LLD Question 6), timeout `UPLOAD_TIMEOUT_SECONDS` (mặc định 600s, đọc từ env var). FastAPI event loop vẫn phục vụ OAuth callback bình thường trong lúc upload chạy nền (threadpool tách biệt)
   - ✅ Strengths: nhất quán LLD, không block REST layer
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 3: Resource Constraints — Network Bandwidth
Upload video lên YouTube tốn băng thông đáng kể (video giáo dục có thể vài trăm MB tới vài GB). Có cần giới hạn/quản lý gì ở tầng NFR không?

A) 💡 Suggested: KHÔNG giới hạn cứng ở tầng ứng dụng — để kết nối mạng cá nhân của Creator tự nhiên giới hạn tốc độ. Xử lý TUẦN TỰ (RabbitMQ consumer prefetch=1, không upload song song nhiều video cùng lúc trong 1 process, nhất quán Unit 2/3/4/5/6) — tránh cạnh tranh băng thông giữa nhiều upload đồng thời trên máy dev cá nhân
   - ✅ Strengths: đơn giản, phù hợp máy dev/kết nối mạng cá nhân
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 4: Availability
A) 💡 Suggested: Chấp nhận unavailability tạm thời — không multi-instance (nhất quán toàn hệ thống). Message ở lại queue `publisher.commands` cho tới khi service khởi động lại; REST endpoint (OAuth) đơn giản không khả dụng trong lúc service down, Creator thử lại sau
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:
a
### Question 5: Security
A) 💡 Suggested: Zero-trust validation (Functional Design Rule 1) cho `publish_video` payload; OAuth flow dùng `state` parameter chuẩn (CSRF protection, LLD's interface-contracts.md). Credential lưu plaintext trong Postgres nội bộ (ADR-0016). Không auth/rate-limit riêng cho AMQP (chỉ Orchestrator gửi command nội bộ); REST OAuth endpoint không cần auth riêng (được gọi trực tiếp bởi trình duyệt của Creator qua redirect, không phải machine-to-machine)
   - ✅ Strengths: nhất quán, đúng threat model đã xác lập ở ADR-0016
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 6: Messaging & Event Participation / Saga Participation
A) 💡 Suggested: Consumer `publish_video` (queue `publisher.commands`), producer `video_published`/`publish_failed` (qua Outbox, 1 event/command). Saga role: Participant trực tiếp, bước "Publish Video" (bước cuối của Saga Publish). Compensating action: không cần rollback — video một khi đã đăng lên YouTube không thể "hoàn tác" tự động (Creator có thể tự xóa thủ công nếu cần, ngoài phạm vi hệ thống)
   - ✅ Strengths: nhất quán
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a

### Question 7: Caching Requirements
A) 💡 Suggested: `OAuthCredential` được đọc từ Postgres mỗi lần `publish_video` (không cache in-memory) — vì token có thể được refresh bất kỳ lúc nào (Business Rule 3) và tần suất publish rất thấp (không phải hot path cần tối ưu latency đọc DB)
   - ✅ Strengths: đơn giản, tránh stale cache (token refresh không đồng bộ đúng nếu cache sai)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]:a
