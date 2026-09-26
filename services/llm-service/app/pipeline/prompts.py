"""The per-call user turns of the code pipeline (CR-039).

The system prompt of every call is the rendered `*_engineer_ai` role from the
prompt library — rules, palette discipline, layout law. What varies per call
lives here: which task this call is, and the slice of the storyboard it owns.
Every turn states what already exists so the model does not redeclare it.
"""

from __future__ import annotations

import json

from app.pipeline import merger
from app.pipeline.checker import Diagnostic
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
    }


def _dump(obj) -> str:
    return json.dumps(obj, ensure_ascii=False, indent=2)


def _world(sb: Storyboard) -> str:
    return f"NHÂN VẬT CHÍNH: {sb.hero}" if sb.hero else f"THẾ GIỚI: {sb.world}"


def _palette_table(sb: Storyboard) -> str:
    keys = merger.palette_keys(sb)
    return "\n".join(f"- {keys[p.role]} = {p.hex} — {p.role}: {p.meaning}" for p in sb.palette)


def _retry_note(problem: str | None) -> str:
    if not problem:
        return ""
    return f"\n\nLẦN TRƯỚC BẠN TRẢ LỜI SAI: {problem}\nHãy trả lời lại, sửa đúng lỗi đó."


# --- Remotion ---------------------------------------------------------------

def remotion_layout(sb: Storyboard, retry: str | None = None) -> str:
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
- Mỗi vật / nhân vật SỐNG QUA NHIỀU SHOT (nhân vật chính, các khối lặp lại, nhãn cố định) có một mục với toạ độ TÂM và kích thước bằng px trong khung 1920x1080, đặt trong vùng an toàn và tránh vùng phụ đề như đã mô tả ở luật bố cục. Ví dụ: hero: {{x: 960, y: 480, size: 320}}.
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
    return f"""NHIỆM VỤ HIỆN TẠI: VIẾT CODE CHO SHOT {shot_ids[0]} → {shot_ids[-1]} ({len(shot_ids)} shot).

Bạn chỉ viết {len(shot_ids)} hàm shot. Hệ thống tự ghép mọi thứ còn lại. ĐÃ CÓ SẴN trong file, TUYỆT ĐỐI KHÔNG viết lại:
- các dòng import (react; remotion: registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig; ./conceptflow-mini/segments; ./conceptflow-mini/primitives: Stage, SAFE_MARGIN, WIDTH, HEIGHT; ./conceptflow-mini/lottie: LottieClip)
- `const clamp = {{extrapolateLeft: 'clamp', extrapolateRight: 'clamp'}} as const;`
- `type ShotProps = {{duration: number}};`
- `export const narrations`, `SHOTS`, `CreatorComposition`, `registerRoot`
- PALETTE (chỉ dùng các khoá dưới đây, không màu nào khác):
{_palette_table(sb)}
- LAYOUT (dùng đúng các khoá này cho vật xuất hiện ở nhiều shot):
{layout.strip()}

Các shot bạn phải viết (theo đúng thứ tự):
{_dump(mine)}

SHOT NGAY TRƯỚC lô này (để chuyển cảnh liền mạch — frame 0 của shot đầu tiên phải nối được với hình cuối của shot này và `scene_end_frame` của nó):
{prev_txt}

SHOT NGAY SAU lô này (chỉ để biết hình cuối của shot cuối lô phải dẫn tới đâu):
{next_txt}

ĐỊNH DẠNG TRẢ VỀ — chỉ một khối ```tsx, gồm ĐÚNG các hàm sau và không gì khác ở cấp cao nhất:
{names}
Mỗi hàm có dạng `function ShotN_M({{duration}}: ShotProps) {{ ... }}` bắt đầu ở cột 0, và ngay phía trên nó là một dòng comment `// Shot n.m — MÁY: ... | HÌNH: ...`. Không khai báo hằng, hàm hay kiểu nào khác ở cấp cao nhất (đặt hằng phụ vào BÊN TRONG hàm shot).{_retry_note(retry)}"""


def remotion_repair(
    sb: Storyboard, layout: str, key: str, code: str, diags: list[Diagnostic], shot_ids_context: list[str]
) -> str:
    listed = "\n".join(f"- dòng {d.line}: {d.message}" if d.line else f"- {d.message}" for d in diags)
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
    return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI BIÊN DỊCH trong shot {key}.

Shot theo kịch bản:
{shot}

Lỗi trình biên dịch TypeScript (số dòng là dòng trong FILE ĐẦY ĐỦ, không phải trong đoạn dưới):
{listed}

Code hiện tại của shot:
```tsx
{code}
```

Bảng màu hợp lệ (chỉ dùng PALETTE.<khoá> dưới đây):
{_palette_table(sb)}

LAYOUT hiện có:
{layout.strip()}

Sửa CHỈ lỗi trên, giữ nguyên hình và chuyển động đã dựng. Trả về ĐÚNG một hàm `{merger.remotion_fn(key)}` đã sửa (kèm comment `// Shot {key} — ...` phía trên) trong một khối ```tsx, không gì khác."""


# --- Manim ------------------------------------------------------------------

def manim_cast(sb: Storyboard, retry: str | None = None) -> str:
    shots = [_shot_json(sc, sh) for sc, sh in sb.all_shots()]
    return f"""NHIỆM VỤ HIỆN TẠI: CAST (bước 1/2 của bước dựng code).

Kịch bản sẽ được dựng thành nhiều method shot do nhiều lượt gọi viết SONG SONG, mỗi lượt chỉ thấy một phần. Để các shot dùng chung được những vật xuyên suốt, bạn dựng trước "cast".

{_world(sb)}

Toàn bộ shot của video:
{_dump(shots)}

Hãy trả về ĐÚNG MỘT method, không gì khác, trong một khối ```python:

