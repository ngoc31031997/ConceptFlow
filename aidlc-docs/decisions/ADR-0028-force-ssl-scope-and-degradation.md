# ADR-0028: Nâng scope lên youtube.force-ssl, và suy giảm êm cho kênh chưa nối lại

## Status
Proposed

## Date
2026-09-09

## Stage
Low-Level Design (CR-015)

## Context
`captions.insert` của YouTube Data API **bắt buộc** scope
`https://www.googleapis.com/auth/youtube.force-ssl`. Scope
`youtube.upload` mà hệ thống đang xin (`oauth_flow.py` dòng 21) không đủ.

`force-ssl` rộng hơn hẳn: nó cấp quyền đọc **và ghi** gần như toàn bộ tài
khoản YouTube — sửa, xoá video, quản lý playlist, bình luận — chứ không chỉ
upload. Đây là leo thang quyền thật, không phải thủ tục.

Hai ràng buộc đi kèm:
1. Refresh token đã lưu **không tự có** scope mới. Google gắn scope vào token
   tại thời điểm consent; kênh đã nối trước CR-015 sẽ nhận 403 khi gọi
   `captions.insert`, dù `videos.insert` vẫn chạy bình thường.
2. OAuth app hiện ở chế độ **Testing** (Creator xác nhận 2026-09-09), nên thêm
   scope không cần Google verify lại. Rủi ro lớn nhất của quyết định này đã
   được loại bỏ trước khi chốt.

## Options Considered

### Option A: Không nâng scope — xuất `.srt` ra cho Creator tự upload
- What it is: assembly ghi file `.srt`, GUI cho tải về, Creator vào YouTube
  Studio nạp tay.
- Strengths: không đụng OAuth, không ai phải nối lại kênh, không có leo thang
  quyền.
- Trade-offs: biến một hệ thống tự động thành nửa thủ công, đúng ở bước cuối.
  Với kênh chạy nhiều video và nhiều kênh con (CR-012), thao tác tay này lặp
  lại mãi mãi. Nó cũng chỉ hoãn quyết định chứ không giải — ngày nào muốn tự
  động thì vẫn phải nâng scope, và lúc đó Creator phải nối lại kênh **lần thứ
  hai**.

### Option B: Nâng scope, bắt buộc nối lại toàn bộ kênh trước khi publish tiếp
- What it is: phát hiện credential thiếu scope thì chặn publish.
- Strengths: trạng thái nhất quán, không có kênh nào chạy nửa vời.
- Trade-offs: một thay đổi về phụ đề lại làm **hỏng chức năng publish** đang
  chạy tốt. Creator đang giữa chừng công việc bỗng không xuất bản được gì cho
  tới khi làm xong thủ tục OAuth cho mọi kênh.

### Option C: Nâng scope, kênh chưa nối lại vẫn publish được nhưng bỏ caption
- What it is: thêm `force-ssl` vào lần consent mới; trước khi upload thì đối
  chiếu scope đã lưu; thiếu thì vẫn `videos.insert` như cũ, bỏ qua
  `captions.insert`, và báo rõ lên GUI kênh nào cần nối lại.
- Strengths: không chức năng nào đang chạy bị hỏng; Creator nối lại từng kênh
  theo nhịp của mình; thông điệp hiện đúng lúc và nói đúng việc cần làm.
- Trade-offs: phải lưu thêm scope vào credential và có thêm một nhánh suy giảm
  để test. Trong một khoảng thời gian, các kênh không đồng nhất về năng lực.

## Decision
**Option C**, và nâng scope **một lần duy nhất** ngay trong CR-015 chứ không
tách thành hai đợt.

Option A hấp dẫn vì tránh được phiền phức, nhưng nó chỉ dời phiền phức ấy sang
tương lai và nhân đôi lên: Creator sẽ phải nối lại kênh vào một ngày nào đó,
mà lúc ấy số kênh đã nhiều hơn hôm nay.

Chọn C thay vì B vì phạm vi thiệt hại không tương xứng: caption là tính năng
mới, publish là tính năng đang chạy. Một tính năng mới thiếu điều kiện không có
quyền làm hỏng tính năng cũ.

Về leo thang quyền: `force-ssl` rộng hơn mức cần thiết, nhưng YouTube không cấp
scope hẹp hơn cho `captions.insert`. Đây là chấp nhận có ý thức, ghi nhận vào
nợ kỹ thuật, và giảm nhẹ bởi mô hình đe doạ local-only đã chốt ở ADR-0016.

## Consequences
- `OAuthCredential` có thêm trường `scopes`. Credential cũ đọc lên không có
  trường này ⇒ coi như chỉ có `youtube.upload`, tức là **không** suy đoán rằng
  chúng có `force-ssl`.
- GUI cần chỗ hiển thị "kênh này chưa nối lại, video sẽ không có phụ đề" ngay
  tại màn hình chọn kênh, chứ không đợi tới lúc publish xong mới báo.
- Token do consent mới cấp có quyền xoá video. Nếu file credential rò rỉ thì
  thiệt hại lớn hơn trước — ghi vào nợ kỹ thuật, gắn với ADR-0016.
- Chế độ Testing gỡ được rủi ro verification, nhưng kéo theo một ràng buộc khác:
  app thuộc kiểu **External**, nên refresh token hết hạn sau **7 ngày** và
  Creator phải nối lại mọi kênh hàng tuần. Creator chấp nhận tạm thời
  (2026-09-09). Lối thoát là chuyển app sang *In production*, nhưng điều đó lại
  đòi Google verification chính vì `force-ssl` là sensitive scope — nên việc này
  hoãn sang một CR riêng, để chỉ nộp hồ sơ **một lần** với bộ scope cuối cùng,
  thay vì verify hôm nay với `youtube.upload` rồi verify lại sau khi đổi scope.
  Đây cũng là lý do CR-015 nâng scope ngay bây giờ chứ không chờ.

## Related
- ADR-0016 (lưu credential plaintext, mô hình đe doạ local-only)
- ADR-0026 (hai tầng app/credential — nơi trường `scopes` được thêm vào)
- ADR-0027 (nhánh `track` là thứ đòi scope này)
- CR-015 FR40
