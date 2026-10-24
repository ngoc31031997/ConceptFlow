import { describe, it, expect } from "vitest";
import { sanitizeIdeaText } from "../../src/utils/sanitizeIdeaText";

describe("sanitizeIdeaText", () => {
  it("đổi dấu nháy kép thẳng thành nháy cong, mở sau khoảng trắng và đóng sau chữ", () => {
    expect(sanitizeIdeaText('Vì sao người ta nói "ăn sung là ăn xác ong"? Có thật không?')).toBe(
      "Vì sao người ta nói “ăn sung là ăn xác ong”? Có thật không?",
    );
  });

  it("nháy ở đầu chuỗi hoặc sau dấu mở là nháy mở", () => {
    expect(sanitizeIdeaText('"Sung" và ("ong")')).toBe("“Sung” và (“ong”)");
  });

  it("gõ từng ký tự cũng cho kết quả giống dán cả đoạn", () => {
    const typed = [...'nói "a" đi'].reduce((acc, ch) => sanitizeIdeaText(acc + ch), "");
    expect(typed).toBe(sanitizeIdeaText('nói "a" đi'));
  });

  it("đổi dấu gạch chéo ngược và bỏ ký tự điều khiển, giữ xuống dòng", () => {
    expect(sanitizeIdeaText("a\\b\u0000c\u0007\nd\r\ne")).toBe("a/bc\nd\ne");
  });

  it("giữ nguyên tiếng Việt, nháy đơn và chữ thường", () => {
    const text = "Rơi vào hố đen: người ở xa thấy gì? It's fine.";
    expect(sanitizeIdeaText(text)).toBe(text);
    expect(sanitizeIdeaText("")).toBe("");
  });
});
