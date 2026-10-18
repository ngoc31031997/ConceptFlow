import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptAuthoringSettingsStepPage } from "../../src/pages/ScriptAuthoringSettingsStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { VideoFormat } from "../../src/types";

const ARCHETYPES: apiClient.VideoArchetype[] = [
  {
    id: "system-A",
    code: "A",
    name: "Nghịch lý",
    when_to_use: "lỗi tư duy",
    playbook: "",
    recommended_format_id: "essay",
    is_system: true,
  },
];
const FORMATS = [
  { id: "essay", name: "Chuỗi ví dụ", version: 1, min_seconds: 300, max_seconds: 600, beats: [] },
  {
    id: "quick",
    name: "Giải thích nhanh",
    version: 1,
    min_seconds: 60,
    max_seconds: 120,
    beats: [],
  },
] as VideoFormat[];

let createDraft: MockInstance<typeof apiClient.createProjectDraft>;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;

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
  vi.spyOn(apiClient, "listVideoArchetypes").mockResolvedValue(ARCHETYPES);
  createDraft = vi
    .spyOn(apiClient, "createProjectDraft")
    .mockResolvedValue({ similarProjects: [] });
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
  it("chia ba nhóm có đánh số, theo đúng thứ tự quyết định", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("settings-section-content")).toBeInTheDocument());

    const sections = [
      "settings-section-content",
      "settings-section-voice",
      "settings-section-authoring",
    ].map((id) => screen.getByTestId(id));
    expect(sections[0]).toHaveTextContent("1. Nội dung video");
    expect(sections[1]).toHaveTextContent("2. Giọng đọc");
    expect(sections[2]).toHaveTextContent("3. Cách soạn");
    for (let i = 0; i < sections.length - 1; i += 1) {
      expect(
        sections[i].compareDocumentPosition(sections[i + 1]) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    }
    // Kiểu video nằm cùng nhóm với format nó gợi ý; không có bộ chọn engine.
    await waitFor(() =>
      expect(sections[0]).toContainElement(screen.getByTestId("video-archetype-picker")),
    );
    expect(sections[0]).toContainElement(screen.getByTestId("video-format-picker"));
    expect(screen.queryByTestId("render-engine-picker")).not.toBeInTheDocument();
  });

  it("hiện chủ đề chỉ đọc, “Xem ở Ý tưởng” đưa về bước 1", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("topic-summary")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("topic-summary-open"));
    expect(screen.getByTestId("step-1-stub")).toBeInTheDocument();
  });

  it("không còn chọn ngôn ngữ nội dung và dạng video: hai thứ đó chốt ở bước Ý tưởng", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("settings-section-content")).toBeInTheDocument());
    expect(screen.queryByTestId("content-language-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("video-output-mode-picker")).not.toBeInTheDocument();
  });

  it("chọn kiểu video là lưu lại chủ đề có “kiểu: X”; dùng format gợi ý là lưu format", async () => {
    renderPage();
    await waitFor(() => screen.getByTestId("video-archetype-select"));

    fireEvent.click(screen.getByTestId("video-archetype-select"));
    fireEvent.click(screen.getByTestId("video-archetype-select-option-A"));

    await waitFor(() =>
      expect(createDraft).toHaveBeenCalledWith(expect.any(String), "kiểu: A", "vi"),
    );
    expect(screen.getByTestId("topic-summary")).toHaveTextContent("kiểu: A");

    fireEvent.click(await screen.findByTestId("video-archetype-use-format"));
    await waitFor(() =>
      expect(patchSettings).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({ videoFormatId: "essay" }),
      ),
    );
  });

  it("nút tiếp theo lưu cấu hình rồi sang bước Nghiên cứu", async () => {
    renderPage();
    await waitFor(() => screen.getByTestId("script-authoring-settings-next"));
    expect(screen.getByTestId("script-authoring-settings-next")).toHaveTextContent("Sang Nghiên cứu");

    fireEvent.click(screen.getByTestId("script-authoring-settings-next"));

    expect(await screen.findByTestId("research-stub")).toBeInTheDocument();
    expect(patchSettings).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ confirm: true }));
  });

  it("báo ở thanh dưới khi không lưu được kiểu video", async () => {
    createDraft.mockRejectedValue(new Error("offline"));
    renderPage();
    await waitFor(() => screen.getByTestId("video-archetype-select"));

    fireEvent.click(screen.getByTestId("video-archetype-select"));
    fireEvent.click(screen.getByTestId("video-archetype-select-option-A"));

    expect(await screen.findByText(/Không lưu được kiểu video/)).toBeInTheDocument();
  });
});
