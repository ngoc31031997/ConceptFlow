# CR-115 · Bước 10 Chuỗi hình (Visual Sequencing) — câu hỏi yêu cầu

Nhánh `feature/cr-115-visual-sequencing` (worktree riêng). Khuôn: bước 9 (CR-114), đã có trên main.

## Creator đã chốt (2026-11-18)
1. Mục tiêu: sắp xếp các visual unit thành dòng chảy hình ảnh; xác định nhịp, motif và chuyển cảnh. Output `visual-sequence.json` theo khuôn Creator đưa.
2. Input và nơi đọc:
   | Input | Nguồn |
   |---|---|
   | visual-plan | `visual_plan_versions` (bản chốt mới nhất, DB của visual-service) |
   | visual-concept | `visual_concept_versions` (bản chốt mới nhất, DB của visual-service) |
   | beat-sheet | `beat:{k}`, `order` trong bản chốt kịch bản, qua HTTP `GET /internal/v1/authoring/{id}/script-versions/latest` |
   | script-final | `script:{k}`, `handoff:{k}`, `glossary`, `brief`, `facts` cùng route trên |
   Không đọc `project_beat_sheet`/`project_beat` (bản đang sửa, không có phiên bản, nằm ở authoring-db).
3. Motif là output của bước 10: bước 10 tự đặt motif; mỗi motif neo vào một mục đã chốt ở bước 8 (`representation_rules`: `scientific_concepts`, `characters`, `objects`) để cách vẽ không lệch. Bước 8 không đổi.

## Câu hỏi

**Q1. Thứ tự có được đổi không?** Lời thoại chạy tuyến tính, unit của bước 9 đã theo thứ tự câu.
- A (đề xuất): giữ nguyên thứ tự bước 9; bước 10 chỉ thêm nhịp, chuyển cảnh, motif cho từng unit. Máy kiểm `sequence_index` khớp `order` của bước 9.
- B: cho đổi thứ tự trong một beat (vd. hình đến trước lời).
=> A
**Q2. Đơn vị duyệt và cách AI viết.**
- A (đề xuất): giống bước 9, đơn vị duyệt là một beat; AI viết từng beat, thấy unit cuối của beat trước (để viết chuyển cảnh nối beat) và danh sách motif. Motif là một đơn vị duyệt riêng, AI đặt trước khi viết các beat.
- B: cả chuỗi là một đơn vị, AI viết một lượt.
=> A
**Q3. Danh sách `pacing`** (AI không tự thêm). Đề xuất: `slow_reveal`, `steady`, `quick_build`, `comparison`, `pause_emphasis`, `climax`, `resolve`. Thêm/bớt?
=> cần càng detail càn tốt lấy dữ liệu từ nhũng bộ phim thật sự, nếu dc phải làm hẳn một bộ master data cho chỗ này và AI sẽ chọn chúng đồng thời user cũng sẽ được chỉnh sửa chỗ này
**Q4. Danh sách `transition_from_previous.type`.** Đề xuất: `cut`, `match_cut`, `dissolve`, `morph`, `zoom_in`, `zoom_out`, `pan`, `wipe`. Máy kiểm loại chuyển cảnh không thuộc `motion_language.avoid` của bước 8; có buộc phải nằm trong `preferred_transitions` không?
=>cần bổ sung càng detail càng tốt và hoạt động tương tự tương tự q3
**Q5. Khuôn motif.** Đề xuất `{motif_id, description, anchor_ref (mã bước 8), role (vai trò trong mạch: neo, đối chiếu, gieo–gặt…), unit_refs (máy gom từ motif_refs)}`. Máy kiểm: `anchor_ref` có thật; motif được dùng ở ≥ 2 unit (dưới 2 thì không còn là motif → cảnh báo).
=>ok 
**Q6. Vòng AI rà soát cả chuỗi** như bước 8 và 9 (nhóm `consistency`, `clarity`, `feasibility`…)? Đề xuất: có.
=> ok
**Q7. Giao diện.** Làm mockup artifact để Creator chốt trước khi code, như bước 9? Đề xuất: có.
=> ok

## Trả lời câu hỏi bổ sung (2026-11-18)
Q8–Q11: Creator duyệt theo đề xuất (khuôn trường như Q8; Q9 A; Q10 A; seed nháp ở `cr-115-library-seed-draft.md` chờ duyệt).

## Câu hỏi bổ sung (theo trả lời Q3, Q4: master data nhịp và chuyển cảnh)

Hướng hiểu: hai thư viện dùng chung cho mọi video, giống thư viện tham chiếu `reference_styles` của bước 8; seed sẵn từ thuật ngữ dựng phim thật; AI chỉ được chọn mã có trong thư viện; Creator sửa được.

**Q8. Khuôn một mục thư viện.** Đề xuất:
- Nhịp (`pacing_patterns`): `id` (`P01`…), `name`, `summary`, `when_to_use`, `avoid_when`, `shot_length` (ngắn/vừa/dài, gợi ý số giây mỗi hình), `motion` (nhịp chuyển động trong hình), `audio_relation` (lời/nhạc/khoảng lặng), `film_examples[]` (phim, cảnh, vì sao), `tags[]`.
- Chuyển cảnh (`transition_types`): `id` (`T01`…), `name`, `summary`, `mechanics` (cách dựng: cái gì giữ, cái gì đổi, mấy giây), `meaning` (cảm giác/nghĩa với người xem), `when_to_use`, `avoid_when`, `needs_motif` (có, ví dụ match cut cần hình neo chung), `code_feasibility` (dựng bằng code dễ/khó), `film_examples[]`, `tags[]`.
Thêm/bớt trường?

**Q9. "Creator sửa được" là sửa gì?**
- A (đề xuất): cả hai — (1) thêm/sửa/xoá mục thư viện ở một màn riêng (như thư viện tham chiếu), (2) ở bước 10 đổi nhịp/chuyển cảnh AI chọn cho từng unit, chọn từ thư viện.
- B: chỉ (2), thư viện cố định trong code.

**Q10. Quan hệ với bước 8** (`motion_language.preferred_transitions`, `avoid`, `pace` slow/medium/fast theo beat — đều là chữ tự do).
- A (đề xuất): không buộc; máy cảnh báo (không chặn) khi chuyển cảnh chọn trùng ý một mục `avoid`, việc so ý để vòng AI rà soát (chữ tự do không so khớp máy được). Nhịp của unit được đưa kèm `pace` của beat cho AI.
- B: Creator chọn trước ở mỗi video một tập con của thư viện; AI chỉ chọn trong tập con đó.

**Q11. Seed ban đầu.** Mình soạn bản nháp seed (khoảng 15–20 nhịp, 25–30 chuyển cảnh, ví dụ từ phim có thật: Kubrick, Nolan, Kurosawa, phim tài liệu khoa học như Cosmos, Kurzgesagt…) để Creator duyệt trước khi đưa vào code. Ví dụ phim chỉ ghi những cảnh mình chắc chắn; mục không chắc để trống chứ không bịa. Đồng ý?
