# CR-072 — Application Design

## Date
2026-10-13 (cập nhật 2026-10-17: Tiền kỳ · Nội dung, Creator duyệt cùng ngày)

Gom bốn tài liệu: [`components.md`](components.md), [`component-methods.md`](component-methods.md), [`services.md`](services.md), [`component-dependency.md`](component-dependency.md). Đầu vào: yêu cầu `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md`, kế hoạch `aidlc-docs/inception/plans/cr-072-workflow-plan.md`, câu trả lời AD-1..AD-9 trong `aidlc-docs/inception/plans/cr-072-application-design-plan.md`.

**Cập nhật 2026-10-17.** Creator thiết kế lại Tiền kỳ tới kịch bản thành bảy bước (FR-27..FR-32, mockup bản 8 đã chốt bước 1–7). Thiết kế thêm: mười một thành phần ở authoring-service cho các bước đó (`CreativeBrief`, `ResearchDesk`, `SourceVerifier`, `SourceFetcher`, `ResearchSource`, `StoryFrame`, `BeatSheet`, `ScriptWriter`, `ScriptReview`, `HandoffNotes`, `ScriptVersions`; `ArchetypeAdvisor` chuyển sang bước Câu chuyện), đường ra Internet để kiểm nguồn (ADR-0043), phiên bản kịch bản đã chốt làm đầu vào của mọi bước từ 8 trở đi (ADR-0044), luồng 21 bước (ADR-0042, thay ADR-0034 phần luồng). Khung đơn vị dời sang U3a; `SynthesizeLines`, `DraftVoiceDispatcher` và cách phát file chung dời sang U3e (người dùng đầu tiên: nghe thử ở Viết kịch bản).

## Hình dạng hệ thống sau CR-072

- **Không thêm service.** Tám service giữ nguyên vai trò (ADR-0029). authoring-service lớn lên nhiều nhất: giữ mọi đơn vị của tiền kỳ và sản xuất, ngôn ngữ hình, bộ mỹ thuật, thư viện nhân vật, từ điển phát âm.
- **Một khung sửa chung cho mọi bước** (ADR-0035): đơn vị có trạng thái, ghi chú, dấu đầu vào. Sửa một đơn vị chỉ làm các đơn vị phụ thuộc thành "cần làm lại". Mỗi loại đơn vị cắm vào khung bằng một `UnitGenerator`.
- **Ngôn ngữ hình** (ADR-0036): luật ở DB có phiên bản, rig là kit Remotion build sẵn, đúng một ngôn ngữ bật, project ghim phiên bản.
- **Ảnh tĩnh là một dịch vụ chung** ở rendering (ADR-0039). Ảnh mẫu, bộ mỹ thuật, khung mẫu, khung animatic, thumbnail đều đi qua `/v1/stills`, có cache theo nội dung.
- **Animatic không phải video** (ADR-0039): trình duyệt phát ảnh tĩnh, file giọng và phụ đề theo mốc thời gian.
- **Giọng sơ bộ qua RabbitMQ** (ADR-0040), ngoài saga render. authoring-service tra từ điển phát âm trước khi gửi.
- **Render theo cảnh** (ADR-0038): mỗi cảnh một lệnh, orchestrator giữ trạng thái từng cảnh, đoạn chồng ở chỗ nối liền mạch, video-assembly ghép theo cảnh.
- **Luồng 21 bước, 3 giai đoạn, Tiền kỳ chia Nội dung và Hình ảnh** (ADR-0042, thay ADR-0034 phần luồng), định nghĩa ở ba bản có test hợp đồng như CR-051. Bước Nghiên cứu tuỳ chọn.
- **Nghiên cứu có nguồn** (ADR-0043): prompt chép ra công cụ nghiên cứu ngoài, Creator dán kết quả; authoring-service tải trang nguồn và kiểm đoạn trích, chặn mọi địa chỉ nội bộ. Nguồn tra cứu sau một cổng để cắm API tìm kiếm sau này.
- **Kịch bản có phiên bản** (ADR-0044): chốt là ảnh chụp bất biến của bước 1–7 kèm bộ bàn giao; bước 8 trở đi chỉ đọc bản đã chốt; chốt bản mới chỉ làm phần đổi thành "cần làm lại".
- **`/shared` là kho file** (ADR-0041): file nhị phân và lớn nằm trên `/shared`, DB và lệnh chỉ giữ đường dẫn; một chủ ghi mỗi thư mục; tên file theo hash nội dung; trình duyệt lấy file qua api-gateway có kiểm; dọn file mồ côi theo lịch.

