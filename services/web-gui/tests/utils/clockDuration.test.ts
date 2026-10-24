import { describe, it, expect } from "vitest";
import { formatClock, parseClock } from "../../src/utils/clockDuration";

describe("formatClock", () => {
  it("writes m:ss under an hour and h:mm:ss from an hour", () => {
    expect(formatClock(0)).toBe("0:00");
    expect(formatClock(59)).toBe("0:59");
    expect(formatClock(270)).toBe("4:30");
    expect(formatClock(3599)).toBe("59:59");
    expect(formatClock(3600)).toBe("1:00:00");
    expect(formatClock(3725)).toBe("1:02:05");
  });
});

describe("parseClock", () => {
  it("reads both clock forms back to seconds", () => {
    expect(parseClock("4:30")).toBe(270);
    expect(parseClock(" 1:02:05 ")).toBe(3725);
    expect(parseClock("0:59")).toBe(59);
  });

  it("reads a bare number as seconds", () => {
    expect(parseClock("90")).toBe(90);
  });

  it("refuses what is not a clock time instead of guessing", () => {
    for (const bad of ["", "abc", "4:60", "1:60:00", "1:2:3:4", "4:3x", "-1:30", "4:"]) {
      expect(parseClock(bad), bad).toBeNull();
    }
  });

  it("round-trips every whole second up to two hours", () => {
    for (let s = 0; s <= 7200; s += 37) expect(parseClock(formatClock(s))).toBe(s);
  });
});
