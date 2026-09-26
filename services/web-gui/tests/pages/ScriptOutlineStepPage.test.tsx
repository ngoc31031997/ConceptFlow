import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptOutlineStepPage } from "../../src/pages/ScriptOutlineStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

// Bước 1a: fetches the story_architect template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring VisualDirectorStepPage.test.tsx's stub.
beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "story_architect",
    id: "system-x",
    name: "Mặc định",
    is_system: true,
    is_active: true,
    template_text: "CHỦ ĐỀ VIDEO: {{topic}}\n{{format_beats}}\n{{narration_language_rule}}",
  });
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "saveAuthoringStory").mockResolvedValue(undefined);
});


// Chuỗi 1a→1b→1c chạy ở SERVER (POST .../authoring/chain trả về ngay, GET cho
// biết kết cục). Helper này dựng một server giả: trước khi bấm chạy chưa có
// chuỗi nào; sau khi bấm, server "đã xong" với kết cục do test quy định.
function mockServerChain(outcome: Partial<apiClient.AuthoringChainState> = {}) {
  let started: apiClient.AuthoringStep[] | null = null;
  const start = vi.spyOn(apiClient, "startAuthoringChain").mockImplementation(async (_id, steps) => {
    started = steps;
  });
  vi.spyOn(apiClient, "getAuthoringChain").mockImplementation(async () =>
    started
      ? {
          running: false,
          steps: started,
          current_index: started.length,
          finished: true,
          started_at: new Date().toISOString(),
          finished_at: new Date().toISOString(),
          ...outcome,
        }
      : { running: false, steps: [], current_index: 0, finished: false },
  );
  return start;
}

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <ProjectDraftProvider>
          <AuthoringRunProvider>
            <ScriptOutlineStepPage />
          </AuthoringRunProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// CR-031 bug report — engine/cách làm đã chốt ở màn chọn tình huống, nên mỗi
