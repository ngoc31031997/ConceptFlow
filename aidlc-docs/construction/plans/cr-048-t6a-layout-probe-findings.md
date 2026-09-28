# CR-048 T6a — Spike: đo bố cục shot Remotion trong trình duyệt (findings)

Nhánh: `feature/cr-048-t6a-layout-probe`. Đây là **prototype**, chưa nối vào pipeline (việc đó là T6b).
Mọi con số dưới đây lấy từ các lần chạy thật trong môi trường spike (mục 5). Chúng **chưa** được đo trong container `rendering` (lý do ở mục 8.1).

## 1. Kết luận

- **Làm được, và nhanh.** Lấy được hộp bao DOM (có tính transform/zoom) của mọi chữ, `svg`, hình trong bộ minh hoạ, clip Lottie và khối màu của từng shot, tại 0% / 50% / 85% / 100% của một độ dài giả định (150 frame). Chữ có thêm `fontSize` và `scrollWidth/clientWidth`.
- **Thời gian:** video 30 shot × 4 frame (120 mẫu) đo trong **1,8–2,0 s**. Một process chạy từ đầu tới cuối mất **3,1–3,5 s**, gồm khởi động Node, bundle harness và mở trình duyệt. Mục tiêu dưới 60 s đạt với dư rất nhiều.
- **Tìm đúng cả 3 lỗi cố ý, không báo nhầm trên script sạch:** chữ tràn, vật ra ngoài vùng an toàn, hai nhãn đè nhau. Mỗi lỗi chỉ đúng **dòng JSX** gây ra nó và đúng **dòng khai báo hàm shot**.
- **Được thêm, ngoài yêu cầu:** probe bắt được cả lỗi **runtime** xảy ra ở một frame cụ thể (shot `throw` khi `frame > 40%`). `tsc` không bắt được loại lỗi này, và nó cũng làm render thật chết.

## 2. Cách làm

```
merged.tsx ──TypeScript transpileModule (+ transformer gắn data-cf-line)──▶ CommonJS
                                                                              │
harness.tsx (esbuild bundle 1 lần: React, remotion*, @remotion/player,       ▼
             conceptflow-mini)  ── Chromium headless (playwright-core) ── cfProbe.load(js)
                                                                              │
             <Thumbnail component={CreatorComposition} inputProps={{segments}} frameToDisplay=g>
                                                                              │
             với mỗi shot i, mỗi pct p: g = i·D + round(p·(D−1)) → flushSync render
             → chờ hết delayRender() → đọc DOM → JSON
```

Các file:

| File | Vai trò |
|---|---|
| `services/rendering/remotion_project/layout_probe.mjs` | CLI Node. Biên dịch script, bundle harness, mở Chromium, gọi đo, áp luật nguyên mẫu (`--check`), ghi JSON. |
| `services/rendering/remotion_project/src/layout-probe/harness.tsx` | Trang đo. `load()` chạy script, `measure()` vẽ bằng `<Thumbnail>` rồi đọc hộp bao, `fontStatus()`. |
| `services/rendering/remotion_project/src/layout-probe/remotion-shim.ts` | `remotion` cho trang đo. Chỉ khác một chỗ: `getInputProps()` (xem 3.4). |
| `services/rendering/remotion_project/layout_probe_samples/` | Script mẫu sinh bằng **merger thật** của llm-service (`build_samples.py`), `*.lines.json` (= `Merged.lines`), và output JSON của probe. |
| `package.json` | Thêm devDependency ghim cứng: `@remotion/player@4.0.525`, `playwright-core@1.56.1`. Repo chưa commit lockfile nào, nên không có lockfile để cập nhật. |

Các điểm chính:

