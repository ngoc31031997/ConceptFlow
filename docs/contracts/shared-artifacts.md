# Hợp đồng volume `shared_artifacts`

Nguồn sự thật cho cây thư mục và **service nào ghi thư mục nào** (CR-040 FR114).
Mỗi service vẫn giữ helper đường dẫn riêng trong `adapters/storage/artifact_paths.py`;
`tests/contracts/test_shared_artifacts_contract.py` kiểm tra các helper đó khớp với tài liệu này.

Mount tại `/shared` (`/shared:ro` với publisher).

## Cây thư mục

| Đường dẫn | Chủ ghi | Ghi chú |
|---|---|---|
| `/shared/{project_id}/audio/{scene}_{voice}_{hash}.wav` | tts | |
| `/shared/{project_id}/audio/{line}_{cache_key}.wav` | tts | giọng từng câu của kịch bản (`b1-2` là câu `b1.2`, dấu chấm đổi thành gạch ngang); `cache_key` là 12 ký tự đầu của sha256 (giọng, chữ), nên cùng chữ và giọng thì dùng lại file; api-gateway phát qua `/v1/projects/{id}/audio/{file}` |
| `/shared/{project_id}/video/rendered.mp4` | rendering | video câm |
| `/shared/{project_id}/video/timing.json` | rendering | sidecar thời gian |
| `/shared/{project_id}/video/final.mp4` | video-assembly | |
| `/shared/{project_id}/video/final.srt` | video-assembly | |
| `/shared/{project_id}/thumbnail/` | api-gateway | upload của Creator |
| `/shared/{project_id}/music/` | api-gateway | upload của Creator |
| `/shared/channel-assets/{kind}/{quality}/normalized.mp4`, `with_music_v{n}.mp4` | video-assembly | không thuộc project nào |
| `/shared/channel-assets/{kind}/…` (file upload gốc) | api-gateway | không thuộc project nào |
| `/shared/stills/{hash}.png` | rendering | ảnh mẫu của ngôn ngữ hình, tên theo hash nội dung; `POST /v1/stills/prune` dọn ảnh không còn ai trỏ tới; api-gateway phát qua `/v1/stills/{file}` |

Publisher chỉ đọc.

## Xoá project

Không service nào được xoá thư mục của service khác, và không ai xoá cả `/shared/{project_id}`.
Orchestrator điều phối saga xoá (`DELETE /v1/projects/{id}` → 202):

1. Từ chối 409 nếu project có bước đang chạy (`IsInFlight`); nếu không, đặt `deleting`.
2. Gửi `purge_project_artifacts` tới `tts`, `rendering`, `video_assembly`. Mỗi service chỉ xoá các đường dẫn
   **của mình** trong bảng trên (rồi `rmdir` thư mục cha nếu đã rỗng), rồi trả `artifacts_purged`
   (hoặc `purge_failed`). Lệnh idempotent.
3. api-gateway xoá `thumbnail/` và `music/` của chính nó khi nhận 202.
4. Khi cả ba service đã trả `artifacts_purged`, orchestrator xoá dòng project. Nếu một service lỗi,
   project ở lại `deleting`; gọi DELETE lần nữa để phát lại lệnh.

Lý do thứ tự: không có bước nào chạy khi project đã `deleting`, nên không có worker nào có thể
ghi lại file sau khi dọn.
