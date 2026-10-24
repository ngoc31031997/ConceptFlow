import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { CreativeBriefStepPage } from "../../src/pages/CreativeBriefStepPage";
import { ProjectDraftContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { VideoFormat } from "../../src/types";

const FORMATS = [
  { id: "essay", name: "Chuỗi ví dụ", version: 1, min_seconds: 300, max_seconds: 600, beats: [] },
  {
    id: "vertical_short_60s",
    name: "Short dọc",
    version: 1,
    min_seconds: 30,
    max_seconds: 60,
    beats: [],
  },
] as VideoFormat[];

beforeEach(() => {
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: false,
    provider: "",
    reason: "Chưa cấu hình",
  });
  vi.spyOn(apiClient, "fetchVideoFormats").mockResolvedValue(FORMATS);
  vi.spyOn(apiClient, "getBriefScreen").mockResolvedValue({
    brief: null,
    idea: null,
    subject: "",
    language: "vi",
    videoMode: "long",
    minDurationSeconds: 121,
    maxDurationSeconds: 1800,
    ideaChanged: false,
    review: null,
    reviewStale: false,
  });
  vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

/** Shows the draft's format and subtitle settings, which the brief step itself does not display. */
function SubtitleProbe() {
  const draft = useContext(ProjectDraftContext);
  return (
    <output data-testid="subtitle-probe">{`${draft.videoFormatId}/${draft.subtitleMode}/${draft.subtitleStyle.position}`}</output>
  );
}

describe("chọn Short dọc", () => {
  it("đặt phụ đề đốt vào hình, ở phía trên — giống mặc định của server", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={["/"]}>
          <ProjectDraftProvider>
            <Routes>
              <Route path="/" element={<CreativeBriefStepPage />} />
            </Routes>
            <SubtitleProbe />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    const short = await screen.findByTestId("video-output-mode-short");
    await waitFor(() => expect(short).not.toBeDisabled());
    fireEvent.click(short);
    await waitFor(() =>
      expect(screen.getByTestId("subtitle-probe")).toHaveTextContent("vertical_short_60s/burn_in/top"),
    );
  });
});
