# Messaging Design — Unit 9: API Gateway

## Delivery Guarantee
At-most-once cho progress message (chấp nhận được — progress chỉ mang tính thông báo UX, không phải nguồn sự thật duy nhất; `Project.Status` ở Orchestrator luôn query được qua `GET /v1/projects/{id}`). Khác các unit khác cần at-least-once cho command/event nghiệp vụ.

## Saga Role: Không áp dụng
Gateway KHÔNG tham gia Saga — không phải orchestrator, không phải participant. Chỉ là AMQP-to-SSE bridge thuần túy cho mục đích UX.

## Consumed
| Exchange/Queue | Nội dung |
|---|---|
| `progress.fanout` (exclusive queue riêng của Gateway) | `ProgressMessage` (ADR-0017): `{project_id, step, status, scene_index?, scene_total?, error_message?}` |

## Published
Không publish gì tới RabbitMQ — Gateway không gửi command/event nghiệp vụ nào.

## Inbox/Outbox Pattern: Không áp dụng
Không có state cần đồng bộ với message gửi/nhận (Gateway không ghi database). Consumer chỉ cần ack message sau khi forward thành công qua SSE (hoặc bỏ qua nếu không có SSE client nào đang theo dõi `project_id` đó — vẫn ack, không phải lỗi).

## SSE Fan-out Detail
1. Nhận AMQP message từ `progress.fanout`.
2. Parse `project_id` từ payload.
3. Lookup `Map<project_id, Response[]>` — nếu có connection đang mở, ghi `data: <json>\n\n` vào từng response.
4. Nếu KHÔNG có connection nào theo dõi `project_id` đó (Creator chưa mở GUI hoặc đã đóng), bỏ qua — không lưu lại để replay sau (at-most-once, chấp nhận mất progress update nếu không ai đang xem).
5. Ack AMQP message ngay sau bước 3/4 (không phụ thuộc kết quả ghi SSE thành công hay không — network write lỗi tới browser không phải lý do để requeue message AMQP).