1. **Một shot tại một frame.** Probe không cắt riêng `SHOTS[i]`. Nó vẽ nguyên `CreatorComposition` của script, với `segments` gồm N đoạn liền nhau dài D frame, rồi đặt frame toàn cục `i·D + f`. Vì `<Segments>` bọc mỗi shot trong `<Sequence>`, tại frame đó chỉ shot i được mount. Script chạy đúng như lúc render: qua `<Stage>`, `<Segments>`, font, `clamp`, `LAYOUT`, và không cần sửa script.
2. **Đọc hộp bao, không chụp ảnh.** Mỗi mẫu gồm một lần commit React đồng bộ (`flushSync`), rồi `getBoundingClientRect()` và `Range.getBoundingClientRect()` cho chữ. Hộp bao đã tính transform, nên vị trí sau zoom đo được trực tiếp. Ví dụ shot 1.2 zoom 1.1: răng 420 px thành 420×462 và cỡ chữ 48 thành 52,8, đúng với tính tay.
3. **Chờ tài nguyên bất đồng bộ.** Font và JSON của Lottie đều giữ `delayRender()`. Harness chờ `window.remotion_delayRenderHandles` rỗng hai macrotask liên tiếp (tối đa 5 s). Handle nào kẹt quá 5 s bị đánh dấu "stale" và bỏ qua ở các mẫu sau, để một clip hỏng không làm mọi mẫu sau chậm thêm 5 s. Đo được: clip id sai tốn 5006 ms đúng một lần, các mẫu sau vẫn khoảng 10 ms.
4. **Truy về dòng code.** Một transformer TypeScript chèn `data-cf-line={dòng}` vào **mọi** phần tử JSX trước khi biên dịch. Mỗi hộp DOM lấy dòng của phần tử gần nhất có thuộc tính này (`closest('[data-cf-line]')`). Phần tử do khung `Stage`/`Thumbnail` vẽ không có thuộc tính nên bị bỏ qua. Nhờ vậy nền `#080E1C` và các div bọc của Player không lẫn vào kết quả.
5. **Gọi tên hình trong bộ minh hoạ.** Mỗi component viết hoa export từ `conceptflow-mini/illustration` và `lottie` được bọc trong `<div data-cf-kit="Tooth" style="display:contents">`. Div `display: contents` không tạo hộp nên không đổi layout. Hộp của hình là hợp các hộp con. `Face` và `GroundShadow` không được bọc vì chúng vẽ bên trong `<svg>`.
6. **Thu thập gì:**
   - chữ: phần tử HTML có text node trực tiếp, và `<text>` trong SVG;
   - `<svg>` gốc, bỏ qua mọi phần tử con của SVG trừ `<text>`;
   - hình trong bộ minh hoạ và clip Lottie (`kind: "kit"`);
   - khối có sơn: nền không trong suốt, `background-image`, viền, hoặc `img/canvas/video` (`kind: "shape"`).

   Bỏ qua phần tử có opacity hiệu dụng (nhân dồn theo các tổ tiên) < 0,01, `display:none`, `visibility:hidden` và phần tử kích thước 0. Phần tử phủ kín khung được gắn `full_frame: true` (Backdrop, nền cảnh).
7. **Cô lập mạng.** Mọi request ngoài origin giả `http://cf-layout-probe.local` bị `abort`. Origin này chỉ trả trang harness, bundle và file trong `public/` (Lottie qua `staticFile`).

## 3. Vì sao chọn cách này

### 3.1 `<Thumbnail>` chứ không phải `renderStill` hay tự điều khiển trang render của Remotion

- `renderStill` chụp ảnh PNG, tốn hàng trăm ms mỗi frame, trong khi ta chỉ cần DOM.
- Trang bundle của `@remotion/bundler` đặt frame qua các global nội bộ (`remotion_setBundleMode`, `remotion_setFrame`) không có tài liệu. Webpack bundle cũng mất nhiều giây cho **mỗi** script.
- `<Thumbnail>` là API công khai, cùng version 4.0.525, và vẽ đúng một frame vào DOM thường.

### 3.2 esbuild cho harness và TypeScript cho script

