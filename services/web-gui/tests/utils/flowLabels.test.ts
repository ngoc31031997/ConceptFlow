import { describe, it, expect } from "vitest";
import {
  AUTO_STEPS,
  FLOW_LABELS,
  FLOW_STEP_PURPOSE,
  FLOW_VIEW,
  FLOW_VIEW_PHASES,
  flowStepName,
  flowTitle,
  phaseOf,
  previewRoute,
} from "../../src/utils/flow";
import { stepLabel } from "../../src/utils/pipelineLabels";

describe("flowTitle", () => {
  it("names a step as the step rail does", () => {
    expect(flowTitle(4)).toBe("Bước 9 — Storyboard");
    expect(flowTitle(FLOW_LABELS.length)).toBe("Bước 21 — Publish");
    expect(flowTitle(12)).toBe("Bước 20 — Kết quả");
  });

  it("gives a screen title the bare name, since the line above it carries the number", () => {
    expect(flowStepName(4)).toBe("Storyboard");
    expect(flowStepName(1)).toBe("Ý tưởng ban đầu");
    expect(flowStepName(2)).toBe("Định hướng sáng tạo");
    expect(flowStepName(99)).toBe("Bước 99");
  });

  // A hand-edited or stale ?step= reaches flowTitle unchecked (RenderPage view mode).
  it("does not print “undefined” for a step outside the flow", () => {
    expect(flowTitle(99)).toBe("Bước 99");
    expect(flowTitle(0)).toBe("Bước 0");
    expect(flowTitle(Number("abc"))).toBe("Bước");
  });
});

describe("saga step labels under the flow titles", () => {
  // Bước 10's only saga step sits under the title "Render", so it
  // carries the same name rather than a second one.
  it("render_scenes carries the rail's name", () => {
    expect(stepLabel("render_scenes")).toBe(FLOW_LABELS[9]);
  });
});

describe("phases of the flow", () => {
  it("shows the 21 steps of the mockup in the four phases, numbered in order", () => {
    expect(FLOW_VIEW_PHASES.map((p) => p.name)).toEqual(["Tiền kỳ · Nội dung", "Tiền kỳ · Hình ảnh", "Sản xuất", "Hậu kỳ"]);
    expect(FLOW_VIEW.map((v) => v.n)).toEqual(Array.from({ length: 21 }, (_, i) => i + 1));
    expect(FLOW_VIEW.map((v) => v.phase)).toEqual([...FLOW_VIEW.map((v) => v.phase)].sort((a, b) => a - b));
    expect(FLOW_VIEW.filter((v) => v.phase === 0)).toHaveLength(7);
  });

  it("puts every running step under exactly one step of the view, in order", () => {
    const shown = FLOW_VIEW.flatMap((v) => [...v.server]);
    expect(shown).toEqual(Array.from({ length: 13 }, (_, i) => i + 1));
    expect(FLOW_VIEW.filter((v) => v.server.length === 0).map((v) => v.n)).toEqual([3, 4, 5, 7, 8, 10, 11, 12, 18]);
  });

  it("finds the phase of a step, and none for a step outside the flow", () => {
    expect(phaseOf(4)).toEqual({ index: 1, name: "Tiền kỳ · Hình ảnh", tone: "accent" });
    expect(phaseOf(13)?.name).toBe("Hậu kỳ");
    expect(phaseOf(0)).toBeNull();
    expect(phaseOf(14)).toBeNull();
  });

  it("marks only the worker-run steps as automatic", () => {
    expect([...AUTO_STEPS].sort((a, b) => a - b)).toEqual([7, 9, 10, 11]);
  });

  it("marks the steps the Creator approves", () => {
    expect(FLOW_VIEW.filter((v) => v.gate).map((v) => v.n)).toEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 16]);
  });
});

describe("step previews", () => {
  it("has one purpose sentence per step", () => {
    expect(FLOW_STEP_PURPOSE).toHaveLength(FLOW_LABELS.length);
  });

  it("keeps the preview under the project when there is one", () => {
    expect(previewRoute(9, "p1")).toBe("/projects/p1/preview/9");
    expect(previewRoute(9, "")).toBe("/create/preview/9");
  });
});
