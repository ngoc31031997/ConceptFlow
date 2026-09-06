import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { NarrationPanel } from "../../src/components/NarrationPanel";
import * as client from "../../src/api/client";

const VOICES = [
  {
    voice_id: "vi_VN-vais1000-medium",
    language: "vi" as const,
    gender: "female" as const,
    quality: "medium",
    label: "Tiếng Việt — Nữ (VAIS)",
    sample_audio_url: "/v1/voices/vi_VN-vais1000-medium/sample",
  },
  {
    voice_id: "vi_VN-vivos-x_low",
    language: "vi" as const,
    gender: "male" as const,
    quality: "x_low",
    label: "Tiếng Việt — Nam (VIVOS)",
    sample_audio_url: "/v1/voices/vi_VN-vivos-x_low/sample",
  },
  {
    voice_id: "en_US-ryan-high",
    language: "en" as const,
    gender: "male" as const,
    quality: "high",
    label: "English — Male (Ryan)",
    sample_audio_url: "/v1/voices/en_US-ryan-high/sample",
  },
];

function renderPanel(overrides: Partial<React.ComponentProps<typeof NarrationPanel>> = {}) {
  const props = {
    voiceLanguage: "vi" as const,
    onVoiceLanguageChange: vi.fn(),
    ttsEnabled: true,
    onTtsEnabledChange: vi.fn(),
    voiceId: "vi_VN-vais1000-medium",
    onVoiceIdChange: vi.fn(),
    subtitlesEnabled: false,
    onSubtitlesEnabledChange: vi.fn(),
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

    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nữ (VAIS)")).toBeInTheDocument());
    expect(screen.getByText("Tiếng Việt — Nam (VIVOS)")).toBeInTheDocument();
    expect(screen.queryByText("English — Male (Ryan)")).not.toBeInTheDocument();
  });

  it("hides voice selection entirely when narration is disabled", async () => {
    renderPanel({ ttsEnabled: false });

    expect(screen.queryByTestId("narration-voice-list")).not.toBeInTheDocument();
  });

  it("selects a voice when its card is clicked", async () => {
    const props = renderPanel();
    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nam (VIVOS)")).toBeInTheDocument());

    fireEvent.click(screen.getByText("Tiếng Việt — Nam (VIVOS)"));

    expect(props.onVoiceIdChange).toHaveBeenCalledWith("vi_VN-vivos-x_low");
  });

  it("plays the sample clip without changing the selection", async () => {
    const play = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal(
      "Audio",
      vi.fn(() => ({ play, pause: vi.fn() })),
    );
    const props = renderPanel();
    await waitFor(() => expect(screen.getByText("Tiếng Việt — Nam (VIVOS)")).toBeInTheDocument());

    fireEvent.click(screen.getByLabelText("Nghe thử Tiếng Việt — Nam (VIVOS)"));

    expect(play).toHaveBeenCalled();
    expect(props.onVoiceIdChange).not.toHaveBeenCalledWith("vi_VN-vivos-x_low");
  });

  it("falls back to a voice of the newly chosen language", async () => {
    // A Vietnamese voice must not survive a switch to English — the TTS
    // Service would have no matching model for the script.
    const props = renderPanel({ voiceLanguage: "en", voiceId: "vi_VN-vais1000-medium" });

    await waitFor(() => expect(props.onVoiceIdChange).toHaveBeenCalledWith("en_US-ryan-high"));
  });

  it("warns when the video would have neither narration nor subtitles", () => {
    renderPanel({ ttsEnabled: false, subtitlesEnabled: false });

    expect(screen.getByRole("status")).toHaveTextContent("không có lời thoại lẫn phụ đề");
  });
});
