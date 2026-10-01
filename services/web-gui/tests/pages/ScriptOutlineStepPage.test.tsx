import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ScriptOutlineStepPage } from "../../src/pages/ScriptOutlineStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import { mockRenderPrompt } from "../helpers/renderPromptMock";

// Bước 1a: fetches the story_architect template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring VisualDirectorStepPage.test.tsx's stub.
beforeEach(() => {
  mockRenderPrompt();
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

/**
 * The topic is typed on step 1; here it arrives from the server, the way the
 * page rehydrates it.
 */
function seedTopic(topic: string) {
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic, story: "", storyboard: "", code: "" });
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

// Engine/cách làm đã chốt ở màn chọn tình huống, nên mỗi
// tab chỉ hiện một dòng tóm tắt, thu gọn hai bộ chọn đầy đủ lại (xem
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

  it("copies the server-rendered prompt for the project's topic", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    seedTopic("Vòng lặp for trong Java");
    renderPage();

    await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for trong Java"));
    await waitFor(() =>
      expect(apiClient.renderPrompt).toHaveBeenCalledWith(
        expect.objectContaining({ role: "story_architect", topic: "Vòng lặp for trong Java" }),
      ),
    );
    await waitFor(() => expect(screen.getByTestId("script-outline-copy")).toBeEnabled());
    fireEvent.click(screen.getByTestId("script-outline-copy"));

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const copied = writeText.mock.calls[0][0] as string;
    expect(copied).toContain("RENDERED[story_architect]");
    expect(copied).toContain("Vòng lặp for trong Java");
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

  // The server renders {{topic}} itself, so the topic must reach it. These
  // two cover the round trip: it goes up with the outline, and it comes back down.
  it("sends the project's topic to the server alongside the outline", async () => {
    seedTopic("Vì sao bầu trời có màu xanh");
      renderPage();

    await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vì sao bầu trời có màu xanh"));
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
      expect(screen.getByTestId("topic-summary")).toHaveTextContent("Thuật toán sắp xếp nổi bọt");
    });
  });

  it("shows the topic read-only: it is edited on step 1, and the video kind on step 2", async () => {
    seedTopic("Vòng lặp for");
    renderPage();

    await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
    expect(screen.queryByTestId("script-outline-topic")).not.toBeInTheDocument();
    expect(screen.queryByTestId("video-archetype-picker")).not.toBeInTheDocument();
  });

  it("puts the settings and AI bar above the prompt and result cards, as on steps 4–6", async () => {
    renderPage();

    const bar = await screen.findByTestId("pipeline-settings-bar");
    const result = screen.getByTestId("script-outline-story-input");
    expect(bar.compareDocumentPosition(result) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByTestId("topic-summary").compareDocumentPosition(bar) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("asks to paste from an outside AI in manual mode, and to run or write it in AI mode", async () => {
    vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
    renderPage();

    expect(await screen.findByText("Dán dàn ý từ AI để tiếp tục.")).toBeInTheDocument();
    expandSettings();
    await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("authoring-mode-ai"));
    expect(await screen.findByText("Bấm Chạy bằng AI, hoặc tự viết dàn ý vào ô bên dưới.")).toBeInTheDocument();
  });

  it("has no 1a/1b/1c tab bar: the three script steps are steps 3/4/5 of the flow", () => {
    renderPage();

    expect(screen.queryByTestId("script-tab-outline")).not.toBeInTheDocument();
    expect(screen.queryByTestId("script-tab-storyboard")).not.toBeInTheDocument();
    expect(screen.queryByTestId("script-tab-code")).not.toBeInTheDocument();
  });

  // Engine phải lên server ngay khi chọn, không phải chỉ lúc nộp
  // render: chuỗi 1a→1b→1c đọc project.RenderEngine từ server để chọn đúng
  // vai trò storyboard/code, nên tới lúc Creator xuống 1c đổi thì đã trễ.
  describe("chọn công cụ render", () => {
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
      seedTopic("Vòng lặp for");
      renderPage();
      expandSettings();

      fireEvent.click(screen.getByTestId("render-engine-remotion"));
      await waitFor(() => expect(createDraft).toHaveBeenCalledWith(expect.any(String), "", "vi", "remotion"));
      createDraft.mockClear();

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(createDraft).toHaveBeenCalledWith(expect.any(String), "Vòng lặp for", "vi", "remotion"),
      );
    });

    // A Remotion video gets its drawings between Visual and Code.
    it("chuỗi của video Remotion có thêm bước Hình minh hoạ trước Code; Manim thì không", async () => {
      vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
      const start = mockServerChain();
      seedTopic("Vòng lặp for");
      renderPage();
      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
      expandSettings();
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      // The button names the steps by the rail's numbers, whatever the
      // engine — Manim skips 5 but the range is still 3–6.
      expect(screen.getByTestId("run-with-ai-story")).toHaveTextContent("Chạy bằng AI các bước 3–6");
      fireEvent.click(screen.getByTestId("render-engine-remotion"));
      await waitFor(() => expect(screen.getByTestId("run-with-ai-story")).toHaveTextContent("Chạy bằng AI các bước 3–6"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));
      await waitFor(() =>
        expect(start).toHaveBeenCalledWith(expect.any(String), ["story", "storyboard", "illustrations", "code"]),
      );
    });
  });

  // Chế độ làm việc là lựa chọn cho CẢ bước 1, không phải một
  // nút riêng từng tab. Mặc định là copy tay.
  describe("chế độ làm bước 1", () => {
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
      // The mode applies to all of steps 3–6, not "bước 3".
      expect(screen.getByRole("radiogroup", { name: "Cách làm các bước 3–6" })).toBeInTheDocument();
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 3 — Kịch bản");
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

    // Một lần bấm chạy cả 1a → 1b → 1c. Chuỗi chạy tuần tự vì mỗi
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

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() => expect(start).toHaveBeenCalledTimes(1));
      expect(start.mock.calls[0][1]).toEqual(["story", "storyboard", "code"]);
      await waitFor(() =>
        expect(screen.getByTestId("script-outline-story-input")).toHaveValue("CÂU HỎI CỐT LÕI: vì sao?"),
      );
    });

    // 1c có thể lưu code vẫn còn lỗi biên dịch sau các vòng sửa. Đó là
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

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
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

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Vòng lặp for"));
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
      seedTopic("Cây nhị phân");
      renderPage();
      expandSettings();

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("Cây nhị phân"));
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(saveDraft).toHaveBeenCalledWith(expect.any(String), "Cây nhị phân", "vi", "manim"),
      );
    });

    // Chưa có key thì chế độ AI không chọn được, và đường copy tay
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
        new apiClient.ApiError("Tài khoản Hive hết số dư — nạp thêm ở dashboard Hive. Hoặc dùng nút Sao chép prompt như cũ."),
      );
      seedTopic("X");
      renderPage();
      expandSettings();

      await waitFor(() => expect(screen.getByTestId("topic-summary")).toHaveTextContent("X"));
      await waitFor(() => expect(screen.getByTestId("authoring-mode-ai")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("authoring-mode-ai"));
      fireEvent.click(screen.getByTestId("run-with-ai-story"));

      await waitFor(() =>
        expect(screen.getByTestId("run-with-ai-error")).toHaveTextContent("hết số dư"),
      );
      expect(screen.getByTestId("run-with-ai-error")).toHaveTextContent("Sao chép prompt");
    });

    // Chế độ nằm ở project trong DB, không chỉ localStorage: đó là cái
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
    setup({ running: true, steps: ["story", "storyboard", "code"], current_index: 0, finished: false });
    renderPage();

    await waitFor(() => expect(screen.getByTestId("run-with-ai-running")).toBeInTheDocument());
    expect(screen.getByTestId("run-with-ai-running")).toHaveTextContent("Đang chạy Bước 3 — Kịch bản");
    expect(screen.getByTestId("run-with-ai-story")).toBeDisabled();
    expect(screen.getByTestId("run-with-ai-cancel")).toBeInTheDocument();
  });

  it("khi AI đã sang bước 4, màn bước 3 không còn trông như đang chạy", async () => {
    setup({ running: true, steps: ["story", "storyboard", "code"], current_index: 1, finished: false });
    renderPage();

    await screen.findByTestId("authoring-run-elsewhere");
    expect(screen.queryByTestId("run-with-ai-running")).not.toBeInTheDocument();
    expect(screen.queryByRole("status", { name: "AI đang chạy" })).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-with-ai-cancel")).not.toBeInTheDocument();
    // The button keeps its own label and stays locked while the chain runs.
    expect(screen.getByTestId("run-with-ai-story")).toHaveTextContent("Chạy bằng AI các bước 3–6");
    expect(screen.getByTestId("run-with-ai-story")).toBeDisabled();
    // The step menu says which step the AI is on.
    await waitFor(() => expect(screen.getByTestId("rail-step-4")).toHaveAttribute("data-status", "running"));
    expect(screen.getByTestId("rail-step-3")).not.toHaveAttribute("data-status", "running");
  });

  // Each screen shows only its own step's progress: once the outline is done
  // and the chain moved on, step 3 says where the AI is and leads there.
  it("chỉ hiện tiến độ của bước mình: dàn ý xong thì nói AI đang chạy bước nào và mở được bước đó", async () => {
    setup({ running: true, steps: ["story", "storyboard", "code"], current_index: 1, finished: false });
    renderPage();

    const elsewhere = await screen.findByTestId("authoring-run-elsewhere");
    expect(elsewhere).toHaveTextContent("Bước 3 — Kịch bản đã xong. AI đang chạy Bước 4 — Hình ảnh.");
    expect(screen.queryByTestId(/^authoring-run-panel-/)).not.toBeInTheDocument();
    expect(screen.queryByTestId("authoring-live-progress")).not.toBeInTheDocument();
    expect(screen.getByTestId("authoring-run-open-current")).toHaveTextContent("Mở Bước 4 — Hình ảnh");
  });

  it("khi AI đang chạy chính bước này, hiện thẻ tiến độ của nó", async () => {
    setup({ running: true, steps: ["story", "storyboard", "code"], current_index: 0, finished: false });
    renderPage();

    expect(await screen.findByTestId("authoring-live-progress")).toBeInTheDocument();
    expect(screen.queryByTestId("authoring-run-elsewhere")).not.toBeInTheDocument();
  });

  it("lỗi ở bước khác: chỉ nói chuỗi dừng ở đâu và mở được bước đó, đóng được", async () => {
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

    const row = await screen.findByTestId("run-with-ai-error-elsewhere");
    expect(row).toHaveTextContent("Chuỗi AI dừng vì lỗi ở Bước 4 — Hình ảnh");
    expect(screen.queryByTestId("run-with-ai-error")).not.toBeInTheDocument();
    expect(screen.getByTestId("run-with-ai-open-error")).toHaveTextContent("Mở Bước 4 — Hình ảnh");

    fireEvent.click(screen.getByTestId("run-with-ai-dismiss"));
    await waitFor(() => expect(screen.queryByTestId("run-with-ai-error-elsewhere")).not.toBeInTheDocument());
  });

  it("lỗi ở chính bước này hiện nguyên văn", async () => {
    setup({
      running: false,
      steps: ["story", "storyboard", "code"],
      current_index: 0,
      finished: true,
      error: "Tài khoản Hive hết số dư.",
      error_step: "story",
      finished_at: new Date().toISOString(),
    });
    renderPage();

    const err = await screen.findByTestId("run-with-ai-error");
    expect(err).toHaveTextContent("hết số dư");
    expect(err).toHaveTextContent("dừng ở Bước 3 — Kịch bản");
  });

  it("không hiện cảnh báo của storyboard ở bước Kịch bản", async () => {
    setup({
      running: false,
      steps: ["story", "storyboard", "code"],
      current_index: 3,
      finished: true,
      warnings: { storyboard: ["Cảnh concrete: ~58 giây, ngân sách 30–45 giây (+29%)"] },
      finished_at: new Date().toISOString(),
    });
    renderPage();

    expect(await screen.findByTestId("run-with-ai-done")).toBeInTheDocument();
    expect(screen.queryByTestId("run-with-ai-storyboard-warnings")).not.toBeInTheDocument();
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
    expect(story).toHaveTextContent("10,2k ký tự");
    expect(story).toHaveTextContent("4.000 token");
    expect(screen.getByTestId("last-run-storyboard")).toHaveAttribute("data-state", "failed");
    expect(screen.queryByTestId("last-run-code")).not.toBeInTheDocument();
  });
});
