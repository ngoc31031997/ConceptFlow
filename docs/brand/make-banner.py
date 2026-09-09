#!/usr/bin/env python3
"""Dựng banner kênh YouTube 2560x1440 từ ảnh torus gốc.

Chạy từ gốc repo:  python3 docs/brand/make-banner.py
Cần Pillow:        pip install Pillow

Xuất ra docs/brand/:
  conceptflow-banner-2560.jpg          <- file mang đi upload
  conceptflow-banner-mobile-preview.png <- đúng 1546x423 mà điện thoại hiển thị

Nguyên tắc bố cục: mọi thứ quan trọng nằm trong ô 1546x423 ở chính giữa;
hai bên chỉ là nền loang tối. Xem docs/brand/youtube-avatar-prompts.md.
"""
from PIL import Image, ImageDraw, ImageFilter, ImageFont, ImageEnhance
import os

BRAND = "docs/brand"
SRC = os.path.join(BRAND, "Liquid_mercury_torus_floating_202609091943.jpeg")
W, H = 2560, 1440
SAFE_W, SAFE_H = 1546, 423

src = Image.open(SRC).convert("RGB")

# --- nền: phóng ảnh gốc phủ kín khung, làm mờ mạnh rồi tối hai bên ---
scale = max(W / src.width, H / src.height)
bg = src.resize((round(src.width * scale), round(src.height * scale)), Image.LANCZOS)
bg = bg.crop(((bg.width - W) // 2, (bg.height - H) // 2,
              (bg.width - W) // 2 + W, (bg.height - H) // 2 + H))
bg = bg.filter(ImageFilter.GaussianBlur(70))
bg = ImageEnhance.Brightness(bg).enhance(0.75)

# vignette: tối dần ra rìa trái/phải để hai bên "trống" như spec banner
vig = Image.new("L", (W, H), 0)
vd = ImageDraw.Draw(vig)
vd.ellipse((-W * 0.35, -H * 0.9, W * 1.35, H * 1.9), fill=255)
vig = vig.filter(ImageFilter.GaussianBlur(220))
bg = Image.composite(bg, Image.new("RGB", (W, H), (5, 10, 22)), vig)

# --- chủ thể: cắt vòng xuyến, thu nhỏ vừa dải an toàn 423px ---
MK = 344
mark = src.crop((440, 130, 960, 650)).resize((MK, MK), Image.LANCZOS)
# Mặt nạ toả tròn: đặc tới sát vành xuyến rồi tan hẳn, không lộ mép vuông ô cắt.
c = MK / 2
mask = Image.new("L", (MK, MK))
r_in, r_out = MK * 0.44, MK * 0.50
mask.putdata([
    255 if (r := ((x - c) ** 2 + (y - c) ** 2) ** 0.5) <= r_in
    else 0 if r >= r_out
    else round(255 * (r_out - r) / (r_out - r_in))
    for y in range(MK) for x in range(MK)
])
mask = mask.filter(ImageFilter.GaussianBlur(6))

# --- chữ ---
HEL = "/System/Library/Fonts/Helvetica.ttc"
title_f = ImageFont.truetype(HEL, 112, index=1)
sub_f = ImageFont.truetype(HEL, 44, index=0)
TITLE, SUB = "ConceptFlow", "Toán · Khoa học · kể bằng hình"

d = ImageDraw.Draw(bg)
tw = d.textlength(TITLE, font=title_f)
sw = d.textlength(SUB, font=sub_f)
GAP = 56
lock_w = MK + GAP + max(tw, sw)
x0 = (W - lock_w) / 2          # canh giữa cả cụm trong ô an toàn
bg.paste(mark, (round(x0), (H - MK) // 2), mask)

tx = round(x0 + MK + GAP)
ty = H // 2 - 108
d.text((tx, ty + 4), TITLE, font=title_f, fill=(8, 14, 28))
d.text((tx, ty), TITLE, font=title_f, fill=(242, 247, 255))
d.text((tx + 4, ty + 152), SUB, font=sub_f, fill=(143, 199, 239))
assert x0 > (W - SAFE_W) / 2 and x0 + lock_w < (W + SAFE_W) / 2, "cụm chữ tràn khỏi ô an toàn"

out = os.path.join(BRAND, "conceptflow-banner-2560.jpg")
bg.save(out, quality=92, subsampling=0)

prev = bg.crop(((W - SAFE_W) // 2, (H - SAFE_H) // 2,
                (W + SAFE_W) // 2, (H + SAFE_H) // 2))
prev.save(os.path.join(BRAND, "conceptflow-banner-mobile-preview.png"))
print(out, os.path.getsize(out), bg.size)
