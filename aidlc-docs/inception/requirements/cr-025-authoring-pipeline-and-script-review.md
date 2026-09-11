# CR-025 — Pipeline soạn script bốn bước và cổng review kịch bản (P1)

## Date
2026-09-11

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: Rework — đổi cách sinh script, không thêm khâu mới vào pipeline render
- **Scope estimate**: 2 unit — `web-gui` (chia prompt thành bốn bước, lưu artefact trung gian), `rendering` (mở rộng `script_lint.py` với luật kể chuyện). `orchestrator` chỉ cần chỗ lưu.
- **Complexity estimate**: Moderate. Việc khó không phải kỹ thuật mà là chọn đúng luật nào kiểm được bằng máy và luật nào chỉ nên là hướng dẫn trong prompt.

## Bối cảnh — quan sát được
Hiện tại `scriptPrompts.ts` sinh **một prompt duy nhất** và Creator dán sang AI
bên ngoài. Prompt đó yêu cầu model làm bốn việc khác loại trong một lượt:

1. hiểu chủ đề và chọn insight cần dạy,
2. thiết kế câu chuyện và viết lời thoại,
3. nghĩ cách biến kiến thức thành hình ảnh và chuyển động,
4. viết code Manim chạy được, đúng API, đúng whitelist của CR-017.

Việc (4) là việc có ràng buộc cứng nhất và tốn "sức chú ý" của model nhất — nên
nó lấn át (1)–(3). Kết quả thường gặp là code chạy được nhưng video kể chuyện
dở: mở bằng title, rồi định nghĩa, rồi bullet point, rồi ví dụ, rồi tổng kết.

**Và không có khâu nào review ý tưởng.** Hệ thống kiểm được rất nhiều thứ về
script, nhưng toàn bộ đều là kiểm **hình thức**, không kiểm **nội dung**:

| Đã kiểm | Ở đâu | Loại |
|---|---|---|
| API lạ, ngoài whitelist conceptflow | `script_lint.py` (CR-017/020) | hình thức |
| Script có chạy thật được không | lượt dry của `validate_script` (CR-020) | hình thức |
| Beat bắt buộc, thứ tự beat | `quality` validate (CR-019) | hình thức |
| Bố cục, âm thanh sau khi dựng | `qc_rules.py` (CR-021) | hình thức |
| Dàn ý có hợp ý Creator không | cổng duyệt (CR-024) | **người duyệt** |
| **Ý tưởng có đáng dạy không** | — | **không có** |
| **Câu chuyện có hook/insight không** | — | **không có** |
| **Có show hay chỉ tell** | — | **không có** |

CR-024 đã mở cổng cho người duyệt dàn ý, nhưng Creator duyệt một dàn ý **đã
được sinh ra rồi** — không có gì giúp họ biết dàn ý đó dở ở đâu so với nguyên
tắc của kênh.

## Vấn đề
1. **Một prompt làm bốn việc thì việc khó nhất lấn át ba việc còn lại.** Ràng
   buộc code đẩy chất lượng kể chuyện xuống.
2. **Không giữ lại artefact trung gian.** Khi video cần sửa phần hình ảnh, cả
   câu chuyện lẫn lời thoại bị sinh lại từ đầu — sửa một chỗ thì mọi chỗ khác
   đổi theo, và Creator mất luôn bản story họ đã thấy ổn.
3. **Nguyên tắc của kênh chỉ tồn tại trong prompt.** "Show, don't tell",
   "concrete trước abstract", "ít chữ" là những câu không ai kiểm chứng — đúng
   lớp vấn đề mà CR-021 đã mô tả về câu "bố cục nằm gọn trong khung an toàn".

## Quyết định
Chia prompt thành **bốn bước tuần tự**, mỗi bước một vai, artefact của bước
trước là đầu vào của bước sau và được **lưu lại trên project**:

```
        CHỦ ĐỀ
           ↓
┌──────────────────────┐
│ 1. STORY ARCHITECT   │  "Phải dạy gì?"
└──────────┬───────────┘
           ↓  story + lời thoại
┌──────────────────────┐
│ 2. VISUAL DIRECTOR   │  "Cho thấy nó thế nào?"
└──────────┬───────────┘
           ↓  visual storyboard
┌──────────────────────┐
│ 3. MANIM ENGINEER    │  "Code nó thế nào?"
└──────────┬───────────┘
           ↓  script .py
┌──────────────────────┐
│ 4. SCRIPT REVIEWER   │  "Ý tưởng, kịch bản và code có ổn không?"
└──────────┬───────────┘
       ┌───┴────┐
      PASS    REVISE ──→ quay lại bước 2 (Visual Director)
```

