"""Shared stills: one frame of a sample composition, rendered once and cached by what it shows."""

import os
import time

import pytest

from application.stills import StillsUseCase
from domain.kits import UnknownKitError


class FakeRenderer:
    """Writes a PNG-looking file where it is told to, and counts the renders."""

    def __init__(self):
        self.calls = []

    def still(self, kit_key, composition, output, frame=10, props=None):
        self.calls.append((kit_key, composition, frame, props))
        with open(output, "wb") as f:
            f.write(b"\x89PNG-" + composition.encode())


@pytest.fixture
def stills(tmp_path):
    renderer = FakeRenderer()
    return StillsUseCase(renderer, {"phatsang": 1}, tmp_path / "stills"), renderer, tmp_path / "stills"


def test_the_same_content_is_rendered_once_and_then_served_from_the_cache(stills):
    use_case, renderer, directory = stills
    first = use_case.render("sample-fig-wide", "phatsang")
    second = use_case.render("sample-fig-wide", "phatsang")
    assert first.cache_hit is False and second.cache_hit is True
    assert first.file == second.file and len(renderer.calls) == 1
    assert (directory / first.file).read_bytes().startswith(b"\x89PNG")
    assert len(first.file) == 64 + len(".png")


def test_anything_that_changes_the_picture_changes_the_file(stills):
    use_case, _, _ = stills
    base = use_case.render("sample-fig-wide", "phatsang").file
    assert use_case.render("sample-fig-tall", "phatsang").file != base
    assert use_case.render("sample-fig-wide", "phatsang", frame=20).file != base
    assert use_case.render("sample-fig-wide", "phatsang", {"x": 1}).file != base


def test_a_new_kit_version_never_reuses_an_old_still(tmp_path):
    renderer = FakeRenderer()
    v1 = StillsUseCase(renderer, {"phatsang": 1}, tmp_path)
    v2 = StillsUseCase(renderer, {"phatsang": 2}, tmp_path)
    assert v1.render("sample-fig-wide", "phatsang").file != v2.render("sample-fig-wide", "phatsang").file
    assert len(renderer.calls) == 2


@pytest.mark.parametrize("kit", ["", "nope"])
def test_a_missing_or_unknown_kit_is_refused_before_anything_is_rendered(stills, kit):
    use_case, renderer, _ = stills
    with pytest.raises(UnknownKitError):
        use_case.render("sample-fig-wide", kit)
    assert renderer.calls == []


def test_a_failed_render_leaves_no_half_written_file(tmp_path):
    class Broken(FakeRenderer):
        def still(self, kit_key, composition, output, frame=10, props=None):
            with open(output, "wb") as f:
                f.write(b"partial")
            raise RuntimeError("browser died")

    use_case = StillsUseCase(Broken(), {"phatsang": 1}, tmp_path)
    with pytest.raises(RuntimeError):
        use_case.render("sample-fig-wide", "phatsang")
    assert list(tmp_path.iterdir()) == []


def test_pruning_deletes_only_old_stills_nobody_keeps(stills):
    use_case, _, directory = stills
    kept = use_case.render("sample-fig-wide", "phatsang").file
    old = use_case.render("sample-fig-tall", "phatsang").file
    fresh = use_case.render("sample-space-wide", "phatsang").file
    foreign = directory / "notes.txt"
    foreign.write_text("not a still")
    long_ago = time.time() - 48 * 3600
    for name in (kept, old):
        os.utime(directory / name, (long_ago, long_ago))
    removed = use_case.prune([kept], min_age_hours=24)
    assert removed == [old]
    assert (directory / kept).exists() and (directory / fresh).exists() and foreign.exists()
    assert not (directory / old).exists()


def test_pruning_with_a_negative_age_is_an_error_and_an_empty_directory_is_fine(tmp_path):
    use_case = StillsUseCase(FakeRenderer(), {"phatsang": 1}, tmp_path / "missing")
    assert use_case.prune([], 24) == []
    with pytest.raises(ValueError):
        use_case.prune([], -1)
