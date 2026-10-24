# Bước 1–7 theo luồng input/output mẫu: một bước một CR

Creator chốt ngày 2026-10-24: mỗi bước của luồng mẫu ("Chuyện gì xảy ra nếu ta rơi vào hố đen?") là một CR riêng. Đầu ra của bước trước là hợp đồng mà bước sau đọc. Làm theo thứ tự, bắt đầu từ bước 1 Idea.

## Các CR

| CR | Bước | Nhánh | Đọc (hợp đồng vào) | Ghi (hợp đồng ra) | File mẫu tương ứng |
|---|---|---|---|---|---|
| CR-093 | 1 Idea | `feature/cr-093-idea-step` | câu hỏi thô của Creator | `idea.v1` | `idea.md` |
| CR-094 | 2 Creative brief | `feature/cr-094-creative-brief` | `idea.v1` | `creative-brief.v1` | `creative-brief.md` |
| CR-095 | 3 Research & fact-check | `feature/cr-095-research-dossier` | `idea.v1`, `creative-brief.v1` | `research-dossier.v1` (claim R…, câu hỏi mở, sổ nguồn) | `research/research-dossier.md`, `research/source-register.csv` |
| CR-096 | 4 Story development | `feature/cr-096-story-structure` | `creative-brief.v1`, `research-dossier.v1` | `story-structure.v1` | `story-structure.md` |
| CR-097 | 5 Outline / beat sheet | `feature/cr-097-beat-sheet` | `story-structure.v1`, `research-dossier.v1` | `beat-sheet.v1` | `beat-sheet.md` |
| CR-098 | 6 Scriptwriting | `feature/cr-098-scriptwriting` | `beat-sheet.v1`, `story-structure.v1`, `research-dossier.v1`, `creative-brief.v1` | `script-draft.v1` | `script-v1.md` |
| CR-099 | 7 Script review & lock | `feature/cr-099-script-review-lock` | `script-draft.v1`, `creative-brief.v1`, `research-dossier.v1`, `beat-sheet.v1`, `review-rules` | `script-review-report.v1`, `script-version.v1` (bản cuối, ghi chú bàn giao) | `script-review-report.md`, `script-final.md`, `script-handoff-notes.md`, `script-version.json` |

## Luật chung cho cả bảy CR

- Mỗi hợp đồng có một tài liệu `docs/contracts/<tên>-v1.md`: các trường, kiểu dữ liệu, trường nào bắt buộc, điều kiện "bước xong", và bước nào đọc nó. Mỗi hợp đồng cũng có một dòng trong `docs/contracts/README.md`.
- Bước sau chỉ đọc bản đã duyệt của hợp đồng vào, không đọc bảng hay đơn vị nội bộ của bước trước.
- Khi một CR đổi đầu ra của bước mình mà bước sau chưa có CR mới, CR đó phải giữ cho bước sau chạy được: bước sau đọc hợp đồng mới qua một lớp chuyển đổi. Luồng luôn chạy tới cuối được.
- Tiêu chí "Hoàn thành khi" trong luồng mẫu là điều kiện để duyệt bước. Máy kiểm những gì kiểm được; phần còn lại Creator xác nhận.
- Phụ thuộc: CR sau cần CR trước đã giao. Nhánh `fix/beat-sheet-from-story` đang mở và trùng phần với CR-096 và CR-097, phải xử lý trước khi hai CR đó bắt đầu.
