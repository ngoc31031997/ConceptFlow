import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptAuthoringSettingsStepPage } from "../../src/pages/ScriptAuthoringSettingsStepPage";
import { ProjectDraftContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
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
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
    topic: "",
    story: "",
    storyboard: "",
    code: "",
  });
  vi.spyOn(apiClient, "fetchVideoFormats").mockResolvedValue(FORMATS);
  vi.spyOn(apiClient, "listVoices").mockResolvedValue([]);
  vi.spyOn(apiClient, "listVideoArchetypes").mockResolvedValue([]);
  vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
  vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

/** Shows the draft's subtitle settings, which step 2 itself does not display. */
function SubtitleProbe() {
  const draft = useContext(ProjectDraftContext);
  return (
    <output data-testid="subtitle-probe">{`${draft.subtitleMode}/${draft.subtitleStyle.position}`}</output>
  );
}

describe("chọn Short dọc", () => {
  it("đặt phụ đề đốt vào hình, ở phía trên — giống mặc định của server", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={["/create/script/settings"]}>
          <ProjectDraftProvider>
            <AuthoringRunProvider>
              <Routes>
                <Route
                  path="/create/script/settings"
                  element={<ScriptAuthoringSettingsStepPage />}
                />
              </Routes>
              <SubtitleProbe />
            </AuthoringRunProvider>
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    fireEvent.click(await screen.findByTestId("video-output-mode-short"));
    await waitFor(() =>
      expect(screen.getByTestId("subtitle-probe")).toHaveTextContent("burn_in/top"),
    );
    expect(screen.getByText(/đốt phụ đề ở phía trên khung/)).toBeInTheDocument();
  });
});
