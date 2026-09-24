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
    && curl -fsSL -o Montserrat.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/montserrat/Montserrat%5Bwght%5D.ttf" \
    && fc-cache -f \
    && rm -rf /var/lib/apt/lists/*

# Node + headless Chrome for the Remotion render engine (feature/remotion-engine).
# Package list and "don't use Alpine" guidance straight from Remotion's own
# Docker docs (https://www.remotion.dev/docs/docker) — this base image is
# already Debian slim (python:3.12-slim), not Alpine, so no base-image change
# was needed, just adding Node + these libs alongside the existing Python/TeX
# toolchain in the same image (feature/remotion-engine keeps both engines in
# one container — see the branch's design notes).
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    gnupg \
    libnss3 \
    libdbus-1-3 \
    libatk1.0-0 \
    libgbm-dev \
    libasound2 \
    libxrandr2 \
    libxkbcommon-dev \
    libxfixes3 \
    libxcomposite1 \
    libxdamage1 \
    libatk-bridge2.0-0 \
    libpango-1.0-0 \
    libcairo2 \
    libcups2 \
    fonts-noto-color-emoji \
    fonts-noto-cjk \
    && curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY remotion_project/package.json remotion_project/
# `npm install`, not `npm ci`: no package-lock.json is committed yet for
# this first cut (nothing has generated one against a real npm registry
# from inside this repo before) — `npm ci` requires one to exist.
RUN cd remotion_project && npm install
COPY . .
# Downloads Chrome Headless Shell into the image at build time (per Remotion's
# Docker docs) rather than on first render — a cold first render in
# production shouldn't pay a surprise download.
RUN cd remotion_project && npx remotion browser ensure

CMD ["python", "main.py"]
