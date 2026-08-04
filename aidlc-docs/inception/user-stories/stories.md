# User Stories

**Persona**: The Creator (xem `personas.md`)
**Tổ chức**: User Journey-Based (Soạn nội dung → Cấu hình → Render → Xem kết quả → Đăng YouTube), mỗi story gắn tag `[Feature: ...]` map tới FR trong `requirements.md`.
**Định dạng Acceptance Criteria**: Gherkin (Given/When/Then)

---

## Epic A: Content Authoring (Soạn nội dung)

### Story A1 — Soạn script nội dung video trong GUI
`[Feature: GUI, Script Processing | FR2.1, FR6.1]`

Là **Creator**, tôi muốn soạn/nhập script (nội dung lời thoại + cấu trúc minh họa) trực tiếp trong GUI, để tôi không phải rời khỏi công cụ để chuẩn bị nội dung.

**Acceptance Criteria**:
```gherkin
Given tôi đang ở màn hình tạo video mới trong GUI
When tôi nhập nội dung script vào vùng soạn thảo
Then hệ thống lưu nội dung script đó gắn với dự án video hiện tại

Given tôi đã có sẵn một file script/markdown trên máy
When tôi chọn chức năng import file trong GUI
Then nội dung file được nạp vào vùng soạn thảo script
```

### Story A2 — Phân tích script thành cấu trúc scene
`[Feature: Script Processing | FR2.2]`

Là **Creator**, tôi muốn hệ thống tự động phân tích script tôi soạn thành cấu trúc scene (lời thoại + điểm cần minh họa), để tôi không phải tự tay chia nhỏ nội dung thành từng scene.

**Acceptance Criteria**:
```gherkin
Given tôi đã nhập script hợp lệ theo cú pháp được công cụ hỗ trợ
When tôi yêu cầu hệ thống phân tích script
Then hệ thống trả về danh sách các scene, mỗi scene gồm đoạn lời thoại tương ứng và loại minh họa cần sinh

Given script có cú pháp không hợp lệ (vd. thiếu cấu trúc scene bắt buộc)
When tôi yêu cầu hệ thống phân tích script
Then hệ thống hiển thị thông báo lỗi rõ ràng chỉ ra vị trí và nguyên nhân lỗi
```

---

## Epic B: Plugin & Configuration (Chọn plugin & cấu hình)

### Story B1 — Chọn content type plugin cho video
`[Feature: Plugin Architecture, GUI | FR1.1, FR1.2, FR6.1]`

Là **Creator**, tôi muốn chọn plugin nội dung (vd. "Lập trình") cho video đang tạo, để hệ thống biết cách sinh animation phù hợp với loại nội dung đó.

**Acceptance Criteria**:
```gherkin
Given tôi đang cấu hình một dự án video mới trong GUI
When tôi mở danh sách content type plugin khả dụng
Then hệ thống hiển thị plugin "Lập trình" là plugin khả dụng để chọn

Given tôi chưa chọn content type plugin nào
When tôi cố gắng tiến tới bước render
Then hệ thống yêu cầu tôi chọn một plugin trước khi tiếp tục
```

### Story B2 — Chọn loại nội dung lập trình cụ thể (thuật toán / khái niệm)
`[Feature: Programming Plugin | FR1.2]`

Là **Creator**, tôi muốn chỉ định trong plugin "Lập trình" rằng scene của tôi là minh họa thuật toán/cấu trúc dữ liệu hay khái niệm lập trình tổng quát, để hệ thống sinh đúng loại animation tương ứng.

**Acceptance Criteria**:
```gherkin
Given tôi đã chọn plugin "Lập trình" và đang cấu hình một scene cụ thể
When tôi chỉ định loại nội dung là "thuật toán/cấu trúc dữ liệu"
Then hệ thống sử dụng bộ animation minh họa thuật toán khi render scene đó

Given tôi đã chọn plugin "Lập trình" và đang cấu hình một scene cụ thể
When tôi chỉ định loại nội dung là "khái niệm lập trình tổng quát"
Then hệ thống sử dụng bộ animation minh họa khái niệm lập trình khi render scene đó
```

### Story B3 — Sử dụng scene minh họa code có syntax highlight
`[Feature: Animation Generation | FR3.1]`

Là **Creator**, tôi muốn chèn một đoạn code vào scene và có nó hiển thị với syntax highlight trong animation, để người xem dễ theo dõi mã nguồn đang minh họa.

**Acceptance Criteria**:
```gherkin
Given tôi đang cấu hình một scene thuộc plugin "Lập trình"
When tôi nhập một đoạn code và chỉ định ngôn ngữ lập trình của đoạn code đó
Then animation sinh ra hiển thị đoạn code với syntax highlight tương ứng ngôn ngữ đã chỉ định
```

### Story B4 — Chọn ngôn ngữ giọng đọc cho video
`[Feature: TTS, GUI | FR4.2, FR6.1]`

