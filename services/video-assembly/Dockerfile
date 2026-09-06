FROM python:3.12-slim
# fonts-dejavu-core is required for burned-in subtitles: the base image ships
# no fonts at all, so libass would render empty boxes. DejaVu covers Vietnamese
# diacritics, which most default fonts do not.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    fonts-dejavu-core \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
