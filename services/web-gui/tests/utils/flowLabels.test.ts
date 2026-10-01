import { describe, it, expect } from "vitest";
import { AUTO_STEPS, FLOW_LABELS, FLOW_PHASES, FLOW_STEP_PURPOSE, flowTitle, phaseOf, previewRoute, skippedReason } from "../../src/utils/flow";
import { stepLabel } from "../../src/utils/pipelineLabels";

describe("flowTitle", () => {
  it("names a step as the step rail does", () => {
    expect(flowTitle(4)).toBe("Bước 4 — Hình ảnh");
    expect(flowTitle(FLOW_LABELS.length)).toBe("Bước 14 — Đăng video");
  });

  // A hand-edited or stale ?step= reaches flowTitle unchecked (RenderPage view mode).
  it("does not print “undefined” for a step outside the flow", () => {
    expect(flowTitle(99)).toBe("Bước 99");
    expect(flowTitle(0)).toBe("Bước 0");
    expect(flowTitle(Number("abc"))).toBe("Bước");
  });
});

describe("saga step labels under the flow titles", () => {
  // Bước 10's only saga step sits under the title "Bước 10 — Dựng hình", so it
  // carries the same name rather than a second one.
  it("render_scenes carries the rail's name", () => {
    expect(stepLabel("render_scenes")).toBe(FLOW_LABELS[9]);
  });
});

describe("phases of the flow", () => {
  it("groups all 14 steps into five phases, each step exactly once", () => {
    expect(FLOW_PHASES.map((p) => p.name)).toEqual(["Chuẩn bị", "Soạn nội dung", "Duyệt", "Sản xuất", "Hoàn tất"]);
    const steps = FLOW_PHASES.flatMap((p) => [...p.steps]);
    expect(steps).toEqual(Array.from({ length: 14 }, (_, i) => i + 1));
  });

  it("finds the phase of a step, and none for a step outside the flow", () => {
    expect(phaseOf(4)).toEqual({ index: 1, name: "Soạn nội dung", steps: [3, 4, 5, 6] });
    expect(phaseOf(14)?.name).toBe("Hoàn tất");
    expect(phaseOf(0)).toBeNull();
    expect(phaseOf(15)).toBeNull();
  });

  it("marks only the worker-run steps as automatic", () => {
    expect([...AUTO_STEPS].sort((a, b) => a - b)).toEqual([7, 9, 10, 11, 12]);
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

  it("says why each unused step does not apply", () => {
    expect(skippedReason(5)).toContain("Remotion");
    expect(skippedReason(12)).toContain("clip dọc");
  });
});
