import { describe, it, expect } from "vitest";
import {
  FLOW_VIEW,
  VIEW_RESEARCH,
  comingRoute,
  isBuilt,
  phaseOf,
  phaseOfView,
  screenRoute,
  screenStatus,
} from "../../src/utils/flow";

describe("steps with a screen of their own", () => {
  const research = FLOW_VIEW[VIEW_RESEARCH - 1];

  it("treats the research step as built though no running step stands behind it", () => {
    expect(research.label).toBe("Nghiên cứu");
    expect(research.server).toEqual([]);
    expect(research.screen).toBe("research");
    expect(research.optional).toBe(true);
    expect(isBuilt(research)).toBe(true);
  });

  it("leaves the steps nobody has built as not built", () => {
    const notBuilt = FLOW_VIEW.filter((v) => !isBuilt(v)).map((v) => v.n);
    expect(notBuilt).toEqual([4, 5, 7, 8, 10, 11, 12, 18]);
  });

  it("addresses the research screen under its video and a not-developed step under a video or alone", () => {
    expect(screenRoute(research, "p1")).toBe("/projects/p1/research");
    expect(comingRoute(4, "p1")).toBe("/projects/p1/coming/4");
    expect(comingRoute(4, "")).toBe("/create/coming/4");
  });

  it("puts the research step open from the config step on and passed once the script is reached", () => {
    expect(screenStatus(0)).toBe("pending");
    expect(screenStatus(1)).toBe("pending");
    expect(screenStatus(2)).toBe("waiting");
    expect(screenStatus(3)).toBe("done");
    expect(screenStatus(13)).toBe("done");
  });

  it("finds the phase of a step by its place in the 21 steps", () => {
    expect(phaseOfView(3)?.name).toBe("Tiền kỳ · Nội dung");
    expect(phaseOfView(9)?.name).toBe("Tiền kỳ · Hình ảnh");
    expect(phaseOfView(21)?.name).toBe("Hậu kỳ");
    expect(phaseOfView(22)).toBeNull();
    // The running-step form still agrees with it.
    expect(phaseOf(3)).toEqual(phaseOfView(6));
  });
});
