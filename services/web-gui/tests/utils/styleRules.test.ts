import { describe, it, expect } from "vitest";
import { describeWarning, fixWarningsNote, warningText } from "../../src/utils/styleRules";

describe("style warnings", () => {
  it("puts the rule code first, then the line and the detail", () => {
    const w = describeWarning({ message: '[S3] nét thiếu strokeLinecap="round"', line: 14 });
    expect(warningText(w)).toBe('S3 — dòng 14: nét thiếu strokeLinecap="round"');
  });

  it("keeps a warning without a rule code", () => {
    expect(warningText(describeWarning({ message: "lạ", line: null }))).toBe("lạ");
    expect(warningText(describeWarning({ message: "[S99] x", line: null }))).toBe("S99 — x");
  });

  it("writes a redraw note listing every warning", () => {
    const note = fixWarningsNote([
      describeWarning({ message: "[S2] <rect> vừa tô màu vừa có viền — khối không có viền", line: 3 }),
      describeWarning({ message: "[S12] màu chính không đổi được qua prop color", line: null }),
    ]);
    expect(note.split("\n")).toEqual([
      "Sửa các cảnh báo style sau, giữ nguyên ý hình, tên component và các prop:",
      "- S2 — dòng 3: <rect> vừa tô màu vừa có viền — khối không có viền",
      "- S12 — màu chính không đổi được qua prop color",
    ]);
  });
});
