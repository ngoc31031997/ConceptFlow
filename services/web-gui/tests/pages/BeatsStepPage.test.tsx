import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { BeatsStepPage } from "../../src/pages/BeatsStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { Beat, BeatFact, BeatState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function beat(key: string, over: Partial<Beat> = {}): Beat {
  return {
    key,
    fn: "Giải thích cơ chế",
    title: `Beat ${key}`,
    goal: `Mục tiêu ${key}`,
    content: `Nội dung ${key}`,
    facts: [],
    dur: 40,
    next: `Nối sang sau ${key}`,
    status: "pending",
    mine: [],
    stale: false,
    onlyMust: [],
    noFact: false,
    step: `Bước ${key}`,
    ...over,
  };
}

function fact(id: string, over: Partial<BeatFact> = {}): BeatFact {
  return { id, claimVi: `Dữ kiện ${id}`, type: "fact", must: false, link: "ok", ...over };
}

function baseState(over: Partial<BeatState> = {}): BeatState {
  return {
    status: "pending",
    storyApproved: true,
    beats: [
      beat("b1", { fn: "Hook", dur: 18, facts: ["F01"], onlyMust: ["F01"] }),
      beat("b2", { facts: ["F02"] }),
      beat("b3", { fn: "Payoff", dur: 50, next: "" }),
    ],
    facts: [fact("F01", { must: true }), fact("F02"), fact("F03", { type: "hypothesis" })],
    input: {
      topic: "Vì sao trong quả sung có xác ong?",
      brief: "Khán giả: Người tò mò\nGiọng điệu: Gần gũi",
      briefApproved: true,
      story: "question (Câu hỏi trung tâm):\nVì sao ong chết trong quả sung?",
    },
    checks: [
      { ok: false, text: "Tổng 1:48 · mục tiêu 4–5 phút" },
      { ok: true, text: "Hook 18 s · nên ≤ 20 s" },
    ],
    totalSeconds: 108,
    targetLabel: "4–5 phút",
    phase: "content",
    unwritten: 0,
    rebalanced: 0,
    scriptApproved: 0,
    ...over,
  };
}

let state: BeatState;
let project: Project;

beforeEach(() => {
  state = baseState();
  project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 2, run_state: "idle", video_output_mode: "long" } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive", reason: "", models: [{ id: "m1", label: "Model 1" }] });
  vi.spyOn(apiClient, "getProject").mockImplementation(async () => project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "getBeats").mockImplementation(async () => state);
  vi.spyOn(apiClient, "getBeatImpact").mockResolvedValue([]);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/beats"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/beats" element={<BeatsStepPage />} />
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
  await screen.findByTestId("beats-structure-card");
}