**Vẫn là copy tay sang AI bên ngoài** (đã chốt 2026-09-11), đúng kiến trúc hiện
tại: hệ thống sinh prompt, Creator dán sang model mạnh, dán kết quả về. Không
gọi API trả phí, không dùng Ollama local cho việc này — CR-014 đã cho thấy
Ollama local vỡ với context dài, và sinh code Manim đúng API là việc nặng hơn
gợi ý metadata rất nhiều.

**Bước 4 review ý tưởng, kịch bản và code — KHÔNG review frame ảnh** (đã chốt
2026-09-11). Review gồm hai phần tách bạch:
- **Phần máy kiểm được**: mở rộng `script_lint.py` với các luật kể chuyện đọc
  được từ AST (xem FR73). Chạy trong mili giây, tất định, không cần AI.
- **Phần người/AI đánh giá**: một prompt review mang story + storyboard + code
  + kết quả lint, để model bên ngoài phê ý tưởng và kịch bản. Đầu ra của nó
  quay về bước 2, không phải bước 1 — vì phần thường cần sửa là cách *cho thấy*,
  không phải chuyện *dạy gì*.

## Functional Requirements

### FR71 — Bốn bước và artefact trung gian
- **FR71.1**: Hệ thống PHẢI sinh được bốn prompt riêng, mỗi prompt đúng một vai,
  và chỉ mang thông tin vai đó cần.
- **FR71.2**: Artefact của bước 1 (story + lời thoại) và bước 2 (visual
  storyboard) PHẢI được lưu cùng project, không chỉ nằm trong clipboard.
- **FR71.3**: Bước sau PHẢI nhận artefact của bước trước tự động trong prompt,
  Creator không phải tự dán lại.
- **FR71.4**: Creator PHẢI quay lại được bước trước mà không mất artefact của
  các bước khác — đây là lý do tồn tại của FR71.2.
- **FR71.5**: Creator PHẢI bỏ qua được bước 1 và 2 để viết script trực tiếp
  (đường hiện tại vẫn phải chạy). Quy trình bốn bước là đường được khuyến nghị,
  không phải cổng chặn.
- **FR71.6**: Lời thoại sinh ở bước 1 PHẢI theo ngôn ngữ nội dung của project
  (giữ nguyên CR-008 FR21.3).
- **FR71.7**: Ngân sách beat đưa vào bước 1 PHẢI tính bằng **số từ**, không phải
  giây (giữ nguyên CR-019 FR54.1 — model đếm được từ, không đếm được giây).

### FR72 — Nguyên tắc kể chuyện trong prompt
- **FR72.1**: Prompt bước 1 PHẢI buộc model nêu rõ: câu hỏi cốt lõi, insight
  cốt lõi, ẩn dụ hình ảnh, và khoảnh khắc "aha" — trước khi viết lời thoại.
- **FR72.2**: Prompt bước 2 PHẢI yêu cầu mỗi cảnh có quan hệ trực quan với cảnh
  trước (ưu tiên `Transform`/`ReplacementTransform` hơn FadeOut-rồi-FadeIn).
- **FR72.3**: Prompt bước 2 PHẢI yêu cầu đi từ cụ thể sang trừu tượng, không mở
  đầu bằng công thức.
- **FR72.4**: Prompt PHẢI yêu cầu visual giải thích **cơ chế**, không chỉ hiện
  kết quả.
- **FR72.5**: Prompt bước 3 PHẢI giữ nguyên mọi ràng buộc kỹ thuật đang có:
  whitelist API của CR-017, `self.narrate()`/`beat()`/`chapter()` của
  CR-018/019, một Scene class, không I/O.

### FR73 — Luật kể chuyện kiểm được bằng máy
Mở rộng `script_lint.py`. Tất cả ở mức **cảnh báo**, không chặn render: đây là
nhận định về chất lượng, và một luật chất lượng chặn sai còn tệ hơn không có.
- **FR73.1**: PHẢI cảnh báo khi script **chỉ** dùng FadeIn/FadeOut mà không có
  một `Transform`/`ReplacementTransform`/`.animate` nào — dấu hiệu rõ nhất của
  slide-show thay vì kể chuyện.
- **FR73.2**: PHẢI cảnh báo khi tỉ lệ mobject dạng chữ trên tổng mobject vượt
  ngưỡng — video biến thành presentation.
