"""Builds the layout-probe sample scripts with the REAL
Code Merger of llm-service, so the probe is tried on exactly the file shape
the pipeline produces (head, PALETTE, LAYOUT, narrations, SHOTS, tail) and
the shot line ranges printed here are the ones `_map_failures` would use.

    cd services/llm-service && python ../rendering/remotion_project/layout_probe_samples/build_samples.py

Writes, next to this file:
  clean.tsx      6 shots, nothing wrong on purpose
  problems.tsx   the same 6 shots, three of them broken on purpose:
                   1.2 a label that cannot fit its width (text overflow)
                   1.3 a tooth whose box ends past x = 1824 (safe area)
                   2.1 two labels drawn over each other (text overlap)
  timing30.tsx   30 shots (the six clean bodies repeated) for timing
  overshoot.tsx  1 shot whose spring overshoots past the safe area between samples
  *.lines.json   shot id -> [first line, last line] from Merged.lines
"""

from __future__ import annotations

import json
import sys
from pathlib import Path

HERE = Path(__file__).resolve().parent
LLM_SERVICE = HERE.parents[2] / "llm-service"
sys.path.insert(0, str(LLM_SERVICE))

from app.pipeline import merger  # noqa: E402
from app.storyboard import Storyboard  # noqa: E402

LAYOUT = """const LAYOUT = {
  hero: {x: 960, y: 560, size: 460},
  tooth: {x: 620, y: 560, size: 420},
  label: {x: 1180, y: 520, w: 560},
  cat: {x: 1500, y: 600, size: 420},
};"""

# --- shot bodies (what the model writes) ------------------------------------------

SHOT_1_1 = """// Shot {id} — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const {{fps}} = useVideoConfig();
  const grow = spring({{frame, fps, config: {{damping: 200}}}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {{x, y, size}} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={{x}} y={{y + 60}} size={{size}} pose="wave" mood="happy" age="child" scale={{grow}} />
      <div style={{{{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}}"""

# 1.2 clean: tooth at left, label to its right, slow push-in.
SHOT_1_2_CLEAN = """// Shot {id} — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {{...clamp, easing: Easing.inOut(Easing.cubic)}});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {{x, y, size}} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{{{position: 'absolute', inset: 0, transformOrigin: `${{x}}px ${{y}}px`, transform: `scale(${{zoom}})`}}}}>
        <Tooth x={{x}} y={{y}} size={{size}} decay={{0.2}} />
        <div style={{{{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}}"""

# 1.2 broken: same shot, the label is one long line that must not wrap in 360 px.
SHOT_1_2_OVERFLOW = """// Shot {id} — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {{...clamp, easing: Easing.inOut(Easing.cubic)}});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {{x, y, size}} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{{{position: 'absolute', inset: 0, transformOrigin: `${{x}}px ${{y}}px`, transform: `scale(${{zoom}})`}}}}>
        <Tooth x={{x}} y={{y}} size={{size}} decay={{0.2}} />
        <div style={{{{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: 360, whiteSpace: 'nowrap', fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}}}>
          Lớp men răng bảo vệ
        </div>
      </div>
    </AbsoluteFill>
  );
}}"""

SHOT_1_3_CLEAN = """// Shot {id} — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {{x, y, size}} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={{x}} y={{y}} size={{size}} decay={{0.5}} />
      <Germ x={{x + 60}} y={{y - 120}} size={{140}} />
      <svg width={{300}} height={{80}} viewBox="0 0 300 80" style={{{{position: 'absolute', left: 860, top: 400, opacity: arrow}}}}>
        <line x1={{290}} y1={{40}} x2={{20}} y2={{40}} stroke={{PALETTE.accent}} strokeWidth={{10}} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={{PALETTE.accent}} />
      </svg>
      <LottieClip id="cat.thinking" x={{LAYOUT.cat.x}} y={{LAYOUT.cat.y}} size={{LAYOUT.cat.size}} />
    </AbsoluteFill>
  );
}}"""

# 1.3 broken: the tooth is pushed right so its box ends past the safe area (x > 1824).
SHOT_1_3_OUTSIDE = """// Shot {id} — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {{y, size}} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={{1700}} y={{y}} size={{size}} decay={{0.5}} />
      <Germ x={{1760}} y={{y - 120}} size={{140}} />
      <svg width={{300}} height={{80}} viewBox="0 0 300 80" style={{{{position: 'absolute', left: 1180, top: 400, opacity: arrow}}}}>
        <line x1={{20}} y1={{40}} x2={{280}} y2={{40}} stroke={{PALETTE.accent}} strokeWidth={{10}} strokeLinecap="round" />
        <polygon points="300,40 264,16 264,64" fill={{PALETTE.accent}} />
      </svg>
      <LottieClip id="cat.thinking" x={{620}} y={{LAYOUT.cat.y}} size={{LAYOUT.cat.size}} />
    </AbsoluteFill>
  );
}}"""

SHOT_2_1_CLEAN = """// Shot {id} — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={{1380}} y={{540}} size={{440}} color={{PALETTE.accent}} />
      <div style={{{{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{{{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}}"""

