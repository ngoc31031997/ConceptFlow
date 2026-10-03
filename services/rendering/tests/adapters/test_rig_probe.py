"""The close-up kit (Person poses and framing, ReachingHand) measured in a real
browser by the layout probe: what a hand or a person holds lands where the
hand is. Runs where remotion_project/node_modules and Chromium are present
(the rendering image)."""

from __future__ import annotations

import shutil

import pytest

from adapters.rendering.layout_checker import LayoutChecker
from tests.adapters.test_layout_checker import REAL, _browser_available

needs_real = pytest.mark.skipif(
    not (REAL / "node_modules" / "playwright-core").exists() or shutil.which("node") is None
    or not _browser_available(),
    reason="cần remotion_project/node_modules và Chromium headless shell của Playwright",
)

SCRIPT = """
import React from 'react';
import {registerRoot, Composition, AbsoluteFill} from 'remotion';
import {calculateMetadataFromSegments, Segments, evenLines, lineSpan} from './conceptflow-mini/segments';
import {Stage} from './conceptflow-mini/primitives';
import {Backdrop, Person, Book, Lightbulb} from './conceptflow-mini/illustration';
import {ReachingHand} from './conceptflow-mini/rig';

export const narrations: string[] = ["một", "hai", "ba", "bốn", "năm"];
export const shotLineCounts: number[] = [1, 1, 1, 2];
type ShotProps = {duration: number; lines: number[]};

function Shot1_1({duration, lines}: ShotProps) {
  return (
    <AbsoluteFill>
      <Backdrop color="#FFD84D" />
      <ReachingHand edge="bottom" target={{x: 960, y: 540}} reach={0} pose="grip" size={260}>
        <Lightbulb x={0} y={0} size={120} />
      </ReachingHand>
    </AbsoluteFill>
  );
}

function Shot1_2({duration, lines}: ShotProps) {
  return (
    <AbsoluteFill>
      <Backdrop color="#FFD84D" />
      <ReachingHand edge="bottom" target={{x: 960, y: 540}} reach={1} pose="grip" size={260}>
        <Lightbulb x={0} y={0} size={120} />
      </ReachingHand>
    </AbsoluteFill>
  );
}

function Shot2_1({duration, lines}: ShotProps) {
  return (
    <AbsoluteFill>
      <Backdrop color="#2EC4F0" />
      <Person x={960} y={560} size={600} pose="hold">
        <Book x={0} y={0} size={140} />
      </Person>
    </AbsoluteFill>
  );
}

function Shot2_2({duration, lines}: ShotProps) {
  return (
    <AbsoluteFill>
      <Backdrop color="#FF9EC4" />
      <Person x={960} y={540} size={700} framing="bust"
        pose="think" toPose="shrug" poseT={0.5} mood="happy" />
    </AbsoluteFill>
  );
}

const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1, Shot2_2];

type Seg = {startFrame: number; durationInFrames: number; lines?: number[]};

function CreatorComposition({segments = []}: {segments?: Seg[]}) {
  return (
    <Stage>
      <Segments segments={segments}>
        {(index, segment) => {
          const Shot = SHOTS[index];
          const lines = segment.lines ?? evenLines(segment.durationInFrames, shotLineCounts[index] ?? 1);
          return Shot ? <Shot duration={segment.durationInFrames} lines={lines} /> : null;
        }}
      </Segments>
    </Stage>
  );
}

registerRoot(() => (
  <Composition id="creator" component={CreatorComposition} width={1920} height={1080} fps={30}
    durationInFrames={150} calculateMetadata={calculateMetadataFromSegments} />
));
"""


@pytest.fixture(scope="module")
def probe():
    checker = LayoutChecker(REAL, timeout_seconds=60)
    try:
        yield checker.measure(SCRIPT)
    finally:
        checker.close()


def _centre_of(shot: dict, component: str) -> tuple[float, float]:
    for sample in shot["samples"]:
        assert not sample.get("error"), sample.get("error")
        for el in sample["elements"]:
            if el.get("component") == component:
                r = el["rect"]
                return r["x"] + r["w"] / 2, r["y"] + r["h"] / 2
    raise AssertionError(f"{component} was not drawn in shot {shot['id']}")


@needs_real
def test_a_hand_that_has_not_reached_in_keeps_what_it_holds_below_the_frame(probe):
    _, y = _centre_of(probe["shots"][0], "Lightbulb")
    assert y > 1080


@needs_real
def test_a_hand_that_reached_in_holds_its_object_at_the_target(probe):
    x, y = _centre_of(probe["shots"][1], "Lightbulb")
    assert abs(x - 960) < 60 and abs(y - 540) < 60


@needs_real
def test_a_person_holding_carries_the_object_in_front_of_the_chest(probe):
    x, y = _centre_of(probe["shots"][2], "Book")
    assert abs(x - 960) < 30
    # The 600 px person spans y 260..860; the hands are at chest height.
    assert 400 < y < 700


@needs_real
def test_a_bust_framing_mid_pose_draws_without_error(probe):
    shot = probe["shots"][3]
    assert shot["samples"] and all(not s.get("error") for s in shot["samples"])
    _centre_of(shot, "Person")


@needs_real
def test_a_stub_with_the_lines_prop_is_skipped():
    stubbed = SCRIPT.replace(
        SCRIPT[SCRIPT.index("function Shot2_2"):SCRIPT.index("const SHOTS")],
        "function Shot2_2({duration, lines}: ShotProps) {\n  return null;\n}\n\n",
    )
    checker = LayoutChecker(REAL, timeout_seconds=60)
    try:
        by_id = {s["id"]: s for s in checker.measure(stubbed)["shots"]}
    finally:
        checker.close()
    assert by_id["2.2"].get("skipped") == "stub"
