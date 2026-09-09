"""Unit tests for AzureTTSAdapter (CR-011)."""

from __future__ import annotations

import urllib.error
import wave

import pytest

from adapters.tts_engines import azure_adapter
from adapters.tts_engines.azure_adapter import AzureTTSAdapter, _build_ssml
from domain.errors import TTSEngineError


def wav_bytes(seconds: float = 1.0, rate: int = 24000) -> bytes:
    import io

    buffer = io.BytesIO()
    with wave.open(buffer, "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(rate)
        w.writeframes(b"\x00\x00" * int(rate * seconds))
    return buffer.getvalue()


class FakeResponse:
    def __init__(self, payload: bytes) -> None:
        self._payload = payload

    def read(self) -> bytes:
        return self._payload

    def __enter__(self):
        return self

    def __exit__(self, *args):
        return False


@pytest.fixture
def adapter():
    return AzureTTSAdapter(key="test-key", region="southeastasia")


def test_synthesize_writes_the_returned_wav_and_reports_its_duration(
    tmp_path, adapter, monkeypatch
):
    captured = {}

    def fake_urlopen(request, timeout=None):
        captured["request"] = request
        return FakeResponse(wav_bytes(seconds=2.0))

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", fake_urlopen)
    output = tmp_path / "out.wav"

    duration = adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(output))

    assert duration == pytest.approx(2.0)
    assert output.exists()


def test_the_catalogue_prefix_is_stripped_before_azure_sees_the_voice(
    tmp_path, adapter, monkeypatch
):
    """Azure knows `vi-VN-NamMinhNeural`; the `azure:` prefix exists only to
    keep our catalogue keys distinct from the identical Edge voice."""
    captured = {}

    def fake_urlopen(request, timeout=None):
        captured["body"] = request.data.decode("utf-8")
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", fake_urlopen)

    adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert 'name="vi-VN-NamMinhNeural"' in captured["body"]
    assert "azure:" not in captured["body"]


def test_credentials_and_output_format_are_sent(tmp_path, adapter, monkeypatch):
    captured = {}

    def fake_urlopen(request, timeout=None):
        captured["headers"] = {k.lower(): v for k, v in request.headers.items()}
        captured["url"] = request.full_url
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", fake_urlopen)

    adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert captured["headers"]["ocp-apim-subscription-key"] == "test-key"
    assert captured["headers"]["x-microsoft-outputformat"] == azure_adapter.OUTPUT_FORMAT
    assert captured["url"].startswith("https://southeastasia.tts.speech.microsoft.com/")


@pytest.mark.parametrize("status", [401, 429, 503])
def test_http_errors_become_domain_errors(tmp_path, adapter, monkeypatch, status):
    """A wrong key or a spent quota has to reach RoutingTTSEngine as a
    TTSEngineError, or the Edge fallback never runs."""

    def failing_urlopen(request, timeout=None):
        raise urllib.error.HTTPError(
            request.full_url, status, "boom", hdrs=None, fp=None
        )

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", failing_urlopen)

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))


def test_network_errors_become_domain_errors(tmp_path, adapter, monkeypatch):
    def failing_urlopen(request, timeout=None):
        raise urllib.error.URLError("no route to host")

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", failing_urlopen)

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))


def test_narration_text_is_xml_escaped():
    """Narration is Creator-authored prose; an unescaped `&` or `<` would make
    Azure reject the whole SSML document."""
    ssml = _build_ssml("Tim & Ted <script>", "vi-VN-NamMinhNeural")

    assert "&amp;" in ssml
    assert "&lt;script&gt;" in ssml
    assert "<script>" not in ssml


def test_locale_is_derived_from_the_voice_name():
    assert 'xml:lang="vi-VN"' in _build_ssml("xin chao", "vi-VN-NamMinhNeural")
    assert 'xml:lang="en-US"' in _build_ssml("hello", "en-US-GuyNeural")


def test_is_configured_needs_both_key_and_region(monkeypatch):
    monkeypatch.delenv(azure_adapter.KEY_ENV_VAR, raising=False)
    monkeypatch.delenv(azure_adapter.REGION_ENV_VAR, raising=False)
    assert not azure_adapter.is_configured()

    monkeypatch.setenv(azure_adapter.KEY_ENV_VAR, "k")
    assert not azure_adapter.is_configured()

    monkeypatch.setenv(azure_adapter.REGION_ENV_VAR, "southeastasia")
    assert azure_adapter.is_configured()
