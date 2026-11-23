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
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

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
  vi.spyOn(apiClient, "listAiRuns").mockResolvedValue([]);
  vi.spyOn(apiClient, "getAiChannelSetting").mockResolvedValue({ defaultChannel: "api", stepChannels: {}, effective: { beats: "api" } });
});

function aiRun(over: Partial<apiClient.AiRun> = {}): apiClient.AiRun {
  return {
    id: "r1",
    project_id: "p1",
    label: "Viết beat",
    method: "POST",
    path: "/v1/projects/p1/beats/b2/redo",
    body: '{"note":""}',
    channel: "api",
    status: "running",
    prompt: "PROMPT BEAT",
    reply: "",
    preview: null,
    error: "",
    created_at: "2026-11-23T00:00:00Z",
    updated_at: "2026-11-23T00:00:00Z",
    ...over,
  };
}

const unwrittenState = () =>
  baseState({ unwritten: 1, beats: [beat("b1", { fn: "Hook" }), beat("b2", { goal: "", content: "" }), beat("b3", { fn: "Payoff", next: "" })] });

/** The step's own request for beat `key`, as the client's describe mode hands it back. */
const described = (path: string, body: unknown) =>
  new apiClient.ExternalRequestDescribed({ gateway: "http://gw", method: "POST", path, body: JSON.stringify(body) });

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
  await screen.findByTestId("beats-sheet-card");
}

