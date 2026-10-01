import { describe, expect, it } from "vitest";
import { formatsFor, isShortFormat } from "../../src/utils/outputMode";
import { skippedReason, stepStatus } from "../../src/utils/flow";
import type { VideoFormat } from "../../src/types";

const format = (id: string, max: number): VideoFormat => ({
  id,
  name: id,
  version: 1,
  min_seconds: 0,
  max_seconds: max,
  beats: [],
});
const FORMATS = [format("case_study_essay_8min", 560), format("quick_explainer_3min", 300), format("vertical_short_60s", 60)];

describe("formats by output mode", () => {
  it("a short only offers short formats and a long video only long ones", () => {
    expect(formatsFor("short", FORMATS).map((f) => f.id)).toEqual(["vertical_short_60s"]);
    expect(formatsFor("long", FORMATS).map((f) => f.id)).toEqual(["case_study_essay_8min", "quick_explainer_3min"]);
    expect(isShortFormat(format("x", 90))).toBe(true);
    expect(isShortFormat(format("x", 91))).toBe(false);
  });

  it("step 12 (cutting clips) does not apply to a vertical short, and says why", () => {
    expect(stepStatus(12, 13, "done", "short")).toBe("skipped");
    expect(stepStatus(12, 13, "done", "both")).toBe("done");
    expect(skippedReason(12, "short")).toContain("9:16");
  });
});
