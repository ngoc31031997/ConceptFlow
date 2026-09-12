import { describe, it, expect } from "vitest";
import { stripMarkdownCodeFence, validateScript } from "../../src/utils/scriptValidation";

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

  it("báo còn dính dòng markdown ``` thay vì lỗi cú pháp Python mơ hồ", () => {
    // Bug report 2026-09-12: Creator copy nguyên khối ```python ... ``` từ AI
    // vào ScriptEditor. ScriptEditor tự gỡ khối trọn vẹn qua
    // stripMarkdownCodeFence; luật này là lưới an toàn cho phần còn sót (ví dụ
    // chỉ còn dòng mở, thiếu dòng đóng).
    const withFence = "```python\nfrom conceptflow import *\n";
    const result = validateScript(withFence, "vi");
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("```");
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

describe("stripMarkdownCodeFence", () => {
  it("gỡ khối ```python ... ``` bọc trọn script", () => {
    const wrapped = "```python\n" + VALID + "\n```";
    expect(stripMarkdownCodeFence(wrapped)).toBe(VALID);
  });

  it("gỡ khối ``` trơn, không kèm tên ngôn ngữ", () => {
    const wrapped = "```\n" + VALID + "\n```";
    expect(stripMarkdownCodeFence(wrapped)).toBe(VALID);
  });

  it("giữ nguyên script không bọc trong khối markdown", () => {
    expect(stripMarkdownCodeFence(VALID)).toBe(VALID);
  });

  it("giữ nguyên nếu chỉ có dòng mở, thiếu dòng ``` đóng", () => {
    // Không khớp trọn khối — đây là ca mà scriptValidation's LEADING_FENCE_RE
    // bắt tiếp, không phải hàm gỡ này.
    const openOnly = "```python\n" + VALID;
    expect(stripMarkdownCodeFence(openOnly)).toBe(openOnly);
  });

  it("gỡ khối fence dù AI thêm câu mở đầu trước khối code", () => {
    const withPreamble = "Đây là script đã chỉnh sửa:\n```python\n" + VALID + "\n```";
    expect(stripMarkdownCodeFence(withPreamble)).toBe(VALID);
  });

  it("gỡ khối fence dù AI thêm lời chào sau khối code", () => {
    const withTrailer = "```python\n" + VALID + "\n```\nNếu cần chỉnh gì thêm, cứ nói nhé!";
    expect(stripMarkdownCodeFence(withTrailer)).toBe(VALID);
  });

  it("gỡ khối fence dù AI thêm cả câu mở đầu lẫn lời chào cuối", () => {
    const wrapped = "Chắc chắn rồi!\n```python\n" + VALID + "\n```\nChúc bạn quay video vui vẻ.";
    expect(stripMarkdownCodeFence(wrapped)).toBe(VALID);
  });
});
