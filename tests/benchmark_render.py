"""Harness benchmark cho Pha 0 (CR-002 / CR-003).

Render fixture tham chiếu bằng đúng cơ chế mà Rendering Service dùng
(thay `self.wait(AUTO)` bằng thời lượng narration, chạy manim qua
subprocess), rồi đo:

- thời gian render thực tế (wall-clock)
- RAM đỉnh của tiến trình con
- thời lượng video thật (ffprobe)
- **độ lệch đồng bộ**: offset thật của từng narration (qua marker
  `_cf_mark`, cơ chế đề xuất ở CR-002 FR10.1) so với offset mà pipeline
  HIỆN TẠI giả định (cộng dồn thời lượng narration).

Chạy trong container rendering:
    docker exec rendering python /tmp/benchmark_render.py -q m
"""

from __future__ import annotations

import argparse
import json
import os
import re
import resource
import subprocess
import sys
import tempfile
import time

AUTO_WAIT_RE = re.compile(r"self\.wait\(\s*AUTO\s*\)")
NARRATION_RE = re.compile(r'^\s*#\s*NARRATION:\s*"(.*)"\s*$', re.M)

WPM_VI = 140.0
MIN_SECONDS = 1.5

PREAMBLE = '''
import json as _cf_json, os as _cf_os
_CF_MARKS_PATH = _cf_os.environ.get("CF_MARKS_PATH", "/tmp/marks.jsonl")
def _cf_mark(scene, index):
    with open(_CF_MARKS_PATH, "a") as _f:
        _f.write(_cf_json.dumps({"index": index, "t": scene.renderer.time}) + "\\n")
'''


def estimate(text: str) -> float:
    words = len(text.split())
    return max(MIN_SECONDS, words / WPM_VI * 60.0) if words else MIN_SECONDS


def patch(script: str, durations: list[float]) -> str:
    counter = iter(range(len(durations)))
    it = iter(durations)

    def sub(_m):
        return f"(_cf_mark(self, {next(counter)}), self.wait({next(it):.4f}))"

    body = AUTO_WAIT_RE.sub(sub, script)
    lines = body.splitlines()
    for i, line in enumerate(lines):
        if line.startswith("from manim import") or line.startswith("import manim"):
            lines.insert(i + 1, PREAMBLE)
            break
    else:
        lines.insert(0, PREAMBLE)
    return "\n".join(lines)


def ffprobe_duration(path: str) -> float:
    out = subprocess.run(
        ["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path],
        capture_output=True, text=True, check=True,
    )
    return float(out.stdout.strip())


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("script")
    ap.add_argument("scene")
    ap.add_argument("-q", "--quality", default="m", choices=["l", "m", "h", "k"])
    args = ap.parse_args()

    script = open(args.script, encoding="utf-8").read()
    narrations = NARRATION_RE.findall(script)
    durations = [estimate(t) for t in narrations]
    n_waits = len(AUTO_WAIT_RE.findall(script))
    if n_waits != len(durations):
        print(f"FATAL: {n_waits} wait(AUTO) vs {len(durations)} narration", file=sys.stderr)
        return 2

    media_dir = tempfile.mkdtemp(prefix="bench-media-")
    marks_path = os.path.join(media_dir, "marks.jsonl")
    script_path = os.path.join(media_dir, "bench.py")
    with open(script_path, "w", encoding="utf-8") as f:
        f.write(patch(script, durations))

    env = dict(os.environ, CF_MARKS_PATH=marks_path)
    ru_before = resource.getrusage(resource.RUSAGE_CHILDREN)
    before = ru_before.ru_maxrss
    started = time.monotonic()
    proc = subprocess.run(
        ["manim", f"-q{args.quality}", "--disable_caching", "--media_dir", media_dir,
         script_path, args.scene],
        cwd=media_dir, env=env, capture_output=True, text=True,
    )
    elapsed = time.monotonic() - started
    ru_after = resource.getrusage(resource.RUSAGE_CHILDREN)
    peak_kb = max(ru_after.ru_maxrss, before)
    cpu_seconds = (ru_after.ru_utime - ru_before.ru_utime) + (ru_after.ru_stime - ru_before.ru_stime)

    if proc.returncode != 0:
        print(proc.stderr[-4000:], file=sys.stderr)
        print(f"FAILED after {elapsed:.1f}s", file=sys.stderr)
        return 1

    video = None
    for root, _d, files in os.walk(media_dir):
        for name in files:
            if name.endswith(".mp4") and "partial" not in root:
                cand = os.path.join(root, name)
                if video is None or os.path.getsize(cand) > os.path.getsize(video):
                    video = cand

    marks = [json.loads(l) for l in open(marks_path, encoding="utf-8")]
    marks.sort(key=lambda m: m["index"])

    # Offset mà pipeline HIỆN TẠI giả định: cộng dồn thời lượng narration.
    assumed, acc = [], 0.0
    for d in durations:
        assumed.append(acc)
        acc += d

    drift = [m["t"] - a for m, a in zip(marks, assumed)]

    report = {
        "quality": args.quality,
        "render_seconds": round(elapsed, 1),
        "peak_child_rss_mb": round(peak_kb / 1024, 1),
        "cpu_seconds": round(cpu_seconds, 1),
        "cpu_to_wallclock_ratio": round(cpu_seconds / elapsed, 2) if elapsed else None,
        "video_duration_seconds": round(ffprobe_duration(video), 2) if video else None,
        "video_size_mb": round(os.path.getsize(video) / 1024 / 1024, 2) if video else None,
        "narration_count": len(durations),
        "narration_total_seconds": round(sum(durations), 1),
        "animation_total_seconds": round(ffprobe_duration(video) - sum(durations), 1) if video else None,
        "real_offsets": [round(m["t"], 2) for m in marks],
        "assumed_offsets_current_pipeline": [round(a, 2) for a in assumed],
        "desync_seconds": [round(d, 2) for d in drift],
        "max_desync_seconds": round(max(drift), 2) if drift else 0.0,
    }
    print(json.dumps(report, indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