// tab giờ chỉ hiện một dòng tóm tắt, thu gọn hai bộ chọn đầy đủ lại (xem
// PipelineSettingsBar). Các test dưới đây thao tác trực tiếp với hai bộ chọn
// đó, nên phải mở panel ra trước — y hệt một Creator bấm "Đổi".
function expandSettings() {
  fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));
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

    await waitFor(() => expect(apiClient.getPromptTemplate).toHaveBeenCalledWith("story_architect"));
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
    });

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("script-outline-topic")).toHaveValue("Thuật toán sắp xếp nổi bọt");
    });
  });

  it("has no 1a/1b/1c tab bar: the three script steps are steps 3/4/5 of the flow", () => {
    renderPage();

    expect(screen.queryByTestId("script-tab-outline")).not.toBeInTheDocument();
    expect(screen.queryByTestId("script-tab-storyboard")).not.toBeInTheDocument();
    expect(screen.queryByTestId("script-tab-code")).not.toBeInTheDocument();
  });

  // CR-030 — engine phải lên server ngay khi chọn, không phải chỉ lúc nộp
  // render: chuỗi 1a→1b→1c đọc project.RenderEngine từ server để chọn đúng
  // vai trò storyboard/code, nên tới lúc Creator xuống 1c đổi thì đã trễ.
  describe("chọn công cụ render (CR-030)", () => {
    it("mặc định chọn Manim và lưu Remotion lên server ngay khi Creator đổi", async () => {
      const createDraft = vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      renderPage();
      expandSettings();

      expect(screen.getByTestId("render-engine-manim")).toHaveAttribute("aria-checked", "true");

      fireEvent.click(screen.getByTestId("render-engine-remotion"));

      await waitFor(() =>
        expect(createDraft).toHaveBeenCalledWith(expect.any(String), "", "vi", "remotion"),
      );
    });

    it("gửi kèm engine hiện tại ngay trước khi chạy chuỗi AI", async () => {
      const createDraft = vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      mockServerChain();
      renderPage();
      expandSettings();

      fireEvent.click(screen.getByTestId("render-engine-remotion"));
      await waitFor(() => expect(createDraft).toHaveBeenCalledWith(expect.any(String), "", "vi", "remotion"));
      createDraft.mockClear();

      fireEvent.change(screen.getByTestId("script-outline-topic"), { target: { value: "Vòng lặp for" } });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(createDraft).toHaveBeenCalledWith(expect.any(String), "Vòng lặp for", "vi", "remotion"),
      );
    });
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
      expandSettings();

      await waitFor(() => expect(screen.getByTestId("authoring-mode-bar")).toBeInTheDocument());
      expect(screen.getByTestId("script-outline-prompt")).toBeInTheDocument();
      expect(screen.getByTestId("script-outline-copy")).toBeInTheDocument();
      expect(screen.queryByTestId("run-with-ai-story")).not.toBeInTheDocument();
    });

    it("chọn 'Gọi API trực tiếp' thì ẩn ô prompt copy tay và hiện nút chạy", async () => {
      mockLlm(true);
      renderPage();
      expandSettings();

      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));

      expect(screen.getByTestId("run-with-ai-story")).toBeInTheDocument();
      expect(screen.queryByTestId("script-outline-prompt")).not.toBeInTheDocument();
      expect(screen.queryByTestId("script-outline-copy")).not.toBeInTheDocument();
    });

    // CR-030 — một lần bấm chạy cả 1a → 1b → 1c. Chuỗi chạy tuần tự vì mỗi
    // lượt gọi tự lưu kết quả lên server, và bước sau render prompt từ đúng
    // dữ liệu bước trước vừa lưu.
    it("chạy cả ba bước bằng API và điền kết quả vào đúng từng ô", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      const start = mockServerChain();
      // Sau mỗi lượt chạy, bản nháp đọc lại ba kết quả từ server — mô phỏng
      // server đã lưu dàn ý của bước 1a.
      vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
        topic: "Vòng lặp for", story: "CÂU HỎI CỐT LÕI: vì sao?", storyboard: "", code: "",
      });
      renderPage();
      expandSettings();

      fireEvent.change(screen.getByTestId("script-outline-topic"), {
        target: { value: "Vòng lặp for" },
      });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() => expect(start).toHaveBeenCalledTimes(1));
      expect(start.mock.calls[0][1]).toEqual(["story"]);
      await waitFor(() =>
        expect(screen.getByTestId("script-outline-story-input")).toHaveValue("CÂU HỎI CỐT LÕI: vì sao?"),
      );
    });

    // CR-039 — 1c có thể lưu code vẫn còn lỗi biên dịch sau các vòng sửa. Đó là
    // ghi chú (code đã lưu, token đã tốn), nhưng phải nói rõ và kèm lỗi, không
    // được im lặng như thể xong sạch.
    it("báo code đã lưu nhưng còn lỗi biên dịch, kèm danh sách lỗi", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      mockServerChain({
        note:
          "Đã sinh và lưu code nhưng vẫn lỗi biên dịch sau 3 vòng sửa: dòng 41: TS2304: Cannot find name 'x'. · dòng 50: TS1005: ';' expected.. Sửa tay trong ô soạn thảo, hoặc chạy lại.",
      });
      vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
        topic: "Vòng lặp for", story: "s", storyboard: "sb", code: "const broken = ;",
      });
      renderPage();
      expandSettings();

      fireEvent.change(screen.getByTestId("script-outline-topic"), { target: { value: "Vòng lặp for" } });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      const note = await screen.findByText(/vẫn lỗi biên dịch sau 3 vòng sửa/);
      expect(note.textContent).toContain("TS2304");
      expect(note.textContent).toContain("TS1005");
      expect(screen.queryByTestId("run-with-ai-error")).not.toBeInTheDocument();
    });

    it("báo lỗi khi bước chạy hỏng, giữ nguyên nội dung ô soạn thảo", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      mockServerChain({ error: "Tài khoản Hive hết số dư.", error_step: "story" });
      // Sau mỗi lượt chạy, bản nháp đọc lại ba kết quả từ server — mô phỏng
      // server đã lưu dàn ý của bước 1a.
      vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
        topic: "Vòng lặp for", story: "CÂU HỎI CỐT LÕI: vì sao?", storyboard: "", code: "",
      });
      renderPage();
      expandSettings();

      fireEvent.change(screen.getByTestId("script-outline-topic"), {
        target: { value: "Vòng lặp for" },
      });
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() => expect(screen.getByTestId("run-with-ai-error")).toBeInTheDocument());
      expect(screen.getByTestId("run-with-ai-error")).toHaveTextContent("hết số dư");
      // Bước 1a đã xong vẫn còn nguyên trong ô soạn thảo.
      expect(screen.getByTestId("script-outline-story-input")).toHaveValue("CÂU HỎI CỐT LÕI: vì sao?");
    });

    it("lưu chủ đề lên server trước khi gọi, vì prompt được render ở server", async () => {
      mockLlm(true);
      const saveDraft = vi
        .spyOn(apiClient, "createProjectDraft")
        .mockResolvedValue({ similarProjects: [] });
      mockServerChain();
      renderPage();
      expandSettings();

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
      expandSettings();

      await waitFor(() =>
        expect(screen.getByTestId("authoring-mode-switch")).toHaveTextContent("HIVE_API_KEY"),
      );
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));

      expect(screen.queryByTestId("run-with-ai-story")).not.toBeInTheDocument();
      expect(screen.getByTestId("script-outline-prompt")).toBeInTheDocument();
      expect(screen.getByTestId("script-outline-copy")).toBeInTheDocument();
    });

    it("nói rõ nguyên nhân khi lượt chạy thất bại", async () => {
      mockLlm(true);
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      vi.spyOn(apiClient, "getAuthoringChain").mockResolvedValue({ running: false, steps: [], current_index: 0, finished: false });
      vi.spyOn(apiClient, "startAuthoringChain").mockRejectedValue(
        new apiClient.ApiError("Tài khoản Hive hết số dư — nạp thêm ở dashboard Hive. Hoặc dùng nút Copy prompt như cũ."),
      );
      renderPage();
      expandSettings();

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
      expandSettings();

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
      });
      renderPage();
      expandSettings();

      await waitFor(() => expect(screen.getByTestId("run-with-ai-story")).toBeInTheDocument());
      expect(screen.queryByTestId("script-outline-prompt")).not.toBeInTheDocument();
    });
  });
});

