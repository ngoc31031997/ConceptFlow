import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext, useEffect, type ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScriptStepPage } from "../../src/pages/ScriptStepPage";
import { ProjectDraftDispatchContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { BriefState, CreativeBrief, ReviewUnit, UnitStatus } from "../../src/api/client";
import type { Project } from "../../src/types";

const BRIEF: CreativeBrief = {
  question: "Ong vào quả sung để làm gì?",
  audience: "Người xem 16–35 tuổi",
  know: "Quả sung là chùm hoa kín",
  understand: "Vòng đời ong gắn với sung",
  feel: "Ngạc nhiên",
  tone: "Tò mò, ấm",
  out_of_scope: "Không tư vấn ăn uống",
};

function unitOf(status: UnitStatus, creatorFields: string[] = [], content: CreativeBrief = BRIEF): ReviewUnit<CreativeBrief> {
  return { projectId: "p", kind: "brief", key: "main", status, content, creatorFields, redoCount: 0, updatedAt: "" };
}

let serverState: BriefState;
let createDraft: MockInstance<typeof apiClient.createProjectDraft>;
let setDuration: MockInstance<typeof apiClient.setTargetDuration>;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;
let saveModels: MockInstance<typeof apiClient.saveAuthoringModels>;
let generate: MockInstance<typeof apiClient.generateBrief>;
let llmEnabled: boolean;

beforeEach(() => {
  llmEnabled = true;
  serverState = { unit: null, targetDuration: "4-5", storyStale: false, storyPromptReadsBrief: true };
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
  vi.spyOn(apiClient, "getBriefState").mockImplementation(async () => serverState);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
  createDraft = vi.spyOn(apiClient, "createProjectDraft").mockResolvedValue({ similarProjects: [] });
  setDuration = vi.spyOn(apiClient, "setTargetDuration").mockResolvedValue(undefined);
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
  saveModels = vi.spyOn(apiClient, "saveAuthoringModels").mockResolvedValue(undefined);
  generate = vi.spyOn(apiClient, "generateBrief").mockImplementation(async () => {
    serverState = { ...serverState, unit: unitOf("pending") };
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
      <Route path="/create/script/settings" element={<div data-testid="landed-on-settings" />} />
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
  await screen.findByText("Tạo project và viết brief →");
  typeTopic(topic);
  fireEvent.click(screen.getByTestId("script-step-create"));
  await screen.findByTestId("brief-field-question");
}

describe("ScriptStepPage (Bước 1 — Ý tưởng)", () => {
  it("khoá Tạo project khi chủ đề còn trống, và Sang cấu hình khi chưa có brief đã duyệt", async () => {
    renderPage();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Ý tưởng · creative brief");
    expect(screen.getByTestId("step-role")).toHaveTextContent("Bước này:");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa tạo project");

    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeTopic("   ");
    expect(screen.getByTestId("script-step-create")).toBeDisabled();
    typeTopic("Vòng lặp for trong Java");
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("tạo project với thời lượng đã chọn, rồi AI viết brief chờ duyệt", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết brief →");
    typeTopic("  Vì sao trong quả sung có xác ong?  ");
    fireEvent.change(screen.getByTestId("idea-duration"), { target: { value: "5-7" } });
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("brief-field-question");
    expect(createDraft).toHaveBeenCalledTimes(1);
    expect(createDraft).toHaveBeenCalledWith(expect.any(String), "Vì sao trong quả sung có xác ong?", "vi");
    expect(setDuration).toHaveBeenCalledWith(expect.any(String), "5-7");
    expect(patchSettings).not.toHaveBeenCalled();
    // The brief runs on the story step's model, so the project gets the models before the run.
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ story: expect.any(String) }));
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
    expect(generate).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("brief-field-question")).toHaveTextContent(BRIEF.question);
    expect(screen.getByTestId("brief-status")).toHaveTextContent("Chờ duyệt");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt brief");
    // Đề tài khoá sau khi tạo: brief viết theo đúng những lựa chọn đó.
    expect(screen.getByTestId("script-step-topic")).toBeDisabled();
    expect(screen.getByTestId("video-output-mode-short")).toBeDisabled();
    expect(screen.getByTestId("idea-duration")).toBeDisabled();
    expect(screen.getByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(screen.getByTestId("script-step-next")).toBeDisabled();
  });

  it("chọn model AI viết brief ngay trước khi tạo: project lưu model đó trước lượt chạy AI", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết brief →");
    fireEvent.click(await screen.findByTestId("idea-brief-model"));
    fireEvent.click(await screen.findByTestId("idea-brief-model-option-ollama"));
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("brief-field-question");
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" }));
    expect(saveModels.mock.invocationCallOrder[0]).toBeLessThan(generate.mock.invocationCallOrder[0]);
  });

  it("project mới bắt đầu bằng model mặc định: chưa chọn thì không ghi đè bằng model nào", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết brief →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));
    await screen.findByTestId("brief-field-question");
    expect(saveModels).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ brief: "" }));
  });

  it("đổi model AI viết brief ở modal làm lại thì lưu lên project", async () => {
    renderPage();
    await createWithAI();
    fireEvent.click(screen.getByTestId("brief-review-redo"));
    fireEvent.click(await screen.findByTestId("brief-redo-model"));
    fireEvent.click(await screen.findByTestId("brief-redo-model-option-ollama"));
    await waitFor(() => expect(saveModels).toHaveBeenLastCalledWith(expect.any(String), expect.objectContaining({ brief: "ollama" })));
  });

  it("short: lưu dạng video short thay cho thời lượng, và hiện dưới 60 giây", async () => {
    renderPage();
    await screen.findByText("Tạo project và viết brief →");
    fireEvent.click(screen.getByTestId("video-output-mode-short"));
    expect(screen.getByTestId("idea-duration")).toHaveTextContent("dưới 60 giây");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    await screen.findByTestId("brief-field-question");
    expect(patchSettings).toHaveBeenCalledWith(expect.any(String), { videoOutputMode: "short" });
    expect(setDuration).not.toHaveBeenCalled();
    expect(screen.getByTestId("idea-duration")).toHaveTextContent("dưới 60 giây");
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
    await screen.findByText("Tạo project và viết brief →");
    typeTopic("Đệ quy");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect((await screen.findAllByText(/Không tạo được dự án/)).length).toBeGreaterThan(0);
    expect(generate).not.toHaveBeenCalled();
    // Lỗi xong thì Creator thử lại được, không bị kẹt ở trạng thái "Đang tạo".
    expect(screen.getByTestId("script-step-create")).not.toBeDisabled();
  });

  it("báo lỗi AI ngay trên thẻ brief, và vẫn cho tự viết hoặc nhờ AI viết lại", async () => {
    generate.mockRejectedValue(new apiClient.ApiError("câu trả lời của AI không dùng được làm brief. Thử làm lại, hoặc tự điền các ô.", undefined, undefined, 502));
    renderPage();
    await screen.findByText("Tạo project và viết brief →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    expect(await screen.findByTestId("brief-error")).toHaveTextContent("không dùng được làm brief");
    expect(screen.getByTestId("brief-write-own")).toBeInTheDocument();
    expect(screen.getByTestId("brief-generate")).toBeInTheDocument();
  });

  it("duyệt brief thì mở Sang cấu hình", async () => {
    const approve = vi.spyOn(apiClient, "approveUnit").mockImplementation(async () => {
      serverState = { ...serverState, unit: unitOf("approved") };
      return unitOf("approved") as never;
    });
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("brief-review-approve"));
    await waitFor(() => expect(screen.getByTestId("brief-status")).toHaveTextContent("Đã duyệt"));
    expect(approve).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("brief-review-unapprove")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("script-step-next"));
    expect(await screen.findByTestId("landed-on-settings")).toBeInTheDocument();
  });

  it("không cho duyệt brief còn ô trống, và nói ô nào", async () => {
    generate.mockImplementation(async () => {
      serverState = { ...serverState, unit: unitOf("pending", [], { ...BRIEF, feel: "" }) };
      return unitOf("pending", [], { ...BRIEF, feel: "" });
    });
    renderPage();
    await createWithAI();

    expect(screen.getByTestId("brief-review-approve")).toBeDisabled();
    expect(screen.getByTestId("brief-missing")).toHaveTextContent("Sau video, người xem sẽ cảm thấy");
  });

  it("Sửa tay chỉ gửi ô đã đổi; khi Kịch bản đã có, modal liệt kê ảnh hưởng và nút lưu màu cam", async () => {
    vi.mocked(apiClient.getUnitImpact).mockResolvedValue([
      "Dàn ý và lời thoại đã viết ở bước Kịch bản: cần viết lại theo brief mới",
    ]);
    const edit = vi.spyOn(apiClient, "editUnit").mockImplementation(async (_p, _r, fields) => {
      const next = unitOf("pending", ["tone"], { ...BRIEF, ...fields });
      serverState = { ...serverState, unit: next };
      return next as never;
    });
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("brief-review-edit"));
    expect(await screen.findByTestId("brief-edit-impact")).toHaveTextContent("bước Kịch bản");
    expect(screen.getByTestId("brief-edit-save").className).toContain("impactBtn");

    fireEvent.change(screen.getByTestId("brief-edit-field-tone"), { target: { value: "Hài hước" } });
    fireEvent.click(screen.getByTestId("brief-edit-save"));

    await waitFor(() => expect(edit).toHaveBeenCalledWith(expect.any(String), apiClient.BRIEF_UNIT, { tone: "Hài hước" }));
    expect(await screen.findByTestId("brief-mine-tone")).toHaveTextContent("Bạn viết");
    expect(screen.getByTestId("brief-field-tone")).toHaveTextContent("Hài hước");
  });

  it("Làm lại kèm ghi chú: nút tím khi chưa có gì phụ thuộc, cần ghi chú, gửi ghi chú cho AI", async () => {
    const redo = vi.spyOn(apiClient, "redoUnit").mockResolvedValue(unitOf("pending") as never);
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("brief-review-redo"));
    const send = await screen.findByTestId("brief-redo-send");
    await waitFor(() => expect(send.className).toContain("runBtn"));
    expect(send).toBeDisabled();

    fireEvent.change(screen.getByTestId("brief-redo-note"), { target: { value: "  nói về ong đực nhiều hơn " } });
    fireEvent.click(screen.getByTestId("brief-redo-send"));

    await waitFor(() =>
      expect(redo).toHaveBeenCalledWith(expect.any(String), apiClient.BRIEF_UNIT, "nói về ong đực nhiều hơn", expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) })),
    );
  });

  it("modal sửa tay không đóng khi bấm ra ngoài, Huỷ thì đóng", async () => {
    renderPage();
    await createWithAI();

    fireEvent.click(screen.getByTestId("brief-review-edit"));
    await screen.findByTestId("brief-edit");
    fireEvent.click(screen.getByTestId("brief-edit").parentElement as HTMLElement);
    expect(screen.getByTestId("brief-edit")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("brief-edit-cancel"));
    await waitFor(() => expect(screen.queryByTestId("brief-edit")).not.toBeInTheDocument());
  });

  it("không có AI: chỉ tạo project, rồi Creator tự viết brief", async () => {
    llmEnabled = false;
    renderPage();
    await screen.findByText("Tạo project →");
    typeTopic("Ong sung");
    fireEvent.click(screen.getByTestId("script-step-create"));

    fireEvent.click(await screen.findByTestId("brief-write-own"));
    expect(generate).not.toHaveBeenCalled();
    expect(await screen.findByText("Tự viết · Creative brief")).toBeInTheDocument();
  });

  it("mở lại project đã có: phần Đề tài chỉ đọc, brief nạp từ server", async () => {
    vi.spyOn(apiClient, "getProject").mockImplementation(async (id) => ({ project_id: id, status: "draft" }) as Project);
    serverState = { ...serverState, unit: unitOf("approved", ["question"]), targetDuration: "7-10" };
    renderPage({ reopen: { projectId: "p-old", topic: "Ong sung" } });

    expect(await screen.findByTestId("idea-created-note")).toBeInTheDocument();
    expect(screen.getByTestId("script-step-topic")).toHaveValue("Ong sung");
    expect(screen.getByTestId("script-step-topic")).toBeDisabled();
    expect(screen.queryByTestId("script-step-create")).not.toBeInTheDocument();
    expect(await screen.findByTestId("brief-mine-question")).toBeInTheDocument();
    expect(screen.getByTestId("idea-duration")).toHaveValue("7-10");
    expect(screen.getByTestId("idea-duration")).toBeDisabled();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Brief đã duyệt");
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
