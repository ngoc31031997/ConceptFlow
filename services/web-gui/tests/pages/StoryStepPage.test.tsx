import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { StoryStepPage } from "../../src/pages/StoryStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Story, StoryFact, StoryState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function arc(): apiClient.ArcBeat[] {
  return [
    { beat: "Hook", text: "Mở từ đĩa sung có xác ong.", facts: [] },
    { beat: "Cơ chế", text: "Sung là chùm hoa kín, ong chui vào và mất cánh.", facts: ["F01", "F02"] },
    { beat: "Payoff", text: "Ficin phân giải xác ong.", facts: ["F03"] },
  ];
}

function story(over: Partial<Story> = {}): Story {
  return {
    centralQuestion: "Vì sao ong chết trong quả sung?",
    coreAnswer: "Ong là cái giá của một quan hệ.",
    narrativeArc: arc(),
    creatorFields: [],
    status: "pending",
    updatedAt: "2026-10-24T10:00:00Z",
    ...over,
  };
}

function storyFact(id: string, over: Partial<StoryFact> = {}): StoryFact {
  return { id, claimVi: `Dữ kiện ${id}`, type: "fact", must: false, cited: true, link: "ok", ...over };
}

function baseState(over: Partial<StoryState> = {}): StoryState {
  const s = "story" in over ? over.story : story();
  return {
    status: s?.status ?? "pending",
    story: s ?? null,
    facts: [storyFact("F01", { must: true }), storyFact("F02"), storyFact("F03", { type: "hypothesis" })],
    brief: { audience: "Học sinh cấp 3", durationSeconds: 360, tone: "Ấm, tò mò", learningObjectives: ["Hiểu quan hệ sung và ong"], avoid: ["Thuật ngữ nặng"] },
    briefApproved: true,
    researchResolved: true,
    researchSkipped: false,
    briefStale: false,
    researchStale: false,
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
    models: [{ id: "m1", label: "Model 1" }],
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
  await screen.findByTestId("story-card");
}

describe("StoryStepPage", () => {
  it("shows the input, the three fields of the story and the approval", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Xây dựng câu chuyện");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 4/21");
    expect(screen.getByTestId("rail-step-4")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("story-input-summary")).toHaveTextContent("Brief đã duyệt · 3 dữ kiện đang giữ");
    expect(screen.getByTestId("story-field-central_question")).toHaveTextContent("Vì sao ong chết trong quả sung?");
    expect(screen.getByTestId("story-field-core_answer")).toHaveTextContent("Ong là cái giá của một quan hệ.");
    const beats = screen.getByTestId("story-arc").querySelectorAll("li");
    expect(beats).toHaveLength(3);
    expect(beats[1]).toHaveTextContent("Cơ chế");
    expect(beats[1]).toHaveTextContent("F01");
    expect(beats[1]).toHaveTextContent("F02");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt câu chuyện");
    expect(screen.queryByTestId("story-stale")).not.toBeInTheDocument();
  });

  it("tags a field the Creator wrote", async () => {
    state = baseState({ story: story({ creatorFields: ["core_answer"] }) });
    await ready();
    expect(screen.getByTestId("story-mine-core_answer")).toHaveTextContent("Bạn viết");
    expect(screen.queryByTestId("story-mine-central_question")).not.toBeInTheDocument();
  });

  it("says the brief or the facts changed since the story was written", async () => {
    state = baseState({ briefStale: true, researchStale: true });
    await ready();
    expect(screen.getByTestId("story-stale")).toHaveTextContent("Brief và bảng dữ kiện đã đổi");
  });

  it("reads the input again from the server each time the dialog opens", async () => {
    await ready();
    const get = vi.mocked(apiClient.getStory);
    const before = get.mock.calls.length;
    fireEvent.click(screen.getByTestId("story-input-open"));
    expect(await screen.findByTestId("story-input-dialog")).toBeInTheDocument();
    await waitFor(() => expect(get.mock.calls.length).toBeGreaterThan(before));
    const brief = await screen.findByTestId("story-input-brief");
    expect(brief).toHaveTextContent("Học sinh cấp 3");
    expect(brief).toHaveTextContent("Hiểu quan hệ sung và ong");
    expect(screen.getByTestId("story-input-facts")).toHaveTextContent("Dữ kiện F02");
  });

  it("says there are no facts when the research step was skipped", async () => {
    state = baseState({ researchSkipped: true, facts: [] });
    await ready();
    expect(screen.getByTestId("story-input-summary")).toHaveTextContent("bước Nghiên cứu đã bỏ qua");
    fireEvent.click(screen.getByTestId("story-input-open"));
    expect(await screen.findByTestId("story-input-no-facts")).toHaveTextContent("không có dữ kiện");
  });

  it("lets the Creator pick the model the story step runs on", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("story-redo-all"));
    expect(await screen.findByTestId("story-redo-all-model-field")).toHaveTextContent("Model AI cho bước Câu chuyện");
  });

  it("says what the step waits for while the brief or the research is not done", async () => {
    state = baseState({ briefApproved: true, researchResolved: false, story: null, brief: null });
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.getByTestId("story-not-open")).toHaveTextContent("Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu");
    expect(screen.queryByTestId("story-generate")).not.toBeInTheDocument();
    expect(screen.queryByTestId("story-write-by-hand")).not.toBeInTheDocument();
  });

  it("lets the AI write the story when none exists yet", async () => {
    state = baseState({ story: null });
    const gen = vi.spyOn(apiClient, "generateStory").mockImplementation(async () => {
      state = baseState();
      return state.story as Story;
    });
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa có câu chuyện");
    expect(screen.getByTestId("story-approve")).toBeDisabled();
    fireEvent.click(screen.getByTestId("story-generate"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.anything()));
    await waitFor(() => expect(screen.getByTestId("story-field-central_question")).toHaveTextContent("Vì sao ong chết trong quả sung?"));
    expect(screen.queryByTestId("story-generate")).not.toBeInTheDocument();
  });

  it("edits the core answer by hand in a dialog and saves only its text", async () => {
    const edit = vi.spyOn(apiClient, "editStory").mockImplementation(async () => {
      state = baseState({ story: story({ coreAnswer: "Đáp án mới", creatorFields: ["core_answer"] }) });
      return state.story as Story;
    });
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-core_answer"));
    const box = await screen.findByTestId("story-edit-answer-dialog-field");
    expect(box).toHaveValue("Ong là cái giá của một quan hệ.");
    fireEvent.change(box, { target: { value: "Đáp án mới" } });
    fireEvent.click(screen.getByTestId("story-edit-answer-dialog-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", { coreAnswer: "Đáp án mới" }));
    await waitFor(() => expect(screen.getByTestId("story-mine-core_answer")).toBeInTheDocument());
  });

  it("edits the arc beat by beat: change, move, add and remove", async () => {
    const edit = vi.spyOn(apiClient, "editStory").mockResolvedValue(story());
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-narrative_arc"));
    await screen.findByTestId("story-arc-edit");
    expect(screen.getByTestId("story-arc-beat-1")).toHaveValue("Cơ chế");
    expect(screen.getByTestId("story-arc-facts-1")).toHaveValue("F01, F02");
    fireEvent.change(screen.getByTestId("story-arc-beat-0"), { target: { value: "Mở đầu" } });
    fireEvent.click(screen.getByTestId("story-arc-down-0"));
    fireEvent.click(screen.getByTestId("story-arc-remove-2"));
    fireEvent.click(screen.getByTestId("story-arc-add"));
    fireEvent.change(screen.getByTestId("story-arc-beat-2"), { target: { value: "Kết" } });
    fireEvent.change(screen.getByTestId("story-arc-text-2"), { target: { value: "Trả lời câu hỏi" } });
    fireEvent.change(screen.getByTestId("story-arc-facts-2"), { target: { value: "f3" } });
    fireEvent.click(screen.getByTestId("story-arc-edit-save"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", {
        narrativeArc: [
          { beat: "Cơ chế", text: "Sung là chùm hoa kín, ong chui vào và mất cánh.", facts: ["F01", "F02"] },
          { beat: "Mở đầu", text: "Mở từ đĩa sung có xác ong.", facts: [] },
          { beat: "Kết", text: "Trả lời câu hỏi", facts: ["F3"] },
        ],
      }),
    );
  });

  it("does not save an arc that was not changed", async () => {
    const edit = vi.spyOn(apiClient, "editStory").mockResolvedValue(story());
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-narrative_arc"));
    await screen.findByTestId("story-arc-edit");
    fireEvent.click(screen.getByTestId("story-arc-edit-save"));
    await waitFor(() => expect(screen.queryByTestId("story-arc-edit")).not.toBeInTheDocument());
    expect(edit).not.toHaveBeenCalled();
  });

  it("shows why a hand edit was refused and keeps the dialog open", async () => {
    vi.spyOn(apiClient, "editStory").mockRejectedValue(new apiClient.ApiError("dữ kiện F09 không có trong bảng dữ kiện"));
    await ready();
    fireEvent.click(screen.getByTestId("story-edit-narrative_arc"));
    await screen.findByTestId("story-arc-edit");
    fireEvent.change(screen.getByTestId("story-arc-facts-0"), { target: { value: "F09" } });
    fireEvent.click(screen.getByTestId("story-arc-edit-save"));
    expect(await screen.findByText(/F09 không có trong bảng dữ kiện/)).toBeInTheDocument();
    expect(screen.getByTestId("story-arc-edit")).toBeInTheDocument();
  });

  it("writes the story by hand when the Creator starts from nothing", async () => {
    state = baseState({ story: null });
    const edit = vi.spyOn(apiClient, "editStory").mockImplementation(async () => {
      state = baseState({ story: story({ centralQuestion: "Câu hỏi của tôi?", creatorFields: ["central_question"] }) });
      return state.story as Story;
    });
    await ready();
    fireEvent.click(screen.getByTestId("story-write-by-hand"));
    fireEvent.change(await screen.findByTestId("story-edit-question-dialog-field"), { target: { value: "Câu hỏi của tôi?" } });
    fireEvent.click(screen.getByTestId("story-edit-question-dialog-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", { centralQuestion: "Câu hỏi của tôi?" }));
    await waitFor(() => expect(screen.getByTestId("story-mine-central_question")).toBeInTheDocument());
  });

  it("has the AI redo the whole story with the Creator's note", async () => {
    const gen = vi.spyOn(apiClient, "generateStory").mockResolvedValue(story());
    await ready();
    fireEvent.click(screen.getByTestId("story-redo-all"));
    fireEvent.change(await screen.findByTestId("story-redo-all-dialog-note"), { target: { value: "ngắn hơn" } });
    await waitFor(() => expect(screen.getByTestId("story-redo-all-dialog-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("story-redo-all-dialog-send"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "ngắn hơn", expect.anything()));
  });

  it("approves the story, then lets the Creator take the approval back and move on", async () => {
    const approve = vi.spyOn(apiClient, "approveStory").mockImplementation(async () => {
      state = baseState({ story: story({ status: "approved" }) });
      return state.story as Story;
    });
    vi.spyOn(apiClient, "unapproveStory").mockImplementation(async () => {
      state = baseState();
      return state.story as Story;
    });
    await ready();
    expect(screen.getByTestId("story-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("story-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1"));
    await waitFor(() => expect(screen.getByTestId("story-unapprove")).toBeInTheDocument());
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Câu chuyện đã duyệt");
    fireEvent.click(screen.getByTestId("story-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/beats");
  });

  it("shows why the approval was refused", async () => {
    vi.spyOn(apiClient, "approveStory").mockRejectedValue(new apiClient.ApiError("còn trống Mạch kể"));
    await ready();
    fireEvent.click(screen.getByTestId("story-approve"));
    expect(await screen.findByTestId("story-error")).toHaveTextContent("còn trống Mạch kể");
  });

  it("shows why an AI run failed", async () => {
    state = baseState({ story: null });
    vi.spyOn(apiClient, "generateStory").mockRejectedValue(new apiClient.ApiError("câu trả lời của AI không dùng được làm câu chuyện"));
    await ready();
    fireEvent.click(screen.getByTestId("story-generate"));
    expect(await screen.findByTestId("story-error")).toHaveTextContent("không dùng được làm câu chuyện");
  });
});
