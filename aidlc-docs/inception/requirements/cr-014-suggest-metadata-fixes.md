# CR-014 — Sửa lỗi "Gợi ý AI" (P0, bug)

## Date
2026-09-09

## Stage
Requirements Analysis → Code Generation

## Triệu chứng Creator báo
Bấm **Gợi ý AI (tiêu đề, mô tả, tags)** ở màn kết quả:

```
TypeError: Cannot read properties of null (reading 'join')
```

## Điều tra
Gọi thẳng endpoint trên project thật:

```json
{"title":"","description":"Nếu video hữu ích, hãy đăng ký kênh...","tags":null}
```

Không phải một lỗi mà là **ba lỗi chồng nhau**, phát hiện lần lượt:

**D1 — `tags: null` phá GUI.** `normalizeTags` trả `nil` khi không đọc được tags.
Trong Go, slice `nil` serialize thành JSON `null`, không phải `[]` — phá vỡ hợp
đồng "tags là mảng". `PublishForm.tsx:64` gọi `suggestion.tags.join(", ")` trên
`null` → nổ.

**D2 — Nội dung vô dụng vẫn trả về 200.** `Suggest` chỉ kiểm tra output có phải
JSON hợp lệ hay không, không kiểm tra có dùng được không. `title: ""` đi qua trót
lọt. Nếu D1 được sửa mà không sửa D2, Creator sẽ nhận ô tiêu đề trống một cách
im lặng — và có thể đăng luôn như thế.

**D3 (gốc rễ thật) — prompt tràn context.** Toàn bộ script được nhét vào prompt.
Đo trên DB:

| Project | Độ dài script | Kết quả trước khi sửa |
|---|---|---|
| a4157e66… | 17.567 ký tự | luôn thất bại |
| 2e05ec9d… | 12.750 ký tự | vừa đủ chạy |
| e2e-* | 259–484 ký tự | luôn chạy tốt |

Ollama mặc định `num_ctx = 2048` token. Script video 10 phút vượt xa mức đó, model
trả về object rỗng. Đây là lỗi **tất định theo độ dài script**, nên retry không
cứu được — và giải thích vì sao chỉ một số project bị.

## Functional Requirements

### FR38 — `tags` luôn là mảng
- **FR38.1**: `normalizeTags` KHÔNG BAO GIỜ được trả `nil`; không đọc được thì trả
  `[]string{}`.
- **FR38.2**: Bao gồm cả trường hợp `json.Unmarshal` vào `[]string` thành công
  nhưng cho ra `nil` (khi model gửi `"tags": null`).
- **FR38.3**: GUI PHẢI chuẩn hoá tại biên (`Array.isArray`) — đây là payload do
  model sinh ra đi qua ranh giới service, không nên tin tưởng hình dạng.

### FR39 — Output vô dụng phải báo lỗi
- **FR39.1**: `title` rỗng sau khi trim ⇒ coi là thất bại, raise lỗi. Ô tiêu đề
  trống im lặng nguy hiểm hơn một thông báo lỗi, vì Creator có thể đăng nhầm.
- **FR39.2**: Thử lại **2 lần** (không phải 4 như TTS) — mỗi lần gọi model rất
  chậm và trình duyệt đang chờ.
- **FR39.3**: Context đã huỷ ⇒ dừng ngay, không thử lại vào một context đã chết.

### FR40 — Giới hạn độ dài script gửi cho model
- **FR40.1**: Cắt script còn tối đa **4.000 ký tự** trước khi dựng prompt.
- **FR40.2**: Cắt theo **rune**, không theo byte — tiếng Việt là multibyte, cắt
  giữa rune sẽ gửi UTF-8 hỏng cho model (cùng lý do với `truncateTitle`).
- **FR40.3**: Chọn cắt script thay vì nâng `num_ctx`: nâng context tốn RAM và thời
  gian cho **mọi** request, để giải quyết một vấn đề mà model vốn không cần cả
  script mới tránh được — tiêu đề và mô tả 2–4 câu rút ra từ phần mở đầu, nơi chủ
  đề được nêu.

## Kiểm chứng
- Go: toàn bộ test xanh, thêm 11 test (normalizeTags 4 hình dạng model hay trả,
  retry, title rỗng, context huỷ, cắt script theo rune).
- Thực tế: project 17.567 ký tự **trước** → `{"title":"","tags":null}` + GUI crash;
  **sau** → HTTP 200 kèm title và 5 tags.

## Việc tồn đọng
Chất lượng tiêu đề tiếng Việt từ `llama3.2` còn kém (sai chính tả, ghép chữ lạ).
Đây là giới hạn của model nhỏ, không phải lỗi code. Đổi `OLLAMA_MODEL` trong
`.env` sang model lớn hơn nếu cần chất lượng cao hơn.

## Liên quan
- CR-006 (metadata SEO — tính năng bị lỗi)
- CR-008 (ngôn ngữ nội dung trong prompt)
