import { describe, it, expect } from "vitest";
import { validateScript } from "../../src/utils/scriptValidation";

const VALID =
  'from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    def construct(self):\n        self.narrate("một hai ba bốn năm")\n        self.narrate("sáu bảy tám chín mười")';

describe("validateScript", () => {
  it("đếm lời thoại từ self.narrate và cộng thời lượng", () => {
    const result = validateScript(VALID, "vi");
    expect(result.isValid).toBe(true);
    expect(result.narrationCount).toBe(2);
    expect(result.totalWords).toBe(10);
    expect(result.estimatedNarrationSeconds).toBeCloseTo((10 / 140) * 60, 6);
  });

  it("nhận cả nháy đơn", () => {
    expect(
      validateScript("class A(ConceptFlowScene):\n    self.narrate('xin chào')", "vi").narrationCount,
    ).toBe(1);
  });

  it("báo script còn dùng chuẩn trước CR-018", () => {
    const legacy =
      'class A(Scene):\n    def construct(self):\n        # NARRATION: "x"\n        self.wait(AUTO)';
    const result = validateScript(legacy, "vi");
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("chuẩn cũ");
  });

  it("báo khi thiếu class Scene", () => {
    expect(validateScript("x = 1", "vi").isValid).toBe(false);
  });

  it("script rỗng không bị coi là sai", () => {
    expect(validateScript("", "vi").isValid).toBe(true);
  });

  it("dùng WPM đã hiệu chỉnh khi được truyền vào", () => {
    const base = validateScript(VALID, "vi").estimatedNarrationSeconds;
    const fast = validateScript(VALID, "vi", 300).estimatedNarrationSeconds;
    expect(fast).toBeLessThan(base);
  });

  it("không đếm được lời thoại sinh trong vòng lặp — đây là ước lượng tối thiểu", () => {
    // Sau CR-018 narrate() nằm trong vòng lặp là hợp lệ, nhưng phân tích tĩnh
    // không thể biết nó chạy mấy lần. Chỉ lượt dry bên Rendering mới biết.
    const looped =
      'class A(ConceptFlowScene):\n    def construct(self):\n        for i in range(3):\n            self.narrate("lặp")';
    expect(validateScript(looped, "vi").narrationCount).toBe(1);
  });
});
