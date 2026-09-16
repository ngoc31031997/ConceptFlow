"""RemotionScriptRenderer — feature/remotion-engine.

Like test_manim_renderer.py, the subprocess layer (`node render.mjs`,
`ffprobe`) is monkeypatched in every test — these never invoke real Node or
Remotion.
"""

from __future__ import annotations

import json

import pytest
from adapters.rendering.remotion_renderer import RemotionScriptRenderer, _extract_narrations
from domain.errors import AnimationEngineError
from domain.models import NarrationSegment, ScriptRenderRequest

ENTRY_SCRIPT = """
import {registerRoot, Composition} from 'remotion';
export const narrations: string[] = ["dong mot", "dong hai"];
registerRoot(() => <Composition id="creator" component={() => null} />);
"""


def make_request(tmp_path, **overrides) -> ScriptRenderRequest:
    kwargs = dict(
        project_id="proj-1",
        script_content=ENTRY_SCRIPT,
        scene_class_name="creator",
        narration_segments=[
            NarrationSegment(scene_index=0, duration_seconds=2.0),
            NarrationSegment(scene_index=1, duration_seconds=3.0),
        ],
        engine="remotion",
    )
    kwargs.update(overrides)
    return ScriptRenderRequest(**kwargs)


class TestExtractNarrations:
    def test_extracts_double_quoted_array(self):
        assert _extract_narrations(ENTRY_SCRIPT) == ["dong mot", "dong hai"]

    def test_extracts_single_quoted_array(self):
        script = "export const narrations = ['a', 'b', 'c'];"
        assert _extract_narrations(script) == ["a", "b", "c"]

    def test_no_narrations_export_returns_empty(self):
        assert _extract_narrations("export default function() {}") == []


class TestDryRun:
    def test_returns_narrations_in_order(self, tmp_path):
        renderer = RemotionScriptRenderer(project_template_dir=str(tmp_path))
        result = renderer.dry_run(make_request(tmp_path))
        assert result.narrations == ["dong mot", "dong hai"]

    def test_raises_when_no_narrations(self, tmp_path):
        renderer = RemotionScriptRenderer(project_template_dir=str(tmp_path))
        request = make_request(tmp_path, script_content="export default function() {}")
        with pytest.raises(AnimationEngineError, match="no narration"):
            renderer.dry_run(request)


class TestRender:
    def test_render_invokes_node_and_moves_output(self, tmp_path, monkeypatch):
        template_dir = tmp_path / "remotion_project"
        (template_dir / "src").mkdir(parents=True)
        media_root = tmp_path / "media"

        captured_cmd = {}

        def dispatch(cmd, *a, **k):
            if cmd[0] == "node":
                captured_cmd["cmd"] = cmd
                # Simulate the Node driver writing the output file it was told to.
                out_index = cmd.index("--out") + 1
                with open(cmd[out_index], "wb") as f:
                    f.write(b"fake video bytes")
                return type("R", (), {"returncode": 0, "stdout": "", "stderr": ""})()
            # ffprobe
            return type("R", (), {"returncode": 0, "stdout": "12.3", "stderr": ""})()

        monkeypatch.setattr("adapters.rendering.remotion_renderer.subprocess.run", dispatch)

        renderer = RemotionScriptRenderer(
            project_template_dir=str(template_dir), cache_root=str(media_root)
        )
        output_path = str(tmp_path / "final.mp4")
        result = renderer.render(make_request(tmp_path), output_path)

        assert captured_cmd["cmd"][0] == "node"
        assert "--entry" in captured_cmd["cmd"]
        assert "--id" in captured_cmd["cmd"]
        assert "creator" in captured_cmd["cmd"]
        assert result.video_path == output_path
        assert result.video_duration_seconds == 12.3
        # segment 0 starts at frame 0 (t=0s); segment 1 starts after 2s * 30fps = 60 frames = 2.0s
        assert result.wait_offsets == [0.0, 2.0]

        # The entry file lands at the fixed src/ path, not the media dir —
        # see RemotionScriptRenderer._entry_path()'s docstring.
        assert (template_dir / "src" / "CreatorEntry.tsx").read_text(encoding="utf-8") == ENTRY_SCRIPT

        # props.json carries frame-based timing, derived from real durations.
        props_files = list(media_root.glob("*/cf_props.json"))
        assert len(props_files) == 1
        props = json.loads(props_files[0].read_text())
        assert props["segments"] == [
            {"startFrame": 0, "durationInFrames": 60},
            {"startFrame": 60, "durationInFrames": 90},
        ]

    def test_render_raises_on_nonzero_exit(self, tmp_path, monkeypatch):
        template_dir = tmp_path / "remotion_project"
        (template_dir / "src").mkdir(parents=True)
        monkeypatch.setattr(
            "adapters.rendering.remotion_renderer.subprocess.run",
            lambda *a, **k: type("R", (), {"returncode": 1, "stdout": "", "stderr": "boom"})(),
        )
        renderer = RemotionScriptRenderer(
            project_template_dir=str(template_dir), cache_root=str(tmp_path / "media")
        )
        with pytest.raises(AnimationEngineError, match="boom"):
            renderer.render(make_request(tmp_path), str(tmp_path / "out.mp4"))

    def test_render_raises_with_no_segments(self, tmp_path):
        renderer = RemotionScriptRenderer(project_template_dir=str(tmp_path))
        request = make_request(tmp_path, narration_segments=[])
        with pytest.raises(AnimationEngineError, match="no narration_segments"):
            renderer.render(request, str(tmp_path / "out.mp4"))