## Quyết định

| Câu | Chọn | ADR |
|---|---|---|
| AD-1 Ai giữ đơn vị duyệt | A: authoring-service (tiền kỳ, sản xuất), orchestrator (hậu kỳ) | ADR-0035 |
| AD-2 Biết phần nào phải làm lại | A: dấu đầu vào trên từng đơn vị | ADR-0035 |
| AD-3 Lưu ngôn ngữ hình | A: luật ở DB, rig là kit Remotion | ADR-0036 |
| AD-4 Ảnh tĩnh | A: một API chung có cache | ADR-0039 |
| AD-5 Animatic | C: phát trong trình duyệt | ADR-0039 |
| AD-6 Giọng sơ bộ | A: lệnh RabbitMQ ngoài saga | ADR-0040 |
| AD-7 Từ điển phát âm | A: authoring-service tra trước khi gửi | ADR-0040 |
| AD-8 Render từng cảnh | A: mỗi cảnh một lệnh, orchestrator giữ trạng thái | ADR-0038 |
| AD-9 Định nghĩa luồng | A: ba bản cùng test hợp đồng | ADR-0034 |
| Câu 15 B, 21 B, C5 B (yêu cầu) | Luồng 3 giai đoạn, bộ mỹ thuật 6 mục, duyệt hai lượt | ADR-0034 |
| C3 B, C4 A (yêu cầu) | Phong cách theo thế giới, cảnh giải thích ý trong thế giới | ADR-0037 |
| Đề nghị của Creator khi duyệt thiết kế | `/shared` là kho file, dữ liệu chỉ giữ đường dẫn | ADR-0041 |
| Tiền kỳ · Nội dung (Creator, 2026-10-17) | Bảy bước, mỗi bước một chỗ duyệt, luồng 21 bước | ADR-0042 |
| Nguồn nghiên cứu (Creator, 2026-10-17) | Chép prompt ra công cụ ngoài, dán JSON, máy kiểm đoạn trích; cổng cho API tìm kiếm sau | ADR-0043 |
| "Lock" (Creator, 2026-10-17) | Đóng băng một phiên bản; sửa sau đó tạo bản mới và đánh phần phụ thuộc "cần làm lại"; tài liệu là đơn vị trong DB | ADR-0044 |

ADR-0034..0041 ở trạng thái **Accepted** (Creator duyệt thiết kế, 2026-10-13). ADR-0042..0044 **Accepted** (Creator duyệt, 2026-10-17); ADR-0034 thành "Superseded by ADR-0042".

## Đối chiếu yêu cầu

