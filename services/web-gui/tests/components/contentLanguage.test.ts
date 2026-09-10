import { describe, it, expect } from "vitest";
import { buildBeatSheetSection, buildGenerationSystemPrompt } from "../../src/components/scriptPrompts";
import {
  END_SCREEN_SNIPPETS,
  HOOK_SNIPPETS,
  SCRIPT_TEMPLATES,
} from "../../src/components/scriptTemplates";

/**
 * CR-008 regressions. Choosing English used to change only the TTS voice: the
 * starter script, the AI prompts and the narration they asked for all stayed
 * Vietnamese.
 */
// The prompts are pure builders now, so the language rules are asserted on the
// text itself rather than through whichever screen happens to display it.
function openSystemPrompt(contentLanguage: "vi" | "en"): { value: string } {
  return { value: buildGenerationSystemPrompt(contentLanguage) };
}

describe("content language drives the AI prompts", () => {
  it("offers a starter script per language", () => {
    expect(SCRIPT_TEMPLATES.vi).toContain("# NARRATION:");
    expect(SCRIPT_TEMPLATES.en).toContain("# NARRATION:");
    expect(SCRIPT_TEMPLATES.vi).not.toEqual(SCRIPT_TEMPLATES.en);
  });

  it("keeps both starter scripts structurally valid for the pipeline", () => {
    // The renderer rejects a script whose NARRATION markers and wait(AUTO)
    // calls do not line up one-to-one (CR-002 FR10.5), so a template that
    // drifts would hand the Creator a script that cannot render.
    for (const template of [SCRIPT_TEMPLATES.vi, SCRIPT_TEMPLATES.en]) {
      const markers = template.match(/# NARRATION: "/g) ?? [];
      const waits = template.match(/self\.wait\(AUTO\)/g) ?? [];
      expect(markers.length).toBeGreaterThan(0);
      expect(waits.length).toBe(markers.length);
    }
  });

  it("asks the AI for English narration when the project is English", () => {
    const prompt = openSystemPrompt("en");

    expect(prompt.value).toContain("TIẾNG ANH");
    expect(prompt.value).not.toContain("phải viết bằng TIẾNG VIỆT");
  });

  it("asks the AI for Vietnamese narration when the project is Vietnamese", () => {
    const prompt = openSystemPrompt("vi");

    expect(prompt.value).toContain("TIẾNG VIỆT");
  });

  it("keeps the prompt's own instructions in Vietnamese for both languages", () => {
    // The two axes are separate: the Creator reads Vietnamese, the audience
    // hears English. Translating the instructions would be the wrong fix.
    const prompt = openSystemPrompt("en");

    expect(prompt.value).toContain("RÀNG BUỘC ĐỊNH DẠNG BẮT BUỘC");
  });

  it("states the render limits that CR-003 actually raised", () => {
    // The prompt used to tell the AI the timeout was 300s and to target 2-4
    // minute videos, both stale after CR-003 and both nudging it to write
    // shorter videos than the monetization goal wants.
    const prompt = openSystemPrompt("vi");

    expect(prompt.value).toContain("1800s");
    expect(prompt.value).not.toContain("timeout 300s");
    // CR-019: mốc 8 phút chỉ có ý nghĩa sau khi bật kiếm tiền; kênh mới phải
    // tối ưu tỉ lệ giữ chân trước đã.
    expect(prompt.value).toContain("6-8 phút");
  });
});

describe("hook and end-screen snippets (CR-006 FR17)", () => {
  it("keeps every snippet's NARRATION and wait(AUTO) balanced", () => {
    // The renderer rejects a script where these drift (CR-002 FR10.5), and
    // pasting a snippet must never be what breaks it.
    const snippets = [
      HOOK_SNIPPETS.vi,
      HOOK_SNIPPETS.en,
      END_SCREEN_SNIPPETS.vi,
      END_SCREEN_SNIPPETS.en,
    ];
    for (const snippet of snippets) {
      const markers = snippet.match(/# NARRATION: "/g) ?? [];
      const autoWaits = snippet.match(/self\.wait\(AUTO\)/g) ?? [];
      expect(markers.length).toBe(1);
      expect(autoWaits.length).toBe(1);
    }
  });

  it("does not put a wait(AUTO) inside a loop in any snippet", () => {
    // A wait(AUTO) in a loop fires more often than its single marker, which is
    // the most common way scripts break the count.
    for (const snippet of [HOOK_SNIPPETS.vi, END_SCREEN_SNIPPETS.en]) {
      expect(snippet).not.toMatch(/for .*:\s*[\s\S]*self\.wait\(AUTO\)/);
    }
  });

  it("leaves a trailing hold for YouTube's end-screen elements", () => {
    // A fixed wait, not AUTO — there is no narration over it.
    expect(END_SCREEN_SNIPPETS.vi).toContain("self.wait(8)");
    expect(END_SCREEN_SNIPPETS.en).toContain("self.wait(8)");
  });

  it("writes each snippet's narration in its own language", () => {
    expect(HOOK_SNIPPETS.en).toMatch(/# NARRATION: "[A-Za-z]/);
    expect(END_SCREEN_SNIPPETS.en).toContain("Subscribe");
    expect(END_SCREEN_SNIPPETS.vi).toContain("đăng ký kênh");
  });
});

describe("beat sheet trong prompt (CR-019 FR54)", () => {
  const FORMAT = {
    id: "t",
    name: "Thử",
    version: 1,
    min_seconds: 360,
    max_seconds: 480,
    beats: [
      { id: "hook", role: "hook", min_seconds: 8, max_seconds: 12, required: true, max_repeat: 1 },
      { id: "concrete", role: "example", min_seconds: 40, max_seconds: 70, required: true, max_repeat: 1 },
      { id: "variation", role: "example", min_seconds: 50, max_seconds: 90, required: false, max_repeat: 2 },
    ],
  };

  it("quy ngân sách giây thành ngân sách TỪ", () => {
    // Model đếm được từ; nó không đếm được giây. Giao cho nó phép quy đổi là
    // giao một việc nó không có cơ sở để làm.
    const section = buildBeatSheetSection(FORMAT, "vi");
    // 8 giây ở 140 wpm ≈ 19 từ; 12 giây ≈ 28 từ.
    expect(section).toContain("19–28 từ");
    expect(section).not.toMatch(/\d+ giây/);
  });

  it("dùng WPM đã hiệu chỉnh khi có (FR54.3)", () => {
    const base = buildBeatSheetSection(FORMAT, "vi");
    const fast = buildBeatSheetSection(FORMAT, "vi", 280);
    expect(fast).not.toEqual(base);
  });

  it("đánh dấu beat bắt buộc và số lần lặp cho phép", () => {
    const section = buildBeatSheetSection(FORMAT, "vi");
    expect(section).toContain('self.beat("hook")');
    expect(section).toContain("BẮT BUỘC");
    expect(section).toContain("lặp tối đa 2 lần");
  });

  it("nêu rõ concrete phải đứng trước pattern", () => {
    // Đây là ràng buộc mang toàn bộ ý nghĩa của format — nếu chỉ nằm trong đầu
    // người viết CR thì nó không tồn tại.
    expect(buildBeatSheetSection(FORMAT, "vi")).toContain("TRƯỚC");
  });

  it("prompt dùng beat sheet khi có format", () => {
    const withFormat = buildGenerationSystemPrompt("vi", FORMAT);
    expect(withFormat).toContain("CẤU TRÚC VIDEO BẮT BUỘC");
    expect(withFormat).not.toContain("ĐỘ DÀI MỤC TIÊU");
  });
});
