# UX/UI design rules

## 1. Flow order: top-to-bottom, left-to-right

Every screen reads and operates from top to bottom and from left to right.

- Inputs and data selections come first (above or to the left).
- The action that consumes them (Run, Generate, Continue) comes after (below or to the right).
- Never place an action above or left of the data it needs. Otherwise the user can pick something lower/right and then press an action higher/left, and the action runs with stale or missing input.
- Settings that affect a run (engine, mode, format) sit before the run button, in that same order.

Example: on the script outline step, "1. Chủ đề" and "2. Dàn ý" come first and the engine/AI-run bar sits below them.

## 2. Visual style: Neubrutalism

The web-gui uses **Neubrutalism**: flat colour, thick black outline, hard offset shadow (no blur, no gradients). Tokens live in `services/web-gui/src/styles/theme.css`; use them instead of hard-coded values.

- Colour: `--accent` mustard yellow (primary action / active), `--accent2` pink (secondary highlight), `--run` purple (AI running), `--danger`, `--success`, `--warning`. Cream page background, white `--card` surfaces. Button fills follow rule 6.
- Outline: `--border-w` (2px) solid `--border-color` (near-black) on cards, buttons, inputs.
- Shadow: hard offset `--shadow` (4px 4px 0) for cards and selected items, `--shadow-sm` (2px 2px 0) for small controls.
- Radius: `--radius` (4px), small and square-ish.
- Type: `--font-display` (Space Grotesk) for titles, `--font-body` (Inter) for text, `--font-mono` (Space Mono) for code and prompts.
- Spacing: `--space-xs/sm/md/lg` (8/16/24/32px).
- Selected vs unselected: see `styles/selectable.module.css` (tinted fill, ring, bolder label, check badge). Long labels shrink to fit their box instead of overflowing.

## 3. Animation: slide out / slide in

Anything that shows or hides (collapsible blocks, panels, banners, revealed fields) slides out from where it was triggered and slides back in when dismissed. It never pops in or vanishes instantly.

- Use the shared `.reveal` (enter, ~0.22s) and `.revealOut` (exit, ~0.18s) classes in `services/web-gui/src/styles/glass.module.css`. They grow down from the top edge (`translateY` + `scaleY` + `clip-path`, fade).
- Keep the block mounted during the exit with `usePresence(open)` (`src/hooks/usePresence.ts`): render while `mounted`, add `.revealOut` while `closing`. See `components/Disclosure.tsx`.
- Do not write a new one-off animation for show/hide. Reuse the shared classes so every screen moves the same way.
- Respect `prefers-reduced-motion`: the shared classes already disable animation, and `usePresence` unmounts immediately in that case.
- Keep motion short (under ~0.25s) and subtle: no bounce, no long fades.
- Modals (rule 4) are the one exception to "slides out from where it was triggered": a modal fades in and rises a few pixels into the centre of the screen, and fades out when closed, within the same timing limits and with the same reduced-motion behaviour.

## 4. Modal for a task, inline reveal for a detail

A task the user starts from a button opens in a modal dialog over the screen. Inline reveal (rule 3) stays for read-only detail that belongs to the flow of the page. A task never expands a form into the middle of a list or table.

Open a modal for:

- Editing an item: "Sửa", "Sửa tay", "Sửa mô tả", "Sửa prompt", "Sửa màu". The modal holds the item's fields as a form; text on the page is never made editable in place.
- Adding an item: "Thêm dữ kiện", "Thêm thuật ngữ", "+ Ghi chú".
- Asking the AI to redo an item with a note ("Làm lại kèm ghi chú").
- Reviewing an AI proposal before it is applied (a pacing proposal, an edit plan, notes the AI will apply).
- Picking from a list or uploading a file to replace an item ("Thay từ thư viện", "Tải SVG lên").
- A generated text the user copies into another tool (a prompt to paste into an outside research or image tool).

Size of a modal follows how much it holds, set with the dialog's `size`:

- `sm` (480px): a confirmation, one or two lines of text.
- `md` (640px): a form of one to four short fields (the default).
- `lg` (900px): one long block of text, or a form of five to eight fields.
- `xl` (1200px): a form of many long text fields ("Sửa tay · Creative brief", "Sửa ngôn ngữ hình").
- Previewing one item's media (a 480p shot preview).
- Every confirmation required by rule 5.

Keep inline (rule 3): a collapsible section, "Xem toàn bộ" of a long text that is part of the step itself, a status or warning banner, a hint under a field.

How a modal behaves:

