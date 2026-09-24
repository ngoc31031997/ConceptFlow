FROM python:3.12-slim
# fonts-dejavu-core is required for burned-in subtitles: the base image ships
# no fonts at all, so libass would render empty boxes. DejaVu covers Vietnamese
# diacritics, which most default fonts do not.
#
# Be Vietnam Pro and Montserrat are the other subtitle fonts a Creator can pick
# (adapters/assembly/subtitle_file.SUBTITLE_FONTS). Fetched from the google/fonts
# repo for the same reason services/rendering/Dockerfile gives: the
# fonts.google.com download endpoint now returns an HTML page, not a zip.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    fonts-dejavu-core \
    fontconfig \
    curl \
    && mkdir -p /usr/share/fonts/truetype/conceptflow \
    && cd /usr/share/fonts/truetype/conceptflow \
    && curl -fsSL -o BeVietnamPro-Regular.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/bevietnampro/BeVietnamPro-Regular.ttf" \
    && curl -fsSL -o Montserrat.ttf \
       "https://raw.githubusercontent.com/google/fonts/main/ofl/montserrat/Montserrat%5Bwght%5D.ttf" \
    && fc-cache -f \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
