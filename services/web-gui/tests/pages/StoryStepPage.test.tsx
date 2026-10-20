import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { StoryStepPage } from "../../src/pages/StoryStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { StoryApproach, StoryField, StoryFact, StoryState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

const LABELS: Record<string, string> = {
  question: "Câu hỏi mở đầu",
  promise: "Lời hứa với khán giả",
  chain: "Chuỗi nhân quả",
  payoff: "Kết luận · payoff",
  logline: "Logline",
  summary: "Tóm tắt",
};

function field(key: apiClient.StoryFieldKey, text: string, over: Partial<StoryField> = {}): StoryField {
  return { key, label: LABELS[key], text, facts: [], mine: false, stale: false, ...over };
}

function fullFields(): StoryField[] {
  return [
    field("question", "Vì sao ong chết trong quả sung?"),
    field("promise", "Bạn sẽ biết vì sao."),
    field("chain", "Sung là chùm hoa kín. [F01]\nOng chui vào và mất cánh. [F02, F01]", { facts: ["F01", "F02"] }),
    field("payoff", "Ong là cái giá của một quan hệ. [F02]", { facts: ["F02"] }),
    field("logline", "Một con ong hy sinh vì quả sung.", { mine: true }),
    field("summary", "Mở từ đĩa sung, đi vào lòng quả."),
  ];
}

function approach(key: string, over: Partial<StoryApproach> = {}): StoryApproach {
  return {
    key,
    code: "HÀNH-TRÌNH",
    name: "Hành trình khám phá",
    angle: "đi vào quả để thấy điều không ngờ",
    why: "F01 cho cú lật",
    refs: ["F01"],
    open: "Quả sung bạn vừa ăn...",
    isNew: false,
    archetypeId: "",
    selected: false,
    ...over,
  };
}

function storyFact(id: string, over: Partial<StoryFact> = {}): StoryFact {
  return { id, claimVi: `Dữ kiện ${id}`, type: "fact", must: false, used: true, cited: false, link: "ok", ...over };
}

function baseState(over: Partial<StoryState> = {}): StoryState {
  return {
    status: "pending",
    approachKey: "",
    fields: fullFields(),
    approaches: [approach("A1"), approach("A2", { code: "VÒNG-ĐỜI", name: "Vòng đời khép kín", isNew: true })],
    facts: [
      storyFact("F01", { must: true, cited: true }),
      storyFact("F02", { cited: true }),
      storyFact("F03", { type: "hypothesis", used: false }),
    ],
    briefApproved: true,
    researchResolved: true,
    researchSkipped: false,
    ...over,
  };
}

let state: StoryState;
let project: Project;

