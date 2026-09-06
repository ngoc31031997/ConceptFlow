FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    tar \
    && rm -rf /var/lib/apt/lists/*

# Install the standalone Piper CLI binary (see adapters/tts_engines/piper_adapter.py
# for why this shells out to the binary instead of using the piper-tts PyPI package).
# TARGETARCH is set automatically by BuildKit (amd64/arm64) — picking the matching
# Piper release avoids running an x86_64 binary under Rosetta emulation inside the
# Linux container, which fails with "failed to open elf at /lib64/ld-linux-x86-64.so.2"
# on Apple Silicon hosts.
ARG TARGETARCH
RUN case "$TARGETARCH" in \
      amd64) PIPER_ARCH=x86_64 ;; \
      arm64) PIPER_ARCH=aarch64 ;; \
      *) echo "Unsupported TARGETARCH: $TARGETARCH" >&2; exit 1 ;; \
    esac && \
    curl -L -o /tmp/piper.tar.gz "https://github.com/rhasspy/piper/releases/latest/download/piper_linux_${PIPER_ARCH}.tar.gz" && \
    tar -xzf /tmp/piper.tar.gz -C /usr/local/bin --strip-components=1 && \
    rm /tmp/piper.tar.gz

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Bundle Piper voice models at build time — avoids any network dependency at
# runtime. Files are named by voice_id to match adapters/tts_engines/voice_registry.py.
# The 4 voices (vi/en × female/male) are CR-001's chosen set; vi_VN-vivos-x_low is
# the only male Vietnamese voice Piper publishes, hence its lower quality tier.
ARG PIPER_VOICES_BASE=https://huggingface.co/rhasspy/piper-voices/resolve/main
RUN mkdir -p /app/voices && \
    for spec in \
      "vi/vi_VN/vais1000/medium/vi_VN-vais1000-medium" \
      "vi/vi_VN/vivos/x_low/vi_VN-vivos-x_low" \
      "en/en_US/lessac/medium/en_US-lessac-medium" \
      "en/en_US/ryan/high/en_US-ryan-high" \
    ; do \
      name="$(basename "$spec")" && \
      curl -fL -o "/app/voices/${name}.onnx" "${PIPER_VOICES_BASE}/${spec}.onnx" && \
      curl -fL -o "/app/voices/${name}.onnx.json" "${PIPER_VOICES_BASE}/${spec}.onnx.json" ; \
    done

COPY . .
CMD ["python", "main.py"]
