import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { VisualPlanStepPage } from "../../src/pages/VisualPlanStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { VisualPlanBeat, VisualPlanState, VisualPlanUnit } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  const l = useLocation();
  return <span data-testid="where">{l.pathname}</span>;
}

function unit(id: string, n: number, beat: string, over: Partial<VisualPlanUnit> = {}): VisualPlanUnit {
  return {
    unit_id: id,
    script_refs: [`${beat}.1`, `${beat}.2`],
    narration_summary: "Tóm tắt",
    communication_goal: `Mục tiêu ${id}`,
    essential_visual_info: [{ item: "Hố đen", priority: "must" }],
    on_screen_text: ["Người rơi"],
    visual_approach: { method: "diagram", description: `Mô tả ${id}`, concept_rule_refs: ["color_system.palette.ink"] },
    narration_relationship: "contrast",
    risk_notes: [],
    open_questions: [],
    duration: { planned_s: 0, extra_reason: "" },
    status: "pending",
    beat_key: beat,
    number: n,
    lines: [
      { id: `${beat}.1`, text: "Câu một.", facts: [] },
      { id: `${beat}.2`, text: "Câu hai.", facts: [] },
    ],
    fact_refs: [],
    narration_min_s: 11,
    effective_planned_s: 11,
    problems: [],
    creator_fields: [],
    ...over,
  };
}

function beat(key: string, number: number, units: VisualPlanUnit[], over: Partial<VisualPlanBeat> = {}): VisualPlanBeat {
  const planned = units.reduce((s, u) => s + u.effective_planned_s, 0);
  return {
    key,
    number,
    fn: "Hook",
    title: `Beat ${number}`,
    dur: 20,
    lines_count: 4,
    written: units.length > 0,
    stale: false,
    stale_sources: [],
    timing: units.length > 0 ? { narration_min_s: planned, planned_s: planned, beat_sheet_s: planned - 3, delta_s: 3 } : null,
    units,
    ...over,
  };
}

function baseState(over: Partial<VisualPlanState> = {}): VisualPlanState {
  const vp1 = unit("VP-001", 1, "b1", { status: "approved" });
  const vp2 = unit("VP-002", 2, "b2", {
    risk_notes: [{ fact_id: "F08", note: "Không dừng hẳn" }],
    open_questions: [{ id: "Q1", question: "Chia đôi hay cắt qua lại?", answer: "" }],
    problems: ["Câu hỏi Q1 chưa trả lời"],
    duration: { planned_s: 14, extra_reason: "Để khán giả kịp so sánh" },
    effective_planned_s: 14,
    fact_refs: ["F08"],
  });
  const vp3 = unit("VP-003", 3, "b2");
  return {
    input: { script_version: 2, concept_version: 1, facts_count: 24, terms_count: 3, rule_refs_count: 9, seconds_per_word: 0.4 },
    beats: [beat("b1", 1, [vp1]), beat("b2", 2, [vp2, vp3]), beat("b3", 3, [])],
    units_total: 3,
    units_approved: 1,
    coverage: { lines_total: 6, lines_covered: 4, uncovered: ["b3.1", "b3.2"], duplicated: [] },
    timing: { beats: {}, planned_s: 36, brief_s: 30, over_brief: true },
    checks: [
      { id: "m-plan-risk", label: "Dữ kiện giả thuyết có ghi chú rủi ro", ok: false, detail: "VP-005 thiếu F14", accepted: false, why: "" },
      { id: "m-plan-avoid", label: "Chữ trên hình không dùng cách nói cấm", ok: true, detail: "", accepted: false, why: "" },
    ],
    review: {
      reviewed: true,
      rounds: 1,
      notes: ["Mạch hình liền"],
      open_count: 2,
      findings: [
        { id: "f1", group: "clarity", group_label: "Rõ ràng", unit_key: "VP-003", field: "communication_goal", text: "Mục tiêu mơ hồ", detail: "d", fix: "Nêu rõ điều cần thấy", state: "open", why: "" },
        { id: "f2", group: "accuracy", group_label: "Độ chính xác khoa học", unit_key: "VP-002", field: "", text: "Cách nói dễ hiểu nhầm", detail: "", fix: "", state: "open", why: "" },
        { id: "f3", group: "clarity", group_label: "Rõ ràng", unit_key: "VP-001", field: "", text: "Quá nhiều chữ", detail: "", fix: "", state: "fixed", why: "" },
      ],
    },
    blockers: ["Còn 2 câu chưa phủ"],
    can_lock: false,
    versions: [],
    locked_version: 0,
    next_version: 1,
    draft: [],
    auto_fixes: [],
    model: "",
    default_model: "ollama",
    models: [{ id: "ollama", label: "Ollama", local: true }],
    ...over,
  };
}

