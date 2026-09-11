FROM python:3.12-slim
# texlive-* packages are required by any Manim script using MathTex/Tex or
# the default Integer/DecimalNumber (which renders digits via MathTex) —
# without them, such scripts fail with "latex failed but did not produce a
# log file" since Rendering now executes arbitrary Creator-authored scripts,
# not just the fixed set of templates the old MVP shipped with.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    libcairo2-dev \
    libpango1.0-dev \
    pkg-config \
    build-essential \
    texlive \
    texlive-latex-extra \
    texlive-fonts-extra \
    texlive-latex-recommended \
    texlive-science \
    tipa \
    dvisvgm \
    && rm -rf /var/lib/apt/lists/*
# Font của bộ nhận diện (CR-017 §Quyết định #2). Manim/Pango **im lặng thay
# bằng font khác** khi font yêu cầu không có, nên thiếu bước này thì mọi video
# vẫn render thành công — chỉ là bằng font của người khác, và không ai phát hiện
# ra cho tới lúc nhìn video đã publish.
#
# Nguồn là repo google/fonts trên GitHub, không phải fonts.google.com/download:
# endpoint download đó đã đổi thành một trang JS trả về HTML thay vì file zip
# (phát hiện lúc build 2026-09-11 — `unzip` báo "End-of-central-directory
# signature not found" vì file tải về là HTML, không phải zip). raw.githubusercontent
# phục vụ đúng file .ttf tĩnh, ổn định hơn một endpoint UI có thể đổi bất cứ lúc nào.
RUN apt-get update && apt-get install -y --no-install-recommends \
    fonts-jetbrains-mono \
    fontconfig \
    curl \
    && mkdir -p /usr/share/fonts/truetype/conceptflow \
    && cd /usr/share/fonts/truetype/conceptflow \
    && curl -fsSL -o CormorantGaramond.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/cormorantgaramond/CormorantGaramond%5Bwght%5D.ttf" \
    && curl -fsSL -o BeVietnamPro-Regular.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/bevietnampro/BeVietnamPro-Regular.ttf" \
    && curl -fsSL -o BeVietnamPro-Bold.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/bevietnampro/BeVietnamPro-Bold.ttf" \
    && curl -fsSL -o BeVietnamPro-SemiBold.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/bevietnampro/BeVietnamPro-SemiBold.ttf" \
    && fc-cache -f \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
