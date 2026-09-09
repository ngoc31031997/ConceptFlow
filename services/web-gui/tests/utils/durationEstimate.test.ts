import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import {
  countWords,
  estimateNarrationDuration,
  formatDuration,
  type ContentLanguage,
} from "../../src/utils/durationEstimate";

interface Vector {
  text: string;
  language: ContentLanguage;
  expected_seconds: number;
  why: string;
}

const fixture = JSON.parse(
  readFileSync(
    resolve(__dirname, "../../../../tests/fixtures/narration-duration-vectors.json"),
    "utf-8",
  ),
) as {
  vectors: Vector[];
  words_per_minute: Record<string, number>;
  min_narration_seconds: number;
};

describe("estimateNarrationDuration khớp bản Go", () => {
  // Cùng một con số được dùng ở hai nơi: ở đây để Creator thấy trước lúc soạn
  // (CR-016 FR42), và ở Orchestrator để định thời lượng thật khi tắt TTS
  // (CR-001). Bộ vector này là thứ giữ hai bản không trôi khỏi nhau.
  it.each(fixture.vectors)("$why", ({ text, language, expected_seconds }) => {
    expect(estimateNarrationDuration(text, language)).toBeCloseTo(expected_seconds, 6);
  });

  it("dùng đúng hằng số WPM và sàn của bản Go", () => {
    expect(fixture.words_per_minute).toEqual({ vi: 140, en: 150 });
    expect(fixture.min_narration_seconds).toBe(1.5);
  });
});

describe("countWords", () => {
  it("gộp khoảng trắng liên tiếp như strings.Fields bên Go", () => {
    expect(countWords("  a   b \n c \t ")).toBe(3);
    expect(countWords("   ")).toBe(0);
  });
});

describe("WPM đã hiệu chỉnh (FR43)", () => {
  it("giọng đọc nhanh hơn cho ra thời lượng ngắn hơn", () => {
    const text = "một hai ba bốn năm sáu bảy tám chín mười";
    expect(estimateNarrationDuration(text, "vi", 200)).toBeLessThan(
      estimateNarrationDuration(text, "vi"),
    );
  });
});

describe("formatDuration", () => {
  it.each([
    [45, "45 giây"],
    [60, "1 phút"],
    [440, "7 phút 20 giây"],
  ])("%i giây -> %s", (seconds, expected) => {
    expect(formatDuration(seconds)).toBe(expected);
  });
});
