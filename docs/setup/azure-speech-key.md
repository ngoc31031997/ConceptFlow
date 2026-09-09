# Lấy key Azure AI Speech

Dùng cho: giọng đọc Azure trong danh mục giọng (CR-011, ADR-0025).

**Không bắt buộc.** Bỏ trống thì mọi giọng Azure tự rơi về giọng Edge y hệt —
cùng bộ giọng neural, chỉ khác là Edge không có SLA và không có quyền thương mại.

Cần: `AZURE_SPEECH_KEY` và `AZURE_SPEECH_REGION` trong `.env`.

---

## 1. Tạo Speech resource

1. Vào https://portal.azure.com (đăng nhập tài khoản Microsoft)
2. Ô tìm kiếm trên cùng, gõ **Speech services** → chọn nó → **Create**
3. Điền:

   | Trường | Điền gì |
   |---|---|
   | Subscription | subscription bạn có (Free Trial hoặc Pay-As-You-Go đều được) |
   | Resource group | tạo mới, ví dụ `conceptflow-rg` |
   | Region | **Southeast Asia** (gần VN nhất) — xem mục 3 về cách viết vào `.env` |
   | Name | tuỳ ý, ví dụ `conceptflow-speech` |
   | Pricing tier | **Free F0** ← quan trọng |

4. **Review + create** → **Create**, đợi khoảng 1 phút

> **F0 cho gì**: 500.000 ký tự neural mỗi tháng, miễn phí, không hết hạn.
> Pipeline này ước tính dùng khoảng một nửa mức đó.
>
> Nếu không thấy tuỳ chọn **F0**, thường là vì subscription đã có sẵn một Speech
> resource bậc F0 rồi — Azure chỉ cho một cái mỗi subscription. Xoá cái cũ, hoặc
> dùng lại chính nó (mục 2 vẫn áp dụng).

## 1b. Nếu không tạo/lấy key được: kiểm tra subscription

Gặp thông báo kiểu này khi mở trang Keys hoặc khi tạo resource:

```
{"error":{"code":"ReadOnlyDisabledSubscription",
 "message":"The subscription '...' is disabled and therefore marked as read only.
 You cannot perform any write actions on this subscription until it is re-enabled."}}
```

**Đây không phải lỗi key.** Subscription Azure đang bị khoá về chế độ chỉ-đọc; lấy
key (`ListKeys`) bị tính là thao tác ghi nên bị chặn theo.

Quan trọng: kể cả moi được key ra thì cũng vô dụng — subscription bị khoá thì
resource không phục vụ request, mọi lệnh TTS sẽ lỗi. Phải mở khoá trước.

Nguyên nhân thường gặp:

| Nguyên nhân | Dấu hiệu |
|---|---|
| Hết hạn Free Trial (30 ngày / $200) | phổ biến nhất — khoá tới khi nâng Pay-As-You-Go |
| Chạm spending limit | Free Trial mặc định bật spending limit |
| Thẻ thanh toán lỗi / quá hạn | Azure có gửi email nhắc |
| Hết credit Azure for Students | nếu dùng gói sinh viên |

Cách xử lý: **Subscriptions** → mở subscription → đọc trường **Status** và banner
trên cùng, Azure ghi rõ lý do kèm nút tương ứng (**Reactivate** / **Upgrade** /
**Remove spending limit**). Thường nằm ở **Cost Management + Billing**.

Nâng lên Pay-As-You-Go **không có nghĩa là bắt đầu mất tiền**: F0 vẫn miễn phí
500.000 ký tự/tháng. Chỉ là F0 cần một subscription còn sống để tồn tại.

**Hoặc bỏ qua Azure.** Xem mục "Có thật sự cần Azure không?" ở cuối file.

## 2. Lấy key

1. Mở resource vừa tạo
2. Menu trái → **Resource Management** → **Keys and Endpoint**
3. Copy **KEY 1** (KEY 2 cũng dùng được, nó tồn tại để xoay key mà không mất dịch vụ)

## 3. Điền vào `.env`

```
AZURE_SPEECH_KEY=<KEY 1 vừa copy>
AZURE_SPEECH_REGION=southeastasia
```