- The title names the item and the task ("Sửa dữ kiện F08", "Xoá beat B3?"). The body follows rule 1: fields first, then the impact (rule 5), then the buttons at the bottom right, "Huỷ" on the left and the main action on the right, coloured by rule 6.
- Focus moves into the modal (the first field, or the main action of a confirmation), Tab stays inside it, Esc, "×" and "Huỷ" close it, and focus returns to the button that opened it. The page behind does not scroll.
- Clicking the backdrop closes a modal that holds no input (a confirmation, a preview). A modal with a form closes only by "Huỷ", "×" or Esc, so a stray click does not lose typed text.
- Text typed into a modal survives a refresh of the screen behind it (for example an AI job finishing).
- One modal at a time. When saving a form changes later steps, the impact list sits inside the same modal above the save button; it is not a second modal on top.
- Reuse the dialog behaviour already in `components/ConfirmModal.tsx` (portal, focus trap, Esc, scroll lock). The first screen that needs a modal with a form extracts a shared dialog shell from it; screens do not each write their own.

## 5. Confirm a change that deletes data or makes later steps redo

Before running an action that does any of the following, ask for confirmation in a modal that says what will happen to the workflow:

- (a) deletes data;
- (b) sends work that a later step has already approved or already run back to "cần làm lại" or back to review;
- (c) discards text the user wrote;
- (d) publishes outside the app.

The confirmation shows:

- What is changed or deleted, by name ("beat B3 “Cửa hẹp”", "dữ kiện F08").
- The impact: each later step that has to redo something and how much ("Lời thoại của beat B3"; "Storyboard, giọng, animatic, hình, code, render của cảnh 3, ở những bước đã chạy"), and what stays as it is ("Các beat khác giữ nguyên"). The list comes from the server's dependency tracker, not from text written into the screen.
- Whether it can be undone: a delete says it cannot be restored; an edit after a version is locked says it creates a new draft and later steps keep using the locked version until the next lock.
- Two buttons: "Huỷ" and an action named by what it does ("Xoá beat", "Đổi góc", "Lưu và làm lại"), never "OK" or "Có".

A change made through a select or a checkbox that falls under (a)–(c) shows the same confirmation and puts the control back if the user cancels.

No confirmation for an action that only moves forward or is easy to undo and touches nothing downstream: approve, save something no later step uses yet, open, filter, copy, play.

When the action's own modal already shows the impact (an edit form, a redo with a note), that modal is the confirmation; do not ask twice.

## 6. Button colour shows how far an action reaches

A button's fill tells how much the action changes before the user reads its label. Pick the highest level that applies.

| Level | Fill (token) | For | Examples |
|---|---|---|---|
| Delete | red `--danger` | Removes data that cannot be restored | Xoá beat, Xoá ngôn ngữ, Bỏ ảnh tham khảo |
| Redo later steps | orange `--impact` | Changes data so that approved or already-run work in later steps has to redo; starts something outside the app that cannot be taken back | Lưu bản sửa (when something depends on it), Đổi góc, Chốt phiên bản v2, Render lại, Đăng lên YouTube |
| Run AI or a long job | purple `--run` | Starts an AI or render job; nothing already done is lost | Gửi cho AI làm lại (when nothing depends on it yet), Bắt đầu render, Đọc lại câu này |
| Approve, save, check | yellow `--accent` | Moves the flow forward or saves; easy to undo; no later step has to redo | Duyệt, Sang bước sau, Đọc kết quả, Kiểm lại link, Chốt phiên bản v1 |
| View | blue tint `--info-soft` | Opens information without changing anything | Xem toàn bộ, Xem ở Cài đặt, Mở câu đó, Xem nhanh 480p |
| Neutral | white `--card` | Cancel, close, back, toggle, filter, copy, play | Huỷ, Đóng, ← Bước trước, Sao chép, ▶ Nghe |

- An action whose reach depends on the data takes the colour of what it will do this time: "Gửi cho AI làm lại" is orange when a later step already used the item and purple when nothing depends on it yet.
- Colour never carries the meaning alone: the label names the action, and every red or orange action goes through rule 5 (or shows its impact in its own modal).
- Text on a filled button keeps a 4.5:1 contrast in both themes. The dark theme puts dark text on the orange, red and purple fills (`--impact-ink`, `--danger-ink`, `--run-ink`).
- `components/ui/Button.tsx` has the `primary`, `ghost`, `danger` and `dangerGhost` variants today. The `impact`, `run` and `view` variants, and the `--impact`, `--impact-ink`, `--danger-ink` and `--run-ink` tokens in `theme.css`, are added under these names by the first change that needs them.

## 7. Every flow step page has the same width

The step pages of the flow (Ý tưởng, Cấu hình, Nghiên cứu, and every step after them) must look equally wide, so moving between steps does not make the content jump from full width to a narrow column.

- A step page passes `wide` to `<AppShell>`, as every existing step page does. Without it the content sits in a narrow centred column with empty space on both sides.
- Do not set a smaller `max-width` on a step's own body to get a narrower column; cards fill the `wide` main column.
- A new step page is checked against the previous step side by side at the same window size before it is done.
- The only pages that stay narrow are standalone forms and callbacks outside the flow (sign-in, OAuth callback, "Đang mở dự án"). A page that is part of the flow is never one of them.
