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

  // CR-027 FR79 — chế độ làm việc là lựa chọn cho CẢ bước 1, không phải một
  // nút riêng từng tab. Mặc định là copy tay, đúng cái mọi project vẫn làm
  // trước CR-027.
  describe("chế độ làm bước 1 (CR-027 FR79)", () => {
    function mockLlm(enabled: boolean, reason?: string) {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
        enabled,
        provider: enabled ? "hive" : "",
        reason,
      });
    }

    it("mặc định là copy tay: có ô prompt và nút Copy, chưa có nút chạy AI", async () => {
      mockLlm(true);
      renderPage();

      await waitFor(() => expect(screen.getByTestId("authoring-mode-bar")).toBeInTheDocument());
      expect(screen.getByTestId("script-outline-prompt")).toBeInTheDocument();
      expect(screen.getByTestId("script-outline-copy")).toBeInTheDocument();
      expect(screen.queryByTestId("run-with-ai-story")).not.toBeInTheDocument();
    });

    it("chọn 'Gọi API trực tiếp' thì ẩn ô prompt copy tay và hiện nút chạy", async () => {
      mockLlm(true);
      renderPage();

      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));

      expect(screen.getByTestId("run-with-ai-story")).toBeInTheDocument();
      expect(screen.queryByTestId("script-outline-prompt")).not.toBeInTheDocument();
      expect(screen.queryByTestId("script-outline-copy")).not.toBeInTheDocument();
    });

    it("chạy bước 1 bằng API và điền kết quả vào ô dàn ý", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      const generate = vi.spyOn(apiClient, "generateAuthoringStep").mockResolvedValue({
        step: "story",
        role: "story_architect",
        content: "CÂU HỎI CỐT LÕI: vì sao?",
        provider: "hive",
        usage: { model: "deepseek" },
      });
      renderPage();

      fireEvent.change(screen.getByTestId("script-outline-topic"), {
        target: { value: "Vòng lặp for" },
      });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() => expect(generate).toHaveBeenCalledWith(expect.any(String), "story", undefined));
      await waitFor(() =>
        expect(screen.getByTestId("script-outline-story-input")).toHaveValue("CÂU HỎI CỐT LÕI: vì sao?"),
      );
    });

    it("lưu chủ đề lên server trước khi gọi, vì prompt được render ở server", async () => {
      mockLlm(true);
      const saveDraft = vi
        .spyOn(apiClient, "createProjectDraft")
        .mockResolvedValue({ similarProjects: [] });
      vi.spyOn(apiClient, "generateAuthoringStep").mockResolvedValue({
        step: "story",
        role: "story_architect",
        content: "dàn ý",
        provider: "hive",
        usage: { model: "deepseek" },
      });
      renderPage();

      fireEvent.change(screen.getByTestId("script-outline-topic"), {
        target: { value: "Cây nhị phân" },
      });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(saveDraft).toHaveBeenCalledWith(expect.any(String), "Cây nhị phân", "vi"),
      );
    });

    // FR79.4 — chưa có key thì chế độ AI không chọn được, và đường copy tay
    // vẫn nguyên vẹn; không bao giờ có một nút bấm vào là lỗi.
    it("không cho chọn chế độ AI khi chưa có API key", async () => {
      mockLlm(false, "Chưa cấu hình HIVE_API_KEY");
      renderPage();

      await waitFor(() =>
        expect(screen.getByTestId("authoring-mode-bar")).toHaveTextContent("HIVE_API_KEY"),
      );
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));

      expect(screen.queryByTestId("run-with-ai-story")).not.toBeInTheDocument();
      expect(screen.getByTestId("script-outline-prompt")).toBeInTheDocument();
      expect(screen.getByTestId("script-outline-copy")).toBeInTheDocument();
    });

    it("nói rõ nguyên nhân khi lượt chạy thất bại", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      vi.spyOn(apiClient, "generateAuthoringStep").mockRejectedValue(
        new apiClient.ApiError("Tài khoản Hive hết số dư — nạp thêm ở dashboard Hive. Hoặc dùng nút Copy prompt như cũ."),
      );
      renderPage();

      fireEvent.change(screen.getByTestId("script-outline-topic"), { target: { value: "X" } });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(screen.getByTestId("run-with-ai-error")).toHaveTextContent("hết số dư"),
      );
      expect(screen.getByTestId("run-with-ai-error")).toHaveTextContent("Copy prompt");
    });

    // FR79 — chế độ nằm ở project trong DB, không chỉ localStorage: đó là cái
    // làm nó sống qua reload, qua trình duyệt khác và qua restart, ở bất cứ
    // bước nào của dự án.
    it("lưu chế độ lên server khi Creator đổi", async () => {
      mockLlm(true);
      const saveMode = vi.spyOn(apiClient, "saveAuthoringMode").mockResolvedValue(undefined);
      renderPage();

      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));

      await waitFor(() => expect(saveMode).toHaveBeenCalledWith(expect.any(String), "ai"));

      fireEvent.click(screen.getByTestId("authoring-mode-manual"));
      await waitFor(() => expect(saveMode).toHaveBeenCalledWith(expect.any(String), "manual"));
    });

    it("nạp lại chế độ từ server, kể cả khi draft trong trình duyệt nói khác", async () => {
      mockLlm(true);
      // Draft (localStorage) nói manual; project trong DB nói ai. Server thắng:
      // nó là bản ghi của dự án, localStorage chỉ là bản nháp của một máy.
      window.localStorage.setItem(
        "conceptflow.draft.v1",
        JSON.stringify({ projectId: "p-123", voiceLanguage: "vi", authoringMode: "manual" }),
      );
      vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
        mode: "ai",
        topic: "Chủ đề đã lưu",
        story: "",
        storyboard: "",
        code: "",
        review: "",
      });
      renderPage();

      await waitFor(() => expect(screen.getByTestId("run-with-ai-story")).toBeInTheDocument());
      expect(screen.queryByTestId("script-outline-prompt")).not.toBeInTheDocument();
    });
  });
});