beforeEach(() => {
  state = baseState();
  project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 2, run_state: "idle", video_output_mode: "long" } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    reason: "",
    default_model: "m1",
    models: [{ id: "m1", label: "Model 1", code_ok: true }],
  });
  vi.spyOn(apiClient, "getProject").mockImplementation(async () => project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "getStory").mockImplementation(async () => state);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/story"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/story" element={<StoryStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

async function ready() {
  renderPage();
  await screen.findByTestId("story-frame-card");
}

describe("StoryStepPage", () => {
  it("shows the three sections: approaches, the six fields and the facts", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Xây dựng câu chuyện");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 4/21");
    expect(screen.getByTestId("rail-step-4")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("approach-A1")).toHaveTextContent("Hành trình khám phá");
    expect(screen.getByTestId("approach-A2")).toHaveTextContent("AI đề xuất kiểu mới");
    for (const key of ["question", "promise", "chain", "payoff", "logline", "summary"]) {
      expect(screen.getByTestId(`story-field-${key}`)).toBeInTheDocument();
    }
    expect(screen.getByTestId("story-chain").querySelectorAll("li")).toHaveLength(2);
    expect(screen.getByTestId("story-payoff-facts")).toHaveTextContent("F02");
    expect(screen.getByTestId("story-mine-logline")).toHaveTextContent("Bạn viết");
    expect(screen.getByTestId("story-facts-count")).toHaveTextContent("2/3 dữ kiện");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt câu chuyện");
  });

  it("lets the Creator pick the model the story step runs on", async () => {
    await ready();
    expect(await screen.findByTestId("story-model-field")).toHaveTextContent("Model AI cho bước Câu chuyện");
  });

  it("says what the step waits for while the brief or the research is not done", async () => {
    state = baseState({ briefApproved: true, researchResolved: false, fields: fullFields().map((f) => ({ ...f, text: "" })), approaches: [] });
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.getByTestId("story-not-open")).toHaveTextContent("Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu");
    expect(screen.queryByTestId("story-generate")).not.toBeInTheDocument();
    expect(screen.getByTestId("story-approach-generate")).toBeDisabled();
  });

  it("lets the AI write the frame when none exists yet", async () => {
    state = baseState({ fields: fullFields().map((f) => ({ ...f, text: "", facts: [], mine: false })) });
    const gen = vi.spyOn(apiClient, "generateStory").mockImplementation(async () => {
      state = baseState();
      return state;
    });
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa có khung");
    fireEvent.click(screen.getByTestId("story-generate"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.anything()));
    await waitFor(() => expect(screen.getByTestId("story-field-question")).toHaveTextContent("Vì sao ong chết trong quả sung?"));
    expect(screen.queryByTestId("story-generate")).not.toBeInTheDocument();
  });

  it("chooses another approach straight away while the story waits for review", async () => {
    const select = vi.spyOn(apiClient, "selectStoryApproach").mockResolvedValue(undefined);
    const gen = vi.spyOn(apiClient, "generateStory");
    await ready();
    fireEvent.click(screen.getByTestId("approach-A2"));
    await waitFor(() => expect(select).toHaveBeenCalledWith("p1", "A2"));
    expect(screen.queryByTestId("story-approach-dialog")).not.toBeInTheDocument();
    expect(gen).not.toHaveBeenCalled();
  });

  it("asks before changing the approach of an approved story, then has the AI write the fields again", async () => {
    state = baseState({ status: "approved", approachKey: "A1", approaches: [approach("A1", { selected: true }), approach("A2")] });
    const select = vi.spyOn(apiClient, "selectStoryApproach").mockResolvedValue(undefined);
    const gen = vi.spyOn(apiClient, "generateStory").mockResolvedValue(state);
    await ready();
    fireEvent.click(screen.getByTestId("approach-A2"));
    const dialog = await screen.findByTestId("story-approach-dialog");
    expect(dialog).toHaveTextContent("Đổi góc tiếp cận?");
    expect(dialog).toHaveTextContent("ô bạn tự viết giữ nguyên");
    expect(select).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("story-approach-confirm"));
    await waitFor(() => expect(select).toHaveBeenCalledWith("p1", "A2"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.anything()));
  });

  it("leaves the approach alone when the Creator cancels the change", async () => {
    state = baseState({ status: "approved", approachKey: "A1", approaches: [approach("A1", { selected: true }), approach("A2")] });
    const select = vi.spyOn(apiClient, "selectStoryApproach").mockResolvedValue(undefined);
    await ready();
    fireEvent.click(screen.getByTestId("approach-A2"));
    fireEvent.click(await screen.findByTestId("story-approach-dialog-cancel"));
    expect(select).not.toHaveBeenCalled();
  });

  it("saves a new kind the AI proposed into the Kiểu video table", async () => {
    state = baseState({ approachKey: "A2", approaches: [approach("A1"), approach("A2", { isNew: true, selected: true, name: "Vòng đời" })] });
    const save = vi.spyOn(apiClient, "saveApproachAsArchetype").mockImplementation(async () => {
      state = baseState({ approachKey: "A2", approaches: [approach("A1"), approach("A2", { isNew: true, selected: true, archetypeId: "row-1" })] });
      return state.approaches[1];
    });
    await ready();
    fireEvent.click(screen.getByTestId("story-archetype-save"));
    await waitFor(() => expect(save).toHaveBeenCalledWith("p1", "A2"));
    await waitFor(() => expect(screen.getByTestId("story-archetype-saved")).toHaveTextContent("Đã lưu vào bảng Kiểu video"));
  });

  it("opens the Kiểu video table in a new tab", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    await ready();
    fireEvent.click(screen.getByTestId("story-approach-all"));
    expect(open).toHaveBeenCalledWith("/settings/video-archetypes", "_blank", "noopener");
  });

  it("edits one field by hand in a dialog and saves only its text", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockImplementation(async () => {
      state = baseState({ fields: fullFields().map((f) => (f.key === "promise" ? { ...f, text: "Lời hứa mới", mine: true } : f)) });
      return {} as never;
    });
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-promise"));
    const box = await screen.findByTestId("story-edit-field-text");
    expect(box).toHaveValue("Bạn sẽ biết vì sao.");
    await waitFor(() => expect(screen.getByTestId("story-edit-save")).not.toBeDisabled());
    fireEvent.change(box, { target: { value: "Lời hứa mới" } });
    fireEvent.click(screen.getByTestId("story-edit-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "story_field", key: "promise" }), { text: "Lời hứa mới" }));
    await waitFor(() => expect(screen.getByTestId("story-mine-promise")).toBeInTheDocument());
  });

  it("shows why a hand edit was refused and keeps the dialog open", async () => {
    vi.spyOn(apiClient, "editUnit").mockRejectedValue(new apiClient.ApiError("dữ kiện F09 không có trong bảng dữ kiện"));
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-payoff"));
    const box = await screen.findByTestId("story-edit-field-text");
    await waitFor(() => expect(screen.getByTestId("story-edit-save")).not.toBeDisabled());
    fireEvent.change(box, { target: { value: "Kết [F09]" } });
    fireEvent.click(screen.getByTestId("story-edit-save"));
    expect(await screen.findByText(/F09 không có trong bảng dữ kiện/)).toBeInTheDocument();
    expect(screen.getByTestId("story-edit")).toBeInTheDocument();
  });

  it("has the AI redo one field with the Creator's note", async () => {
    const redo = vi.spyOn(apiClient, "redoUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("story-redo-logline"));
    fireEvent.change(await screen.findByTestId("story-redo-note"), { target: { value: "ngắn hơn" } });
    await waitFor(() => expect(screen.getByTestId("story-redo-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("story-redo-send"));
    await waitFor(() =>
      expect(redo).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "story_field", key: "logline" }), "ngắn hơn", expect.anything()),
    );
  });

  it("does not let a fact the script must state be turned off, and asks before dropping a cited one", async () => {
    const use = vi.spyOn(apiClient, "setStoryFactUse").mockResolvedValue(undefined);
    await ready();
    expect(screen.getByTestId("story-fact-use-F01")).toBeDisabled();
    fireEvent.click(screen.getByTestId("story-fact-use-F02"));
    const dialog = await screen.findByTestId("story-fact-dialog");
    expect(dialog).toHaveTextContent("Thôi dùng F02 trong chuyện?");
    expect(use).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("story-fact-confirm"));
    await waitFor(() => expect(use).toHaveBeenCalledWith("p1", "F02", false));
  });

  it("turns on an unused fact without asking", async () => {
    const use = vi.spyOn(apiClient, "setStoryFactUse").mockResolvedValue(undefined);
    await ready();
    expect(screen.getByTestId("story-fact-F03")).toHaveTextContent("Giả thuyết");
    fireEvent.click(screen.getByTestId("story-fact-use-F03"));
    await waitFor(() => expect(use).toHaveBeenCalledWith("p1", "F03", true));
    expect(screen.queryByTestId("story-fact-dialog")).not.toBeInTheDocument();
  });

  it("marks facts without a source when the research step was skipped", async () => {
    state = baseState({ researchSkipped: true, facts: [storyFact("F01", { link: "none", cited: true })] });
    await ready();
    expect(screen.getByTestId("story-fact-F01")).toHaveTextContent("chưa có nguồn");
  });

  it("approves the story, then lets the Creator take the approval back and move on", async () => {
    const approve = vi.spyOn(apiClient, "approveUnit").mockImplementation(async () => {
      state = baseState({ status: "approved" });
      return {} as never;
    });
    vi.spyOn(apiClient, "unapproveUnit").mockImplementation(async () => {
      state = baseState({ status: "pending" });
      return {} as never;
    });
    await ready();
    expect(screen.getByTestId("story-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("story-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "story", key: "main" })));
    await waitFor(() => expect(screen.getByTestId("story-unapprove")).toBeInTheDocument());
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Câu chuyện đã duyệt");
    fireEvent.click(screen.getByTestId("story-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/coming/5");
  });

  it("cannot approve a story with an empty field and says which", async () => {
    state = baseState({ fields: fullFields().map((f) => (f.key === "summary" ? { ...f, text: "" } : f)) });
    await ready();
    expect(screen.getByTestId("story-approve")).toBeDisabled();
    expect(screen.getByTestId("story-missing")).toHaveTextContent("Còn trống: Tóm tắt");
  });

  it("shows why an AI run failed", async () => {
    state = baseState({ fields: fullFields().map((f) => ({ ...f, text: "" })) });
    vi.spyOn(apiClient, "generateStory").mockRejectedValue(new apiClient.ApiError("câu trả lời của AI không dùng được làm khung câu chuyện"));
    await ready();
    fireEvent.click(screen.getByTestId("story-generate"));
    expect(await screen.findByTestId("story-error")).toHaveTextContent("không dùng được làm khung câu chuyện");
  });
});