| FR | Thành phần chính | CR con |
|---|---|---|
| FR-1 Luồng 3 giai đoạn, 21 bước | `FlowDefinition` (OR, AU, web-gui), `StepRail` | U2a; mỗi CR con thêm bước |
| FR-2 Logline, tóm tắt, theo cảnh | thay bằng FR-27..FR-32 | — |
| FR-27 Ý tưởng · creative brief | `CreativeBrief` | U3a |
| FR-28 Nghiên cứu và kiểm chứng | `ResearchDesk`, `SourceVerifier`, `SourceFetcher`, `ResearchSource` | U3b |
| FR-29 Xây dựng câu chuyện | `StoryFrame`, `ArchetypeAdvisor` | U3c |
| FR-30 Beat sheet | `BeatSheet` | U3d |
| FR-31 Viết kịch bản | `ScriptWriter`, `VoiceLines`, `DraftVoiceDispatcher`, `SynthesizeLines`, phát file chung | U3e |
| FR-32 Rà soát và chốt, bộ bàn giao | `ScriptReview`, `HandoffNotes`, `ScriptVersions` | U3f |
| FR-3 Bộ mỹ thuật | `ArtBible`, `StillRenderService`, `ArtSheet`, `StyleFrame` | U4, U5 |
| FR-4 Nhân vật, thư viện nhân vật | `CharacterLibrary`, schema storyboard | U4 |
| FR-5 Phong cách theo thế giới | `ArtBible` (thế giới), luật theo ngôn ngữ | U4 |
| FR-6 Hành trình màu | `ArtBible` (color key), `ColorKeyFrame`, kiểm tương phản | U5 |
| FR-7 Cảnh giải thích ý trong thế giới | prompt Đạo diễn, schema storyboard | U5 |
| FR-8 Máy quay liền mạch | schema storyboard (cách nối), `CodeSegments`, `SceneRenderer` | U5, U7, U8 |
| FR-9 Giọng sơ bộ sớm | `DraftVoiceDispatcher`, `VoiceLines`, `SynthesizeLines` | U6 |
| FR-10 Animatic khung tĩnh | `AnimaticPlan`, `AnimaticStill`, `AnimaticPlayer` | U6 |
| FR-11 Key pose | `rig` | U7 |
| FR-12 VFX | `vfx` | U7 |
| FR-13 Hình minh hoạ theo bộ mỹ thuật | `IllustrationPlanner` | U4 |
| FR-14 Render từng cảnh | `RenderSaga`, `SceneRenderer`, `SceneAssembler` | U8 |
| FR-15 Prompt và luật viết lại | `PromptSeeds`, luật theo ngôn ngữ | U1..U9 |
| FR-16 Xoá dữ liệu cũ | script vận hành một lần | U2a (U1 gỡ phần kit cũ) |
| FR-17 Sửa đúng chỗ | `ReviewUnitStore`, `DependencyTracker`, `RedoService`, bộ thành phần đơn vị | U3a (khung); U5 (áp lên Storyboard) |
| FR-18 Ngôn ngữ hình | `VisualLanguageStore`, `LanguageKits`, `LanguageSample` | U1 |
| FR-19 Gợi ý kiểu video | `ArchetypeAdvisor` (ở bước Câu chuyện) | U3c |
| FR-20 Sửa giọng, từ điển phát âm | `PronunciationDictionary`, `VoiceLines`, `SynthesizeLines` | U6 |
| FR-21 Thumbnail Remotion | `ThumbnailComposer`, `Thumbnail` | U9 |
| FR-22 Code theo cảnh, lỗi theo shot | `CodeSegments`, llm-service `pipeline`, `ShotPreview` | U7 |
| FR-23 Validate đúng chỗ | `ValidationFindings` | U7 |
| FR-24 Review đủ thông tin | `ReviewSummary` | U7 |
| FR-25 Dựng theo ghi chú | `EditSession`, `EditPlanner` | U8 |
| FR-26 Merge có tiến độ, QC | `SceneAssembler`, `qc_rules` | U8 |

NFR: NFR-1 (chi phí đo theo bước, như `llm_usage` hiện nay), NFR-2 (`ReviewUnitStore` đếm lần làm lại), NFR-3 (mọi bước mới đi qua cơ chế lỗi và chạy lại hiện có, không có đường "giả thành công"), NFR-4 (mọi composition nhận khung ngang và dọc, ADR-0031), NFR-5..NFR-7 (luật hiện có).

## Điểm để Functional Design của từng CR con chốt

- Schema bảng `review_units` và cách tính dấu đầu vào (U3a).
- Mẫu JSON `conceptflow.research.v1`, cách chuẩn hoá chữ khi so đoạn trích, số liệu giới hạn của `SourceFetcher` (U3b).
- Schema `script_version`, phần nào của phiên bản được hash riêng, `HandoffFor` trả gì cho từng vai (U3f).
- Cách các bước có sẵn (Kịch bản, Storyboard, Hình minh hoạ, Code) đọc đầu vào mới trong lúc Tiền kỳ · Nội dung chưa làm xong (U3a..U3f).
- Luật của từng ngôn ngữ hình, kit Remotion, hai cảnh thử của ảnh mẫu, luật dọn `/shared/stills` (U1).
- Schema storyboard mới (U4, U5), prompt các vai.
- Số worker render song song, kích thước đoạn chồng, phân loại lỗi tạm thời (U8, NFR Design).
- Thời gian mục tiêu của xem nhanh một shot (U7, NFR Design).
- Model vẽ đọc được ảnh tham khảo hay không (U4, P-5 A).
