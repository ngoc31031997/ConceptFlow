import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useState } from "react";
import {
  VideoArchetypePicker,
  forcedArchetypeCode,
  withForcedArchetype,
} from "../../src/components/VideoArchetypePicker";
import * as apiClient from "../../src/api/client";
import type { VideoFormat } from "../../src/types";

const ROWS: apiClient.VideoArchetype[] = [
  { id: "system-A", code: "A", name: "Nghịch lý", when_to_use: "lỗi tư duy", playbook: "", recommended_format_id: "essay", is_system: true },
  { id: "system-B", code: "B", name: "Họ khái niệm", when_to_use: "nhiều cách", playbook: "", recommended_format_id: "", is_system: true },
];
const FORMATS = [
  { id: "essay", name: "Chuỗi ví dụ" },
  { id: "quick", name: "Giải thích nhanh" },
] as unknown as VideoFormat[];

describe("forced archetype in topic", () => {
  it("reads and rewrites the 'kiểu: X' suffix", () => {
    expect(forcedArchetypeCode("vòng lặp — kiểu: b")).toBe("B");
    expect(forcedArchetypeCode("vòng lặp")).toBe("");
    expect(withForcedArchetype("vòng lặp", "B")).toBe("vòng lặp — kiểu: B");
    expect(withForcedArchetype("vòng lặp — kiểu: B", "A")).toBe("vòng lặp — kiểu: A");
    expect(withForcedArchetype("vòng lặp kiểu: B", "")).toBe("vòng lặp");
  });
});

function Harness({ onFormat }: { onFormat: (id: string) => void }) {
  const [topic, setTopic] = useState("vòng lặp");
  return (
    <>
      <span data-testid="topic">{topic}</span>
      <VideoArchetypePicker topic={topic} onTopicChange={setTopic} formats={FORMATS} formatId="quick" onFormatChange={onFormat} />
    </>
  );
}

describe("VideoArchetypePicker", () => {
  afterEach(() => vi.restoreAllMocks());

  it("writes the chosen kind into the topic and suggests its format", async () => {
    vi.spyOn(apiClient, "listVideoArchetypes").mockResolvedValue(ROWS);
    const onFormat = vi.fn();
    render(<Harness onFormat={onFormat} />);
    await waitFor(() => screen.getByTestId("video-archetype-select"));

    fireEvent.click(screen.getByTestId("video-archetype-select"));
    fireEvent.click(screen.getByTestId("video-archetype-select-option-A"));
    expect(screen.getByTestId("topic").textContent).toBe("vòng lặp — kiểu: A");

    fireEvent.click(screen.getByTestId("video-archetype-use-format"));
    expect(onFormat).toHaveBeenCalledWith("essay");

    fireEvent.click(screen.getByTestId("video-archetype-select"));
    fireEvent.click(screen.getByTestId("video-archetype-select-option-none"));
    expect(screen.getByTestId("topic").textContent).toBe("vòng lặp");
    expect(screen.queryByTestId("video-archetype-format-suggestion")).toBeNull();
  });
});
