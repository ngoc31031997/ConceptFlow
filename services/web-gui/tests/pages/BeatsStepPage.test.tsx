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
    checks: [
      { ok: false, text: "Tổng 1:48 · mục tiêu 4–5 phút" },
      { ok: true, text: "Hook 18 s · nên ≤ 20 s" },
    ],
    totalSeconds: 108,
    targetLabel: "4–5 phút",
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
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
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

  it("says what the step waits for while the story is not approved", async () => {
    state = baseState({ storyApproved: false, beats: [], checks: [] });
    renderPage();
    expect(await screen.findByTestId("beats-not-open")).toHaveTextContent("Duyệt câu chuyện");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.queryByTestId("beats-generate")).not.toBeInTheDocument();
  });

  it("has the AI write the sheet when there is none", async () => {
    state = baseState({ beats: [], checks: [] });
    const generate = vi.spyOn(apiClient, "generateBeats").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("beats-generate"));
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "", expect.anything()));
  });

  it("adds an empty beat by hand to an empty sheet", async () => {
    state = baseState({ beats: [], checks: [] });
    const add = vi.spyOn(apiClient, "addBeat").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("beats-add-first"));
    await waitFor(() => expect(add).toHaveBeenCalledWith("p1", "", true, ""));
  });

  it("saves the text of a beat from the hand-edit dialog", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("beat-review-b2-edit"));
    fireEvent.change(await screen.findByTestId("beat-edit-b2-field-goal"), { target: { value: "Mục tiêu mới" } });
    await waitFor(() => expect(screen.getByTestId("beat-edit-b2-save")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beat-edit-b2-save"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "beat", key: "b2" }), { goal: "Mục tiêu mới" }),
    );
  });

  it("moves the target length by five seconds and refuses to go below the minimum", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    state = baseState({ beats: [beat("b1", { dur: 18 }), beat("b2", { dur: 5 })] });
    await ready();
    fireEvent.click(screen.getByTestId("beat-dur-up-b1"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ key: "b1" }), { dur: "23" }));
    expect(screen.getByTestId("beat-dur-down-b2")).toBeDisabled();
  });

  it("changes the function of a beat", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("beat-fn-b2"));
    fireEvent.click(await screen.findByTestId("beat-fn-b2-option-Cao trào"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ key: "b2" }), { fn: "Cao trào" }));
  });

  it("attaches and drops facts of a beat", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("beat-fact-drop-b2-F02"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ key: "b2" }), { facts: "" }));
    fireEvent.click(screen.getByTestId("beat-fact-add-b2"));
    fireEvent.click(await screen.findByTestId("beat-fact-add-b2-option-F03"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ key: "b2" }), { facts: "F02, F03" }));
  });

  it("reorders beats and keeps the first up and the last down disabled", async () => {
    const reorder = vi.spyOn(apiClient, "reorderBeats").mockResolvedValue(baseState());
    await ready();
    expect(screen.getByTestId("beat-up-b1")).toBeDisabled();
    expect(screen.getByTestId("beat-down-b3")).toBeDisabled();
    fireEvent.click(screen.getByTestId("beat-down-b1"));
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
    const redo = vi.spyOn(apiClient, "redoUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("beat-review-b2-redo"));
    fireEvent.change(await screen.findByTestId("beat-redo-b2-note"), { target: { value: "ngắn hơn" } });
    await waitFor(() => expect(screen.getByTestId("beat-redo-b2-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beat-redo-b2-send"));
    await waitFor(() =>
      expect(redo).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "beat", key: "b2" }), "ngắn hơn", expect.anything()),
    );
  });

  it("cannot approve a beat without a goal and says so", async () => {
    state = baseState({ beats: [beat("b1", { goal: "" })] });
    await ready();
    expect(screen.getByTestId("beat-review-b1-approve")).toBeDisabled();
    expect(screen.getByTestId("beat-b1")).toHaveTextContent("Viết mục tiêu của beat rồi mới duyệt được");
  });

  it("approves every beat still waiting, then lets the Creator move on", async () => {
    const approve = vi.spyOn(apiClient, "approveUnit").mockImplementation(async (_p, ref) => {
      state = baseState({ status: "approved", beats: state.beats.map((b) => ({ ...b, status: "approved" as const })) });
      void ref;
      return {} as never;
    });
    await ready();
    expect(screen.getByTestId("beats-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("beats-approve-all"));
    await waitFor(() => expect(approve).toHaveBeenCalled());
    await waitFor(() => expect(screen.getByTestId("step-strip")).toHaveTextContent("Beat sheet đã duyệt"));
    expect(approve.mock.calls.map((c) => (c[1] as { key: string }).key)).toEqual(["b1", "b2", "b3"]);
    fireEvent.click(screen.getByTestId("beats-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/coming/6");
  });

  it("rewrites the whole sheet with a note after saying what it keeps", async () => {
    const generate = vi.spyOn(apiClient, "generateBeats").mockResolvedValue(baseState());
    await ready();
    fireEvent.click(screen.getByTestId("beats-rewrite"));
    fireEvent.change(await screen.findByTestId("beats-rewrite-dialog-note"), { target: { value: "gọn hơn" } });
    await waitFor(() => expect(screen.getByTestId("beats-rewrite-dialog-send")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("beats-rewrite-dialog-send"));
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "gọn hơn", expect.anything()));
  });

  it("shows why a change failed", async () => {
    vi.spyOn(apiClient, "reorderBeats").mockRejectedValue(new apiClient.ApiError("thứ tự không hợp lệ"));
    await ready();
    fireEvent.click(screen.getByTestId("beat-down-b1"));
    expect(await screen.findByTestId("beats-error")).toHaveTextContent("thứ tự không hợp lệ");
  });
});
