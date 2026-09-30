import { describe, it, expect } from "vitest";
import { describeWarning, fixWarningsNote, parseRuleNames, warningText } from "../../src/utils/styleRules";

// Lines as they appear in authoring-service/.../prompts/illustration_style_vi.txt.
const RULES = `## 1. Hình khối
- [S2] KHÔNG VIỀN quanh khối: khối có màu (fill) không có stroke. → cảnh báo.
- [S3] BO TRÒN: góc chữ nhật có rx ≥ 6 (khối lớn rx 20–40); đầu nét strokeLinecap="round". → cảnh báo.
- [S10] TỐI ĐA 6 MÀU gốc trong một hình (không tính màu dẫn xuất bằng shadeOf). → cảnh báo.
- [S12] Màu chính của vật phải đổi được qua prop color (và các prop màu phụ nếu cần), để Kỹ sư gán màu.
- [S20] Chuyển động NHỎ và ĐỀU: không quá 5% cạnh hộp.`;

describe("style rule names", () => {
  it("reads each rule's name from the served rules text", () => {
    expect(parseRuleNames(RULES)).toEqual({
      S2: "Không viền quanh khối",
      S3: "Bo tròn",
      S10: "Tối đa 6 màu gốc trong một hình",
      S12: "Màu chính của vật phải đổi được qua prop color",
      S20: "Chuyển động nhỏ và đều",
    });
  });

  it("puts the rule name first, then the line and the detail", () => {
    const w = describeWarning({ message: '[S3] nét thiếu strokeLinecap="round"', line: 14 }, parseRuleNames(RULES));
    expect(warningText(w)).toBe('S3 · Bo tròn — dòng 14: nét thiếu strokeLinecap="round"');
  });

  it("keeps a warning without a rule code, or a rule the text does not know", () => {
    expect(warningText(describeWarning({ message: "lạ", line: null }, {}))).toBe("lạ");
    expect(warningText(describeWarning({ message: "[S99] x", line: null }, {}))).toBe("S99 — x");
  });

  it("writes a redraw note listing every warning", () => {
    const names = parseRuleNames(RULES);
    const note = fixWarningsNote([
      describeWarning({ message: "[S2] <rect> vừa tô màu vừa có viền — khối không có viền", line: 3 }, names),
      describeWarning({ message: "[S12] màu chính không đổi được qua prop color", line: null }, names),
    ]);
    expect(note.split("\n")).toEqual([
      "Sửa các cảnh báo style sau, giữ nguyên ý hình, tên component và các prop:",
      "- S2 · Không viền quanh khối — dòng 3: <rect> vừa tô màu vừa có viền — khối không có viền",
      "- S12 · Màu chính của vật phải đổi được qua prop color — màu chính không đổi được qua prop color",
    ]);
  });
});