describe("BeatsStepPage", () => {
  it("shows the structure strip, its checks and one card per beat", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Dàn ý · beat sheet");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 5/23");
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

  it("puts every action of the whole sheet above the beats, and writes only the beats without content from its button", async () => {
    state = baseState({ unwritten: 1, beats: [beat("b1", { fn: "Hook" }), beat("b2", { goal: "", content: "" }), beat("b3", { fn: "Payoff", next: "" })] });
    const generate = vi.spyOn(apiClient, "generateBeats").mockResolvedValue(baseState());
    await ready();
    const run = screen.getByTestId("beats-generate");
    expect(run).toHaveTextContent("AI viết nội dung các beat chưa có (1)");
    expect(screen.getByTestId("beats-sheet-card")).toContainElement(run);
    expect(screen.getByTestId("beats-sheet-card")).toContainElement(screen.getByTestId("beats-approve-all"));
    expect(run.compareDocumentPosition(screen.getByTestId("beat-b1")) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    fireEvent.click(run);
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "", expect.anything(), true));
  });

  it("offers the run for the missing beats only while some beat has no content", async () => {
    await ready();
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
    expect(screen.getByTestId("beats-rewrite")).toBeInTheDocument();
  });

  it("gives a beat without content its own run on the step's channel, and starts it with the note", async () => {
    state = unwrittenState();
    const start = vi.spyOn(apiClient, "startAiRun").mockResolvedValue(aiRun());
    vi.spyOn(apiClient, "redoBeat").mockRejectedValue(described("/v1/projects/p1/beats/b2/redo", { note: "nhấn vào cái giá" }));
    await ready();
    expect(screen.queryByTestId("beat-write-b1")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("beat-write-b2"));
    fireEvent.change(await screen.findByTestId("beat-write-dialog-b2-note"), { target: { value: "nhấn vào cái giá" } });
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-run"));
    await waitFor(() =>
      expect(start).toHaveBeenCalledWith("authoring", "p1", {
        label: expect.any(String),
        request: expect.objectContaining({ method: "POST", path: "/v1/projects/p1/beats/b2/redo", body: JSON.stringify({ note: "nhấn vào cái giá" }) }),
        channel: "api",
      }),
    );
  });

  it("reads a pasted reply of one beat, shows the beat that would be stored, and applies it only on the Creator's click", async () => {
    state = unwrittenState();
    vi.spyOn(apiClient, "redoBeat").mockRejectedValue(described("/v1/projects/p1/beats/b2/redo", { note: "" }));
    vi.spyOn(apiClient, "startAiRun").mockResolvedValue(aiRun({ channel: "manual", status: "waiting" }));
    const preview = { key: "b2", beat: { fn: "Giải thích cơ chế", title: "Cửa hẹp", goal: "Ong mất cánh", content: "Ong chui qua cửa", facts: ["F02"], dur: 40, next: "Vào trong" } };
    const reply = vi.spyOn(apiClient, "replyAiRun").mockResolvedValue(aiRun({ channel: "manual", status: "ready", reply: "{...}", preview }));
    const apply = vi.spyOn(apiClient, "applyAiRun").mockResolvedValue({ run: aiRun({ channel: "manual", status: "applied" }), state });
    const get = vi.mocked(apiClient.getBeats);
    await ready();
    fireEvent.click(screen.getByTestId("beat-write-b2"));
    fireEvent.click(await screen.findByTestId("beat-write-dialog-b2-run-more"));
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-run-manual"));
    expect(await screen.findByTestId("beat-write-dialog-b2-prompt")).toHaveTextContent("PROMPT BEAT");
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-tab-reply"));
    fireEvent.change(screen.getByTestId("beat-write-dialog-b2-reply"), { target: { value: "{...}" } });
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-read"));
    await waitFor(() => expect(reply).toHaveBeenCalledWith("authoring", "p1", "r1", "{...}"));
    const shown = await screen.findByTestId("ai-run-preview");
    expect(shown).toHaveTextContent("Cửa hẹp");
    expect(shown).toHaveTextContent("Ong mất cánh");
    expect(apply).not.toHaveBeenCalled();
    const loads = get.mock.calls.length;
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-apply"));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("authoring", "p1", "r1"));
    await waitFor(() => expect(get.mock.calls.length).toBeGreaterThan(loads));
  });

  it("reopens an open run of a beat from its pill", async () => {
    state = unwrittenState();
    vi.spyOn(apiClient, "listAiRuns").mockResolvedValue([aiRun({ status: "unreadable", reply: "lỗi", error: "Không đọc được JSON" })]);
    await ready();
    fireEvent.click(await screen.findByTestId("beat-write-run-b2"));
    expect(await screen.findByTestId("beat-write-dialog-b2-error")).toHaveTextContent("Không đọc được JSON");
  });

  it("folds an approved beat to one line and opens it again", async () => {
    state = baseState({ beats: [beat("b1", { fn: "Hook", status: "approved" }), beat("b2"), beat("b3", { fn: "Payoff", next: "" })] });
    await ready();
    expect(screen.queryByTestId("beat-goal-b1")).not.toBeInTheDocument();
    expect(screen.getByTestId("beat-status-b1")).toBeInTheDocument();
    expect(screen.getByTestId("beat-goal-b2")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("beat-expand-b1"));
    expect(await screen.findByTestId("beat-goal-b1")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("beat-collapse-b1"));
    await waitFor(() => expect(screen.queryByTestId("beat-goal-b1")).not.toBeInTheDocument());
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

  it("has the AI write a new beat after another through a run, and shows its pill only on that beat", async () => {
    const body = { after: "b1", blank: false, note: "x" };
    const start = vi.spyOn(apiClient, "startAiRun").mockResolvedValue(aiRun({ path: "/v1/projects/p1/beats", body: JSON.stringify(body) }));
    vi.spyOn(apiClient, "addBeat").mockRejectedValue(described("/v1/projects/p1/beats", body));
    await ready();
    fireEvent.click(screen.getByTestId("beat-add-b1"));
    fireEvent.change(await screen.findByTestId("beat-add-dialog-b1-note"), { target: { value: "x" } });
    fireEvent.click(screen.getByTestId("beat-add-dialog-b1-run"));
    await waitFor(() =>
      expect(start).toHaveBeenCalledWith("authoring", "p1", expect.objectContaining({ request: expect.objectContaining({ path: "/v1/projects/p1/beats" }), channel: "api" })),
    );
    expect(await screen.findByTestId("beat-add-run-b1")).toBeInTheDocument();
    expect(screen.queryByTestId("beat-add-run-b2")).not.toBeInTheDocument();
  });

  it("shows the new beat a run would add before it is stored", async () => {
    vi.spyOn(apiClient, "listAiRuns").mockResolvedValue([
      aiRun({
        path: "/v1/projects/p1/beats",
        body: '{"after":"b1","blank":false,"note":""}',
        status: "ready",
        preview: { after: "b1", beat: { fn: "Cao trào", title: "Mới", goal: "G", content: "C", facts: [], dur: 20, next: "N" } },
      }),
    ]);
    await ready();
    fireEvent.click(await screen.findByTestId("beat-add-run-b1"));
    const shown = await screen.findByTestId("ai-run-preview");
    expect(shown).toHaveTextContent("Beat mới sẽ được thêm sau b1");
    expect(shown).toHaveTextContent("Mới");
  });

  it("adds an empty beat from its own button without asking the AI", async () => {
    const add = vi.spyOn(apiClient, "addBeat").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beat-add-blank-b1"));
    await waitFor(() => expect(add).toHaveBeenCalledWith("p1", "b1", true, ""));
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

  it("has the AI redo one beat with the Creator's note through a run", async () => {
    const start = vi.spyOn(apiClient, "startAiRun").mockResolvedValue(aiRun());
    vi.spyOn(apiClient, "redoBeat").mockRejectedValue(described("/v1/projects/p1/beats/b2/redo", { note: "ngắn hơn" }));
    await ready();
    fireEvent.click(screen.getByTestId("beat-review-b2-redo"));
    fireEvent.change(await screen.findByTestId("beat-write-dialog-b2-note"), { target: { value: "ngắn hơn" } });
    fireEvent.click(screen.getByTestId("beat-write-dialog-b2-run"));
    await waitFor(() =>
      expect(start).toHaveBeenCalledWith(
        "authoring",
        "p1",
        expect.objectContaining({ request: expect.objectContaining({ path: "/v1/projects/p1/beats/b2/redo", body: JSON.stringify({ note: "ngắn hơn" }) }) }),
      ),
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

  describe("with the extension's queue", () => {
    let bridge: FakeAiBridge;
    afterEach(() => bridge.remove());

    const threeUnwritten = () =>
      baseState({ unwritten: 2, beats: [beat("b1", { fn: "Hook" }), beat("b2", { goal: "", content: "" }), beat("b3", { goal: "", content: "", fn: "Payoff", next: "" })] });

    it("disables the extension and manual channels of the run-all button when there is no queue", async () => {
      bridge = installFakeAiBridge();
      state = threeUnwritten();
      await ready();
      fireEvent.click(screen.getByTestId("beats-generate-more"));
      expect(screen.getByTestId("beats-generate-extension")).toBeDisabled();
      expect(screen.getByTestId("beats-generate-manual")).toBeDisabled();
      expect(screen.getByTestId("beats-generate-api")).not.toBeDisabled();
    });

    it("queues one item per beat without content, in the order of the sheet, without calling the AI", async () => {
      bridge = installFakeAiBridge({ queue: true });
      state = threeUnwritten();
      const redo = vi.spyOn(apiClient, "redoBeat");
      await ready();
      const button = await screen.findByTestId("beats-generate");
      expect(button).toHaveTextContent("AI viết nội dung các beat chưa có (2) (API)");
      fireEvent.click(screen.getByTestId("beats-generate-more"));
      await waitFor(() => expect(screen.getByTestId("beats-generate-extension")).not.toBeDisabled());
      fireEvent.click(screen.getByTestId("beats-generate-extension"));
      await waitFor(() => expect(bridge.queued).toHaveLength(2));
      expect(bridge.queued.map((q) => q.item.label)).toEqual(["B2 · Beat b2", "B3 · Beat b3"]);
      const [first] = bridge.queued.map((q) => q.item.request as { method: string; path: string; body: string });
      expect(first.method).toBe("POST");
      expect(first.path).toBe("/v1/projects/p1/beats/b2/redo");
      expect(JSON.parse(first.body)).toEqual({ note: "" });
      expect(bridge.queued[0].item.modes).toEqual({ deepThink: true, search: false });
      expect(await screen.findByTestId("beats-queue-note")).toHaveTextContent("Đã thêm 2 beat vào hàng đợi AI");
      // Describing a request makes no AI call: redoBeat ran only in its "describe" mode.
      redo.mock.calls.forEach((call) => expect(call[3]).toEqual({ external: { mode: "describe" } }));
    });

    it("hides the run button when every beat has content", async () => {
      bridge = installFakeAiBridge({ queue: true });
      await ready();
      await waitFor(() => expect(screen.getByTestId("beats-rewrite")).toBeInTheDocument());
      expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
    });

    it("says how many are waiting, running or failed, and loads the sheet again when a beat is done", async () => {
      bridge = installFakeAiBridge({ queue: true });
      await ready();
      bridge.send({ type: "queue_state", waiting: 2, running: 1, done: 0, failed: 1, paused: false });
      expect(await screen.findByTestId("beats-queue-state")).toHaveTextContent("1 đang chạy · 2 chờ · 1 lỗi");
      const get = vi.mocked(apiClient.getBeats);
      const before = get.mock.calls.length;
      bridge.send({ type: "queue_item", status: "done", item_id: "q1", path: "/v1/projects/p1/beats/b2/redo", project: "p1" });
      await waitFor(() => expect(get.mock.calls.length).toBeGreaterThan(before));
      const afterOther = get.mock.calls.length;
      bridge.send({ type: "queue_item", status: "done", item_id: "q2", path: "/v1/projects/other/beats/b2/redo", project: "other" });
      await new Promise((r) => setTimeout(r, 30));
      expect(get.mock.calls.length).toBe(afterOther);
    });
  });
});