- **Harness** được bundle một lần bằng esbuild: 144–220 ms mỗi process. Có thể cache ra file ở T6b. esbuild đã có sẵn trong `node_modules` nhờ `@remotion/bundler`, và `render.mjs` đã import nó.
- **Script** được biên dịch bằng `ts.transpileModule`, không dùng esbuild, vì esbuild không cho sửa AST. Cần sửa AST để gắn `data-cf-line`. Gói `typescript` đã có (dùng cho `tscheck.mjs`). Chi phí: 297–380 ms lần đầu trong process (nạp TypeScript), 46–130 ms khi đã nóng.

### 3.3 `playwright-core` chứ không phải `openBrowser()` của `@remotion/renderer`

- `openBrowser()` trả về `HeadlessBrowser`, bản fork puppeteer nội bộ của Remotion. `newPage()` của nó đòi các tham số nội bộ (context, sourceMapGetter…) và không có tài liệu.
- `playwright-core` chỉ là thư viện (~8 MB, **không tự tải trình duyệt**), API ổn định (`evaluate`, `route`). Nó chạy được với bất kỳ Chromium nào qua `executablePath`.
- Thứ tự probe chọn trình duyệt:
  1. `--browser`;
  2. biến `LAYOUT_PROBE_BROWSER`;
  3. `ensureBrowser()` của `@remotion/renderer`. Trong container, hàm này trả về Chrome Headless Shell mà Dockerfile đã tải sẵn (`npx remotion browser ensure`).

### 3.4 Shim `getInputProps`

Trong Player, Remotion **cấm** `getInputProps()`: nó throw "You cannot call getInputProps() from a <Player>". Nhưng `conceptflow-mini/primitives.tsx` gọi hàm này để chọn font dự án. Plugin esbuild vì vậy trỏ mọi import `remotion` trong bundle đo (kit, script, `@remotion/lottie`) sang `remotion-shim.ts`: `export * from 'remotion'` và chỉ thay `getInputProps` bằng props do probe truyền vào (`--font`). Không sửa file nào của `conceptflow-mini`.

## 4. Cách chạy

```bash
cd services/rendering/remotion_project
npm install
# (sinh lại mẫu, cần pydantic của llm-service)
cd ../../llm-service && python ../rendering/remotion_project/layout_probe_samples/build_samples.py && cd -
LAYOUT_PROBE_BROWSER=/opt/pw-browsers/chromium_headless_shell-1194/chrome-linux/headless_shell \
  node layout_probe.mjs --check --out-dir layout_probe_samples \
  layout_probe_samples/clean.tsx layout_probe_samples/problems.tsx
```

Các tuỳ chọn khác: `--duration 150`, `--frames 0,0.5,0.85,1`, `--font "Be Vietnam Pro"`, `--subtitle-band bottom:240`. Truyền nhiều script vào một lần chạy thì chúng dùng chung một trình duyệt, giống một checker chạy nóng.

## 5. Thời gian đo được

Môi trường: sandbox 4 vCPU, 16 GB RAM, Node 22.22.2. Trình duyệt là Chromium headless shell **141.0.7390.37** của Playwright (`/opt/pw-browsers/chromium_headless_shell-1194`). Font Be Vietnam Pro được cài cho user (cùng nguồn file với Dockerfile), và probe xác nhận `font.available = true`.

| Lần chạy | Mẫu | Đo (measure) | Tổng cho script | Wall trong Node | Wall ngoài (gồm khởi động `node`) |
|---|---|---|---|---|---|
| `timing30.tsx` (30 shot × 4 frame), process mới, lần 1 | 120 | 1517 ms | 1919 ms | 2468 ms | 3413 ms |
| như trên, lần 2 | 120 | 1457 ms | 1770 ms | 2296 ms | 3129 ms |
| như trên, lần 3 | 120 | 1658 ms | 1994 ms | 2579 ms | 3457 ms |
| `timing30.tsx` × **16 frame** | 480 | 5532 ms | 5853 ms | 6445 ms | 7406 ms |
| Nóng: `timing30`, `clean`, `timing30` trong một process | 120 / 24 / 120 | — | 1842 / 298 / **1545** ms | 4208 ms | 5067 ms |

