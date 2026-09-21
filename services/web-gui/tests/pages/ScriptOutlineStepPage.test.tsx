import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptOutlineStepPage } from "../../src/pages/ScriptOutlineStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

// Bước 1a: fetches the story_architect template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring VisualDirectorStepPage.test.tsx's stub.
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "story_architect",
    language: "vi",
    version: 1,
    template_text: "CHỦ ĐỀ VIDEO: {{topic}}\n{{format_beats}}\n{{narration_language_rule}}",
  });
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "", review: "" });
  vi.spyOn(apiClient, "saveAuthoringStory").mockResolvedValue(undefined);
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <ProjectDraftProvider>
          <ScriptOutlineStepPage />
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ScriptOutlineStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("copies a prompt with the Creator's topic already substituted in", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderPage();

    await waitFor(() => expect(apiClient.getPromptTemplate).toHaveBeenCalledWith("story_architect", "vi"));
    fireEvent.change(screen.getByTestId("script-outline-topic"), {
      target: { value: "Vòng lặp for trong Java" },
    });
    fireEvent.click(screen.getByTestId("script-outline-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = writeText.mock.calls[0][0] as string;
    expect(copied).toContain("Vòng lặp for trong Java");
    expect(copied).not.toContain("[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]");
  });

  it("blocks the step until a story outline is pasted, then saves and advances", async () => {
    renderPage();

    expect(screen.getByTestId("script-outline-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("script-outline-story-input"), {
      target: { value: "CÂU HỎI CỐT LÕI: ...\nBEAT 1 — ..." },
    });
    expect(screen.getByTestId("script-outline-step-next")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("script-outline-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringStory).toHaveBeenCalledWith(
        expect.any(String),
        "CÂU HỎI CỐT LÕI: ...\nBEAT 1 — ...",
        expect.any(String),
      );
    });
  });

  // CR-027 D0 — the topic used to live only in this browser, so the server
  // could not render {{topic}} itself. These two cover the round trip that
  // FR77 depends on: it goes up with the outline, and it comes back down.
  it("sends the Creator's topic to the server alongside the outline", async () => {
    renderPage();

    fireEvent.change(screen.getByTestId("script-outline-topic"), {
      target: { value: "Vì sao bầu trời có màu xanh" },
    });
    fireEvent.change(screen.getByTestId("script-outline-story-input"), {
      target: { value: "CÂU HỎI CỐT LÕI: ..." },
    });
    fireEvent.click(screen.getByTestId("script-outline-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringStory).toHaveBeenCalledWith(
        expect.any(String),
        "CÂU HỎI CỐT LÕI: ...",
        "Vì sao bầu trời có màu xanh",
      );
    });
  });

  it("rehydrates the topic from the server on reload", async () => {
    // Rehydration only runs for a draft that already has a project_id — a
    // reload, not a fresh wizard. Seed the persisted draft the way a real
    // reload would find it.
    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({ projectId: "p-123", voiceLanguage: "vi" }),
    );
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
      topic: "Thuật toán sắp xếp nổi bọt",
      story: "dàn ý đã lưu",
      storyboard: "",
      code: "",
      review: "",
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("script-outline-topic")).toHaveValue("Thuật toán sắp xếp nổi bọt");
    });
  });

  it("shows the pipeline tab bar with 1a active", () => {
    renderPage();

    expect(screen.getByTestId("script-tab-outline")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("script-tab-storyboard")).not.toBeDisabled();
    expect(screen.getByTestId("script-tab-code")).not.toBeDisabled();
    expect(screen.getByTestId("script-tab-review")).not.toBeDisabled();
  });
});
