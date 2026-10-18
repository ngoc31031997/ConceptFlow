"""The per-call user turns of the code pipeline.

The system prompt of every call is the rendered `*_engineer_ai` role from the
prompt library — rules, palette discipline, layout law. What varies per call
lives here: which task this call is, and the slice of the storyboard it owns.
Every turn states what already exists so the model does not redeclare it.

The system prompt must stay byte-identical across every call of one run so the
provider's prompt cache keeps working: anything that varies per call belongs in
these user turns. The code step's system prompt carries only the story's core
lines, so each chunk turn opens with the storyboard's hero/world line itself.
"""

from __future__ import annotations

import json

from app.frame import LANDSCAPE, Frame
from app.pipeline import merger
from app.pipeline.checker import LAYOUT, Diagnostic
from app.storyboard import Scene, Shot, Storyboard


def _shot_json(scene: Scene, shot: Shot) -> dict:
    return {
        "shot": shot.id,
        "scene": scene.id,
        "scene_title": scene.title,
        "scene_invariant": scene.invariant,
        "scene_transition_in": scene.transition_in,
        "scene_mood": scene.mood,
        "scene_end_frame": scene.end_frame,
        "camera": shot.camera,
        "visual": shot.visual,
        "narration": shot.narration,
    } | ({"scene_setting": scene.setting} if scene.setting else {}) | (
        {"motion_blur": True} if shot.motion_blur else {}
    ) | (
        {"lines": [{"say": line.say, "show": line.show} for line in shot.lines]} if shot.lines else {}
    )


def _dump(obj) -> str:
    return json.dumps(obj, ensure_ascii=False, indent=2)


def _world(sb: Storyboard) -> str:
    return f"NHÂN VẬT CHÍNH: {sb.hero}" if sb.hero else f"THẾ GIỚI: {sb.world}"


def _palette_table(sb: Storyboard) -> str:
    keys = merger.palette_keys(sb)
    return "\n".join(f"- PALETTE.{keys[p.role]} = {p.hex} — {p.role}: {p.meaning}" for p in sb.palette)


def _looks_block(sb: Storyboard) -> str:
    """The LOOKS constant the frame declares, and how a shot uses it."""
    return (
        merger.looks_text(sb)
        + "\nMỗi shot bọc TOÀN BỘ hình của nó trong `<Shot look={LOOKS.sceneN}>` (N là số đứng trước dấu chấm của "
        "mã shot, shot 2.3 dùng `LOOKS.scene2`). Không tự viết look, không tự chọn màu ánh sáng: màu ánh sáng, "
        "bóng và không khí do LOOKS và kit tính."
    )


def _retry_note(problem: str | None) -> str:
    if not problem:
        return ""
    return f"\n\nLẦN TRƯỚC BẠN TRẢ LỜI SAI: {problem}\nHãy trả lời lại, sửa đúng lỗi đó."


# --- Remotion ---------------------------------------------------------------

def remotion_layout(sb: Storyboard, retry: str | None = None, canvas: Frame = LANDSCAPE) -> str:
    shots = [_shot_json(sc, sh) for sc, sh in sb.all_shots()]
    return f"""NHIỆM VỤ HIỆN TẠI: LAYOUT (bước 1/2 của bước dựng code).

Toàn bộ kịch bản sẽ được dựng thành nhiều đoạn code do nhiều lượt gọi viết SONG SONG, mỗi lượt chỉ thấy một phần. Để chúng khớp nhau, bạn viết trước bảng toạ độ chung.

{_world(sb)}

Bảng màu (đã có sẵn thành hằng PALETTE, đừng khai báo lại):
{_palette_table(sb)}

Toàn bộ shot của video:
{_dump(shots)}

Hãy trả về ĐÚNG MỘT khai báo TypeScript, không gì khác:

const LAYOUT = {{ ... }};

Yêu cầu:
- Mỗi vật / nhân vật SỐNG QUA NHIỀU SHOT (nhân vật chính, các khối lặp lại, nhãn cố định) có một mục với toạ độ TÂM và kích thước bằng px trong khung {canvas.size}, đặt trong vùng an toàn {canvas.safe_text()} và tránh vùng phụ đề như đã mô tả ở luật bố cục. Ví dụ: hero: {{x: 960, y: 480, size: 320}}.
- Vật chỉ xuất hiện trong một shot thì KHÔNG cần mục ở đây.
- Chỉ dùng số và đối tượng thuần; không import, không hàm, không tham chiếu PALETTE.
- Tên khoá camelCase ASCII, mô tả đúng vai trò.{_retry_note(retry)}"""