# 2.1 broken: the second label starts 40 px below the first, which wraps to two lines.
SHOT_2_1_OVERLAP = """// Shot {id} — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={{1380}} y={{540}} size={{440}} color={{PALETTE.accent}} />
      <div style={{{{position: 'absolute', left: 200, top: 380, width: 520, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{{{position: 'absolute', left: 200, top: 420, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}}"""

SHOT_2_2 = """// Shot {id} — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const {{fps}} = useVideoConfig();
  const pop = spring({{frame: frame - Math.round(duration * 0.3), fps, config: {{damping: 12}}}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={{820}} />
      <Person x={{760}} y={{580}} size={{520}} pose="stand" mood="happy" age="child" />
      <Candy x={{960}} y={{640}} size={{120}} />
      <Bubble x={{1320}} y={{340}} w={{420}} h={{170}} scale={{pop}} fontSize={{40}}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}}"""

SHOT_2_3 = """// Shot {id} — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={{120}} y={{120}} w={{820}} h={{840}} color={{PALETTE.muted}} radius={{32}} />
      <Panel x={{980}} y={{120}} w={{820}} h={{840}} color={{PALETTE.muted}} radius={{32}} />
      <Lollipop x={{530}} y={{480}} size={{380}} />
      <Apple x={{1390}} y={{480}} size={{360}} />
      <div style={{{{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}}}>
        Kẹo dính răng
      </div>
      <div style={{{{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}}"""

# A spring that overshoots: at rest (scale 1) the sun ends at x = 1790, inside the
# safe area; around frame 10 it is ~1.3x and pokes past x = 1824. Four samples
# (0 / 50 / 85 / 100 %) never land on the peak.
SHOT_OVERSHOOT = """// Shot {id} — MÁY: đứng yên | HÌNH: mặt trời bật ra ở góc phải
function {fn}({{duration}}: ShotProps) {{
  const frame = useCurrentFrame();
  const {{fps}} = useVideoConfig();
  const pop = spring({{frame, fps, config: {{damping: 6}}}});
  return (
    <AbsoluteFill>
      <Person x={{760}} y={{600}} size={{520}} pose="point" mood="happy" />
      <Sun x={{1640}} y={{300}} size={{300}} scale={{pop}} />
    </AbsoluteFill>
  );
}}"""

CLEAN = [SHOT_1_1, SHOT_1_2_CLEAN, SHOT_1_3_CLEAN, SHOT_2_1_CLEAN, SHOT_2_2, SHOT_2_3]
PROBLEMS = [SHOT_1_1, SHOT_1_2_OVERFLOW, SHOT_1_3_OUTSIDE, SHOT_2_1_OVERLAP, SHOT_2_2, SHOT_2_3]

NARRATION = [
    "Bạn có biết vì sao răng bị sâu không?",
    "Bên ngoài răng có một lớp men rất cứng.",
    "Vi khuẩn bám trên răng và tiết ra axit.",
    "Đánh răng đều đặn giúp bảo vệ lớp men.",
    "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
    "Hãy chọn táo thay cho kẹo nhé.",
]


def storyboard(n_scenes: int, per_scene: int) -> Storyboard:
    scenes = []
    k = 0
    for s in range(1, n_scenes + 1):
        shots = []
        for m in range(1, per_scene + 1):
            shots.append({"id": f"{s}.{m}", "camera": "", "visual": "mẫu kiểm tra bố cục", "narration": NARRATION[k % len(NARRATION)]})
            k += 1
        scenes.append({"id": f"s{s}", "title": f"Cảnh {s}", "shots": shots})
    return Storyboard.model_validate({
        "hero": "bạn nhỏ và chiếc răng",
        "palette": [
            {"role": "accent", "hex": "#F5B841", "meaning": "thứ đang được chú ý"},
            {"role": "muted", "hex": "#4A5670", "meaning": "nền phụ"},
            {"role": "ink", "hex": "#F2F7FF", "meaning": "nhãn và chữ"},
        ],
        "scenes": scenes,
    })


def build(name: str, sb: Storyboard, bodies: list[str]) -> None:
    ids = [sh.id for _, sh in sb.all_shots()]
    shots = {sid: bodies[i % len(bodies)].format(id=sid, fn=merger.remotion_fn(sid)) for i, sid in enumerate(ids)}
    merged = merger.merge_remotion(sb, LAYOUT, shots)
    (HERE / f"{name}.tsx").write_text(merged.code, encoding="utf-8")
    (HERE / f"{name}.lines.json").write_text(json.dumps(merged.lines, indent=2) + "\n", encoding="utf-8")
    print(f"{name}.tsx: {len(ids)} shots, {merged.code.count(chr(10))} lines")


if __name__ == "__main__":
    build("clean", storyboard(2, 3), CLEAN)
    build("problems", storyboard(2, 3), PROBLEMS)
    build("timing30", storyboard(5, 6), CLEAN)
    build("overshoot", storyboard(1, 1), [SHOT_OVERSHOOT])