Chi tiết lần 1: compile 380 ms, load 9 ms, trung bình **11,8 ms/mẫu**, mẫu chậm nhất 45,7 ms (mẫu đầu, khi font và Lottie nạp lần đầu). Phần khởi động mỗi process: bundle harness 144–197 ms, mở trình duyệt và trang 322–412 ms. Output JSON cho 30 shot × 4 frame nặng 184 KB (390 phần tử).

⇒ Chi phí gần tuyến tính theo số mẫu, khoảng 11 ms/mẫu. Tăng lên 16 frame/shot (để bắt spring vọt lố) vẫn chỉ khoảng 6 s cho 30 shot.

## 6. Kết quả trên mẫu

Các script mẫu do `merger.merge_remotion` thật sinh ra: cùng head/PALETTE/LAYOUT/narrations/SHOTS/tail như pipeline. Chúng qua `tscheck.mjs` với 0 diagnostic.

**`clean.tsx`** (6 shot, dùng Person, Tooth, Germ, Shield, Backdrop, Candy, Bubble, Panel, Lollipop, Apple, `svg`, LottieClip `cat.thinking`, zoom) → **0 vi phạm**.

**`problems.tsx`** (cùng 6 shot, 3 shot bị làm hỏng) → **4 vi phạm, đều đúng**:

| Shot | Luật | `line` (hàm shot) | `element_line` | Thông điệp probe sinh ra |
|---|---|---|---|---|
| 1.2 | `text_overflow` | 53 | 62 | Shot 1.2, frame 50%/85%/100%: nhãn 'Lớp men răng bảo vệ' tràn khung chữ (rộng 502px > width 360px) |
| 1.3 | `safe_area` | 71 | 77 | Shot 1.3, frame 0%/50%/85%/100%: hình Tooth ra ngoài vùng an toàn (phải x=1891 > 1824) |
| 1.3 | `safe_area` | 71 | 78 | Shot 1.3, frame 0%/50%/85%/100%: hình Germ ra ngoài vùng an toàn (phải x=1830 > 1824) |
| 2.1 | `text_overlap` | 89 | 96 | Shot 2.1, frame 50%/85%/100%: nhãn 'Đánh răng hai lần mỗi ngày' (dòng 96) đè lên nhãn 'Kem có fluor giúp men chắc hơn' (dòng 99) — giao nhau 456×56px |

Germ (x=1760, size 140 → mép phải 1830) cũng thật sự vượt vùng an toàn. Đó là lỗi đúng, không phải báo nhầm. Tại frame 0%, nhãn của shot 1.2 và 2.1 có opacity 0 nên không bị tính; lỗi chỉ xuất hiện từ 50%.

Đã kiểm tra tay: dòng 53/71/89 là `function Shot1_2(`/`Shot1_3(`/`Shot2_1(`, còn dòng 62/77/78/96/99 đúng là thẻ `<div>`/`<Tooth>`/`<Germ>`/hai `<div>` nhãn. Các dòng đều nằm trong khoảng `Merged.lines` của shot tương ứng (`problems.lines.json`: 1.2 → 52–68, 1.3 → 70–86, 2.1 → 88–104).

Output đầy đủ: `layout_probe_samples/clean.layout.json` và `problems.layout.json`.

**Thí nghiệm phụ** (chạy thật; output không commit, trừ các script mẫu):

- **Spring vọt lố (`overshoot.tsx`)**: mặt trời `spring({damping: 6})` lúc nghỉ nằm trong vùng an toàn (mép phải 1790), nhưng ở frame 10 phóng tới 411,6 px và vượt ra (mép phải 1846, mép trên 94).
  - 4 frame → **0 vi phạm, bỏ sót**.
  - 16 frame → bắt được: "Shot 1.1, frame 7%: hình Sun ra ngoài vùng an toàn (trên y=94 < 96, phải x=1846 > 1824)".
