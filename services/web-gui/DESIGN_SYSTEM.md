# Design system — web-gui

Quy ước bắt buộc cho mọi màn hình/component mới trong `services/web-gui`, để
UI đồng bộ mà không cần nhớ tên class hay copy style từ trang khác.

## Nguyên tắc

1. **Không viết `style={{...}}` inline** cho spacing/màu/border/font. Dùng
   token trong `src/styles/theme.css` (`--space-xs/sm/md/lg`, `--ink*`,
   `--accent*`, `--surface*`, `--danger/success/warning`) hoặc component
   trong `src/components/ui/`.
2. **Không tự viết lại "glass card", nút, input, select.** Dùng component
   trong `src/components/ui/` — chúng bọc sẵn `src/styles/glass.module.css`,
   là bộ class dùng chung duy nhất trong app.
3. Mọi trang bọc nội dung trong `<AppShell>` (`src/components/AppShell.tsx`)
   — không tự tạo topbar/background riêng.
4. CSS riêng của một trang/component (layout grid, spacing đặc thù) đặt
   trong file `.module.css` cùng tên, import qua CSS Modules — không dùng
   global class ngoài các trường hợp đã có trong `theme.css`.

## Component dùng chung (`src/components/ui`)

| Component | Thay cho | Ghi chú |
|---|---|---|
| `Card` | `<div className={glass.card}>` + header thủ công | props `title`, `hint`, `headerAction` |
| `Button` | `<button>` trần | `variant`: `primary` (mặc định) \| `ghost` \| `danger` \| `dangerGhost` |
| `FormField` | `<label>` + tự canh margin | props `label`, `className` |
| `Select` / `TextArea` / `TextInput` | `<select>/<textarea>/<input>` trần | style sẵn theo `glass.module.css` |
| `CtaRow` | `<div style={{display:"flex", justifyContent:"flex-end"}}>` | hàng nút hành động cuối form, `helperText` bên trái |

Ví dụ đầy đủ: `src/pages/PromptSettingsPage.tsx` — màn cấu hình prompt, là
trang tham chiếu cho bộ quy ước này.

## Khi cần style chưa có sẵn

- Cần spacing dọc giữa 2 khối → dùng `glass.mtXs/mtSm/mtMd/mtLg` (từ
  `src/styles/glass.module.css`), không viết `marginTop: N`.
- Cần badge trạng thái → dùng `StatusBadge` (`src/components/StatusBadge.tsx`).
- Cần layout 2 cột riêng cho 1 trang (vd. cột điều khiển sticky + nội dung)
  → viết trong `<TênTrang>.module.css`, tham khảo `RenderPage.module.css`
  hoặc `PromptSettingsPage.module.css`.
- Thật sự cần một pattern mới (không có sẵn trong `components/ui` hay
  `glass.module.css`) → thêm class dùng chung vào `glass.module.css` hoặc
  component mới vào `components/ui/`, **không** giải quyết cục bộ trong 1
  trang rồi để các trang khác lệch chuẩn.

## Theme / dark mode

Không hardcode màu (`#fff`, `rgba(0,0,0,...)`). Mọi màu phải qua CSS custom
property trong `theme.css`, vì các property đó được định nghĩa lại dưới
`[data-theme="dark"]` — hardcode màu sẽ vỡ dark mode.
