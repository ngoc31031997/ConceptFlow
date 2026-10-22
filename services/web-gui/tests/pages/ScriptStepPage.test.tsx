import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext, useEffect, type ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftDispatchContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Idea, IdeaState, ReviewUnit, UnitStatus } from "../../src/api/client";
import type { Project } from "../../src/types";

const IDEA: Idea = {
  question: "Ong vào quả sung để làm gì?",
  why: "Ai cũng nghe ăn sung là ăn xác ong",
  known: "Quả sung là chùm hoa kín",
  unknown: "Quả ngoài chợ có còn xác ong không",
};

function unitOf(status: UnitStatus, creatorFields: string[] = [], content: Idea = IDEA): ReviewUnit<Idea> {
  return { projectId: "p", kind: "idea", key: "main", status, content, creatorFields, redoCount: 0, updatedAt: "", stale: false };
}

let serverState: IdeaState;
let createDraft: MockInstance<typeof apiClient.createProjectDraft>;
let setDuration: MockInstance<typeof apiClient.setTargetDuration>;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;
let saveModels: MockInstance<typeof apiClient.saveAuthoringModels>;
let generate: MockInstance<typeof apiClient.generateIdea>;
let llmEnabled: boolean;

beforeEach(() => {
  llmEnabled = true;
  serverState = { unit: null };
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
  vi.spyOn(apiClient, "getIdeaState").mockImplementation(async () => serverState);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
  createDraft = vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
  setDuration = vi.spyOn(apiClient, "setTargetDuration").mockResolvedValue(undefined);
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
  saveModels = vi.spyOn(apiClient, "saveAuthoringModels").mockResolvedValue(undefined);
  generate = vi.spyOn(apiClient, "generateIdea").mockImplementation(async () => {
    serverState = { unit: unitOf("pending") };
    return unitOf("pending");
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

function typeTopic(value: string) {
  fireEvent.change(screen.getByTestId("script-step-topic"), { target: { value } });
}

async function createWithAI(topic = "Vì sao trong quả sung có xác ong?") {
  await screen.findByText("Tạo project và viết ý tưởng →");
  typeTopic(topic);
  fireEvent.click(screen.getByTestId("script-step-create"));
  await screen.findByTestId("idea-field-question");
}

describe("ScriptStepPage (Bước 1 — Ý tưởng)", () => {
  it("khoá Tạo project khi chủ đề còn trống, và Sang Creative brief khi chưa có ý tưởng đã duyệt", async () => {
    renderPage();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Ý tưởng ban đầu");
    expect(screen.getByTestId("step-role")).toHaveTextContent("Bước này:");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa tạo project");

    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeTopic("   ");
    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeTopic("Vòng lặp for trong Java");
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("không hỏi dạng video, ngôn ngữ, thời lượng: những lựa chọn đó ở bước Creative brief", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    expect(screen.queryByTestId("video-output-mode-short")).not.toBeInTheDocument();
    expect(screen.queryByTestId("content-language-en")).not.toBeInTheDocument();
    expect(screen.queryByTestId("idea-duration")).not.toBeInTheDocument();
  });

  it("tạo project với thời lượng mặc định, rồi AI viết ý tưởng chờ duyệt", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeTopic("  Vì sao trong quả sung có xác ong?  ");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("idea-field-question");
    expect(createDraft).toHaveBeenCalledTimes(1);
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), "Vì sao trong quả sung có xác ong?", "vi");
    expect(setDuration).toHaveBeenCalledWith(expect.any(String), apiClient.DEFAULT_TARGET_DURATION);
    expect(patchSettings).not.toHaveBeenCalled();
    // The idea runs on the story step's model, so the project gets the models before the run.
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ story: expect.any(String) }));
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
    expect(generate).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("idea-field-question")).toHaveTextContent(IDEA.question);
    expect(screen.getByTestId("idea-field-unknown")).toHaveTextContent(IDEA.unknown);
    expect(screen.getByTestId("idea-status")).toHaveTextContent("Chờ duyệt");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt ý tưởng");
    expect(screen.getByTestId("script-step-topic")).toBeDisabled();
    expect(screen.getByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("chọn model AI ngay trước khi tạo: project lưu model đó trước lượt chạy AI", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    fireEvent.click(await screen.findByTestId("idea-brief-model"));
    fireEvent.click(await screen.findByTestId("idea-brief-model-option-ollama"));
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("idea-field-question");
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" }));
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
  });

  it("project mới bắt đầu bằng model mặc định: chưa chọn thì không ghi đè bằng model nào", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));
    await screen.findByTestId("idea-field-question");
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ brief: "" }));
  });

  it("đổi model AI ở modal làm lại thì lưu lên project", async () => {
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("idea-review-redo"));
    fireEvent.click(await screen.findByTestId("idea-redo-model"));
    fireEvent.click(await screen.findByTestId("idea-redo-model-option-ollama"));
    await waitFor(() => expect(saveModels).toHaveBeenLastCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" })));
  });

  it("cảnh báo chủ đề trùng ngay sau khi tạo project, không chặn bước", async () => {
    createDraft.mockResolvedValue({
      similarProjects: [{ projectId: "old-1", topic: "Vòng lặp for", status: "published", createdAt: "" }],
    });
    renderPage();
    await createWithAI("Vòng lặp for");

    expect(screen.getByTestId("topic-collision-banner")).toHaveTextContent("Vòng lặp for (published)");
  });

  it("báo lỗi và ở lại phần Đề tài khi không tạo được project", async () => {
    createDraft.mockRejectedValue(new Error("network down"));
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeTopic("Đệ quy");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect((await screen.findAllByText(/Không tạo được dự án/)).length).toBeGreaterThan(0);
    expect(generate).not.toHaveBeenCalled();
    // Lỗi xong thì Creator thử lại được, không bị kẹt ở trạng thái "Đang tạo".
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
  });

  it("báo lỗi AI ngay trên thẻ ý tưởng, và vẫn cho tự viết hoặc nhờ AI viết lại", async () => {
    generate.mockRejectedValue(new apiClient.ApiError("câu trả lời của AI không dùng được làm ý tưởng. Thử làm lại, hoặc tự điền các ô.", undefined, undefined, 502));
    renderPage();
    await screen.findByText("Tạo project và viết ý tưởng →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect(await screen.findByTestId("idea-error")).toHaveTextContent("không dùng được làm ý tưởng");
    expect(screen.getByTestId("idea-write-own")).toBeInTheDocument();
    expect(screen.getByTestId("idea-generate")).toBeInTheDocument();
  });

  it("duyệt ý tưởng thì mở Sang Creative brief", async () => {
    const approve = vi.spyOn(apiClient, "approveUnit").mockImplementation(async () => {
      serverState = { unit: unitOf("approved") };
      return unitOf("approved") as never;
    });
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-review-approve"));
    await waitFor(() => expect(screen.getByTestId("idea-status")).toHaveTextContent("Đã duyệt"));
    expect(approve).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("idea-review-unapprove")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("script-step-next"));
    expect(await screen.findByTestId("landed-on-brief")).toBeInTheDocument();
  });

  it("không cho duyệt ý tưởng còn ô trống, và nói ô nào", async () => {
    generate.mockImplementation(async () => {
      serverState = { unit: unitOf("pending", [], { ...IDEA, why: "" }) };
      return unitOf("pending", [], { ...IDEA, why: "" });
    });
    renderPage();
    await createWithAI();

    expect(screen.getByTestId("idea-review-approve")).toBeDisabled();
    expect(screen.getByTestId("idea-missing")).toHaveTextContent("Vì sao câu hỏi này thú vị");
  });

  it("Sửa tay chỉ gửi ô đã đổi; khi brief đã có, modal liệt kê ảnh hưởng và nút lưu màu cam", async () => {
    vi.mocked(apiClient.getUnitImpact).mockResolvedValue(["Creative brief: cần xem lại theo ý tưởng mới"]);
    const edit = vi.spyOn(apiClient, "editUnit").mockImplementation(async (_p, _r, fields) => {
      const next = unitOf("pending", ["why"], { ...IDEA, ...fields });
      serverState = { unit: next };
      return next as never;
    });
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-review-edit"));
    expect(await screen.findByTestId("idea-edit-impact")).toHaveTextContent("Creative brief");
    expect(screen.getByTestId("idea-edit-save").className).toContain("impactBtn");

    fireEvent.change(screen.getByTestId("idea-edit-field-why"), { target: { value: "Vì một nghịch lý" } });
    fireEvent.click(screen.getByTestId("idea-edit-save"));

    await waitFor(() => expect(edit).toHaveBeenCalledWith(expect.any(String), apiClient.IDEA_UNIT, { why: "Vì một nghịch lý" }));
    expect(await screen.findByTestId("idea-mine-why")).toHaveTextContent("Bạn viết");
    expect(screen.getByTestId("idea-field-why")).toHaveTextContent("Vì một nghịch lý");
  });

  it("Làm lại kèm ghi chú: nút tím khi chưa có gì phụ thuộc, cần ghi chú, gửi ghi chú cho AI", async () => {
    const redo = vi.spyOn(apiClient, "redoUnit").mockResolvedValue(unitOf("pending") as never);
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-review-redo"));
    const send = await screen.findByTestId("idea-redo-send");
    await waitFor(() => expect(send.className).toContain("runBtn"));
    expect(send).toBeDisabled();

    fireEvent.change(screen.getByTestId("idea-redo-note"), { target: { value: "  nhấn mạnh ngộ nhận " } });
    fireEvent.click(screen.getByTestId("idea-redo-send"));

    await waitFor(() =>
      expect(redo).toHaveBeenCalledWith(expect.any(String), apiClient.IDEA_UNIT, "nhấn mạnh ngộ nhận", expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) })),
    );
  });

  it("modal sửa tay không đóng khi bấm ra ngoài, Huỷ thì đóng", async () => {
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("idea-review-edit"));
    await screen.findByTestId("idea-edit");
    fireEvent.click(screen.getByTestId("idea-edit").parentElement as HTMLElement);
    expect(screen.getByTestId("idea-edit")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("idea-edit-cancel"));
    await waitFor(() => expect(screen.queryByTestId("idea-edit")).not.toBeInTheDocument());
  });

  it("không có AI: chỉ tạo project, rồi Creator tự viết ý tưởng", async () => {
    llmEnabled = false;
    renderPage();
    await screen.findByText("Tạo project →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    fireEvent.click(await screen.findByTestId("idea-write-own"));
    expect(generate).not.toHaveBeenCalled();
    expect(await screen.findByText("Tự viết · Ý tưởng")).toBeInTheDocument();
  });

  it("mở lại project đã có: đề tài chỉ đọc, ý tưởng nạp từ server", async () => {
    vi.spyOn(apiClient, "getProject").mockImplementation(async (id) => ({ project_id: id, status: "draft" }) as Project);
    serverState = { unit: unitOf("approved", ["question"]) };
    renderPage({ reopen: { projectId: "p-old", topic: "Ong sung" } });

    expect(await screen.findByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.getByTestId("script-step-topic")).toHaveValue("Ong sung");
    expect(screen.getByTestId("script-step-topic")).toBeDisabled();
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(await screen.findByTestId("idea-mine-question")).toBeInTheDocument();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Ý tưởng đã duyệt");
    await waitFor(() => expect(screen.getByTestId("script-step-next")).not.toBeDisabled());
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
