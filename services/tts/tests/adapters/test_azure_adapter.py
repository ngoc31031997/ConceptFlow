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


@pytest.fixture(autouse=True)
def no_retry_sleep(monkeypatch):
    """CR-013 made synthesis retry with backoff. Sleeping for real would add
    ~9s to every failure test for no coverage — the delays are asserted
    explicitly in the retry tests below instead."""
    slept: list[float] = []
    monkeypatch.setattr(azure_adapter.time, "sleep", slept.append)
    return slept


@pytest.fixture
def adapter():
    return AzureTTSAdapter(key="test-key", region="southeastasia")


def _http_error(status: int, headers: dict | None = None) -> urllib.error.HTTPError:
    return urllib.error.HTTPError(
        "https://southeastasia.tts.speech.microsoft.com/cognitiveservices/v1",
        status,
        "boom",
        hdrs=headers or {},
        fp=None,
    )


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
    """After the retries are exhausted, a wrong key or a spent quota still has
    to reach RoutingTTSEngine as a TTSEngineError, or the Edge fallback never
    runs (CR-013 FR37.5)."""

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


# --- CR-013: retry with backoff ---


def test_a_transient_failure_is_retried_and_then_succeeds(tmp_path, adapter, monkeypatch):
    """The measured failure mode: scattered bare 401s from a valid key. Without
    this, roughly one scene in five fell back to Edge inaudibly, losing the
    commercial rights and SLA that are the reason to use Azure at all."""
    attempts = {"n": 0}

    def flaky_urlopen(request, timeout=None):
        attempts["n"] += 1
        if attempts["n"] < 3:
            raise _http_error(401)
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", flaky_urlopen)

    output = str(tmp_path / "o.wav")
    duration = adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", output)

    assert attempts["n"] == 3
    assert duration == pytest.approx(1.0, abs=0.05)


def test_it_gives_up_after_max_attempts(tmp_path, adapter, monkeypatch):
    attempts = {"n": 0}

    def always_failing(request, timeout=None):
        attempts["n"] += 1
        raise _http_error(503)

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", always_failing)

    with pytest.raises(TTSEngineError) as exc_info:
        adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert attempts["n"] == azure_adapter.MAX_ATTEMPTS
    assert str(azure_adapter.MAX_ATTEMPTS) in str(exc_info.value)


def test_backoff_grows_between_attempts(tmp_path, adapter, monkeypatch, no_retry_sleep):
    monkeypatch.setattr(
        azure_adapter.urllib.request,
        "urlopen",
        lambda request, timeout=None: (_ for _ in ()).throw(_http_error(503)),
    )

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    # One sleep fewer than attempts — the last failure raises instead of waiting.
    assert no_retry_sleep == [1.5, 3.0, 4.5]


@pytest.mark.parametrize("status", [400, 404])
def test_a_deterministic_rejection_is_not_retried(tmp_path, adapter, monkeypatch, status):
    """Bad SSML or an unknown voice answers the same way every time, so retrying
    only delays the fallback (FR37.2)."""
    attempts = {"n": 0}

    def failing(request, timeout=None):
        attempts["n"] += 1
        raise _http_error(status)

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", failing)

    with pytest.raises(TTSEngineError):
        adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert attempts["n"] == 1


def test_retry_after_header_is_honoured(tmp_path, adapter, monkeypatch, no_retry_sleep):
    attempts = {"n": 0}

    def throttled(request, timeout=None):
        attempts["n"] += 1
        if attempts["n"] == 1:
            raise _http_error(429, headers={"Retry-After": "7"})
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", throttled)

    adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert no_retry_sleep == [7.0]


def test_an_outsized_retry_after_is_clamped(tmp_path, adapter, monkeypatch, no_retry_sleep):
    """One scene must not be able to stall a whole render."""
    attempts = {"n": 0}

    def throttled(request, timeout=None):
        attempts["n"] += 1
        if attempts["n"] == 1:
            raise _http_error(429, headers={"Retry-After": "3600"})
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", throttled)

    adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert no_retry_sleep == [float(azure_adapter.MAX_RETRY_AFTER_SECONDS)]


def test_an_http_date_retry_after_falls_back_to_normal_backoff(
    tmp_path, adapter, monkeypatch, no_retry_sleep
):
    """Retry-After may be an HTTP-date; that form is not parsed, and must not
    crash the retry loop."""
    attempts = {"n": 0}

    def throttled(request, timeout=None):
        attempts["n"] += 1
        if attempts["n"] == 1:
            raise _http_error(429, headers={"Retry-After": "Wed, 09 Sep 2026 12:00:00 GMT"})
        return FakeResponse(wav_bytes())

    monkeypatch.setattr(azure_adapter.urllib.request, "urlopen", throttled)

    adapter.synthesize("xin chao", "azure:vi-VN-NamMinhNeural", str(tmp_path / "o.wav"))

    assert no_retry_sleep == [azure_adapter.RETRY_BACKOFF_SECONDS]


def test_the_outer_ceiling_outlasts_every_attempt_and_its_backoff():
    """If this ever inverts, the ceiling cuts the retry loop short and the
    retries silently stop happening (FR37.3)."""
    worst_case = azure_adapter.MAX_ATTEMPTS * azure_adapter.ATTEMPT_TIMEOUT_SECONDS + (
        azure_adapter.RETRY_BACKOFF_SECONDS * sum(range(1, azure_adapter.MAX_ATTEMPTS))
    )
    assert azure_adapter.SYNTHESIS_TIMEOUT_SECONDS > worst_case