- **FR73.3**: PHẢI cảnh báo khi một đoạn lời thoại dài quá ngưỡng mà không có
  animation nào giữa nó và đoạn trước (trùng ý với `static_frame` của CR-021
  nhưng bắt được **trước khi render**, tức là trước khi tốn TTS).
- **FR73.4**: PHẢI cảnh báo khi số màu dùng vượt ngưỡng — màu phải có nghĩa, và
  bảng màu đã nằm trong theme của CR-017.
- **FR73.5**: Mọi ngưỡng PHẢI cấu hình được, không hardcode (cùng lý do CR-021
  FR61.5: ngưỡng chất lượng không thể chọn đúng bằng suy luận).
- **FR73.6**: Kết quả các luật này PHẢI đi cùng đường `validation_warnings` đã
  có, để hiện ở màn duyệt dàn ý của CR-024 — không mở màn báo cáo thứ hai.

### FR74 — Prompt review và vòng REVISE
- **FR74.1**: Hệ thống PHẢI sinh được prompt review mang: story, storyboard,
  code, và kết quả lint.
- **FR74.2**: Prompt review PHẢI yêu cầu model trả về kết luận rõ ràng
  PASS/REVISE kèm lý do theo từng mục, không phải một đoạn văn chung chung.
- **FR74.3**: Khi REVISE, hệ thống PHẢI ghép nhận xét đó vào prompt bước 2, để
  Creator không phải tự cắt dán giữa hai khung.
- **FR74.4**: Số lần revise PHẢI hữu hạn và do Creator quyết định dừng — không
  có vòng lặp tự chạy (hệ quả trực tiếp của việc chọn copy tay).

## Non-goals
- **Không review frame ảnh, không VLM** (chốt 2026-09-11). Ollama trong Docker
  trên macOS không truy cập được Metal nên chạy CPU-only; model nhìn ảnh đủ tốt
  thì không chạy nổi, model chạy nổi thì phán đoán thiết kế không đáng tin.
- Không tự gọi API trả phí. Không vòng lặp tự động.
- Không chấm chất lượng **nội dung khoa học** (giải thích có đúng không). Ngoài
  tầm — đó là việc của Creator.
- Không bỏ đường viết script trực tiếp (FR71.5).
- Không đụng cơ chế `narrate()`/lượt dry của CR-018, cổng `validate_script` của
  CR-020, hay QC sau render của CR-021. CR này chỉ thêm ở phía **soạn**.

## Rủi ro
- **Bốn bước copy tay là bốn lần Creator có thể bỏ giữa.** Nếu bước 1 và 2 không
  cho lợi ích thấy được ngay, Creator sẽ bỏ qua về đường viết trực tiếp
  (FR71.5). Giảm nhẹ: artefact phải hiện ra dưới dạng đọc được, không phải một
  khối JSON.
- **Luật FR73 báo động giả.** Cùng lớp rủi ro số một của CR-021, nên cùng cách
  xử lý: tất cả ở mức cảnh báo, ngưỡng cấu hình được, siết sau khi có số liệu.
- **Lint tỉ lệ chữ/mobject có thể hiểu sai `VGroup`.** Cùng bài học CR-021: đếm
  mobject theo AST không phải đếm cái mắt thấy.

## Câu hỏi cần Creator chốt
Đã chốt: cách chạy (copy tay, 2026-09-11) và phạm vi bước 4 (review ý tưởng,
kịch bản, code — không review ảnh, 2026-09-11). Còn lại quyết ở Low-Level
Design: ngưỡng khởi đầu của các luật FR73.

## Kiểm chứng
- Unit test: từng luật FR73, có case đạt và không đạt.
- Unit test: prompt bước sau có chứa artefact bước trước (FR71.3).
- Unit test: quay lại bước 2 không làm mất story của bước 1 (FR71.4).
- Unit test: luật FR73 không bao giờ là blocking (không chặn render).
- Thủ công: chạy cùng một chủ đề qua đường một-prompt cũ và đường bốn bước, so
  chất lượng kể chuyện của video ra.

## Liên quan
- CR-008 FR21.3 (ngôn ngữ nội dung của lời thoại)
- CR-017 (whitelist API và theme — ràng buộc bước 3)
- CR-018 (`self.narrate()` — ràng buộc bước 3)
- CR-019 FR54 (beat sheet và ngân sách từ — đầu vào bước 1)
- CR-020 (`validate_script` — nơi lint FR73 chạy)
- CR-021 (QC sau render — phần bù: CR này bắt lỗi trước khi tốn TTS)
- CR-024 (cổng duyệt dàn ý — nơi cảnh báo FR73 hiện ra)