⚠️ **Region phải viết dạng mã, không phải tên hiển thị.** Trang portal hiện
"Southeast Asia" nhưng giá trị cần điền là `southeastasia` — viết thường, không
dấu cách. Code ghép thẳng chuỗi này vào URL
`https://<region>.tts.speech.microsoft.com/...`, nên viết sai sẽ ra lỗi DNS hoặc
HTTP 401 chứ không phải thông báo "sai region".

Giá trị đúng nằm ngay ở trang **Keys and Endpoint**, dòng **Location/Region**,
hoặc đọc từ chuỗi Endpoint. Vài mã hay dùng:

| Tên hiển thị | Mã điền vào `.env` |
|---|---|
| Southeast Asia | `southeastasia` |
| East US | `eastus` |
| East Asia | `eastasia` |
| Japan East | `japaneast` |
| West Europe | `westeurope` |

## 4. Áp dụng

```bash
docker compose up -d tts
```

TTS đọc biến lúc khởi động, nên phải restart container thì key mới có tác dụng.

## 5. Kiểm tra key có chạy không

Chạy đúng request mà code gửi (`services/tts/adapters/tts_engines/azure_adapter.py`):

```bash
KEY="<key của bạn>"
REGION="southeastasia"

curl -v "https://${REGION}.tts.speech.microsoft.com/cognitiveservices/v1" \
  -H "Ocp-Apim-Subscription-Key: ${KEY}" \
  -H "Content-Type: application/ssml+xml" \
  -H "X-Microsoft-OutputFormat: riff-24khz-16bit-mono-pcm" \
  -H "User-Agent: ConceptFlow" \
  -d '<speak version="1.0" xml:lang="vi-VN"><voice name="vi-VN-NamMinhNeural">Xin chào</voice></speak>' \
  --output /tmp/test.wav

# Đúng thì file nghe được:
afplay /tmp/test.wav
```

Đọc kết quả:

| Kết quả | Nghĩa là |
|---|---|
| `HTTP 200` + file WAV nghe được | ✅ xong |
| `HTTP 401` / `403` | sai key, hoặc **sai region** (key chỉ hợp lệ ở đúng region tạo ra nó) |
| `Could not resolve host` | sai region — viết sai chính tả hoặc còn dấu cách |
| `HTTP 429` | hết 500k ký tự trong tháng |

## 6. Xác nhận trong ứng dụng

Vào màn tạo video, mở danh sách giọng. Nếu key đúng, mỗi giọng sẽ có **hai** mục
với badge engine khác nhau (Edge và Azure). Nếu key sai hoặc thiếu, danh mục chỉ
hiện giọng Edge — hệ thống cố ý không liệt kê giọng của engine chưa cấu hình, để
bạn không chọn nhầm rồi thắc mắc sao nghe vẫn thế.

## Có thật sự cần Azure không?

Không, trong hầu hết trường hợp.

Azure và Edge phát **cùng một bộ giọng neural** — `vi-VN-NamMinhNeural`,
`vi-VN-HoaiMyNeural` là y hệt nhau, vì Edge dùng chính bộ giọng Azure bán. Nghe
không phân biệt được. Khác biệt nằm ở giấy tờ, không nằm ở âm thanh:

| | Edge (mặc định) | Azure F0 |
|---|---|---|
| Chất lượng giọng | giống hệt | giống hệt |
| Cần tài khoản | không | có |
| SLA | không | có |
| Quyền thương mại | không | có |
| Endpoint | không có tài liệu chính thức | API chính thức |

Nên đi lấy key Azure khi bạn cần **quyền thương mại** hoặc **SLA**, hoặc muốn có
engine thứ hai để dự phòng khi Edge chết (ADR-0024 ghi nhận đây là rủi ro thật).
Làm video cá nhân thì Edge là đủ, và không cần cấu hình gì cả.

## Xoay key khi bị lộ

**Keys and Endpoint** → **Regenerate Key1**. Key cũ chết ngay lập tức. Nếu đang
chạy production, đổi sang KEY 2 trước, cập nhật `.env`, restart, rồi mới
regenerate KEY 1.
