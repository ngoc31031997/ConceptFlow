import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { NarrationPanel } from "../../src/components/NarrationPanel";
import { defaultSubtitleStyle } from "../../src/context/ProjectDraftContext";
import * as client from "../../src/api/client";

const VOICES = [
  {
    voice_id: "vi-VN-HoaiMyNeural",
    language: "vi" as const,
    gender: "female" as const,
    quality: "neural",
    label: "Tiếng Việt — Nữ",
    engine: "edge",
    sample_audio_url: "/v1/voices/vi-VN-HoaiMyNeural/sample",
  },
  {
    // Same label as the Edge voice above — only the badge tells them apart.
    voice_id: "azure:vi-VN-HoaiMyNeural",
    language: "vi" as const,
    gender: "female" as const,
    quality: "neural",
    label: "Tiếng Việt — Nữ",
    engine: "azure",
    sample_audio_url: "/v1/voices/azure%3Avi-VN-HoaiMyNeural/sample",
  },
  {
    voice_id: "vi-VN-NamMinhNeural",
    language: "vi" as const,
    gender: "male" as const,
    quality: "neural",
    label: "Tiếng Việt — Nam",
    engine: "edge",
    sample_audio_url: "/v1/voices/vi-VN-NamMinhNeural/sample",
  },
  {
    voice_id: "en-US-GuyNeural",
    language: "en" as const,
    gender: "male" as const,
    quality: "neural",
    label: "English — Male",
    engine: "edge",
    sample_audio_url: "/v1/voices/en-US-GuyNeural/sample",
  },
];

function renderPanel(overrides: Partial<React.ComponentProps<typeof NarrationPanel>> = {}) {
  const props = {
    voiceLanguage: "vi" as const,
    ttsEnabled: true,
    onTtsEnabledChange: vi.fn(),
    voiceId: "vi-VN-HoaiMyNeural",
    onVoiceIdChange: vi.fn(),
    subtitlesEnabled: false,
    onSubtitlesEnabledChange: vi.fn(),
    subtitleStyle: defaultSubtitleStyle,
    onSubtitleStyleChange: vi.fn(),
    ...overrides,
  };
  render(<NarrationPanel {...props} />);
  return props;
}

describe("NarrationPanel", () => {
  beforeEach(() => {
    vi.spyOn(client, "listVoices").mockResolvedValue(VOICES);
  });

  it("lists only the voices for the selected language", async () => {
    renderPanel();

    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nam")).toBeInTheDocument());
    expect(screen.getAllByText("Tiếng Việt — Nữ")).toHaveLength(2);
    expect(screen.queryByText("English — Male")).not.toBeInTheDocument();
  });

  it("hides voice selection entirely when narration is disabled", async () => {
    renderPanel({ ttsEnabled: false });

    expect(screen.queryByTestId("narration-voice-list")).not.toBeInTheDocument();
  });

  it("selects a voice when its card is clicked", async () => {
    const props = renderPanel();
    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nam")).toBeInTheDocument());

    fireEvent.click(screen.getByText("Tiếng Việt — Nam"));

    expect(props.onVoiceIdChange).toHaveBeenCalledWith("vi-VN-NamMinhNeural");
  });

  it("plays the sample clip without changing the selection", async () => {
    const play = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal(
      "Audio",
      vi.fn(() => ({ play, pause: vi.fn() })),
    );
    const props = renderPanel();
    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nam")).toBeInTheDocument());

    fireEvent.click(screen.getByLabelText("Nghe thử Tiếng Việt — Nam"));

    expect(play).toHaveBeenCalled();
    expect(props.onVoiceIdChange).not.toHaveBeenCalledWith("vi-VN-NamMinhNeural");
  });

  it("falls back to a voice of the newly chosen language", async () => {
    // A Vietnamese voice must not survive a switch to English — the TTS
    // Service would have no matching model for the script.
    const props = renderPanel({ voiceLanguage: "en", voiceId: "vi-VN-HoaiMyNeural" });

    await waitFor(() => expect(props.onVoiceIdChange).toHaveBeenCalledWith("en-US-GuyNeural"));
  });

  it("badges each voice with the engine that will actually produce it", async () => {
    // Edge and Azure publish the same voice under the same label, so the badge
    // is the only signal telling the Creator which engine their audio comes
    // from — and which one spends their Azure quota.
    renderPanel();

    await waitFor(() => expect(screen.getByText("Azure")).toBeInTheDocument());
    expect(screen.getAllByText("Edge")).toHaveLength(2);
  });

  it("warns when the video would have neither narration nor subtitles", () => {
    renderPanel({ ttsEnabled: false, subtitlesEnabled: false });

    expect(screen.getByRole("status")).toHaveTextContent("không có lời thoại lẫn phụ đề");
  });
});