// Chuỗi chạy ở server nên sống qua việc đóng/tải lại trang: mở lại phải thấy
// nó đang chạy, hoặc kết cục của nó — không phải một màn im lặng.
describe("chuỗi AI chạy ở server (mở lại trang giữa/sau lượt chạy)", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  function setup(chain: apiClient.AuthoringChainState) {
    vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
    vi.spyOn(apiClient, "getAuthoringChain").mockResolvedValue(chain);
    vi.spyOn(apiClient, "saveAuthoringMode").mockResolvedValue(undefined);
    window.localStorage.setItem(
      "conceptflow.draft.v1",
      JSON.stringify({ projectId: "p-reopen", voiceLanguage: "vi", authoringMode: "ai", authoringTopic: "Vòng lặp for" }),
    );
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
      mode: "ai", topic: "Vòng lặp for", story: "", storyboard: "", code: "",
    });
  }

  it("thấy chuỗi đang chạy dù trang này không bấm chạy, và nút chạy bị khoá", async () => {
    setup({ running: true, steps: ["story", "storyboard", "code"], current_index: 1, finished: false });
    renderPage();

    await waitFor(() => expect(screen.getByTestId("run-with-ai-running")).toBeInTheDocument());
    expect(screen.getByTestId("run-with-ai-running")).toHaveTextContent("Visual");
    expect(screen.getByTestId("run-with-ai-story")).toBeDisabled();
  });

  it("hiện lỗi của lượt chạy đã dừng khi trang được mở lại, và đóng được", async () => {
    setup({
      running: false,
      steps: ["story", "storyboard", "code"],
      current_index: 1,
      finished: true,
      error: "Tài khoản Hive hết số dư.",
      error_step: "storyboard",
      finished_at: new Date().toISOString(),
    });
    renderPage();

    const err = await screen.findByTestId("run-with-ai-error");
    expect(err).toHaveTextContent("hết số dư");
    expect(err).toHaveTextContent("Visual");

    fireEvent.click(screen.getByTestId("run-with-ai-dismiss"));
    await waitFor(() => expect(screen.queryByTestId("run-with-ai-error")).not.toBeInTheDocument());
  });

  it("shows how the last AI run of each step went — time, size, tokens — from the journal", async () => {
    setup({ running: false, steps: [], current_index: 0, finished: false });
    vi.spyOn(apiClient, "listProjectEvents").mockResolvedValue([
      { id: 1, project_id: "p-reopen", at: "t", flow_step: 3, step_label: "Kịch bản", run_state: "running", source: "authoring" },
      { id: 2, project_id: "p-reopen", at: "t", flow_step: 3, step_label: "Kịch bản", run_state: "done", source: "authoring", duration_ms: 72000, content_chars: 10178, prompt_tokens: 900, completion_tokens: 3100 },
      { id: 3, project_id: "p-reopen", at: "t", flow_step: 4, step_label: "Visual", run_state: "failed", source: "authoring", duration_ms: 15000, detail: "hết số dư" },
    ]);
    renderPage();

    const story = await screen.findByTestId("last-run-story");
    expect(story).toHaveTextContent("Kịch bản");
    expect(story).toHaveTextContent("1m 12s");
    expect(story).toHaveTextContent("10.2k ký tự");
    expect(story).toHaveTextContent("4.000 token");
    expect(screen.getByTestId("last-run-storyboard")).toHaveAttribute("data-state", "failed");
    expect(screen.queryByTestId("last-run-code")).not.toBeInTheDocument();
  });
});
