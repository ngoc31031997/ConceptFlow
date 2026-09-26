"""Công cụ cho Creator quản lý danh mục Lottie (CR-038).

    python tools/lottie_catalog.py validate     kiểm manifest + file + giấy phép
    python tools/lottie_catalog.py gallery      sinh trang xem trước, mở bằng trình duyệt để DUYỆT clip
    python tools/lottie_catalog.py prompt       sinh khối {{lottie_catalog}} cho Orchestrator
    python tools/lottie_catalog.py credits      danh sách ghi công (dán vào mô tả video)
    python tools/lottie_catalog.py info FILE    kích thước, thời lượng, màu gốc của một file Lottie

Quy trình thêm clip: bỏ file vào remotion_project/public/lottie/<id>.json, thêm mục
`candidate` vào remotion_project/lottie/manifest.json, chạy `gallery` để xem, đổi sang
`approved` nếu ưng, rồi chạy `prompt` và rebuild orchestrator.
"""

from __future__ import annotations

import argparse
import html
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from domain import lottie_catalog as catalog  # noqa: E402

PROJECT = ROOT / "remotion_project"
MANIFEST = PROJECT / "lottie" / "manifest.json"
PUBLIC_DIR = PROJECT / "public" / "lottie"
PROMPT_TARGET = ROOT.parent / "orchestrator" / "internal" / "domain" / "prompts" / "lottie_catalog_vi.txt"
GALLERY_TARGET = PROJECT / "lottie" / "gallery.html"

BACKGROUND = "#080E1C"  # cùng nền với Stage, để thấy clip đúng như khi lên hình


def cmd_validate(_: argparse.Namespace) -> int:
    assets = catalog.load_manifest(MANIFEST)
    problems = catalog.validate(assets, PUBLIC_DIR)
    for problem in problems:
        print("LỖI", problem)
    ready = len(catalog.approved(assets))
    print(f"{len(assets)} clip ({ready} đã duyệt, {len(assets) - ready} ứng viên), {len(problems)} lỗi")
    return 1 if problems else 0


def cmd_prompt(_: argparse.Namespace) -> int:
    assets = catalog.load_manifest(MANIFEST)
    problems = catalog.validate(assets, PUBLIC_DIR)
    if problems:
        print("Manifest chưa hợp lệ, không sinh prompt:\n  " + "\n  ".join(problems))
        return 1
    PROMPT_TARGET.write_text(catalog.render_prompt_block(assets, "vi"), encoding="utf-8")
    print("wrote", PROMPT_TARGET)
    return 0


def cmd_credits(_: argparse.Namespace) -> int:
    lines = catalog.credits(catalog.load_manifest(MANIFEST))
    print("\n".join(lines) if lines else "(không clip nào cần ghi công)")
    return 0


def cmd_info(args: argparse.Namespace) -> int:
    data = json.loads(Path(args.file).read_text(encoding="utf-8"))
    print(json.dumps(catalog.lottie_info(data), ensure_ascii=False, indent=2))
    return 0


def cmd_gallery(_: argparse.Namespace) -> int:
    assets = catalog.load_manifest(MANIFEST)
    cards = []
    for asset in assets:
        file_path = PUBLIC_DIR / f"{asset.id}.json"
        if not file_path.exists():
            continue
        data = json.loads(file_path.read_text(encoding="utf-8"))
        info = catalog.lottie_info(data)
        colors = "".join(
            f'<span class="sw" title="{c}" style="background:{c}"></span>' for c in info["palette"]
        )
        # `</` trong JSON nhúng sẽ đóng thẻ script sớm.
        payload = json.dumps(data).replace("</", "<\\/")
        cards.append(
            f"""<article class="card {asset.status}" data-status="{asset.status}">
  <div class="stage" data-anim="{html.escape(asset.id)}" data-loop="{str(asset.loop).lower()}"></div>
  <script type="application/json" id="d-{html.escape(asset.id)}">{payload}</script>
  <h3>{html.escape(asset.id)} <em>{asset.status}</em></h3>
  <p>{html.escape(asset.description)}</p>
  <p class="meta">{info['seconds']}s · {info['width']}×{info['height']} · {html.escape(asset.license)} ·
     <a href="{html.escape(asset.source_url)}" target="_blank" rel="noreferrer">nguồn</a></p>
  <div class="pal">{colors}</div>
</article>"""
        )
    page = f"""<!doctype html><html lang="vi"><meta charset="utf-8">
<title>Lottie catalog — duyệt clip</title>
<style>
  body{{margin:0;padding:24px;background:#0b1020;color:#e8eefc;font:14px system-ui,sans-serif}}
  header{{display:flex;gap:16px;align-items:center;margin-bottom:20px}} h1{{font-size:18px;margin:0}}
  button{{background:#1b2440;color:inherit;border:1px solid #33406b;padding:6px 12px;border-radius:6px;cursor:pointer}}
  button.on{{border-color:#F5B841}}
  .grid{{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:16px}}
  .card{{background:#121a33;border-radius:10px;padding:12px;border:2px solid transparent}}
  .card.approved{{border-color:#2f8f5b}} .stage{{background:{BACKGROUND};border-radius:8px;height:220px}}
  h3{{margin:10px 0 4px;font-size:14px}} em{{font-weight:400;color:#8fa0cf;margin-left:6px}}
  p{{margin:4px 0}} .meta{{color:#8fa0cf;font-size:12px}} a{{color:#8fb4ff}}
  .sw{{display:inline-block;width:16px;height:16px;border-radius:4px;margin:2px 4px 0 0;border:1px solid #fff3}}
</style>
<header><h1>Lottie catalog ({len(cards)} clip)</h1>
  <button data-f="all" class="on">Tất cả</button><button data-f="candidate">Ứng viên</button><button data-f="approved">Đã duyệt</button>
  <span class="meta">Đổi status trong manifest.json rồi chạy lại "gallery"; nền xem thử = nền video.</span></header>
<div class="grid">{''.join(cards) or '<p>Manifest chưa có clip nào (hoặc thiếu file).</p>'}</div>
<script src="https://cdn.jsdelivr.net/npm/lottie-web@5.12.2/build/player/lottie.min.js"></script>
<script>
  document.querySelectorAll('.stage').forEach(el => {{
    const data = JSON.parse(document.getElementById('d-' + el.dataset.anim).textContent);
    lottie.loadAnimation({{container: el, renderer: 'svg', loop: el.dataset.loop === 'true', autoplay: true, animationData: data}});
  }});
  document.querySelectorAll('button[data-f]').forEach(b => b.onclick = () => {{
    document.querySelectorAll('button[data-f]').forEach(x => x.classList.toggle('on', x === b));
    document.querySelectorAll('.card').forEach(c => c.style.display = b.dataset.f === 'all' || c.dataset.status === b.dataset.f ? '' : 'none');
  }});
</script></html>"""
    GALLERY_TARGET.write_text(page, encoding="utf-8")
    print("wrote", GALLERY_TARGET)
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)
    for name, fn in (("validate", cmd_validate), ("gallery", cmd_gallery), ("prompt", cmd_prompt), ("credits", cmd_credits)):
        sub.add_parser(name).set_defaults(fn=fn)
    info = sub.add_parser("info")
    info.add_argument("file")
    info.set_defaults(fn=cmd_info)
    args = parser.parse_args()
    return args.fn(args)


if __name__ == "__main__":
    sys.exit(main())