describe("BeatsStepPage", () => {
  it("shows the structure strip, its checks and one card per beat", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Dàn ý · beat sheet");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 5/21");
    expect(screen.getByTestId("rail-step-5")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("beats-timeline").children).toHaveLength(3);
    expect(screen.getByTestId("beats-checks").querySelectorAll("[data-ok='false']")).toHaveLength(1);
    expect(screen.getByTestId("beat-b2")).toHaveTextContent("B2 · Beat b2");
    expect(screen.getByTestId("beat-dur-b1")).toHaveTextContent("18 s");
    expect(screen.getByTestId("beat-facts-b1")).toHaveTextContent("F01");
    expect(screen.getByTestId("beat-goal-b3")).toHaveTextContent("Mục tiêu b3");
    expect(screen.queryByTestId("beat-next-b3")).not.toBeInTheDocument();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt · 0/3 beat");
  });

  it("shows the input the AI reads, read again from the server, before any call", async () => {
    await ready();
    expect(screen.getByTestId("beats-input-summary")).toHaveTextContent("Brief đã duyệt · 3 dữ kiện đang giữ");
    const loads = vi.mocked(apiClient.getBeats).mock.calls.length;
    fireEvent.click(screen.getByTestId("beats-input-open"));
    const dialog = await screen.findByTestId("beats-input-dialog");
    await waitFor(() => expect(vi.mocked(apiClient.getBeats).mock.calls.length).toBe(loads + 1));
    await waitFor(() => expect(screen.queryByTestId("beats-input-loading")).not.toBeInTheDocument());
    expect(dialog).toHaveTextContent("Vì sao trong quả sung có xác ong?");
    expect(screen.getByTestId("beats-input-brief")).toHaveTextContent("Khán giả: Người tò mò");
    expect(screen.getByTestId("beats-input-facts")).toHaveTextContent("Dữ kiện F01 (bắt buộc nói)");
    expect(screen.getByTestId("beats-input-story")).toHaveTextContent("Vì sao ong chết trong quả sung?");
    fireEvent.click(screen.getByTestId("beats-input-close"));
    await waitFor(() => expect(screen.queryByTestId("beats-input-dialog")).not.toBeInTheDocument());
  });

  it("says the brief is not approved when the input has none", async () => {
    state = baseState({ input: { topic: "Đề tài", brief: "", briefApproved: false, story: "question (Câu hỏi trung tâm):\nQ" } });
    await ready();
    fireEvent.click(screen.getByTestId("beats-input-open"));
    expect(await screen.findByTestId("beats-input-no-brief")).toBeInTheDocument();
  });

  it("says what the step waits for while the story is not approved", async () => {
    state = baseState({ storyApproved: false, beats: [], checks: [], phase: "empty" });
    renderPage();
    expect(await screen.findByTestId("beats-not-open")).toHaveTextContent("Duyệt câu chuyện");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
  });

  it("builds the skeleton from the story when there is no sheet, without the AI", async () => {
    state = baseState({ beats: [], checks: [], phase: "empty" });
    const build = vi.spyOn(apiClient, "buildBeatSkeleton").mockResolvedValue(baseState({ phase: "skeleton" }));
    renderPage();
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
    fireEvent.click(await screen.findByTestId("beats-build-skeleton"));
    await waitFor(() => expect(build).toHaveBeenCalledWith("p1"));
  });

  it("shows only the function, the story line and the facts while the skeleton is reviewed", async () => {
    state = baseState({
      phase: "skeleton",
      unwritten: 3,
      beats: [beat("b1", { fn: "Hook", goal: "", content: "" }), beat("b2", { goal: "", content: "", noFact: true }), beat("b3", { fn: "Payoff", goal: "", content: "", next: "" })],
    });
    await ready();
    expect(screen.getByTestId("beat-step-b2")).toHaveTextContent("Bước b2");
    expect(screen.getByTestId("beat-fn-b2")).toBeInTheDocument();
    expect(screen.getByTestId("beat-nofact-mark-b2")).toBeInTheDocument();
    expect(screen.queryByTestId("beat-dur-b2")).not.toBeInTheDocument();
    expect(screen.queryByTestId("beat-goal-b2")).not.toBeInTheDocument();
    expect(screen.queryByTestId("beat-review-b2")).not.toBeInTheDocument();
    expect(screen.queryByTestId("beats-timeline")).not.toBeInTheDocument();
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt bộ khung");
    expect(screen.getByTestId("beat-add-b1")).toHaveTextContent("+ Thêm beat trống sau");
  });

  it("approves the skeleton, and asks first before building it again", async () => {
    state = baseState({ phase: "skeleton", unwritten: 3 });
    const approve = vi.spyOn(apiClient, "approveBeatSkeleton").mockResolvedValue(baseState());
    const build = vi.spyOn(apiClient, "buildBeatSkeleton").mockResolvedValue(baseState({ phase: "skeleton" }));
    await ready();
    fireEvent.click(screen.getByTestId("beats-rebuild-skeleton"));
    expect(build).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByTestId("beats-rebuild-confirm"));
    await waitFor(() => expect(build).toHaveBeenCalledWith("p1"));
    fireEvent.click(screen.getByTestId("beats-approve-skeleton"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1"));
  });

  it("puts every action that runs after the beats, and writes only the beats without content from the bottom button", async () => {
    state = baseState({ unwritten: 1, beats: [beat("b1", { fn: "Hook" }), beat("b2", { goal: "", content: "" }), beat("b3", { fn: "Payoff", next: "" })] });
    const generate = vi.spyOn(apiClient, "generateBeats").mockResolvedValue(baseState());
    await ready();
    const run = screen.getByTestId("beats-generate");
    expect(run).toHaveTextContent("AI viết nội dung các beat chưa có (1)");
    expect(screen.getByTestId("beat-b3").compareDocumentPosition(run) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByTestId("beat-b3").compareDocumentPosition(screen.getByTestId("beats-rewrite")) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    fireEvent.click(run);
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "", expect.anything(), true));
  });

  it("offers the bottom run only while some beat has no content", async () => {
    await ready();
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
    expect(screen.getByTestId("beats-rewrite")).toBeInTheDocument();
  });

  it("gives a beat without content its own run, and a written beat none", async () => {
    state = baseState({ unwritten: 1, beats: [beat("b1", { fn: "Hook" }), beat("b2", { goal: "", content: "" }), beat("b3", { fn: "Payoff", next: "" })] });
    const redo = vi.spyOn(apiClient, "redoBeat").mockResolvedValue(baseState());
    await ready();
    expect(screen.queryByTestId("beat-write-b1")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("beat-write-b2"));
    await waitFor(() => expect(redo).toHaveBeenCalledWith("p1", "b2", "", expect.anything()));
  });

  it("says how many beats were resized to fit the target", async () => {
    state = baseState({ rebalanced: 2 });
    await ready();
    expect(screen.getByTestId("beats-rebalanced")).toHaveTextContent("Đã cân lại thời lượng của 2 beat");
  });

  it("marks a beat that cites no fact with a warning and leaves the others plain", async () => {
    state = baseState({
      beats: [beat("b1", { fn: "Hook", facts: [] }), beat("b2", { facts: ["F02"] }), beat("b3", { facts: [], noFact: true, fn: "Payoff", next: "" })],
    });
    await ready();
    expect(screen.getByTestId("beat-nofact-mark-b3")).toHaveAccessibleName("Beat chưa có dữ kiện");
    expect(screen.getByTestId("beat-nofact-b3")).toHaveTextContent("Chưa gắn dữ kiện");
    expect(screen.queryByTestId("beat-nofact-mark-b1")).not.toBeInTheDocument();
    expect(screen.queryByTestId("beat-nofact-mark-b2")).not.toBeInTheDocument();
    expect(screen.getByTestId("beat-facts-b1")).toHaveTextContent("chưa gắn dữ kiện");
  });

  it("adds an empty beat by hand to an empty sheet", async () => {
    state = baseState({ beats: [], checks: [], phase: "empty" });
    const add = vi.spyOn(apiClient, "addBeat").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("beats-add-first"));
    await waitFor(() => expect(add).toHaveBeenCalledWith("p1", "", true, ""));
  });

  it("saves the text of a beat from the hand-edit dialog", async () => {
    const edit = vi.spyOn(apiClient, "editBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-review-b2-edit"));
    fireEvent.change(await screen.findByTestId("beat-edit-b2-field-goal"), { target: { value: "Mục tiêu mới" } });
    await waitFor(() => expect(screen.getByTestId("beat-edit-b2-save")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beat-edit-b2-save"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", "b2", { goal: "Mục tiêu mới" }),
    );
  });

  it("moves the target length by five seconds and refuses to go below the minimum", async () => {
    const edit = vi.spyOn(apiClient, "editBeat").mockResolvedValue(baseState());
    state = baseState({ beats: [beat("b1", { dur: 18 }), beat("b2", { dur: 5 })] });
    await ready();
    fireEvent.click(screen.getByTestId("beat-dur-up-b1"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "b1", { dur: "23" }));
    expect(screen.getByTestId("beat-dur-down-b2")).toBeDisabled();
  });

  it("changes the function of a beat", async () => {
    const edit = vi.spyOn(apiClient, "editBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-fn-b2"));
    fireEvent.click(await screen.findByTestId("beat-fn-b2-option-Cao trào"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "b2", { fn: "Cao trào" }));
  });

  it("attaches and drops facts of a beat", async () => {
    const edit = vi.spyOn(apiClient, "editBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-fact-drop-b2-F02"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "b2", { facts: "" }));
    fireEvent.click(screen.getByTestId("beat-fact-add-b2"));
    fireEvent.click(await screen.findByTestId("beat-fact-add-b2-option-F03"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "b2", { facts: "F02, F03" }));
  });

  it("reorders beats and keeps the first up and the last down disabled", async () => {
    const reorder = vi.spyOn(apiClient, "reorderBeats").mockResolvedValue(baseState());
    await ready();
    expect(screen.getByTestId("beat-up-b1")).toBeDisabled();
    expect(screen.getByTestId("beat-down-b3")).toBeDisabled();
    fireEvent.click(screen.getByTestId("beat-down-b1"));
    await waitFor(() => expect(reorder).toHaveBeenCalledWith("p1", ["b2", "b1", "b3"]));
  });

  it("asks before moving a beat once dialogue is approved, and says how many beats it touches", async () => {
    state = baseState({ scriptApproved: 2 });
    const reorder = vi.spyOn(apiClient, "reorderBeats").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-down-b1"));
    expect(screen.getByTestId("beat-move-dialog-b1")).toHaveTextContent("2 beat đã có lời thoại được duyệt");
    expect(reorder).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("beat-move-confirm-b1"));
    await waitFor(() => expect(reorder).toHaveBeenCalledWith("p1", ["b2", "b1", "b3"]));
  });

  it("has the AI write a new beat after another", async () => {
    const add = vi.spyOn(apiClient, "addBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-add-b1"));
    await waitFor(() => expect(add).toHaveBeenCalledWith("p1", "b1", false, "", expect.anything()));
  });

  it("asks before deleting a beat and names the fact the script must state that only it holds", async () => {
    const remove = vi.spyOn(apiClient, "deleteBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-delete-b1"));
    const list = await screen.findByTestId("beat-delete-dialog-b1-list");
    expect(list).toHaveTextContent("F01 bắt buộc phải nói nhưng chỉ có trong beat này");
    expect(remove).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("beat-delete-confirm-b1"));
    await waitFor(() => expect(remove).toHaveBeenCalledWith("p1", "b1"));
  });

  it("does not mention a must fact when another beat holds it", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("beat-delete-b2"));
    expect(await screen.findByTestId("beat-delete-dialog-b2-list")).not.toHaveTextContent("bắt buộc");
  });

  it("has the AI redo one beat with the Creator's note", async () => {
    const redo = vi.spyOn(apiClient, "redoBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-review-b2-redo"));
    fireEvent.change(await screen.findByTestId("beat-redo-b2-note"), { target: { value: "ngắn hơn" } });
    await waitFor(() => expect(screen.getByTestId("beat-redo-b2-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beat-redo-b2-send"));
    await waitFor(() =>
      expect(redo).toHaveBeenCalledWith("p1", "b2", "ngắn hơn", expect.anything()),
    );
  });

  it("cannot approve a beat without a goal and says so", async () => {
    state = baseState({ beats: [beat("b1", { goal: "" })] });
    await ready();
    expect(screen.getByTestId("beat-review-b1-approve")).toBeDisabled();
    expect(screen.getByTestId("beat-b1")).toHaveTextContent("Viết mục tiêu của beat rồi mới duyệt được");
  });

  it("approves every beat still waiting, then lets the Creator move on", async () => {
    const approve = vi.spyOn(apiClient, "approveBeat").mockImplementation(async () => {
      state = baseState({ status: "approved", beats: state.beats.map((b) => ({ ...b, status: "approved" as const })) });
      return state;
    });
    await ready();
    expect(screen.getByTestId("beats-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("beats-approve-all"));
    await waitFor(() => expect(approve).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByTestId("step-strip")).toHaveTextContent("Beat sheet đã duyệt"));
    expect(approve.mock.calls.map((c) => c[1])).toEqual(["b1", "b2", "b3"]);
    fireEvent.click(screen.getByTestId("beats-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/script");
  });

  it("rewrites the whole sheet with a note after saying what it keeps", async () => {
    const generate = vi.spyOn(apiClient, "generateBeats").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beats-rewrite"));
    fireEvent.change(await screen.findByTestId("beats-rewrite-dialog-note"), { target: { value: "gọn hơn" } });
    await waitFor(() => expect(screen.getByTestId("beats-rewrite-dialog-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beats-rewrite-dialog-send"));
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "gọn hơn", expect.anything(), false));
  });

  it("shows why a change failed", async () => {
    vi.spyOn(apiClient, "reorderBeats").mockRejectedValue(new apiClient.ApiError("thứ tự không hợp lệ"));
    await ready();
    fireEvent.click(screen.getByTestId("beat-down-b1"));
    expect(await screen.findByTestId("beats-error")).toHaveTextContent("thứ tự không hợp lệ");
  });
});
