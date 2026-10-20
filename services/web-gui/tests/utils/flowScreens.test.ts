import { describe, it, expect } from "vitest";
import {
  FLOW_VIEW,
  VIEW_RESEARCH,
  VIEW_STORY,
  comingRoute,
  isBuilt,
  phaseOf,
  phaseOfView,
  screenRoute,
  screenStatus,
  storyRoute,
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

  it("treats the story step as built with a screen under its video", () => {
    const story = FLOW_VIEW[VIEW_STORY - 1];
    expect(story.label).toBe("Câu chuyện");
    expect(story.screen).toBe("story");
    expect(isBuilt(story)).toBe(true);
    expect(screenRoute(story, "p1")).toBe("/projects/p1/story");
    expect(storyRoute("p1")).toBe("/projects/p1/story");
  });

  it("leaves the steps nobody has built as not built", () => {
    const notBuilt = FLOW_VIEW.filter((v) => !isBuilt(v)).map((v) => v.n);
    expect(notBuilt).toEqual([5, 7, 8, 10, 11, 12, 18]);
  });

  it("addresses the research screen under its video and a not-developed step under a video or alone", () => {
    expect(screenRoute(research, "p1")).toBe("/projects/p1/research");
    expect(comingRoute(5, "p1")).toBe("/projects/p1/coming/5");
    expect(comingRoute(5, "")).toBe("/create/coming/5");
  });

  it("opens the research step once the config is confirmed and passes it with approval or the script", () => {
    expect(screenStatus(research, 0, false)).toBe("pending");
    expect(screenStatus(research, 2, false)).toBe("pending");
    expect(screenStatus(research, 3, false)).toBe("waiting");
    expect(screenStatus(research, 3, true)).toBe("done");
    expect(screenStatus(research, 4, false)).toBe("done");
  });

  it("opens the story step only once research is approved or skipped", () => {
    const story = FLOW_VIEW[VIEW_STORY - 1];
    expect(screenStatus(story, 2, true)).toBe("pending");
    expect(screenStatus(story, 3, false)).toBe("pending");
    expect(screenStatus(story, 3, true)).toBe("waiting");
    expect(screenStatus(story, 4, false)).toBe("done");
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
