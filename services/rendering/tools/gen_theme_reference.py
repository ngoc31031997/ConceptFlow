"""Sinh lại khối theme cho prompt: `python tools/gen_theme_reference.py`.

Ghi vào `orchestrator/internal/domain/prompts/` — nơi Orchestrator embed. Chạy lại
mỗi khi sửa `conceptflow/theme.py` hoặc `conceptflow/api.py`.
"""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from conceptflow.reference import render_theme_reference  # noqa: E402

PROMPTS = ROOT.parent / "orchestrator" / "internal" / "domain" / "prompts"

if __name__ == "__main__":
    for lang in ("vi", "en"):
        target = PROMPTS / f"theme_reference_{lang}.txt"
        target.write_text(render_theme_reference(lang), encoding="utf-8")
        print("wrote", target)
