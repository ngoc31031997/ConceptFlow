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
RUN apt-get update && apt-get install -y --no-install-recommends \
    fonts-jetbrains-mono \
    fontconfig \
    unzip \
    curl \
    && mkdir -p /usr/share/fonts/truetype/conceptflow \
    && curl -fsSL -o /tmp/cormorant.zip \
       "https://fonts.google.com/download?family=Cormorant%20Garamond" \
    && curl -fsSL -o /tmp/bevietnam.zip \
       "https://fonts.google.com/download?family=Be%20Vietnam%20Pro" \
    && unzip -jo /tmp/cormorant.zip '*.ttf' -d /usr/share/fonts/truetype/conceptflow \
    && unzip -jo /tmp/bevietnam.zip '*.ttf' -d /usr/share/fonts/truetype/conceptflow \
    && fc-cache -f \
    && rm -rf /tmp/*.zip /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
