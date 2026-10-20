import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptAuthoringSettingsStepPage } from "../../src/pages/ScriptAuthoringSettingsStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;

beforeEach(() => {
  vi.spyOn(apiClient, "listVoices").mockResolvedValue([]);
  vi.spyOn(apiClient, "getProjectVisualLanguage").mockResolvedValue({
    visual_language_id: "phatsang",
    visual_language_version: 1,
    kit_key: "phatsang",
    name: "Khoa học phát sáng",
    rules: { illustration_rules: "a", backdrop_rules: "b", scene_rules: "c" },
  });
  vi.spyOn(apiClient, "getMusicInfo").mockResolvedValue({ exists: false, background_music_path: null });
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/create/script/settings"]}>
        <ProjectDraftProvider>
          <AuthoringRunProvider>
            <Routes>
              <Route path="/create/script/settings" element={<ScriptAuthoringSettingsStepPage />} />
              <Route path="/" element={<div data-testid="step-1-stub" />} />
              <Route path="/projects/:id/research" element={<div data-testid="research-stub" />} />
            </Routes>
          </AuthoringRunProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ScriptAuthoringSettingsStepPage (Bước 2 — Cấu hình)", () => {
  it("có năm mục đánh số theo thứ tự mockup, rồi các chỗ sẽ duyệt", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("settings-section-voice")).toBeInTheDocument());

    const sections = [
      "settings-section-voice",
      "settings-section-font",
      "settings-section-music",
      "settings-section-quality",
      "settings-section-language",
      "settings-section-gates",
    ].map((id) => screen.getByTestId(id));
    expect(sections[0]).toHaveTextContent("1. Giọng kể");
    expect(sections[1]).toHaveTextContent("2. Chữ trên video");
    expect(sections[2]).toHaveTextContent("3. Nhạc nền");
    expect(sections[3]).toHaveTextContent("4. Chất lượng render");
    expect(sections[4]).toHaveTextContent("5. Ngôn ngữ hình");
    for (let i = 0; i < sections.length - 1; i += 1) {
      expect(
        sections[i].compareDocumentPosition(sections[i + 1]) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    }
  });

  it("không còn chọn kiểu video, format và cách soạn ở bước này", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("settings-section-voice")).toBeInTheDocument());
    expect(screen.queryByTestId("video-archetype-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("video-format-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("settings-section-authoring")).not.toBeInTheDocument();
    expect(screen.queryByTestId("content-language-picker")).not.toBeInTheDocument();
  });

  it("liệt kê các bước có chỗ duyệt, đánh dấu bước tuỳ chọn", async () => {
    renderPage();
    const gates = await screen.findByTestId("settings-gates");
    expect(gates).toHaveTextContent("1. Ý tưởng · creative brief");
    expect(gates).toHaveTextContent("3. Nghiên cứu và kiểm chứng · tuỳ chọn");
    expect(gates).toHaveTextContent("16. Review");
  });

  it("hiện chủ đề chỉ đọc, “Xem ở Ý tưởng” đưa về bước 1", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("topic-summary")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("topic-summary-open"));
    expect(screen.getByTestId("step-1-stub")).toBeInTheDocument();
  });

  it("đổi font, chất lượng rồi lưu từng mục lên project", async () => {
    renderPage();
    await waitFor(() => screen.getByTestId("video-font-Montserrat"));

    fireEvent.click(screen.getByTestId("video-font-Montserrat"));
    await waitFor(() =>
      expect(patchSettings).toHaveBeenCalledWith(expect.any(String), { videoFont: "Montserrat" }),
    );

    fireEvent.click(screen.getByTestId("render-quality-720p30"));
    await waitFor(() =>
      expect(patchSettings).toHaveBeenCalledWith(expect.any(String), { renderQuality: "720p30" }),
    );
  });

  it("nút lưu chốt bước rồi sang bước Nghiên cứu", async () => {
    renderPage();
    await waitFor(() => screen.getByTestId("script-authoring-settings-next"));
    expect(screen.getByTestId("script-authoring-settings-next")).toHaveTextContent("Lưu và sang nghiên cứu");

    fireEvent.click(screen.getByTestId("script-authoring-settings-next"));

    expect(await screen.findByTestId("research-stub")).toBeInTheDocument();
    expect(patchSettings).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ confirm: true }));
  });

  it("báo ở thanh dưới và ở lại bước khi không lưu được", async () => {
    patchSettings.mockRejectedValue(new Error("offline"));
    renderPage();
    await waitFor(() => screen.getByTestId("script-authoring-settings-next"));

    fireEvent.click(screen.getByTestId("script-authoring-settings-next"));

    expect(await screen.findByText(/Không lưu được cấu hình/)).toBeInTheDocument();
    expect(screen.queryByTestId("research-stub")).not.toBeInTheDocument();
  });
});
