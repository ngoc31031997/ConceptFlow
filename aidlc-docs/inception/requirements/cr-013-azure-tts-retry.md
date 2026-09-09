# CR-013 — Retry có backoff cho AzureTTSAdapter (P1)

## Date
2026-09-09

## Stage
Requirements Analysis → Code Generation

## Intent Analysis
- **Request type**: Bug fix (độ tin cậy), không đổi domain
- **Scope estimate**: Một file — `services/tts/adapters/tts_engines/azure_adapter.py`
- **Complexity estimate**: Simple — có tiền lệ trực tiếp là `EdgeTTSAdapter` (CR-010)

## Bối cảnh — đo được, không phải phỏng đoán
Ngay sau khi Creator kích hoạt lại subscription Azure (2026-09-09), đo 20 lần gọi
liên tiếp tới endpoint TTS với cùng một key:

```
200 200 401 200 200 200 200 200 401 200 200 200 200 200 401 401 200 200 200 200
→ 16/20 thành công, 4/20 lỗi HTTP 401
```

Lỗi rải rác chứ không dồn ở đầu chuỗi, nên **không phải** độ trễ kích hoạt còn
sót. Các response lỗi có `server: istio-envoy`, `content-length: 0`, **không có
body JSON** — khác hẳn 401 "sai key" thật, vốn trả JSON nêu rõ lý do. Dấu hiệu
này chỉ tới việc gateway của Azure chưa đồng bộ trạng thái subscription giữa các
instance, tức là lỗi **tạm thời**.

## Vấn đề
`EdgeTTSAdapter` có retry với backoff (CR-010: burst 8 scene chỉ đạt 1/8 nếu
không retry, 8/8 sau khi có). `AzureTTSAdapter` **không có** — nó cố ý giao việc
fallback cho `RoutingTTSEngine`.

Hệ quả ở tỉ lệ hỏng 20%: một video 8 scene có trung bình 1–2 scene âm thầm rơi
về Edge. Creator **không nghe ra** vì hai engine phát cùng bộ giọng neural, và
cảnh báo chỉ nằm trong log container.

Điều đó xoá đúng thứ khiến Azure được chọn. Creator dùng Azure để có **quyền
thương mại và SLA** (ADR-0025); phần rơi về Edge không có hai thứ đó — và Creator
không biết phần nào đã rơi. Fallback là lưới an toàn cho sự cố thật, không phải
thứ để hấp thụ một lỗi tạm thời mà retry xử lý được.

## Functional Requirements

### FR37 — Retry có backoff
- **FR37.1**: `AzureTTSAdapter` PHẢI thử lại tối đa `MAX_ATTEMPTS` lần với backoff
  tuyến tính, theo đúng khuôn `EdgeTTSAdapter` — cùng repo, cùng vấn đề, không có
  lý do để hai adapter hành xử khác nhau.
- **FR37.2**: PHẢI phân biệt lỗi **tạm thời** (đáng thử lại) và lỗi **cố định**
  (thử lại là lãng phí):

  | Tình huống | Xử lý | Vì sao |
  |---|---|---|
  | 401, 403 | thử lại | chính là lỗi đo được ở trên; key sai thật thì 4 lần cũng hỏng nhanh |
  | 429 | thử lại, tôn trọng `Retry-After` | hết hạn mức tức thời, Azure nói rõ chờ bao lâu |
  | 408, 5xx | thử lại | lỗi phía server |
  | timeout, lỗi mạng | thử lại | không có phản hồi để kết luận |
  | 400 và 4xx còn lại | **raise ngay** | SSML sai / voice không tồn tại là tất định, thử lại chỉ tốn thời gian |

- **FR37.3**: Mỗi lần thử PHẢI có trần thời gian riêng, và trần tổng PHẢI dài hơn
  tổng mọi lần thử cộng backoff. Trần tổng hiện tại là 60s cố định dùng cho cả
  hai vai trò; giữ nguyên sẽ cắt ngang vòng retry — đúng cái bẫy mà comment trong
  `EdgeTTSAdapter` đã cảnh báo.
- **FR37.4**: Mỗi lần thử hỏng PHẢI ghi log ở mức warning kèm số lần thử và
  nguyên nhân, để về sau còn phân biệt được "Azure chập chờn" với "Azure chết".
- **FR37.5**: Sau khi hết lượt thử, PHẢI raise `TTSEngineError` như cũ. CR này
  **không** đụng tới cơ chế fallback của `RoutingTTSEngine` — chỉ làm cho fallback
  hiếm khi phải chạy.

## Non-goals
- Không bỏ fallback sang Edge — nó vẫn là lưới an toàn cho sự cố thật.
- Không thêm retry cho `GoogleTTSAdapter` (nhánh đang ngủ, không có dữ liệu đo).
- Không đưa cảnh báo fallback lên GUI — đáng làm, nhưng là việc khác.

## Kiểm chứng
- Unit test cho: thành công ngay lần đầu; thành công sau vài lần hỏng; hết lượt
  thử thì raise; 400 raise ngay không thử lại; 429 tôn trọng `Retry-After`.
- Đo lại thực tế với key của Creator sau khi triển khai.

## Liên quan
- ADR-0024 (Edge làm engine nền, và retry của nó)
- ADR-0025 (Azure là lựa chọn riêng vì quyền thương mại + SLA — thứ CR này bảo vệ)
- CR-010, CR-011
