import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { act, render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext, useEffect, type ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftDispatchContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { IdeaReview, IdeaScreen, ProjectIdea } from "../../src/api/client";
import type { Project } from "../../src/types";
import { installFakeAiBridge } from "../helpers/fakeAiBridge";

const SUBJECT = "Chuyện gì xảy ra nếu ta rơi vào hố đen?";

function ideaOf(over: Partial<ProjectIdea> = {}): ProjectIdea {
  return {
    centralQuestion: "Người rơi thấy gì, người ở xa thấy gì?",
    curiosityPoints: ["Chân trời sự kiện là gì?", "Cơ thể có bị kéo dài không?"],
    scope: "Giải thích trực quan, không đi sâu vào toán.",
    creatorFields: [],
    status: "pending",
    redoCount: 0,
    approvedAt: null,
    updatedAt: "",
    ...over,
  };
}

const REVIEW: IdeaReview = {
  clarity: { rating: "good", comment: "Câu hỏi rõ." },
  researchability: { rating: "fair", comment: "Cần nguồn chuyên môn." },
  scopeFit: { rating: "weak", comment: "Phạm vi hơi rộng." },
  unclear: ["Khán giả là ai?"],
  suggestions: ["Nêu rõ hố đen sao hay siêu nặng."],
  createdAt: "",
};

let server: IdeaScreen;
let createDraft: MockInstance<typeof apiClient.createProjectDraft>;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;
let saveModels: MockInstance<typeof apiClient.saveAuthoringModels>;
let generate: MockInstance<typeof apiClient.generateProjectIdea>;
let review: MockInstance<typeof apiClient.reviewProjectIdea>;
let llmEnabled: boolean;

beforeEach(() => {
  llmEnabled = true;
  server = { idea: null, subject: SUBJECT, language: "vi", inputChanged: false, review: null, reviewStale: false };
  vi.spyOn(apiClient, "getLlmStatus").mockImplementation(async () => ({
    enabled: llmEnabled,
    provider: llmEnabled ? "hive" : "",
    reason: llmEnabled ? "" : "Chưa cấu hình",
    models: llmEnabled
      ? [
          { id: "deepseek-ai/deepseek-v4.1-flash", label: "DeepSeek V4.1 Flash" },
          { id: "ollama", label: "Ollama (local AI)" },
        ]
      : [],
  }));
  vi.spyOn(apiClient, "fetchVideoFormats").mockResolvedValue([]);
  vi.spyOn(apiClient, "getIdeaScreen").mockImplementation(async () => server);
  createDraft = vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
  saveModels = vi.spyOn(apiClient, "saveAuthoringModels").mockResolvedValue(undefined);
  generate = vi.spyOn(apiClient, "generateProjectIdea").mockImplementation(async () => {
    server = { ...server, idea: ideaOf() };
    return ideaOf();
  });
  review = vi.spyOn(apiClient, "reviewProjectIdea").mockImplementation(async () => {
    server = { ...server, review: REVIEW, reviewStale: false };
    return REVIEW;
  });
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

/** Loads a project into the draft the way reopening it from the list does. */
function LoadedDraft({ projectId, topic, children }: { projectId: string; topic: string; children: ReactNode }) {
  const dispatch = useContext(ProjectDraftDispatchContext);
  useEffect(() => {
    dispatch({ type: "LOAD_PROJECT", payload: { projectId, authoringTopic: topic } });
  }, [dispatch, projectId, topic]);
  return <>{children}</>;
}

function renderPage({ reopen }: { reopen?: { projectId: string; topic: string } } = {}) {
  const routes = (
    <Routes>
      <Route path="/" element={<ScriptStepPage />} />
      <Route path="/create/script/brief" element={<div data-testid="landed-on-brief" />} />
    </Routes>
  );
  render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/"]}>
        <ProjectDraftProvider>
          {reopen ? (
            <LoadedDraft projectId={reopen.projectId} topic={reopen.topic}>
              <ProjectFlowProvider>{routes}</ProjectFlowProvider>
            </LoadedDraft>
          ) : (
            routes
          )}
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

function typeText(value: string) {
  fireEvent.change(screen.getByTestId("script-step-topic"), { target: { value } });
}

async function createWithAI(text = SUBJECT) {
  await screen.findByText("Tạo project và viết ý tưởng →");
  typeText(text);
  fireEvent.click(screen.getByTestId("script-step-create"));
  await screen.findByTestId("idea-field-central_question");
}

describe("ScriptStepPage (Bước 1 — Ý tưởng)", () => {
  it("chỉ có ô nhập lớn và ngôn ngữ; khoá Tạo project khi còn trống", async () => {
    renderPage();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Ý tưởng ban đầu");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa tạo project");

    const box = screen.getByTestId("script-step-topic");
    expect(box.tagName).toBe("TEXTAREA");
    expect(box).toHaveAttribute("rows", "8");
    expect(screen.getByTestId("content-language-picker")).toBeInTheDocument();
    expect(screen.queryByTestId("video-output-mode-short")).not.toBeInTheDocument();
    expect(screen.queryByTestId("idea-duration")).not.toBeInTheDocument();

    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeText("   ");
    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeText("Vòng lặp for trong Java");
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("tạo project rồi mở hộp thoại AI ngoài để viết ý tưởng, không gọi AI của hệ thống", async () => {
    generate.mockImplementation(async (_id, _note, call) => {
      if (call && "external" in call && call.external.mode === "prompt") throw new apiClient.ExternalPromptReady("PROMPT Ý TƯỞNG");
      return ideaOf() as never;
    });
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeText(SUBJECT);
    fireEvent.click(screen.getByTestId("script-step-create-external"));

    expect(await screen.findByTestId("idea-create-external-dialog-prompt")).toHaveTextContent("PROMPT Ý TƯỞNG");
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), SUBJECT, "vi");
    expect(generate).toHaveBeenCalledTimes(1);
    expect(generate).toHaveBeenCalledWith(expect.any(String), "", { external: { mode: "prompt" } });
  });

  it("tạo project rồi chạy ý tưởng trên DeepSeek; câu trả lời được đọc như dán tay, không tự đánh giá", async () => {
    const bridge = installFakeAiBridge();
    try {
      generate.mockImplementation(async (_id, _note, call) => {
        if (call && "external" in call) {
          if (call.external.mode === "prompt") throw new apiClient.ExternalPromptReady("PROMPT Ý TƯỞNG");
          server = { ...server, idea: ideaOf() };
        }
        return ideaOf() as never;
      });
      renderPage();
      await screen.findByText("Tạo project và viết ý tưởng →");
      typeText(SUBJECT);
      fireEvent.click(await screen.findByTestId("script-step-create-deepseek"));

      await waitFor(() => expect(bridge.runs).toHaveLength(1));
      expect(createDraft).toHaveBeenCalledWith(expect.any(String), SUBJECT, "vi");
      expect(bridge.runs[0]).toMatchObject({ provider: "deepseek", prompt: "PROMPT Ý TƯỞNG" });
      act(() => bridge.send({ type: "result", request_id: bridge.runs[0].requestId, text: "ý tưởng của DeepSeek" }));

      expect(await screen.findByTestId("idea-field-central_question")).toHaveTextContent("Người rơi thấy gì");
      expect(generate).toHaveBeenLastCalledWith(expect.any(String), "", { external: { mode: "paste", reply: "ý tưởng của DeepSeek" } });
      expect(review).not.toHaveBeenCalled();
    } finally {
      bridge.remove();
    }
  });

  it("tạo project với giá trị mặc định, AI viết ba trường; đánh giá chỉ chạy khi Creator nhờ", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeText("  " + SUBJECT + "  ");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("idea-field-central_question");
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), SUBJECT, "vi");
    expect(patchSettings).not.toHaveBeenCalled();
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
    expect(generate).toHaveBeenCalledWith(expect.any(String), "", expect.objectContaining({ id: expect.any(String) }));

    expect(screen.queryByTestId("idea-field-refined_topic")).not.toBeInTheDocument();
    expect(screen.getByTestId("idea-field-central_question")).toHaveTextContent("Người rơi thấy gì");
    expect(screen.getByTestId("idea-field-curiosity_points")).toHaveTextContent("Cơ thể có bị kéo dài không?");
    expect(screen.getByTestId("idea-field-scope")).toHaveTextContent("không đi sâu vào toán");
    expect(screen.getByTestId("idea-status")).toHaveTextContent("Chờ duyệt");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt ý tưởng");

    // Writing the idea does not start a review; the Creator asks for one.
    expect(review).not.toHaveBeenCalled();
    expect(screen.getByTestId("idea-review-none")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("idea-review-run"));
    await waitFor(() => expect(review).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId("idea-review-unclear")).toHaveTextContent("Khán giả là ai?");
    expect(screen.getByTestId("idea-review-suggestions")).toHaveTextContent("hố đen sao hay siêu nặng");
    expect(screen.getByTestId("idea-review-panel")).toHaveTextContent("Phạm vi hơi rộng");
    expect(screen.getByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("làm sạch dấu nháy kép và gạch chéo ngược ngay khi gõ, để AI không chép chúng vào JSON", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeText('Vì sao người ta nói "ăn sung là ăn xác ong"? Đường dẫn C:\\tmp');
    expect(screen.getByTestId("script-step-topic")).toHaveValue("Vì sao người ta nói “ăn sung là ăn xác ong”? Đường dẫn C:/tmp");

    fireEvent.click(screen.getByTestId("script-step-create"));
    await screen.findByTestId("idea-field-central_question");
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), "Vì sao người ta nói “ăn sung là ăn xác ong”? Đường dẫn C:/tmp", "vi");
  });

  it("ghi chú làm lại cũng được làm sạch dấu nháy kép", async () => {
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-actions-redo"));
    fireEvent.change(await screen.findByTestId("idea-redo-note"), { target: { value: 'nhấn mạnh "ngộ nhận"' } });
    expect(screen.getByTestId("idea-redo-note")).toHaveValue("nhấn mạnh “ngộ nhận”");
  });

  it("ngôn ngữ đã chọn đi cùng lúc tạo project", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    fireEvent.click(screen.getByTestId("content-language-en"));
    typeText("What if we fell into a black hole?");
    fireEvent.click(screen.getByTestId("script-step-create"));
    await screen.findByTestId("idea-field-central_question");
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), "What if we fell into a black hole?", "en");
  });

  it("chọn model AI trước khi tạo: project lưu model đó trước lượt chạy AI", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    fireEvent.click(await screen.findByTestId("idea-brief-model"));
    fireEvent.click(await screen.findByTestId("idea-brief-model-option-ollama"));
    typeText("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("idea-field-central_question");
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" }));
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
  });

  it("cảnh báo chủ đề trùng ngay sau khi tạo project, không chặn bước", async () => {
    createDraft.mockResolvedValue({
      similarProjects: [{ projectId: "old-1", topic: "Vòng lặp for", status: "published", createdAt: "" }],
    });
    renderPage();
    await createWithAI("Vòng lặp for");
    expect(screen.getByTestId("topic-collision-banner")).toHaveTextContent("Vòng lặp for (published)");
  });

  it("báo lỗi và ở lại phần nhập khi không tạo được project", async () => {
    createDraft.mockRejectedValue(new Error("network down"));
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeText("Đệ quy");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect((await screen.findAllByText(/Không tạo được dự án/)).length).toBeGreaterThan(0);
    expect(generate).not.toHaveBeenCalled();
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
  });

  it("báo lỗi AI ngay trên thẻ ý tưởng, và vẫn cho tự viết hoặc nhờ AI viết lại", async () => {
    generate.mockRejectedValue(new apiClient.ApiError("câu trả lời của AI không dùng được làm ý tưởng. Thử làm lại, hoặc tự điền các ô.", undefined, undefined, 502));
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeText("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect(await screen.findByTestId("idea-error")).toHaveTextContent("không dùng được làm ý tưởng");
    expect(screen.getByTestId("idea-write-own")).toBeInTheDocument();
    expect(screen.getByTestId("idea-generate")).toBeInTheDocument();
    expect(review).not.toHaveBeenCalled();
  });

  it("đánh giá lỗi không làm mất ý tưởng; chạy lại được", async () => {
    review.mockRejectedValueOnce(new apiClient.ApiError("câu trả lời của AI không dùng được làm đánh giá ý tưởng.", undefined, undefined, 502));
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-review-run"));
    expect(await screen.findByTestId("idea-error")).toHaveTextContent("đánh giá ý tưởng");
    expect(screen.getByTestId("idea-field-central_question")).toHaveTextContent("Người rơi thấy gì");
    expect(screen.getByTestId("idea-review-none")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("idea-review-run"));
    await screen.findByTestId("idea-review-unclear");
    expect(screen.queryByTestId("idea-review-stale")).not.toBeInTheDocument();
  });

  it("đánh giá chỉ để tham khảo: vẫn duyệt được khi đánh giá yếu, nút duyệt nằm sau đánh giá", async () => {
    const approve = vi.spyOn(apiClient, "approveProjectIdea").mockImplementation(async () => {
      server = { ...server, idea: ideaOf({ status: "approved" }) };
      return ideaOf({ status: "approved" });
    });
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-review-run"));
    await screen.findByTestId("idea-review-unclear");

    const panel = screen.getByTestId("idea-review-panel");
    const approveBtn = screen.getByTestId("idea-actions-approve");
    expect(approveBtn).not.toBeDisabled();
    expect(panel.compareDocumentPosition(approveBtn) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    fireEvent.click(approveBtn);
    await waitFor(() => expect(screen.getByTestId("idea-status")).toHaveTextContent("Đã duyệt"));
    expect(approve).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("idea-actions-unapprove")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("script-step-next"));
    expect(await screen.findByTestId("landed-on-brief")).toBeInTheDocument();
  });

  it("không cho duyệt ý tưởng còn ô trống, và nói ô nào", async () => {
    generate.mockImplementation(async () => {
      server = { ...server, idea: ideaOf({ scope: "", curiosityPoints: [] }) };
      return ideaOf();
    });
    renderPage();
    await createWithAI();

    expect(screen.getByTestId("idea-actions-approve")).toBeDisabled();
    expect(screen.getByTestId("idea-missing")).toHaveTextContent("Điểm gây tò mò; Phạm vi");
    expect(screen.getByTestId("idea-review-run")).toBeDisabled();
  });

  it("Sửa tay chỉ gửi ô đã đổi, mỗi dòng một điểm gây tò mò, và đánh dấu Bạn viết", async () => {
    const edit = vi.spyOn(apiClient, "editProjectIdea").mockImplementation(async (_id, patch) => {
      const next = ideaOf({ ...patch, creatorFields: ["curiosity_points", "scope"] });
      server = { ...server, idea: next, reviewStale: true };
      return next;
    });
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-review-run"));
    await screen.findByTestId("idea-review-unclear");

    fireEvent.click(screen.getByTestId("idea-actions-edit"));
    expect(screen.getByTestId("idea-edit-field-curiosity_points")).toHaveValue("Chân trời sự kiện là gì?\nCơ thể có bị kéo dài không?");
    fireEvent.change(screen.getByTestId("idea-edit-field-curiosity_points"), { target: { value: "Một điểm mới\n\n  Điểm thứ hai  " } });
    fireEvent.change(screen.getByTestId("idea-edit-field-scope"), { target: { value: "Phạm vi mới" } });
    fireEvent.click(screen.getByTestId("idea-edit-save"));

    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith(expect.any(String), { curiosityPoints: ["Một điểm mới", "Điểm thứ hai"], scope: "Phạm vi mới" }),
    );
    expect(await screen.findByTestId("idea-mine-scope")).toHaveTextContent("Bạn viết");
    expect(screen.getByTestId("idea-mine-curiosity_points")).toBeInTheDocument();
    expect(await screen.findByTestId("idea-review-stale")).toBeInTheDocument();
  });

  it("Sửa tay một ý tưởng đã duyệt nói rõ là nó về chờ duyệt", async () => {
    vi.spyOn(apiClient, "getProject").mockImplementation(async (id) => ({ project_id: id, status: "draft" }) as Project);
    server = { ...server, idea: ideaOf({ status: "approved" }) };
    renderPage({ reopen: { projectId: "p-old", topic: SUBJECT } });
    fireEvent.click(await screen.findByTestId("idea-actions-edit"));
    expect(await screen.findByTestId("idea-edit")).toHaveTextContent("về chờ duyệt");
  });

  it("Làm lại kèm ghi chú: nút tím, cần ghi chú, gửi ghi chú cho AI; không tự đánh giá", async () => {
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-actions-redo"));
    const send = await screen.findByTestId("idea-redo-send");
    await waitFor(() => expect(send.className).toContain("runBtn"));
    expect(send).toBeDisabled();

    fireEvent.change(screen.getByTestId("idea-redo-note"), { target: { value: "  nhấn mạnh ngộ nhận " } });
    fireEvent.click(screen.getByTestId("idea-redo-send"));

    await waitFor(() =>
      expect(generate).toHaveBeenLastCalledWith(expect.any(String), "nhấn mạnh ngộ nhận", expect.objectContaining({ id: expect.any(String) })),
    );
    await waitFor(() => expect(screen.queryByTestId("idea-redo")).not.toBeInTheDocument());
    expect(review).not.toHaveBeenCalled();
  });

  it("đổi model AI ở modal làm lại thì lưu lên project", async () => {
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-actions-redo"));
    fireEvent.click(await screen.findByTestId("idea-redo-model"));
    fireEvent.click(await screen.findByTestId("idea-redo-model-option-ollama"));
    await waitFor(() => expect(saveModels).toHaveBeenLastCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" })));
  });

  it("modal sửa tay không đóng khi bấm ra ngoài, Huỷ thì đóng", async () => {
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-actions-edit"));
    await screen.findByTestId("idea-edit");
    fireEvent.click(screen.getByTestId("idea-edit").parentElement as HTMLElement);
    expect(screen.getByTestId("idea-edit")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("idea-edit-cancel"));
    await waitFor(() => expect(screen.queryByTestId("idea-edit")).not.toBeInTheDocument());
  });

  it("không có AI: chỉ tạo project, rồi Creator tự viết ý tưởng và không có nút đánh giá", async () => {
    llmEnabled = false;
    renderPage();
    await screen.findByText("Tạo project →");
    typeText("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    fireEvent.click(await screen.findByTestId("idea-write-own"));
    expect(generate).not.toHaveBeenCalled();
    expect(await screen.findByText("Tự viết · Ý tưởng")).toBeInTheDocument();
  });

  it("mở lại project đã có: nội dung sửa được, ý tưởng và cờ nội dung đã đổi nạp từ server", async () => {
    vi.spyOn(apiClient, "getProject").mockImplementation(async (id) => ({ project_id: id, status: "draft" }) as Project);
    server = { ...server, idea: ideaOf({ status: "approved", creatorFields: ["central_question"] }), inputChanged: true, language: "en" };
    renderPage({ reopen: { projectId: "p-old", topic: SUBJECT } });

    expect(await screen.findByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.getByTestId("script-step-topic")).toHaveValue(SUBJECT);
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(screen.queryByTestId("content-language-picker")).not.toBeInTheDocument();
    expect(await screen.findByTestId("idea-mine-central_question")).toBeInTheDocument();
    expect(screen.getByTestId("idea-input-changed")).toBeInTheDocument();
    expect(screen.getByTestId("idea-language-note")).toHaveTextContent("English");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Ý tưởng đã duyệt");
    await waitFor(() => expect(screen.getByTestId("script-step-next")).not.toBeDisabled());
  });

  it("sửa nội dung sau khi tạo: có nút Lưu nội dung, gọi cập nhật chủ đề rồi nạp lại", async () => {
    vi.spyOn(apiClient, "getProject").mockImplementation(async (id) => ({ project_id: id, status: "draft" }) as Project);
    const update = vi.spyOn(apiClient, "updateProjectTopic").mockResolvedValue({ similarProjects: [] });
    server = { ...server, idea: ideaOf() };
    renderPage({ reopen: { projectId: "p-old", topic: SUBJECT } });

    await screen.findByTestId("idea-created-note");
    expect(screen.queryByTestId("script-step-save-text")).not.toBeInTheDocument();
    typeText(SUBJECT + " Và tại sao?");
    fireEvent.click(await screen.findByTestId("script-step-save-text"));
    await waitFor(() => expect(update).toHaveBeenCalledWith("p-old", SUBJECT + " Và tại sao?"));
  });

  it("bắt đầu draft mới khi draft đã chạy render", async () => {
    // Không có việc này thì quay lại "/" sau một lần render sẽ dùng lại
    // project_id cũ và lần tạo tiếp theo ghi đè video trước.
    function SpentDraft({ children }: { children: ReactNode }) {
      const dispatch = useContext(ProjectDraftDispatchContext);
      useEffect(() => {
        dispatch({ type: "LOAD_PROJECT", payload: { projectId: "spent-project", authoringTopic: "chủ đề cũ" } });
        dispatch({ type: "MARK_SUBMITTED" });
      }, [dispatch]);
      return <>{children}</>;
    }
    render(
      <ThemeProvider>
        <MemoryRouter initialEntries={["/"]}>
          <ProjectDraftProvider>
            <SpentDraft>
              <ScriptStepPage />
            </SpentDraft>
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => expect((screen.getByTestId("script-step-topic") as HTMLTextAreaElement).value).toBe(""));
    expect(screen.getByTestId("script-step-create")).toBeDisabled();
  });
});