- **Trạng thái chuyển tiếp**: chạy 16 frame trên `timing30.tsx` (bản "sạch") sinh 5 cảnh báo `min_font`. Nguyên nhân: bong bóng thoại đang `spring` phóng lên, tại 33% chữ mới ở cỡ hiển thị 25,8 px. Với 4 frame thì không có cảnh báo nào.
- **Thiếu font** (`--font Montserrat`, không cài ở đây, nên chữ rơi về sans-serif): nhãn tràn đo được 488 px thay vì 502 px, vùng đè 436×49 thay vì 456×56. Vi phạm vẫn được phát hiện, nhưng số đo lệch khoảng 3–10%. `font.available` trong JSON báo `false` để T6b biết.
- **Lỗi runtime** (bản sao `clean.tsx`, shot 1.2 `throw` khi `frame > 40%`): mẫu 50/85/100% của shot 1.2 có `error` ("Error: boom at 75"…). Các shot khác vẫn đo bình thường, vì harness remount `<Thumbnail>` sau lỗi.

## 7. Đề xuất schema JSON cho T6b

Output của probe (`version: 1`), dạng rút gọn:

```jsonc
{
  "version": 1,
  "script": "problems.tsx",
  "composition": {"id": "creator", "width": 1920, "height": 1080, "fps": 30},
  "narrations": 6,                    // độ dài mảng narrations export ra
  "nominal_duration": 150,
  "frames": [{"pct": 0, "frame": 0}, {"pct": 0.5, "frame": 75}, {"pct": 0.85, "frame": 127}, {"pct": 1, "frame": 149}],
  "font": {"family": "Be Vietnam Pro", "available": true},
  "browser": {"path": "...", "version": "141.0.7390.37"},
  "shots": [{
    "index": 1, "id": "1.2", "component": "Shot1_2",
    "line": 53,                        // dòng `function Shot1_2(` trong file đã ghép
    "samples": [{
      "pct": 1, "frame": 149, "global_frame": 299, "ms": 10.3,
      "error": "…",                    // chỉ có khi shot throw ở frame này
      "pending_delay_render": 1,       // chỉ có khi chờ quá 5 s
      "elements": [
        {"kind": "kit", "component": "Tooth", "line": 61,
         "rect": {"x": 410, "y": 329, "w": 420, "h": 462}, "opacity": 1},
        {"kind": "text", "tag": "div", "line": 62, "text": "Lớp men răng bảo vệ",
         "rect": {"x": 1038, "y": 483, "w": 396, "h": 66},        // hộp phần tử (sau transform)
         "text_rect": {"x": 1038, "y": 481.9, "w": 552.4, "h": 67.1}, // hộp của chính các dòng chữ
         "opacity": 1, "lines": 1,
         "font_size": 48, "font_size_rendered": 52.8,              // CSS / sau scale
         "scroll_width": 502, "client_width": 360,                  // px layout (chưa scale)
         "scroll_height": 60, "client_height": 60,
         "text_parent": null}                                       // chỉ số chữ bao ngoài (chữ lồng nhau)
      ]
    }]
  }],
  "page_log": [{"type": "pageerror|error|warning|http404|blocked", "text": "…"}],
  "timing_ms": {"compile": 46, "load": 5, "measure": 275, "samples": 24, "sample_avg": 10.2, "sample_max": 20.3,
                "script_total": 327, "startup": {"bundle_harness": 197, "launch_browser_and_page": 327}},
  "violations": [                      // chỉ khi --check; T6b nên chuyển phần này sang Python
    {"shot": "1.2", "shot_index": 1, "line": 53, "element_line": 62,
     "rule": "text_overflow", "severity": "blocking", "frames": ["50%", "85%", "100%"],
     "detail": "…", "message": "Shot 1.2, frame 50%/85%/100%: nhãn '…' tràn khung chữ (rộng 502px > width 360px)"}
  ]
}
```

- `kind` ∈ `text | svg | kit | shape`. Toạ độ tính theo px của khung 1920×1080, gốc ở góc trên trái của composition, làm tròn 0,1.
- `rule` ∈ `safe_area | subtitle_zone | text_overflow | text_overlap | runtime_error` (blocking) và `min_font | hero_size | not_settled` (warning).

Luật nguyên mẫu trong `layout_probe.mjs --check` (để T6b dựa vào, không phải bản cuối):

| Luật | Cách tính |
|---|---|
| `safe_area` | Hộp (chữ: `text_rect`; còn lại: `rect`) không `full_frame` vượt 96..1824 × 96..984, dung sai 0,5 px. |
| `subtitle_zone` | Theo `--subtitle-band top|bottom:<px>`, cùng số với `subtitleBandPx` trong `authoring-service/internal/domain/narration.go` (200/240/280). |
| `text_overflow` | Chiều ngang: `scroll_width > client_width + 1`. Chiều dọc: `scroll_height > client_height + font_size/2`. Ngưỡng nửa dòng là cần thiết: lần chạy đầu với ngưỡng 1 px báo nhầm tiêu đề `lineHeight: 1.2` (74 > 72 px), do dấu tiếng Việt cao hơn line box. |
| `text_overlap` | Hai `text_rect` giao nhau > 1 px mỗi chiều, bỏ qua cặp chữ lồng nhau (`text_parent`). |
| `min_font` | `font_size_rendered < 32`. |
| `hero_size` | Vật (không phải chữ, không `full_frame`) lớn nhất của shot có max(w/1920, h/1080) < 30%. |
| `runtime_error` | Mẫu có `error`. |
| `not_settled` | Mẫu có `pending_delay_render`. |

Vi phạm giống nhau (cùng shot, luật, dòng, phần tử) được gộp lại, kèm danh sách frame.

## 8. Giới hạn đã biết

### 8.1 Trình duyệt và container

Chưa chạy trong container `rendering`:

- Docker daemon không có trong sandbox (`/var/run/docker.sock` không tồn tại).
- `npx remotion browser ensure` bị proxy chặn: `remotion.media` không nằm trong allowlist, trả về 403.

Vì thế probe mới được thử với Chromium **141** của Playwright, chứ chưa với Chrome Headless Shell **149.0.7790.0** (`TESTED_VERSION` của `@remotion/renderer` 4.0.525) mà image dùng. `playwright-core` 1.56.1 được ghim cho Chromium 141. Điều khiển Chrome 149 qua CDP nhiều khả năng vẫn chạy, nhưng **T6b phải chạy thử trong container trước**. Nếu không được, có hai hướng: nâng `playwright-core` cho khớp, hoặc dùng `openBrowser()` của Remotion (API nội bộ, xem 3.3). Image cũng **chưa rebuild**. Hiện chỉ `package.json` đổi (thêm 2 devDependency) và chưa có gì chạy probe.

### 8.2 Font

- Probe đo bằng font **đã cài trong hệ thống**. Thiếu font thì số đo lệch khoảng 3–10% (mục 6). Probe tự phát hiện font bằng cách đo chữ với hai fallback khác nhau (`font.available`). T6b nên coi `available: false` là "không kiểm tra được" chứ không phải "đạt".
- Phát hiện phụ, không sửa trong spike: `useFontLoaded()` trong `primitives.tsx` dùng `document.fonts.load()`. API này chỉ biết web font (`@font-face`), nên với font cài hệ thống nó **luôn** log "font 'Be Vietnam Pro' not found". Log này xuất hiện ngay cả khi font có và Chromium đang vẽ bằng nó (đã xác nhận ở đây: `available: true` nhưng warning vẫn có trong `page_log`). Lúc render thật, cảnh báo này nhiều khả năng cũng sai.

### 8.3 Lottie

Hộp đo là **khung vuông của clip** (x, y, size theo w/h của JSON), không phải hình dáng thật. Manifest ghi rõ ví dụ "mèo nằm nửa dưới, hơi lệch trái", nên phần lớn khung là trong suốt. Hệ quả:

- `safe_area` có thể quá khắt khe (khung chạm biên dù con mèo không chạm);
- nếu kiểm tra "chữ đè Lottie" thì dễ báo nhầm.

Nội dung SVG bên trong clip không được đo. Hướng khắc phục: lưu "hộp thực" của mỗi clip trong manifest, hoặc đo hợp các `path` của lottie-web.

### 8.4 Bộ minh hoạ

- Hộp của một hình là hộp `<svg>` của `Figure`, theo tỉ lệ viewBox. `overflow: visible` nên phần vẽ lố ra ngoài viewBox (tay giơ cao ở `cheer`, bóng đổ) **không** được tính. Ngược lại, khoảng trống trong viewBox lại được tính.
- Chỉ bọc được component export trực tiếp. Component mới vẽ bên trong `<svg>` (kiểu `Face`, `GroundShadow`) phải được thêm vào `SVG_PARTS` trong harness, nếu không sẽ thành `<div>` trong `<svg>` và hình bị hỏng. Hình thư viện của Creator (CR-044) hiện ra dưới tên `Figure`, không có tên riêng.

### 8.5 Spring vọt lố và trạng thái chuyển tiếp

Lấy mẫu rời rạc thì bỏ sót đỉnh: đã chứng minh ở mục 6, 4 frame không thấy, 16 frame thấy. Đề xuất cho T6b:

- lấy mẫu dày ở đoạn vào cảnh, ví dụ 0, 5, 10, 15, 20, 30, 50, 85, 100% (9 mẫu/shot, khoảng 3 s cho 30 shot);
- hoặc quét `spring(` có `damping` thấp để thêm mẫu.

Ngược lại, luật `min_font` và `text_overlap` không nên áp lên frame giữa animation. Bong bóng đang phóng to có chữ 25,8 px là bình thường. Nên áp `min_font` trên `font_size` CSS hoặc chỉ ở mẫu ≥ 85%.

### 8.6 Độ dài giả định

Mốc thời gian là tỉ lệ của 150 frame. Nhưng `spring()` và các mốc theo `fps` chạy theo frame tuyệt đối, nên "50%" của một đoạn TTS thật dài 90 frame khác với 50% của 150 frame. Ở bước code chưa có TTS. T6b có thể ước lượng độ dài từng shot từ số chữ lời thoại, dùng cùng nguồn WPM với T8.

### 8.7 Hình học

- `getBoundingClientRect` của phần tử xoay là hộp thẳng trục bao ngoài, nên lớn hơn hình thật.
- Phần tử bị cắt bởi tổ tiên `overflow: hidden` (ví dụ con của `Panel`) vẫn báo đủ hộp.
- `text_rect` là hộp hợp của mọi dòng. Với chữ nhiều dòng căn giữa, hộp này rộng hơn phần mực thật, nên có thể gây `text_overlap` giả. Có thể cải thiện bằng cách so từng dòng (`range.getClientRects()`).

### 8.8 Phụ thuộc nội bộ

Harness đọc `window.remotion_delayRenderHandles`. Đây là global không nằm trong tài liệu, nhưng `delayRender()` của Remotion ghi vào nó ở mọi môi trường. Khi nâng Remotion phải kiểm tra lại, cùng với thông báo lỗi và hành vi `getInputProps` trong Player.

### 8.9 Lỗi runtime

Stack trace chứa dòng của **JS đã biên dịch** (`eval at load`), không phải dòng TSX. Muốn map lỗi runtime về dòng chính xác thì T6b cần source map. Còn không, chỉ cần dùng dòng hàm shot, vì probe đã biết shot nào đang chạy.

### 8.10 Bảo mật

Script là output của model và chạy trong trang Chromium (sandbox của Chromium, giống render thật). Mọi request mạng ra ngoài đều bị chặn. Import chỉ giới hạn trong các module mà script được phép dùng.

## 9. Map vi phạm về dòng code (cho T6b)

- **Rendering service chỉ nhận `code`.** Nó không biết `Merged.lines`. Probe tự đọc thứ tự shot từ `const SHOTS … = [Shot1_1, …]` và tìm `^function ShotN_M(`, đúng regex `REMOTION_SHOT` của `merger.py`, để ra `shots[i].line`. Nó cũng đổi `ShotN_M` thành id `N.M`.
- **Mỗi hộp mang `line` của thẻ JSX** đã vẽ ra nó (`data-cf-line`). Với hình trong bộ minh hoạ, đó là dòng dùng `<Tooth …/>` trong shot. Chữ bên trong một hình (ví dụ `Bubble`) mang dòng của `<Bubble>`. Hình thư viện truyền `{...props}` xuống `<Figure>` thì dòng của nơi gọi thắng, vì thuộc tính được chèn **trước** spread.
- **`CheckDiagnostic.line` mà T6b trả về nên là:**
  1. `element_line`, nếu nó nằm giữa `shots[i].line` và `shots[i+1].line` (hoặc trước `const SHOTS` với shot cuối);
  2. nếu không, `shots[i].line` (dòng khai báo hàm shot).

  Trường hợp thứ hai gồm: phần tử vẽ từ helper ở LAYOUT hoặc từ khối hình thư viện dán ngoài vùng shot, và lỗi runtime.

  Cả hai giá trị đều rơi vào khoảng `Merged.lines[shot]`, nên `_map_failures` (`llm-service/app/pipeline/run.py`, `merged.shot_at(d.line)`) gán đúng shot. Vòng repair `_repair` sẽ gửi đúng shot đó, không cần sửa llm-service. `element_line` giúp prompt repair chỉ đúng thẻ cần sửa. Nên đưa cả dòng này vào message, ví dụ "(dòng 62)".
- **Message**: dùng `message` đã sinh sẵn (tiếng Việt, có số), ví dụ `"Shot 1.2, frame 50%/85%/100%: nhãn 'Lớp men răng bảo vệ' tràn khung chữ (rộng 502px > width 360px)"`, khớp mẫu mà plan T6b yêu cầu.

## 10. Gợi ý triển khai T6b

1. **Checker chạy nóng**, giống `tscheck.mjs`: một process Node đọc/ghi JSON theo dòng qua stdin/stdout, giữ sẵn trình duyệt, trang và harness. Mỗi lần kiểm tra khi đó chỉ tốn compile + measure (khoảng 1,5–2 s cho 30 shot). Adapter Python đặt cạnh `adapters/rendering/typescript_checker.py`. Timeout theo `LAYOUT_CHECK_TIMEOUT_SECONDS`, bật/tắt bằng `LAYOUT_CHECK_ENABLED`. Trình duyệt chết hoặc quá giờ thì trả kết quả "không kiểm tra được" kèm warning, không coi là đạt.
2. **Tách phần đo và phần luật.** Node chỉ đo và trả JSON theo schema ở mục 7. Luật và message tiếng Việt viết bằng Python (`check_script.py`), để test bằng `pytest` với JSON mẫu mà không cần trình duyệt. Hai file `problems.layout.json`/`clean.layout.json` dùng làm fixture được.
3. **Chỉ chạy sau khi `tsc` và lint đạt.** Lint Lottie id chạy trước, nên tránh được trường hợp chờ 5 s vì clip id sai.
4. **Vùng phụ đề.** `POST /check` hiện chỉ nhận `engine, code, scene_class_name`. Để áp `subtitle_zone`, request phải mang thêm chế độ phụ đề và cỡ/vị trí (authoring → llm-service → rendering), cùng nguồn với `SubtitleZone()` ở `authoring-service/internal/domain/narration.go`.
5. **Dockerfile.** Không cần thêm gì ngoài `npm install` đã có: `playwright-core` không tải trình duyệt, còn Chrome Headless Shell đã được tải bởi `npx remotion browser ensure`. Phải chạy thử probe trong image (mục 8.1). Ba font dự án (Be Vietnam Pro, Montserrat, Cormorant Garamond) đã được Dockerfile cài vào hệ thống, nên `font.available` phải ra `true` với cả ba; nên kiểm tra điều này trong lần chạy thử đó.
