import { describe, it, expect } from "vitest";
import { FLOW_LABELS, flowTitle } from "../../src/utils/flow";
import { stepLabel } from "../../src/utils/pipelineLabels";

describe("flowTitle", () => {
  it("names a step as the step rail does", () => {
    expect(flowTitle(4)).toBe("Bước 4 — Visual");
    expect(flowTitle(FLOW_LABELS.length)).toBe("Bước 14 — Publish");
  });

  // A hand-edited or stale ?step= reaches flowTitle unchecked (RenderPage view mode).
  it("does not print “undefined” for a step outside the flow", () => {
    expect(flowTitle(99)).toBe("Bước 99");
    expect(flowTitle(0)).toBe("Bước 0");
    expect(flowTitle(Number("abc"))).toBe("Bước");
  });
});

describe("saga step labels under the flow titles", () => {
  // Bước 10's only saga step sits under the title "Bước 10 — Render"; it used
  // to say "Render hoạt hình", a second name for the same step.
  it("render_scenes carries the rail's name", () => {
    expect(stepLabel("render_scenes")).toBe(FLOW_LABELS[9]);
  });
});