Là **Creator**, tôi muốn chọn ngôn ngữ giọng đọc (Tiếng Việt hoặc Tiếng Anh) cho video, để phù hợp với đối tượng người xem của từng video cụ thể.

**Acceptance Criteria**:
```gherkin
Given tôi đang cấu hình dự án video
When tôi chọn ngôn ngữ giọng đọc là "Tiếng Việt"
Then giọng đọc được sinh ra ở bước render sử dụng TTS engine tiếng Việt

Given tôi đang cấu hình dự án video
When tôi chọn ngôn ngữ giọng đọc là "Tiếng Anh"
Then giọng đọc được sinh ra ở bước render sử dụng TTS engine tiếng Anh
```

---

## Epic C: Rendering (Sinh video)

### Story C1 — Khởi chạy render video từ GUI
`[Feature: GUI, Animation Generation | FR3.2, FR6.1]`

Là **Creator**, tôi muốn bấm nút để bắt đầu render toàn bộ video từ cấu hình đã thiết lập, để tôi không cần chạy lệnh dòng lệnh thủ công.

**Acceptance Criteria**:
```gherkin
Given tôi đã hoàn tất soạn script, chọn plugin và cấu hình ngôn ngữ
When tôi bấm nút "Render video" trong GUI
Then hệ thống bắt đầu quá trình render và chuyển sang trạng thái "đang xử lý"
```

### Story C2 — Sinh giọng đọc tự động từ script
`[Feature: TTS | FR4.1, FR4.2]`

Là **Creator**, tôi muốn hệ thống tự động sinh giọng đọc từ nội dung lời thoại trong script bằng TTS mã nguồn mở/offline, để tôi không cần tự thu âm.

**Acceptance Criteria**:
```gherkin
Given script đã được phân tích thành các scene với lời thoại tương ứng
When quá trình render bắt đầu
Then hệ thống sinh file audio giọng đọc cho từng scene bằng TTS engine offline đã cấu hình, không gọi API cloud nào
```

### Story C3 — Đồng bộ animation với thời lượng giọng đọc
`[Feature: TTS, Animation Generation | FR4.3]`

Là **Creator**, tôi muốn animation của mỗi scene tự động khớp thời lượng với đoạn giọng đọc tương ứng, để video không bị lệch giữa hình ảnh và âm thanh.

**Acceptance Criteria**:
```gherkin
Given một scene đã có audio giọng đọc được sinh ra với thời lượng xác định
When hệ thống render animation cho scene đó
Then thời lượng animation của scene được điều chỉnh để khớp với thời lượng audio giọng đọc, sai lệch không quá ngưỡng cho phép được cấu hình
```

### Story C4 — Ghép animation và audio thành video hoàn chỉnh
`[Feature: Video Assembly | FR5.1]`

Là **Creator**, tôi muốn hệ thống tự động ghép toàn bộ animation clip và audio giọng đọc của các scene thành một file video .mp4 hoàn chỉnh, để tôi có sản phẩm cuối cùng sẵn sàng sử dụng.

**Acceptance Criteria**:
```gherkin
Given tất cả scene trong dự án đã được render animation và sinh audio thành công
When bước ghép video được thực thi
Then hệ thống xuất ra một file .mp4 duy nhất chứa toàn bộ nội dung video theo đúng thứ tự scene
```

### Story C5 — Thêm nhạc nền tùy chọn cho video
`[Feature: Video Assembly | FR5.2]`

Là **Creator**, tôi muốn tùy chọn thêm một file nhạc nền vào video, để video sinh động hơn khi cần.

**Acceptance Criteria**:
```gherkin
Given tôi đang cấu hình dự án video và có một file nhạc nền hợp lệ
When tôi bật tùy chọn nhạc nền và chọn file nhạc
Then video hoàn chỉnh sau khi ghép (Story C4) có chứa nhạc nền đó ở âm lượng không lấn át giọng đọc

Given tôi không bật tùy chọn nhạc nền
When video được ghép hoàn chỉnh
Then video không chứa nhạc nền, chỉ có giọng đọc và âm thanh animation (nếu có)
```

### Story C6 — Theo dõi tiến trình render trong GUI
`[Feature: GUI | FR6.1]`

Là **Creator**, tôi muốn thấy tiến trình render (scene nào đang xử lý, phần trăm hoàn thành) trong GUI, để tôi biết còn bao lâu thì video xong mà không cần đoán.

**Acceptance Criteria**:
```gherkin
Given quá trình render đang chạy (bắt đầu từ Story C1)
When tôi quan sát màn hình GUI
Then GUI hiển thị scene hiện tại đang được xử lý và tiến trình tổng thể của quá trình render

Given quá trình render gặp lỗi ở một scene cụ thể
When lỗi xảy ra
Then GUI hiển thị rõ scene nào lỗi và thông tin lỗi, dừng hoặc cho phép tiếp tục theo lựa chọn của Creator
```

---

## Epic D: Preview & Review (Xem kết quả)

