"""The kits and skeletons the Remotion bundle draws with, read by Python from the same JSON the bundle imports."""

import json
from pathlib import Path

import pytest

from domain.kits import UnknownKitError, load_kits, load_skeletons, require_kit

PROJECT = Path(__file__).resolve().parents[2] / "remotion_project"


def test_the_shipped_kit_has_a_version():
    kits = load_kits(PROJECT)
    assert kits["phatsang"] >= 1 and all(isinstance(v, int) for v in kits.values())


def test_every_skeleton_has_exactly_one_root_and_only_known_parents():
    for name, tree in load_skeletons(PROJECT).items():
        roots = [bone for bone, parent in tree.items() if parent is None]
        assert len(roots) == 1, name
        assert all(parent is None or parent in tree for parent in tree.values()), name


def test_the_humanoid_has_the_bones_a_person_is_drawn_with():
    humanoid = load_skeletons(PROJECT)["humanoid"]
    assert {"hips", "spine", "head", "armL", "foreArmL", "armR", "foreArmR", "thighL", "shinL", "thighR", "shinR"} == set(humanoid)


@pytest.mark.parametrize("key", ["", "nope"])
def test_a_missing_or_unknown_kit_key_names_the_kits_that_exist(key):
    with pytest.raises(UnknownKitError) as err:
        require_kit({"phatsang": 1}, key)
    assert ("kit_key" in str(err.value)) and (key == "" or "phatsang" in str(err.value))


def test_a_known_kit_returns_its_version(tmp_path):
    (tmp_path / "src" / "conceptflow-mini").mkdir(parents=True)
    (tmp_path / "src" / "conceptflow-mini" / "kits.json").write_text(json.dumps({"a": {"version": 7}}))
    assert load_kits(tmp_path) == {"a": 7}
    assert require_kit(load_kits(tmp_path), "a") == 7
