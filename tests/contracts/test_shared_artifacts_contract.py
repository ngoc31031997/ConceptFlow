"""Every service's artifact_paths helper must agree with docs/contracts/shared-artifacts.md."""

import importlib.util
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2] / "services"


def _load(service: str):
    path = ROOT / service / "adapters" / "storage" / "artifact_paths.py"
    spec = importlib.util.spec_from_file_location(f"{service.replace('-', '_')}_artifact_paths", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_paths_match_the_contract():
    tts, rendering, va = _load("tts"), _load("rendering"), _load("video-assembly")
    assert tts.compute_audio_path("P", 3, "v", "x").startswith("/shared/P/audio/")
    assert rendering.compute_video_path("P") == "/shared/P/video/rendered.mp4"
    assert rendering.compute_timing_path("P") == "/shared/P/video/timing.json"
    assert va.video_output_path("P") == "/shared/P/video/final.mp4"
    assert va.caption_output_path("P") == "/shared/P/video/final.srt"
    assert va.clip_output_path("P", "s", "tiktok") == "/shared/P/clips/s_tiktok.mp4"
    assert rendering.compute_channel_asset_path("intro", "q").startswith("/shared/channel-assets/intro/q/")
    assert va.normalized_channel_asset_path("intro", "q").startswith("/shared/channel-assets/intro/q/")


def test_each_purge_touches_only_its_own_files(tmp_path):
    tts, rendering, va = _load("tts"), _load("rendering"), _load("video-assembly")
    files = {
        "tts": "audio/0.wav", "rendering": "video/rendered.mp4", "rendering2": "video/timing.json",
        "va": "video/final.mp4", "va2": "video/final.srt", "va3": "clips/a_x.mp4",
        "gateway": "thumbnail/t.png", "gateway2": "music/m.mp3",
    }

    def build():
        for rel in files.values():
            p = tmp_path / "P" / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_bytes(b"x")

    def present():
        return {k for k, rel in files.items() if (tmp_path / "P" / rel).exists()}

    for mod in (tts, rendering, va):
        mod.SHARED_VOLUME_ROOT = str(tmp_path)
    build()
    tts.purge_project_artifacts("P")
    assert present() == set(files) - {"tts"}
    rendering.purge_project_artifacts("P")
    assert present() == set(files) - {"tts", "rendering", "rendering2"}
    va.purge_project_artifacts("P")
    assert present() == {"gateway", "gateway2"}  # the Gateway's uploads are not theirs to delete