let state: VisualPlanState;

beforeEach(() => {
  state = baseState();
  const project = {
    project_id: "p1",
    status: "draft",
    voice_language: "vi",
    scenes: [],
    flow_step: 3,
    run_state: "idle",
    video_output_mode: "long",
  } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive", reason: "", models: [] });
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getResearch").mockRejectedValue(new Error("no"));
  vi.spyOn(apiClient, "getVisualPlan").mockImplementation(async () => state);
  Element.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/visual-plan"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/visual-plan" element={<VisualPlanStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("VisualPlanStepPage", () => {
  it("says to lock the visual concept first when step 8 is not locked", async () => {
    vi.spyOn(apiClient, "getVisualPlan").mockRejectedValue(
      new apiClient.ApiError("Chốt Ngôn ngữ hình ảnh ở bước 8 trước", "concept_not_locked", undefined, 409),
    );
    renderPage();
    expect(await screen.findByTestId("plan-not-open")).toHaveTextContent("Chốt Ngôn ngữ hình ảnh ở bước 8 trước");
    fireEvent.click(screen.getByTestId("plan-go-concept"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/visual-concept");
  });

  it("says to lock the script first when step 7 is not locked", async () => {
    vi.spyOn(apiClient, "getVisualPlan").mockRejectedValue(
      new apiClient.ApiError("Chốt kịch bản ở bước 7 trước", "script_not_locked", undefined, 409),
    );
    renderPage();
    expect(await screen.findByTestId("plan-not-open")).toHaveTextContent("Chốt kịch bản ở bước 7 trước");
    fireEvent.click(screen.getByTestId("plan-go-lock"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/lock");
  });

  it("is step 9 of the visual phase: inputs, coverage with filters, and units grouped by beat", async () => {
    renderPage();
    expect(await screen.findByTestId("plan-head-card")).toBeInTheDocument();
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Hình ảnh · Bước 9/23");
    expect(screen.getByTestId("rail-step-9")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("plan-input")).toHaveTextContent("Kịch bản v2 · Ngôn ngữ hình ảnh v1");
    expect(screen.getByTestId("plan-head-count")).toHaveTextContent("1/3 unit đã duyệt · 2 vấn đề cần xử lý");
    expect(screen.getByTestId("plan-coverage-count")).toHaveTextContent("4 / 6 câu");
    expect(screen.getByTestId("plan-uncovered")).toHaveTextContent("b3.1, b3.2");
    expect(screen.getByTestId("plan-beat-b2-timing")).toHaveTextContent("dự kiến 25s · beat-sheet 22s · lệch +3s · ⚠ 1 rủi ro");
    expect(screen.getByTestId("plan-unit-VP-002")).toHaveTextContent("b2.1–b2.2");
    expect(screen.getByTestId("plan-beat-b3-empty")).toBeInTheDocument();
  });

  it("shows the least the lines need next to the planned time and why it is longer", async () => {
    renderPage();
    const dur = await screen.findByTestId("plan-unit-VP-002-duration");
    expect(dur).toHaveTextContent("Tối thiểu (theo lời)11s");
    expect(dur).toHaveTextContent("14s · dài hơn lời 3s");
    expect(dur).toHaveTextContent("Để khán giả kịp so sánh");
    expect(screen.getByTestId("plan-unit-VP-003-duration")).toHaveTextContent("bằng mức tối thiểu");
  });

  it("filters the units and goes to the first uncovered line", async () => {
    renderPage();
    await screen.findByTestId("plan-beats");
    expect(screen.getByTestId("plan-filter-all")).toHaveTextContent("Tất cả · 3");
    expect(screen.getByTestId("plan-filter-question")).toHaveTextContent("Có câu hỏi · 1");
    fireEvent.click(screen.getByTestId("plan-filter-question"));
    expect(screen.getByTestId("plan-unit-VP-002")).toBeInTheDocument();
    expect(screen.queryByTestId("plan-unit-VP-001")).not.toBeInTheDocument();
    expect(screen.queryByTestId("plan-unit-VP-003")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("plan-filter-approved"));
    expect(screen.getByTestId("plan-unit-VP-001")).toBeInTheDocument();
    expect(screen.queryByTestId("plan-unit-VP-002")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("plan-go-uncovered"));
    await waitFor(() => expect(Element.prototype.scrollIntoView).toHaveBeenCalled());
    expect(screen.getByTestId("plan-go-uncovered")).toHaveTextContent("b3.1");
  });

  it("cannot approve a unit with an open question and says why, and approves one that is ready", async () => {
    const approve = vi.spyOn(apiClient, "approveVisualPlanUnit").mockResolvedValue(baseState({ units_approved: 2 }));
    renderPage();
    const blocked = await screen.findByTestId("plan-unit-VP-002-approve");
    expect(blocked).toBeDisabled();
    expect(blocked).toHaveAttribute("title", expect.stringContaining("Câu hỏi Q1 chưa trả lời"));
    expect(screen.getByTestId("plan-unit-VP-002-problems")).toHaveTextContent("Câu hỏi Q1 chưa trả lời");
    fireEvent.click(screen.getByTestId("plan-unit-VP-003-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", "VP-003"));
  });

  it("does not approve a unit of a stale beat", async () => {
    state = baseState({
      beats: [beat("b1", 1, [unit("VP-001", 1, "b1")], { stale: true, stale_sources: ["kịch bản"] })],
    });
    renderPage();
    expect(await screen.findByTestId("plan-beat-b1-stale")).toHaveTextContent("kịch bản đã đổi");
    expect(screen.getByTestId("plan-unit-VP-001-approve")).toBeDisabled();
  });

  it("answers an open question through a dialog", async () => {
    const answer = vi.spyOn(apiClient, "answerVisualPlanQuestion").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-002-answer-Q1"));
    const dialog = await screen.findByTestId("plan-answer-dialog");
    expect(within(dialog).getByTestId("plan-answer-dialog-confirm")).toBeDisabled();
    fireEvent.change(within(dialog).getByTestId("plan-answer-text"), { target: { value: "Chia đôi" } });
    fireEvent.click(within(dialog).getByTestId("plan-answer-dialog-confirm"));
    await waitFor(() => expect(answer).toHaveBeenCalledWith("p1", "VP-002", "Q1", "Chia đôi"));
  });

  it("edits a unit and sends only the fields that changed", async () => {
    const edit = vi.spyOn(apiClient, "editVisualPlanUnit").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-003-edit"));
    const dialog = await screen.findByTestId("plan-edit-dialog");
    fireEvent.change(within(dialog).getByTestId("plan-edit-goal"), { target: { value: "Mục tiêu mới" } });
    fireEvent.click(within(dialog).getByTestId("plan-edit-dialog-confirm"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "VP-003", { communication_goal: "Mục tiêu mới" }));
  });

  it("asks for a reason when the planned time is longer than the narration, and blocks saving without one", async () => {
    const edit = vi.spyOn(apiClient, "editVisualPlanUnit").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-003-edit"));
    const dialog = await screen.findByTestId("plan-edit-dialog");
    expect(within(dialog).getByTestId("plan-edit-min")).toHaveTextContent("11s");
    expect(within(dialog).queryByTestId("plan-edit-reason")).not.toBeInTheDocument();
    fireEvent.change(within(dialog).getByTestId("plan-edit-planned"), { target: { value: "15" } });
    expect(within(dialog).getByTestId("plan-edit-dialog-confirm")).toBeDisabled();
    fireEvent.change(within(dialog).getByTestId("plan-edit-reason"), { target: { value: "Nhịp lắng" } });
    fireEvent.click(within(dialog).getByTestId("plan-edit-dialog-confirm"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", "VP-003", { duration: { planned_s: 15, extra_reason: "Nhịp lắng" } }),
    );
  });

  it("shows why a hand edit was refused and keeps the dialog open", async () => {
    vi.spyOn(apiClient, "editVisualPlanUnit").mockRejectedValue(new apiClient.ApiError("Mục tiêu không được rỗng", "bad", undefined, 400));
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-003-edit"));
    const dialog = await screen.findByTestId("plan-edit-dialog");
    fireEvent.change(within(dialog).getByTestId("plan-edit-goal"), { target: { value: " " } });
    fireEvent.click(within(dialog).getByTestId("plan-edit-dialog-confirm"));
    expect(await within(dialog).findByTestId("plan-edit-dialog-error")).toHaveTextContent("Mục tiêu không được rỗng");
  });

  it("splits a unit before a line and merges a unit with the next one", async () => {
    const split = vi.spyOn(apiClient, "splitVisualPlanUnit").mockResolvedValue(baseState());
    const merge = vi.spyOn(apiClient, "mergeVisualPlanUnit").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-002-split"));
    const dialog = await screen.findByTestId("plan-split-dialog");
    expect(within(dialog).getByTestId("plan-split-at")).toHaveValue("b2.2");
    fireEvent.click(within(dialog).getByTestId("plan-split-dialog-confirm"));
    await waitFor(() => expect(split).toHaveBeenCalledWith("p1", "VP-002", "b2.2"));
    expect(screen.getByTestId("plan-unit-VP-003-split")).toBeInTheDocument();
    // Only a unit with a unit after it in its beat can merge.
    expect(screen.queryByTestId("plan-unit-VP-003-merge")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("plan-unit-VP-002-merge"));
    fireEvent.click(await screen.findByTestId("plan-merge-dialog-confirm"));
    await waitFor(() => expect(merge).toHaveBeenCalledWith("p1", "VP-002", "VP-003"));
  });

  it("plans one beat with a note on the system's AI", async () => {
    const gen = vi.spyOn(apiClient, "generateVisualPlanBeat").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-beat-b3-generate"));
    expect(screen.getByTestId("plan-beat-b3-generate")).toHaveTextContent("Lập kế hoạch beat này");
    expect(screen.getByTestId("plan-beat-b2-generate")).toHaveTextContent("Lập lại beat này");
    const dialog = await screen.findByTestId("plan-beat-b3-dialog");
    fireEvent.change(within(dialog).getByTestId("plan-beat-b3-dialog-note"), { target: { value: "ít chữ" } });
    fireEvent.click(within(dialog).getByTestId("plan-beat-b3-dialog-send"));
    await waitFor(() =>
      expect(gen).toHaveBeenCalledWith("p1", "b3", "ít chữ", expect.objectContaining({ signal: expect.anything() })),
    );
  });

  it("redoes a unit's beat with a note from the unit", async () => {
    const gen = vi.spyOn(apiClient, "generateVisualPlanBeat").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-unit-VP-003-redo"));
    const dialog = await screen.findByTestId("plan-unit-VP-003-redo-dialog");
    fireEvent.click(within(dialog).getByTestId("plan-unit-VP-003-redo-dialog-send"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "b2", "", expect.anything()));
  });

  it("plans every empty beat in one run", async () => {
    const gen = vi.spyOn(apiClient, "generateVisualPlan").mockResolvedValue(baseState());
    renderPage();
    const open = await screen.findByTestId("plan-generate");
    expect(open).toHaveTextContent("AI lập 1 beat còn trống");
    fireEvent.click(open);
    fireEvent.click(await screen.findByTestId("plan-generate-confirm"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.objectContaining({ signal: expect.anything() })));
  });

  it("approves every unit that is ready", async () => {
    const all = vi.spyOn(apiClient, "approveAllVisualPlan").mockResolvedValue(baseState());
    renderPage();
    const button = await screen.findByTestId("plan-approve-all");
    expect(button).toHaveTextContent("Duyệt tất cả còn lại · 1");
    fireEvent.click(button);
    await waitFor(() => expect(all).toHaveBeenCalledWith("p1"));
  });

  it("lists the findings on the review tab and acts on each", async () => {
    const apply = vi.spyOn(apiClient, "applyVisualPlanFinding").mockResolvedValue(baseState());
    const accept = vi.spyOn(apiClient, "acceptVisualPlanFinding").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    expect(screen.getByTestId("plan-tab-review")).toHaveTextContent("2 vấn đề");
    expect(screen.getByTestId("plan-coverage-row")).toHaveTextContent("2 câu chưa thuộc unit nào");
    expect(screen.getByTestId("plan-check-row-m-plan-risk")).toHaveTextContent("VP-005 thiếu F14");
    expect(screen.queryByTestId("plan-check-row-m-plan-avoid")).not.toBeInTheDocument();
    // Only a finding with a fix on a field can apply it.
    expect(screen.queryByTestId("plan-finding-apply-f2")).not.toBeInTheDocument();
    expect(screen.getByTestId("plan-finding-state-f3")).toHaveTextContent("Đã sửa");
    fireEvent.click(screen.getByTestId("plan-finding-apply-f1"));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("p1", "f1"));
    fireEvent.click(screen.getByTestId("plan-finding-accept-quick-f2"));
    await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "f2", ""));
  });

  it("accepts a failing machine check without a reason", async () => {
    const accept = vi.spyOn(apiClient, "acceptVisualPlanFinding").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    fireEvent.click(screen.getByTestId("plan-accept-quick-m-plan-risk"));
    await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "m-plan-risk", ""));
  });

  it("auto-fixes the obvious failures on request and lists what it changed", async () => {
    state = baseState({
      beats: baseState().beats.map((b) => ({
        ...b,
        units: b.units.map((u) =>
          u.unit_id === "VP-003" ? { ...u, problems: ["Thời lượng dự kiến 5.0s ngắn hơn lời thoại (tối thiểu 5.1s)"] } : u,
        ),
      })),
    });
    const fix = vi
      .spyOn(apiClient, "autoFixVisualPlan")
      .mockResolvedValue(baseState({ auto_fixes: ["VP-003: Đã nâng thời lượng 5.00s → 5.10s (mức tối thiểu theo lời thoại)"] }));
    renderPage();
    const button = await screen.findByTestId("plan-auto-fix");
    expect(button).toHaveTextContent("· 1");
    fireEvent.click(button);
    await waitFor(() => expect(fix).toHaveBeenCalledWith("p1"));
    expect(await screen.findByTestId("plan-auto-fix-list")).toHaveTextContent("VP-003: Đã nâng thời lượng");
  });

  it("shows an approved unit on one line, opens it and folds it again", async () => {
    renderPage();
    const card = await screen.findByTestId("plan-unit-VP-001");
    expect(within(card).getByTestId("plan-unit-VP-001-open")).toBeInTheDocument();
    expect(within(card).queryByTestId("plan-unit-VP-001-unapprove")).not.toBeInTheDocument();
    fireEvent.click(within(card).getByTestId("plan-unit-VP-001-open"));
    expect(await screen.findByTestId("plan-unit-VP-001-unapprove")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("plan-unit-VP-001-fold"));
    await waitFor(() =>
      expect(screen.queryByTestId("plan-unit-VP-001-unapprove")).not.toBeInTheDocument(),
    );
  });

  it("opens a finding's unit on the plan tab", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    fireEvent.click(screen.getByTestId("plan-finding-open-f1"));
    expect(await screen.findByTestId("plan-unit-VP-003")).toBeInTheDocument();
    expect(screen.getByTestId("plan-tab-plan")).toHaveAttribute("aria-selected", "true");
  });

  it("runs the review on the system's AI with a note", async () => {
    const review = vi.spyOn(apiClient, "reviewVisualPlan").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    expect(screen.getByTestId("plan-review-run")).toHaveTextContent("Rà soát lại");
    fireEvent.click(screen.getByTestId("plan-review-run"));
    const dialog = await screen.findByTestId("plan-review-dialog");
    fireEvent.click(within(dialog).getByTestId("plan-review-dialog-send"));
    await waitFor(() => expect(review).toHaveBeenCalledWith("p1", "", expect.objectContaining({ signal: expect.anything() })));
  });

  it("locks with a reason when it cannot: the blockers and the criteria say what is missing", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    expect(screen.getByTestId("plan-lock")).toBeDisabled();
    expect(screen.getByTestId("plan-lock")).toHaveTextContent("Chốt kế hoạch hình v1");
    expect(screen.getByTestId("plan-blockers")).toHaveTextContent("Còn 2 câu chưa phủ");
    const criteria = screen.getByTestId("plan-criteria");
    expect(criteria).toHaveTextContent("Phủ 100% câu kịch bản · 4/6 câu");
    expect(criteria).toHaveTextContent("Mọi unit đã duyệt · 1/3");
    expect(criteria).toHaveTextContent("Mọi câu hỏi mở đã trả lời · 1 unit còn câu hỏi");
    expect(screen.getByTestId("plan-over-brief")).toHaveTextContent("vượt thời lượng creative brief");
  });

  it("locks a version with a note, and the second lock lists what changed", async () => {
    const lock = vi.spyOn(apiClient, "lockVisualPlan").mockResolvedValue(baseState());
    state = baseState({
      can_lock: true,
      blockers: [],
      locked_version: 1,
      next_version: 2,
      draft: ["VP-002 đã đổi"],
      versions: [{ version: 1, script_version: 2, concept_version: 1, note: "Bản đầu", locked_at: "2026-11-11T07:00:00Z" }],
    });
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    expect(screen.getByTestId("plan-versions")).toHaveTextContent("v1 · kịch bản v2 · ngôn ngữ hình ảnh v1");
    fireEvent.click(screen.getByTestId("plan-lock"));
    const dialog = await screen.findByTestId("plan-lock-dialog");
    expect(within(dialog).getByTestId("plan-lock-draft")).toHaveTextContent("VP-002 đã đổi");
    fireEvent.change(within(dialog).getByTestId("plan-lock-note"), { target: { value: "Bản 2" } });
    fireEvent.click(within(dialog).getByTestId("plan-lock-confirm"));
    await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", "Bản 2"));
  });

  it("shows a locked plan and opens the next step", async () => {
    state = baseState({ locked_version: 1, next_version: 2, draft: [], can_lock: true, blockers: [] });
    renderPage();
    fireEvent.click(await screen.findByTestId("plan-tab-review"));
    expect(screen.getByTestId("plan-locked")).toHaveTextContent("Đã chốt kế hoạch hình v1");
    expect(screen.getByTestId("plan-next")).not.toBeDisabled();
  });

  it("keeps the next step closed until a plan is locked", async () => {
    renderPage();
    expect(await screen.findByTestId("plan-next")).toBeDisabled();
  });
});