def remotion_chunk(
    sb: Storyboard,
    layout: str,
    shot_ids: list[str],
    prev: tuple[Scene, Shot] | None,
    nxt: tuple[Scene, Shot] | None,
    retry: str | None = None,
) -> str:
    by_id = {sh.id: (sc, sh) for sc, sh in sb.all_shots()}
    mine = [_shot_json(*by_id[i]) for i in shot_ids]
    prev_txt = _dump(_shot_json(*prev)) if prev else "(không có — đây là lô đầu tiên)"
    next_txt = _dump(_shot_json(*nxt)) if nxt else "(không có — đây là lô cuối cùng)"
    names = ", ".join(merger.remotion_fn(i) for i in shot_ids)
    return f"""{_world(sb)}

NHIỆM VỤ HIỆN TẠI: VIẾT CODE CHO SHOT {shot_ids[0]} → {shot_ids[-1]} ({len(shot_ids)} shot).

Bạn chỉ viết {len(shot_ids)} hàm shot. Hệ thống tự ghép mọi thứ còn lại. ĐÃ CÓ SẴN trong file, TUYỆT ĐỐI KHÔNG viết lại:
- các dòng import ({merger.available_names_text()})
- `const clamp = {{extrapolateLeft: 'clamp', extrapolateRight: 'clamp'}} as const;`
- `type ShotProps = {{duration: number; lines: number[]}};` — `lines[i]` là frame bắt đầu câu thoại thứ i của shot (tính từ đầu shot, `lines[0] = 0`); shot có mục "lines" thì mỗi câu một mốc
- `export const narrations`, `shotLineCounts`, `SHOTS`, `CreatorComposition`, `registerRoot`
- PALETTE (chỉ dùng các khoá dưới đây, không màu nào khác):
{_palette_table(sb)}
- LAYOUT (dùng đúng các khoá này cho vật xuất hiện ở nhiều shot):
{layout.strip()}
- LOOKS (ánh sáng của từng cảnh, đã có sẵn):
{_looks_block(sb)}

Các shot bạn phải viết (theo đúng thứ tự):
{_dump(mine)}

SHOT NGAY TRƯỚC lô này (để chuyển cảnh liền mạch — frame 0 của shot đầu tiên phải nối được với hình cuối của shot này và `scene_end_frame` của nó):
{prev_txt}

SHOT NGAY SAU lô này (chỉ để biết hình cuối của shot cuối lô phải dẫn tới đâu):
{next_txt}

ĐỊNH DẠNG TRẢ VỀ — chỉ một khối ```tsx, gồm ĐÚNG các hàm sau và không gì khác ở cấp cao nhất:
{names}
Mỗi hàm có dạng `function ShotN_M({{duration, lines}}: ShotProps) {{ ... }}` bắt đầu ở cột 0, và ngay phía trên nó là một dòng comment `// Shot n.m — MÁY: ... | HÌNH: ...`. Không khai báo hằng, hàm hay kiểu nào khác ở cấp cao nhất (đặt hằng phụ vào BÊN TRONG hàm shot).{_retry_note(retry)}"""


def _listed(diags: list[Diagnostic]) -> str:
    return "\n".join(f"- dòng {d.line}: {d.message}" if d.line else f"- {d.message}" for d in diags)


