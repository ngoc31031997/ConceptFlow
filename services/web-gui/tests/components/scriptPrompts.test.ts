import { describe, it, expect } from "vitest";
import { buildShortScriptSystemPrompt } from "../../src/components/scriptPrompts";

describe("buildShortScriptSystemPrompt", () => {
  it('bắt buộc bọc toàn bộ construct() trong with self.clip("short")', () => {
    // CR-026 FR70.1: đây là điều kiện DUY NHẤT để generate_clips (CR-007)
    // nhận ra script này là một clip — thiếu dòng này, video vẫn render
    // được nhưng không có clip nào, và Creator không có cảnh báo nào khác
    // ngoài "Chưa có clip nào" ở màn kết quả.
    const prompt = buildShortScriptSystemPrompt("vi", "Vòng lặp for trong Java");
    expect(prompt).toContain('with self.clip("short"):');
  });

  it("gắn chủ đề vào prompt khi có", () => {
    const prompt = buildShortScriptSystemPrompt("vi", "Vòng lặp for trong Java");
    expect(prompt).toContain("Vòng lặp for trong Java");
  });

  it("giữ placeholder khi chưa có chủ đề", () => {
    const prompt = buildShortScriptSystemPrompt("vi", "");
    expect(prompt).toContain("[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]");
  });

  it("chuyển ngôn ngữ lời thoại theo content language, không phải ngôn ngữ hướng dẫn", () => {
    const en = buildShortScriptSystemPrompt("en", "For loops");
    expect(en).toContain("TIẾNG ANH");
    const vi = buildShortScriptSystemPrompt("vi", "Vòng lặp for");
    expect(vi).toContain("TIẾNG VIỆT");
  });

  it("nhấn mạnh đây không phải bản rút gọn của video dài", () => {
    const prompt = buildShortScriptSystemPrompt("vi", "chủ đề");
    expect(prompt).toContain("KHÔNG PHẢI bản rút gọn");
  });
});