def setup_cast(self):
    ...

Yêu cầu:
- Chỉ TẠO các vật sống qua nhiều shot (nhân vật chính, thế giới, số đọc, sơ đồ dùng lại) và gán vào `self.<tên>` (snake_case ASCII, mô tả đúng vai trò). KHÔNG gọi reveal / play / add / narrate — chỉ tạo và gán, chưa hiện lên màn hình.
- Vật chỉ dùng trong một shot thì KHÔNG đặt ở đây.
- Chỉ dùng API được phép ở system prompt; không màu hex, không font_size tuỳ ý, không toạ độ tuyệt đối.
- Nếu không có vật nào xuyên suốt, viết `def setup_cast(self):` với thân là `pass`.{_retry_note(retry)}"""


def manim_chunk(
    sb: Storyboard,
    cast: str,
    cast_names: list[str],
    shot_ids: list[str],
    prev: tuple[Scene, Shot] | None,
    nxt: tuple[Scene, Shot] | None,
    retry: str | None = None,
) -> str:
    by_id = {sh.id: (sc, sh) for sc, sh in sb.all_shots()}
    mine = [_shot_json(*by_id[i]) for i in shot_ids]
    prev_txt = _dump(_shot_json(*prev)) if prev else "(không có — đây là lô đầu tiên)"
    next_txt = _dump(_shot_json(*nxt)) if nxt else "(không có — đây là lô cuối cùng)"
    names = ", ".join(merger.manim_fn(i) for i in shot_ids)
    cast_list = ", ".join(f"self.{n}" for n in cast_names) or "(không có vật nào)"
    return f"""NHIỆM VỤ HIỆN TẠI: VIẾT CODE CHO SHOT {shot_ids[0]} → {shot_ids[-1]} ({len(shot_ids)} shot).

Bạn chỉ viết {len(shot_ids)} method shot. Hệ thống tự ghép class, dòng import, `construct`, các lời gọi `self.beat(...)` và `setup_cast`. ĐÃ CÓ SẴN, TUYỆT ĐỐI KHÔNG viết lại: `from conceptflow import *`, class Scene, `construct`, `setup_cast`.

Vật xuyên suốt đã được `setup_cast` tạo sẵn (chỉ tạo, CHƯA hiện lên màn hình — shot nào cần thì tự `self.reveal(...)`): {cast_list}
```python
{cast.strip()}
```
Vật chỉ dùng trong shot của bạn thì tạo ngay trong method (biến cục bộ). Không tự gán thêm thuộc tính `self.<tên>` mới cho thứ shot khác cần dùng.

Các shot bạn phải viết (theo đúng thứ tự):
{_dump(mine)}

SHOT NGAY TRƯỚC lô này (khung hình còn lại khi shot đầu của bạn bắt đầu — chưa chắc rỗng):
{prev_txt}

SHOT NGAY SAU lô này (chỉ để biết hình cuối của shot cuối lô phải dẫn tới đâu):
{next_txt}

Mỗi shot chạy nối tiếp shot trước trên cùng một khung hình: đừng `self.clear_stage()` trừ khi kịch bản nói "cắt thẳng sang cảnh trống"; dọn (`self.dismiss`) những gì shot của bạn thêm vào mà kịch bản không giữ lại.

ĐỊNH DẠNG TRẢ VỀ — chỉ một khối ```python, gồm ĐÚNG các method sau, viết ở cột 0 (KHÔNG thụt vào trong class, hệ thống tự thụt), mỗi method là `def shot_N_M(self):`:
{names}
Không khai báo gì khác ở cấp cao nhất.{_retry_note(retry)}"""


def manim_repair(
    sb: Storyboard, cast: str, key: str, code: str, diags: list[Diagnostic], raw: str
) -> str:
    listed = "\n".join(f"- dòng {d.line}: {d.message}" if d.line else f"- {d.message}" for d in diags)
    tail = f"\n\nĐầu ra của lượt chạy thử:\n{raw[-3000:]}" if raw else ""
    if key == merger.CAST_KEY:
        return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI trong `setup_cast`.

Lỗi:
{listed}{tail}

Code hiện tại (viết ở cột 0):
```python
{code}
```

Trả về ĐÚNG method `def setup_cast(self):` đã sửa, ở cột 0, trong một khối ```python; giữ nguyên tên mọi thuộc tính `self.<tên>` đang có."""
    by_id = {sh.id: (sc, sh) for sc, sh in sb.all_shots()}
    shot = _dump(_shot_json(*by_id[key]))
    return f"""NHIỆM VỤ HIỆN TẠI: SỬA LỖI trong shot {key}.

Shot theo kịch bản:
{shot}

Lỗi khi chạy thử (số dòng là dòng trong FILE ĐẦY ĐỦ, không phải trong đoạn dưới):
{listed}{tail}

Code hiện tại của shot (viết ở cột 0):
```python
{code}
```

Vật xuyên suốt do `setup_cast` tạo:
```python
{cast.strip()}
```

Sửa CHỈ lỗi trên, giữ nguyên hình và lời thoại đã dựng. Trả về ĐÚNG một method `def {merger.manim_fn(key)}(self):` đã sửa, ở cột 0, trong một khối ```python, không gì khác."""