### Story D1 — Xem trước video sau khi render xong
`[Feature: GUI | FR6.1]`

Là **Creator**, tôi muốn xem trước video ngay trong GUI sau khi render hoàn tất, để tôi kiểm tra chất lượng trước khi quyết định đăng tải.

**Acceptance Criteria**:
```gherkin
Given quá trình render đã hoàn tất và file video .mp4 đã được tạo ra
When tôi mở màn hình kết quả trong GUI
Then GUI phát video trực tiếp trong trình phát tích hợp mà không cần mở ứng dụng ngoài
```

---

## Epic E: Publishing (Đăng tải)

### Story E1 — Xác thực tài khoản YouTube qua OAuth
`[Feature: YouTube Publishing | FR7.3]`

Là **Creator**, tôi muốn xác thực tài khoản YouTube của mình một lần qua OAuth, để hệ thống có quyền đăng video thay tôi ở các lần sau mà không cần đăng nhập lại mỗi lần.

**Acceptance Criteria**:
```gherkin
Given tôi chưa từng xác thực tài khoản YouTube với công cụ
When tôi bấm "Kết nối tài khoản YouTube" trong GUI
Then hệ thống dẫn tôi qua luồng OAuth 2.0 của Google và lưu credential hợp lệ trên máy local sau khi tôi cấp quyền

Given tôi đã xác thực thành công trước đó
When tôi mở lại công cụ
Then hệ thống sử dụng credential đã lưu mà không yêu cầu tôi đăng nhập lại
```

### Story E2 — Cấu hình metadata video trước khi đăng
`[Feature: YouTube Publishing, GUI | FR7.2]`

Là **Creator**, tôi muốn nhập tiêu đề, mô tả, tag và chọn chế độ hiển thị (public/unlisted/private) cho video trước khi đăng, để video được đăng đúng như tôi mong muốn.

**Acceptance Criteria**:
```gherkin
Given video đã render xong và tôi đang ở màn hình đăng tải
When tôi điền tiêu đề, mô tả, tag và chọn chế độ hiển thị
Then hệ thống lưu các giá trị này để sử dụng khi thực hiện upload

Given tôi chưa điền tiêu đề video (trường bắt buộc)
When tôi cố gắng bấm nút đăng tải
Then hệ thống ngăn việc đăng tải và thông báo tiêu đề là bắt buộc
```

### Story E3 — Tự động đăng video lên YouTube
`[Feature: YouTube Publishing | FR7.1]`

Là **Creator**, tôi muốn bấm một nút để tự động tải video lên kênh YouTube của tôi với metadata đã cấu hình, để tôi không phải thao tác thủ công trên website YouTube.

**Acceptance Criteria**:
```gherkin
Given tôi đã xác thực YouTube (Story E1) và cấu hình metadata (Story E2)
When tôi bấm nút "Đăng lên YouTube"
Then hệ thống tải video .mp4 lên kênh YouTube của tôi kèm đúng metadata đã cấu hình

Given quá trình upload thất bại (vd. mất kết nối mạng)
When lỗi xảy ra trong lúc upload
Then GUI hiển thị thông báo lỗi rõ ràng và cho phép tôi thử lại mà không phải cấu hình lại metadata từ đầu
```

---

## Epic F: Platform & Runtime (Nền tảng chạy)

### Story F1 — Chạy toàn bộ công cụ qua Docker trên máy cá nhân
`[Feature: Containerized Runtime | FR8.1]`

Là **Creator**, tôi muốn khởi chạy toàn bộ công cụ (GUI + pipeline render + TTS) chỉ bằng một lệnh Docker, để tôi không phải cài đặt thủ công Manim, LaTeX, ffmpeg và các dependency khác trên máy.

**Acceptance Criteria**:
```gherkin
Given tôi đã cài Docker trên máy cá nhân và chưa cài bất kỳ dependency nào khác của công cụ
When tôi chạy lệnh khởi động container được cung cấp bởi công cụ
Then GUI khả dụng và tôi có thể tạo, render và xem video hoàn chỉnh mà không cần cài thêm bất kỳ phần mềm nào trên máy host
```

---

## Traceability Summary

| FR | Story |
|---|---|
| FR1.1 | B1 |
| FR1.2 | B1, B2, B3 |
| FR1.3 | (Kiến trúc — xác nhận ở Low-Level Design, không có story GUI trực tiếp) |
| FR2.1 | A1 |
| FR2.2 | A2 |
| FR3.1 | B3 |
| FR3.2 | C1 |
| FR4.1 | C2 |
| FR4.2 | B4, C2 |
| FR4.3 | C3 |
| FR5.1 | C4 |
| FR5.2 | C5 |
| FR6.1 | A1, B1, B4, C1, C6, D1, E2 |
| FR6.2 | (Bao trùm toàn bộ Epic A-E — không có CLI bắt buộc ở bất kỳ story nào) |
| FR7.1 | E3 |
| FR7.2 | E2 |
| FR7.3 | E1 |
| FR8.1 | F1 |
