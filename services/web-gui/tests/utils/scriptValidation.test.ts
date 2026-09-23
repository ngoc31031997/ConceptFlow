import { describe, it, expect } from "vitest";
import { stripMarkdownCodeFence, validateScript, validateRemotionScript } from "../../src/utils/scriptValidation";

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

const VALID_REMOTION = [
  "import {registerRoot, Composition} from 'remotion';",
  "import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';",
  "import {TitleText} from './conceptflow-mini/primitives';",
  "",
  'export const narrations: string[] = ["một hai", "ba bốn"];',
  "",
  "function CreatorComposition({segments = []}) {",
  "  return <Segments segments={segments}>{(index) => <TitleText>{narrations[index]}</TitleText>}</Segments>;",
  "}",
  "",
  "registerRoot(() => (",
  '  <Composition id="creator" component={CreatorComposition} width={1920} height={1080} fps={30} durationInFrames={150} calculateMetadata={calculateMetadataFromSegments} />',
  "));",
].join("\n");

describe("validateRemotionScript", () => {
  it("chấp nhận code Remotion đúng chuẩn và đếm đúng số câu trong narrations", () => {
    const result = validateRemotionScript(VALID_REMOTION);
    expect(result.isValid).toBe(true);
    expect(result.narrationCount).toBe(2);
  });

  it("báo thiếu export const narrations", () => {
    const result = validateRemotionScript(VALID_REMOTION.replace("export const narrations", "const narrations"));
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("narrations");
  });

  it("báo narrations rỗng", () => {
    const result = validateRemotionScript(
      VALID_REMOTION.replace('["một hai", "ba bốn"]', "[]"),
    );
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("rỗng");
  });

  it("báo thiếu id=\"creator\" trên Composition", () => {
    const result = validateRemotionScript(VALID_REMOTION.replace('id="creator"', 'id="other"'));
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("creator");
  });

  it("báo thiếu calculateMetadata", () => {
    const result = validateRemotionScript(
      VALID_REMOTION.replace("calculateMetadata={calculateMetadataFromSegments}", ""),
    );
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("calculateMetadata");
  });

  it("báo thiếu <Segments>", () => {
    const result = validateRemotionScript(
      VALID_REMOTION.replace(
        "<Segments segments={segments}>{(index) => <TitleText>{narrations[index]}</TitleText>}</Segments>",
        "<TitleText>{narrations[0]}</TitleText>",
      ),
    );
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("Segments");
  });

  it("báo dính dòng markdown ``` ở đầu", () => {
    const result = validateRemotionScript("```tsx\n" + VALID_REMOTION + "\n```");
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("markdown");
  });

  it("báo ngoặc không cân khi code bị cắt cụt", () => {
    // Drop just the closing "));" — everything else (including the required
    // Composition id="creator" tag) stays intact, so this is a pure
    // unbalanced-parens case, not a missing-structure one.
    const truncated = VALID_REMOTION.slice(0, VALID_REMOTION.lastIndexOf("\n));"));
    const result = validateRemotionScript(truncated);
    expect(result.isValid).toBe(false);
    expect(result.message).toContain("cắt cụt");
  });

  it("coi script rỗng là hợp lệ (chưa có gì để báo lỗi)", () => {
    expect(validateRemotionScript("").isValid).toBe(true);
  });

  it("đếm đúng khi lời thoại chứa dấu ] (không bị cắt cụt mảng)", () => {
    const result = validateRemotionScript(
      VALID_REMOTION.replace('["một hai", "ba bốn"]', '["Xem mục [1] nhé", "ba bốn"]'),
    );
    expect(result.isValid).toBe(true);
    expect(result.narrationCount).toBe(2);
  });

  it("đếm đúng khi narrations dùng template literal", () => {
    const result = validateRemotionScript(
      VALID_REMOTION.replace('["một hai", "ba bốn"]', "[`một hai`, `ba bốn`]"),
    );
    expect(result.isValid).toBe(true);
    expect(result.narrationCount).toBe(2);
  });

  it("bỏ qua một mảng narrations để lại trong comment, dùng đúng mảng export const", () => {
    const result = validateRemotionScript(
      '// narrations = ["giả 1", "giả 2", "giả 3"]\n' + VALID_REMOTION,
    );
    expect(result.isValid).toBe(true);
    expect(result.narrationCount).toBe(2);
  });
});