def _remotion_errors(diags: list[Diagnostic], canvas: Frame) -> tuple[str, str]:
    """The compile errors and the measured layout errors of one shot as two
    labelled lists, and the word naming which kinds are present."""
    compile_diags = [d for d in diags if d.kind != LAYOUT]
    layout_diags = [d for d in diags if d.kind == LAYOUT]
    sections: list[str] = []
    if compile_diags:
        sections.append(
            "Lỗi trình biên dịch TypeScript (số dòng là dòng trong FILE ĐẦY ĐỦ, không phải trong đoạn dưới):\n"
            + _listed(compile_diags))
    if layout_diags:
        # Measured on the shot as it is really drawn, at the moments named.
        sections.append(
            "Lỗi bố cục — đo trên hình thật của shot, ở các thời điểm ghi trong từng dòng (số dòng là dòng trong "
            f"FILE ĐẦY ĐỦ; vùng an toàn {canvas.safe_text()}; chữ tối thiểu 32px):\n" + _listed(layout_diags)
            + "\nSửa bằng cách đổi vị trí, kích thước, width, cỡ chữ hoặc biên độ chuyển động (spring vọt lố) "
            "của đúng vật được nêu; không bỏ vật hay chữ mà kịch bản yêu cầu.")
    what = "BIÊN DỊCH" if not layout_diags else "BỐ CỤC" if not compile_diags else "BIÊN DỊCH VÀ BỐ CỤC"
    return what, "\n\n".join(sections)


def remotion_repair_many(
    sb: Storyboard, layout: str, items: list[tuple[str, str, list[Diagnostic]]], canvas: Frame = LANDSCAPE,
) -> str:
    """One repair turn for several failing shots of the same segment.

    `items` is `(shot id, current code, diagnostics)` per shot. A single item
    gives the one-shot turn; several share one palette and LAYOUT block and
    ask for every function back in one code block.
    """
    if len(items) == 1:
        key, code, diags = items[0]
        return remotion_repair(sb, layout, key, code, diags, [], canvas)
    by_id = {sh.id: (sc, sh) for sc, sh in sb.all_shots()}
    keys = [key for key, _, _ in items]
    blocks = []
    for key, code, diags in items:
        what, errors_block = _remotion_errors(diags, canvas)
        blocks.append(f"""### Shot {key} — lỗi {what}

Shot theo kịch bản:
{_dump(_shot_json(*by_id[key]))}

{errors_block}

Code hiện tại của shot:
```tsx
{code}
```""")
    names = ", ".join(merger.remotion_fn(k) for k in keys)
    return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI trong {len(keys)} shot: {", ".join(keys)}.

Mỗi shot dưới đây có lỗi riêng của nó; sửa từng shot, không đụng sang shot khác.

{chr(10).join(blocks)}

Bảng màu hợp lệ (chỉ dùng PALETTE.<khoá> dưới đây):
{_palette_table(sb)}

LAYOUT hiện có:
{layout.strip()}

LOOKS hiện có:
{_looks_block(sb)}

Sửa CHỈ lỗi đã nêu, giữ nguyên hình và chuyển động đã dựng. Trả về ĐÚNG các hàm {names} đã sửa (mỗi hàm kèm comment `// Shot n.m — ...` phía trên) trong MỘT khối ```tsx, không gì khác."""


def remotion_repair(
    sb: Storyboard, layout: str, key: str, code: str, diags: list[Diagnostic], shot_ids_context: list[str],
    canvas: Frame = LANDSCAPE,
) -> str:
    listed = _listed(diags)
    if key == merger.LAYOUT_KEY:
        return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI BIÊN DỊCH trong khai báo LAYOUT.

Lỗi trình biên dịch TypeScript:
{listed}

Code hiện tại:
```tsx
{code}
```

Trả về ĐÚNG khai báo `const LAYOUT = {{ ... }};` đã sửa trong một khối ```tsx, giữ nguyên mọi khoá đang có (các shot đang dùng chúng)."""
    by_id = {sh.id: (sc, sh) for sc, sh in sb.all_shots()}
    shot = _dump(_shot_json(*by_id[key]))
    what, errors_block = _remotion_errors(diags, canvas)
    return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI {what} trong shot {key}.

Shot theo kịch bản:
{shot}

{errors_block}

Code hiện tại của shot:
```tsx
{code}
```

Bảng màu hợp lệ (chỉ dùng PALETTE.<khoá> dưới đây):
{_palette_table(sb)}

LAYOUT hiện có:
{layout.strip()}

LOOKS hiện có:
{_looks_block(sb)}

Sửa CHỈ lỗi trên, giữ nguyên hình và chuyển động đã dựng. Trả về ĐÚNG một hàm `{merger.remotion_fn(key)}` đã sửa (kèm comment `// Shot {key} — ...` phía trên) trong một khối ```tsx, không gì khác."""
