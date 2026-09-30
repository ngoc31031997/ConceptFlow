"""CR-051: the 14-step flow is defined three times and must say the same thing.

authoring-service and orchestrator each carry domain/flow.go (separate Go
modules, separate Docker build contexts — ADR-0001/ADR-0029), and the web-gui
names the steps in utils/flow.ts. A step renumbered or renamed in one place only
shows the Creator two different flows, so this pins them together.
"""

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2] / "services"
GO_FLOWS = [
    ROOT / "authoring-service" / "internal" / "domain" / "flow.go",
    ROOT / "orchestrator" / "internal" / "domain" / "flow.go",
]
WEB_FLOW = ROOT / "web-gui" / "src" / "utils" / "flow.ts"


def _go_steps(src: str) -> dict[str, int]:
    """Flow* = N constants, without FlowStepsTotal."""
    return {
        name: int(num)
        for name, num in re.findall(r"^\s*(Flow[A-Za-z]+)\s*=\s*(\d+)\b", src, re.M)
        if name != "FlowStepsTotal"
    }


def _go_labels(src: str, steps: dict[str, int]) -> dict[int, str]:
    block = re.search(r"var FlowStepLabel = map\[int\]string\{(.*?)\n\}", src, re.S)
    assert block, "FlowStepLabel map not found"
    return {steps[name]: label for name, label in re.findall(r"(Flow[A-Za-z]+):\s*\"([^\"]*)\"", block.group(1))}


def _web_labels(src: str) -> list[str]:
    block = re.search(r"export const FLOW_LABELS = \[(.*?)\] as const;", src, re.S)
    assert block, "FLOW_LABELS not found"
    return re.findall(r"\"([^\"]*)\"", block.group(1))


def _web_consts(src: str) -> dict[str, int]:
    return {name: int(num) for name, num in re.findall(r"export const (FLOW_[A-Z_]+) = (\d+);", src)}


def test_both_go_flow_files_are_identical():
    authoring, orchestrator = (p.read_bytes() for p in GO_FLOWS)
    assert authoring == orchestrator, (
        "services/*/internal/domain/flow.go differ: change both copies together"
    )


def test_go_labels_match_the_web_step_rail():
    src = GO_FLOWS[0].read_text(encoding="utf-8")
    steps = _go_steps(src)
    labels = _go_labels(src, steps)
    web = _web_labels(WEB_FLOW.read_text(encoding="utf-8"))
    assert sorted(steps.values()) == list(range(1, len(web) + 1))
    assert [labels[n] for n in range(1, len(web) + 1)] == web


def test_web_step_numbers_match_go():
    steps = _go_steps(GO_FLOWS[0].read_text(encoding="utf-8"))
    web = _web_consts(WEB_FLOW.read_text(encoding="utf-8"))
    # FLOW_ILLUSTRATIONS ↔ FlowIllustrations, FLOW_TTS ↔ FlowTTS, ...
    go_by_upper = {name[len("Flow"):].upper(): num for name, num in steps.items()}
    assert web, "no FLOW_* constants found in flow.ts"
    for name, num in web.items():
        key = name[len("FLOW_"):].replace("_", "")
        assert key in go_by_upper, f"{name} has no Go counterpart"
        assert go_by_upper[key] == num, f"{name} = {num}, Go says {go_by_upper[key]}"
