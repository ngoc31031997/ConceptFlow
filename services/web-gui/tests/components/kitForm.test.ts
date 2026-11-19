import { describe, expect, it } from "vitest";
import { draftToKit, kitDraftProblem, kitToDraft } from "../../src/components/reference/kitForm";

describe("kitForm", () => {
  it("starts a new kit with valid starting values", () => {
    const draft = kitToDraft(null);
    expect(kitDraftProblem(draft)).toBeNull();
    expect(draft["previewLook.focus"]).toBe("figures");
  });

  it("round-trips a kit through the draft", () => {
    const kit = draftToKit({ ...kitToDraft(null), rimWidth: "0.035", "dof.near": "10", ink: "#2B2118" });
    expect(kit.rimWidth).toBe(0.035);
    expect(kit.dof.near).toBe(10);
    expect(kit.ink).toBe("#2B2118");
    expect(kitToDraft(kit).rimWidth).toBe("0.035");
  });

  it("names the field at fault", () => {
    const base = kitToDraft(null);
    expect(kitDraftProblem({ ...base, grain: "1.5" })).toContain("Hạt");
    expect(kitDraftProblem({ ...base, "dof.sky": "-1" })).toContain("Trời");
    expect(kitDraftProblem({ ...base, "previewLook.grade.warmth": "-1" })).toBeNull();
    expect(kitDraftProblem({ ...base, "previewLook.grade.warmth": "-1.5" })).toContain("-1 đến 1");
    expect(kitDraftProblem({ ...base, ink: "red" })).toContain("#RRGGBB");
    expect(kitDraftProblem({ ...base, bloom: "" })).toContain("Bloom");
  });
});
