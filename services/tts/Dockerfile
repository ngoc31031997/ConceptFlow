FROM python:3.12-slim
# ffmpeg transcodes the MP3 the Edge voices return into the 24 kHz mono WAV the
# rest of the pipeline reads (adapters/tts_engines/edge_adapter.py).
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .
CMD ["python", "main.py"]
