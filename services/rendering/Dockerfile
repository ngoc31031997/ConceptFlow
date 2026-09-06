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
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
