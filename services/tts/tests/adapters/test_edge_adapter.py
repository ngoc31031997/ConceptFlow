"""Unit tests for EdgeTTSAdapter (ADR-0024)."""

from __future__ import annotations

import glob
import os
import sys
import types

import pytest

from adapters.tts_engines import edge_adapter
from adapters.tts_engines.edge_adapter import EdgeTTSAdapter
from domain.errors import TTSEngineError


@pytest.fixture(autouse=True)
def no_retry_backoff(monkeypatch):
    """Keeps the retry path from costing the suite its real backoff seconds."""
    monkeypatch.setattr(edge_adapter, "RETRY_BACKOFF_SECONDS", 0)


@pytest.fixture
def fake_edge_tts(monkeypatch):
    """Stands in for the edge_tts package, which edge_adapter imports lazily."""

    class FakeCommunicate:
        instances: list[FakeCommunicate] = []
        raises: Exception | None = None
        fail_first: int = 0

        def __init__(self, text: str, voice: str, **kwargs) -> None:
            self.text = text
            self.voice = voice
            self.kwargs = kwargs
            FakeCommunicate.instances.append(self)

        def save_sync(self, path: str) -> None:
            if FakeCommunicate.fail_first > 0:
                FakeCommunicate.fail_first -= 1
                raise RuntimeError("No audio was received.")
            if FakeCommunicate.raises is not None:
                raise FakeCommunicate.raises
            with open(path, "wb") as f:
                f.write(b"fake-mp3")

    FakeCommunicate.instances = []
    FakeCommunicate.raises = None
    FakeCommunicate.fail_first = 0
    module = types.ModuleType("edge_tts")
    module.Communicate = FakeCommunicate
    monkeypatch.setitem(sys.modules, "edge_tts", module)
    return FakeCommunicate


def test_service_failure_becomes_a_domain_error(tmp_path, fake_edge_tts):
    """RoutingTTSEngine only catches TTSEngineError, so anything the library
    raises has to arrive as one or the fallback never runs."""
    fake_edge_tts.raises = RuntimeError("no audio received")
    adapter = EdgeTTSAdapter()

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "out.wav"))


def test_a_refused_request_is_retried(tmp_path, fake_edge_tts, monkeypatch):
    """The endpoint refuses intermittently and there is no second engine to
    degrade to, so a render must not die on the first refusal."""
    monkeypatch.setattr(edge_adapter, "_transcode_to_wav", lambda mp3, out: None)
    fake_edge_tts.fail_first = 2

    EdgeTTSAdapter()._fetch_mp3("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "x.mp3"))

    assert len(fake_edge_tts.instances) == 3


def test_each_attempt_is_capped_so_a_stall_cannot_eat_the_retry_budget(tmp_path, fake_edge_tts, monkeypatch):
    """The endpoint stalls as well as refusing. Without per-attempt caps one
    hung request would consume the whole ceiling and no retry would run."""
    monkeypatch.setattr(edge_adapter, "_transcode_to_wav", lambda mp3, out: None)

    EdgeTTSAdapter()._fetch_mp3("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "x.mp3"))

    assert fake_edge_tts.instances[0].kwargs == {
        "connect_timeout": edge_adapter.CONNECT_TIMEOUT_SECONDS,
        "receive_timeout": edge_adapter.RECEIVE_TIMEOUT_SECONDS,
    }


def test_the_outer_ceiling_outlasts_every_attempt():
    """If the ceiling were tighter than the retry budget it would abort the
    recovery this adapter depends on."""
    worst_case_attempts = edge_adapter.MAX_ATTEMPTS * (
        edge_adapter.CONNECT_TIMEOUT_SECONDS + edge_adapter.RECEIVE_TIMEOUT_SECONDS
    )
    worst_case_backoff = edge_adapter.RETRY_BACKOFF_SECONDS * sum(
        range(1, edge_adapter.MAX_ATTEMPTS)
    )

    assert edge_adapter.SYNTHESIS_TIMEOUT_SECONDS > worst_case_attempts + worst_case_backoff


def test_retries_are_bounded(tmp_path, fake_edge_tts):
    """Retrying forever would hang the saga instead of failing it."""
    fake_edge_tts.fail_first = edge_adapter.MAX_ATTEMPTS + 5

    with pytest.raises(TTSEngineError):
        EdgeTTSAdapter().synthesize(
            "xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "out.wav")
        )

    assert len(fake_edge_tts.instances) == edge_adapter.MAX_ATTEMPTS


def test_transcode_failure_becomes_a_domain_error(tmp_path, fake_edge_tts, monkeypatch):
    def failing_ffmpeg(*args, **kwargs):
        raise TTSEngineError("ffmpeg exited with code 1: boom")

    monkeypatch.setattr(edge_adapter, "_transcode_to_wav", failing_ffmpeg)
    adapter = EdgeTTSAdapter()

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "out.wav"))


def test_intermediate_mp3_is_cleaned_up_after_a_failure(tmp_path, fake_edge_tts):
    fake_edge_tts.raises = RuntimeError("no audio received")
    adapter = EdgeTTSAdapter()

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "out.wav"))

    assert glob.glob(str(tmp_path / "*.mp3")) == []


def test_timeout_becomes_a_domain_error(tmp_path, fake_edge_tts):
    """A hung endpoint has to surface as a failed render rather than pinning the
    saga open forever."""
    import threading

    released = threading.Event()

    def blocking_save(self, path):
        released.wait(timeout=5)

    fake_edge_tts.save_sync = blocking_save
    adapter = EdgeTTSAdapter(timeout_seconds=0)

    try:
        with pytest.raises(TTSEngineError):
            adapter.synthesize("xin chao", "vi-VN-NamMinhNeural", str(tmp_path / "out.wav"))
    finally:
        # The worker outlives the timeout, so let it finish before the next test
        # rather than leaving it writing files underneath one.
        released.set()
        adapter._executor.shutdown(wait=True)


@pytest.mark.skipif(
    os.system("ffmpeg -version > /dev/null 2>&1") != 0, reason="ffmpeg not installed"
)
def test_synthesize_writes_a_readable_wav_and_reports_its_duration(tmp_path, fake_edge_tts):
    """The rest of the pipeline opens this file with the `wave` module, so a
    real transcode has to land as PCM WAV, not the MP3 the service returns."""
    import subprocess

    silence = tmp_path / "silence.mp3"
    subprocess.run(
        ["ffmpeg", "-y", "-loglevel", "error", "-f", "lavfi", "-i",
         "anullsrc=r=24000:cl=mono", "-t", "1", str(silence)],
        check=True,
    )
    mp3_bytes = silence.read_bytes()

    def save_real_mp3(self, path):
        with open(path, "wb") as f:
            f.write(mp3_bytes)

    fake_edge_tts.save_sync = save_real_mp3
    output = tmp_path / "out.wav"

    duration = EdgeTTSAdapter().synthesize("xin chao", "vi-VN-NamMinhNeural", str(output))

    assert output.exists()
    assert duration == pytest.approx(1.0, abs=0.2)
