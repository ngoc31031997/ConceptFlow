"""The skeletons the Remotion bundle draws with, read by Python from the same JSON the bundle imports."""

from pathlib import Path

from domain.kits import load_skeletons

PROJECT = Path(__file__).resolve().parents[2] / "remotion_project"


def test_every_skeleton_has_exactly_one_root_and_only_known_parents():
    for name, tree in load_skeletons(PROJECT).items():
        roots = [bone for bone, parent in tree.items() if parent is None]
        assert len(roots) == 1, name
        assert all(parent is None or parent in tree for parent in tree.values()), name


def test_the_humanoid_has_the_bones_a_person_is_drawn_with():
    humanoid = load_skeletons(PROJECT)["humanoid"]
    bones = {"hips", "spine", "head", "armL", "foreArmL", "armR", "foreArmR"}
    bones |= {"thighL", "shinL", "thighR", "shinR"}
    assert bones == set(humanoid)
