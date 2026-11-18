import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { VisualSequenceStepPage } from "../../src/pages/VisualSequenceStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { SequenceItem, SequenceUnit, VisualSequenceState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  const l = useLocation();
  return <span data-testid="where">{l.pathname}</span>;
}

function item(id: string, over: Partial<SequenceItem> = {}): SequenceItem {
  return {
    unit_id: id,
    purpose_in_flow: `Vai trò ${id}`,
    pacing: { name: "Đối chiếu", tempo: "medium", description: "luân phiên" },
    transition_from_previous: {
      type: "Match cut",
      description: "giữ hố đen",
      via_motif: "MOTIF-01",
    },
    motif_refs: ["MOTIF-01"],
    status: "pending",
    ...over,
  };
}

function unit(
  id: string,
  order: number,
  prev: string,
  it: SequenceItem | null,
  over: Partial<SequenceUnit> = {},
): SequenceUnit {
  return {
    unit_id: id,
    order,
    script_refs: ["b1.1", "b1.2"],
    planned_s: 8,
    communication_goal: `Mục tiêu ${id}`,
    visual_approach: { method: "diagram", description: `Mô tả ${id}`, concept_rule_refs: [] },
    narration_relationship: "contrast",
    previous_unit: prev,
    item: it,
    problems: [],
    creator_fields: [],
    ...over,
  };
}

function baseState(over: Partial<VisualSequenceState> = {}): VisualSequenceState {
  const vp1 = unit(
    "VP-001",
    1,
    "",
    item("VP-001", {
      transition_from_previous: null,
      pacing: { name: "Hé lộ chậm", tempo: "slow", description: "trôi" },
      status: "approved",
    }),
  );
  const vp2 = unit("VP-002", 2, "VP-001", item("VP-002"));
  const vp3 = unit("VP-003", 3, "VP-002", null);
  return {
    input: { plan_version: 1, concept_version: 1, script_version: 2, units_count: 3, outdated: [] },
    motifs: {
      status: "approved",
      stale: false,
      stale_sources: [],
      motifs: [
        {
          motif_id: "MOTIF-01",
          name: "Hố đen làm điểm neo",
          description: "Hình tròn đen có vành sáng",
          anchor_ref: "representation_rules.scientific_concepts.bh",
          role: "Điểm neo",
          planned_units: ["VP-001", "VP-002"],
          anchor_text: "tròn đen",
          unit_refs: ["VP-001", "VP-002"],
          problems: [],
          creator_fields: [],
        },
        {
          motif_id: "MOTIF-02",
          name: "Đồng hồ",
          description: "chậm dần",
          anchor_ref: "representation_rules.objects.clock",
          role: "Gieo và gặt",
          planned_units: [],
          anchor_text: "mặt tròn",
          unit_refs: [],
          problems: [],
          creator_fields: [],
        },
      ],
      anchors: {
        "representation_rules.scientific_concepts.bh": "tròn đen",
        "representation_rules.objects.clock": "mặt tròn",
      },
    },
    beats: [
      {
        key: "b1",
        number: 1,
        fn: "Hook",
        title: "Tiến về hố đen",
        pace: "slow",
        written: true,
        approved: false,
        stale: false,
        stale_sources: [],
        units: [vp1, vp2],
      },
      {
        key: "b2",
        number: 2,
        fn: "Payoff",
        title: "Hai góc nhìn",
        pace: "medium",
        written: false,
        approved: false,
        stale: false,
        stale_sources: [],
        units: [vp3],
      },
    ],
    items_approved: 1,
    coverage: { units_total: 3, units_sequenced: 2, missing: ["VP-003"], extra: [] },
    checks: [
      {
        id: "m-motif-count",
        label: "Mỗi motif có ở ít nhất 2 unit",
        ok: false,
        detail: "MOTIF-02 ở 0 unit",
        accepted: false,
        why: "",
      },
      {
        id: "m-seq-cover",
        label: "Mỗi unit có đúng một item",
        ok: false,
        detail: "VP-003",
        accepted: false,
        why: "",
      },
    ],
    review: {
      reviewed: true,
      rounds: 1,
      notes: ["Nhịp đi từ chậm lên nhanh"],
      open_count: 1,
      findings: [
        {
          id: "f1",
          group: "clarity",
          group_label: "Rõ ràng",
          unit_key: "VP-002",
          field: "purpose_in_flow",
          text: "Vai trò mơ hồ",
          detail: "d",
          fix: "Nêu rõ",
          state: "open",
          why: "",
        },
        {
          id: "f2",
          group: "consistency",
          group_label: "Nhất quán",
          unit_key: "MOTIF-01",
          field: "",
          text: "Lệch cách vẽ",
          detail: "",
          fix: "",
          state: "fixed",
          why: "",
        },
      ],
    },
    blockers: ["Còn 1 unit chưa có chuỗi."],
    can_lock: false,
    versions: [],
    locked_version: 0,
    next_version: 1,
    draft: [],
    pending_units: ["VP-002"],
    model: "",
    default_model: "ollama",
    models: [{ id: "ollama", label: "Ollama", local: true }],
    ...over,
  };
}

let state: VisualSequenceState;

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
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    reason: "",
    models: [],
  });
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getResearch").mockRejectedValue(new Error("no"));
  vi.spyOn(apiClient, "getVisualSequence").mockImplementation(async () => state);
  Element.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/visual-sequence"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/visual-sequence" element={<VisualSequenceStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("VisualSequenceStepPage", () => {
  it("says to lock the visual plan first when step 9 is not locked, and leads there", async () => {
    vi.spyOn(apiClient, "getVisualSequence").mockRejectedValue(
      new apiClient.ApiError(
        "Chốt Kế hoạch hình ở bước 9 trước",
        "plan_not_locked",
        undefined,
        409,
      ),
    );
    renderPage();
    expect(await screen.findByTestId("seq-not-open")).toHaveTextContent(
      "Chốt Kế hoạch hình ở bước 9 trước",
    );
    fireEvent.click(screen.getByTestId("seq-go-back"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/visual-plan");
  });

  it("is step 10 of the visual phase: inputs, motifs, run, flow strip and beats in that order", async () => {
    renderPage();
    expect(await screen.findByTestId("seq-head-card")).toBeInTheDocument();
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Hình ảnh · Bước 10/23");
    expect(screen.getByTestId("rail-step-10")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("seq-input")).toHaveTextContent(
      "Kế hoạch hình v1 · 3 unit · Ngôn ngữ hình ảnh v1 · Kịch bản v2",
    );
    const order = ["seq-head-card", "seq-motifs", "seq-run", "seq-flow", "seq-beats"].map((id) =>
      screen.getByTestId(id),
    );
    for (let i = 1; i < order.length; i++) {
      expect(
        order[i - 1].compareDocumentPosition(order[i]) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    }
    expect(screen.getByTestId("seq-motif-MOTIF-01-units")).toHaveTextContent("VP-001, VP-002");
    expect(screen.getByTestId("seq-flow-count")).toHaveTextContent(
      "3 unit · 0:24 · 2/3 đã có chuỗi",
    );
    expect(screen.getByTestId("seq-flow-VP-001")).toHaveTextContent("Hé lộ chậm");
    expect(screen.getByTestId("seq-flow-tr-VP-002")).toHaveTextContent("Match cut");
    expect(screen.getByTestId("seq-flow-VP-003")).toHaveTextContent("chưa viết");
    expect(screen.getByTestId("seq-unit-VP-002-transition")).toHaveTextContent("Match cut");
    expect(screen.getByTestId("seq-unit-VP-002-transition")).toHaveTextContent("Nối bằng MOTIF-01");
    expect(screen.getByTestId("seq-unit-VP-003-empty")).toHaveTextContent("Chưa có nhịp");
    expect(screen.getByTestId("seq-unit-VP-003-redo")).toHaveTextContent("Viết chuỗi unit này");
  });

  it("warns when the plan is behind the latest locked script or concept", async () => {
    state = baseState({
      input: {
        plan_version: 1,
        concept_version: 2,
        script_version: 2,
        units_count: 3,
        outdated: ["kế hoạch hình viết từ ngôn ngữ hình ảnh v1"],
      },
    });
    renderPage();
    expect(await screen.findByTestId("seq-outdated")).toHaveTextContent("ngôn ngữ hình ảnh v1");
  });

  it("holds the AI writing beats until the motifs are approved", async () => {
    state = baseState({ motifs: { ...baseState().motifs, status: "pending" } });
    renderPage();
    expect(await screen.findByTestId("seq-generate")).toBeDisabled();
    expect(screen.getByTestId("seq-beat-b2-generate")).toBeDisabled();
    expect(screen.getByTestId("seq-run-needs-motifs")).toBeInTheDocument();
    const approve = vi.spyOn(apiClient, "approveSequenceMotifs").mockResolvedValue(baseState());
    fireEvent.click(screen.getByTestId("seq-motifs-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1"));
    await waitFor(() => expect(screen.getByTestId("seq-generate")).toBeEnabled());
  });

  it("writes every empty beat with the Creator's note", async () => {
    const gen = vi.spyOn(apiClient, "generateVisualSequence").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-generate"));
    const dialog = await screen.findByTestId("seq-generate-dialog");
    expect(dialog).toHaveTextContent("1 unit chưa có chuỗi: VP-003");
    fireEvent.change(screen.getByTestId("seq-generate-note"), { target: { value: " nhanh dần " } });
    fireEvent.click(screen.getByTestId("seq-generate-confirm"));
    await waitFor(() =>
      expect(gen).toHaveBeenCalledWith(
        "p1",
        "nhanh dần",
        expect.objectContaining({ id: expect.any(String) }),
      ),
    );
  });

  it("adds a motif anchored to a drawing rule of step 8", async () => {
    const add = vi.spyOn(apiClient, "addSequenceMotif").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-motif-add"));
    await screen.findByTestId("seq-motif-dialog");
    expect(screen.getByTestId("seq-motif-dialog-confirm")).toBeDisabled();
    fireEvent.change(screen.getByTestId("seq-motif-name"), { target: { value: "Đồng hồ cát" } });
    fireEvent.click(screen.getByTestId("seq-motif-anchor"));
    fireEvent.click(
      await screen.findByRole("option", { name: /representation_rules\.objects\.clock/ }),
    );
    expect(screen.getByTestId("seq-motif-anchor-text")).toHaveTextContent("mặt tròn");
    fireEvent.click(screen.getByTestId("seq-motif-dialog-confirm"));
    await waitFor(() =>
      expect(add).toHaveBeenCalledWith("p1", {
        name: "Đồng hồ cát",
        anchor_ref: "representation_rules.objects.clock",
      }),
    );
  });

  it("asks before deleting a motif and names the units that lose it", async () => {
    const del = vi.spyOn(apiClient, "deleteSequenceMotif").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-motif-MOTIF-01-delete"));
    expect(await screen.findByTestId("seq-motif-delete-impact")).toHaveTextContent(
      "VP-001, VP-002",
    );
    expect(del).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("seq-motif-delete-dialog-confirm"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("p1", "MOTIF-01"));
  });

  it("sends only the fields the Creator changed in a unit's sequence", async () => {
    const edit = vi.spyOn(apiClient, "editSequenceItem").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-unit-VP-002-edit"));
    await screen.findByTestId("seq-item-dialog");
    fireEvent.change(screen.getByTestId("seq-item-tempo"), { target: { value: "fast" } });
    fireEvent.click(screen.getByTestId("seq-item-motif-MOTIF-02"));
    fireEvent.click(screen.getByTestId("seq-item-dialog-confirm"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", "VP-002", {
        pacing: { name: "Đối chiếu", tempo: "fast", description: "luân phiên" },
        motif_refs: ["MOTIF-01", "MOTIF-02"],
      }),
    );
  });

  it("shows the unit that opens the video without a transition into it", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-unit-VP-001-open"));
    expect(screen.getByTestId("seq-unit-VP-001-transition")).toHaveTextContent("Unit đầu video");
  });

  it("lets a stale beat be confirmed and blocks approving its units", async () => {
    state = baseState();
    state.beats[0] = { ...state.beats[0], stale: true, stale_sources: ["Danh sách motif"] };
    const confirm = vi.spyOn(apiClient, "confirmSequenceBeat").mockResolvedValue(baseState());
    renderPage();
    expect(await screen.findByTestId("seq-beat-b1-stale")).toHaveTextContent("Danh sách motif");
    expect(screen.getByTestId("seq-unit-VP-002-approve")).toBeDisabled();
    fireEvent.click(screen.getByTestId("seq-beat-b1-confirm"));
    await waitFor(() => expect(confirm).toHaveBeenCalledWith("p1", "b1"));
  });

  it("review tab: accepts a machine check in one click, applies a fix, opens a motif and keeps the lock closed", async () => {
    const accept = vi
      .spyOn(apiClient, "acceptVisualSequenceFinding")
      .mockResolvedValue(baseState());
    const apply = vi.spyOn(apiClient, "applyVisualSequenceFinding").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-tab-review"));
    expect(
      within(screen.getByTestId("seq-check-row-m-seq-cover")).queryByRole("button"),
    ).toBeNull();
    fireEvent.click(screen.getByTestId("seq-accept-m-motif-count"));
    await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "m-motif-count", ""));

    fireEvent.click(screen.getByTestId("seq-finding-apply-f1"));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("p1", "f1"));
    expect(screen.getByTestId("seq-finding-state-f2")).toHaveTextContent("Đã sửa");
    expect(screen.getByTestId("seq-lock")).toBeDisabled();
    expect(screen.getByTestId("seq-blockers")).toHaveTextContent("Còn 1 unit chưa có chuỗi.");
    expect(screen.getByTestId("seq-pending-units")).toHaveTextContent("VP-002");
    fireEvent.click(screen.getByTestId("seq-go-pending"));
    expect(await screen.findByTestId("seq-unit-VP-002")).toBeInTheDocument();
  });

  it("locks v2 with the orange button and lists what changed", async () => {
    state = baseState({
      locked_version: 1,
      next_version: 2,
      draft: ["VP-002"],
      can_lock: true,
      blockers: [],
    });
    const lock = vi
      .spyOn(apiClient, "lockVisualSequence")
      .mockResolvedValue(baseState({ locked_version: 2 }));
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-tab-review"));
    fireEvent.click(screen.getByTestId("seq-lock"));
    expect(await screen.findByTestId("seq-lock-draft")).toHaveTextContent("VP-002");
    fireEvent.click(screen.getByTestId("seq-lock-confirm"));
    await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", ""));
  });

  it("shows the run in flight at the top with a stop button, wherever the Creator scrolled", async () => {
    let signal: AbortSignal | undefined;
    vi.spyOn(apiClient, "generateVisualSequence").mockImplementation(
      (_p, _n, call) =>
        new Promise((_, reject) => {
          signal = call?.signal;
          signal?.addEventListener("abort", () =>
            reject(new DOMException("aborted", "AbortError")),
          );
        }),
    );
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-generate"));
    fireEvent.click(await screen.findByTestId("seq-generate-confirm"));
    expect(await screen.findByTestId("seq-running")).toHaveTextContent(
      "AI đang viết chuỗi các unit còn trống",
    );
    fireEvent.click(screen.getByTestId("seq-stop"));
    expect(signal?.aborted).toBe(true);
    await waitFor(() => expect(screen.queryByTestId("seq-running")).toBeNull());
  });

  it("keeps approving open on other beats while the AI writes the empty ones", async () => {
    vi.spyOn(apiClient, "generateVisualSequence").mockImplementation(() => new Promise(() => {}));
    const approve = vi.spyOn(apiClient, "approveSequenceItem").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-generate"));
    fireEvent.click(await screen.findByTestId("seq-generate-confirm"));
    await screen.findByTestId("seq-running");
    expect(screen.getByTestId("seq-unit-VP-002-redo")).toBeDisabled();
    fireEvent.click(screen.getByTestId("seq-unit-VP-002-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", "VP-002"));
  });

  it("folds a beat once every unit is approved and opens it on demand", async () => {
    state = baseState();
    state.beats[0] = {
      ...state.beats[0],
      approved: true,
      units: state.beats[0].units.map((u) => ({
        ...u,
        item: u.item && { ...u.item, status: "approved" as const },
      })),
    };
    renderPage();
    expect(await screen.findByTestId("seq-beat-b1-open")).toHaveTextContent("Mở 2 unit");
    expect(screen.queryByTestId("seq-unit-VP-002")).toBeNull();
    fireEvent.click(screen.getByTestId("seq-beat-b1-open"));
    expect(screen.getByTestId("seq-unit-VP-002")).toBeInTheDocument();
  });

  it("writes one unit in one turn, on the system's AI or one outside", async () => {
    const gen = vi.spyOn(apiClient, "generateSequenceUnit").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-unit-VP-003-redo"));
    expect(await screen.findByTestId("seq-unit-VP-003-redo-dialog")).toHaveTextContent(
      "AI viết riêng VP-003",
    );
    fireEvent.click(screen.getByTestId("seq-unit-VP-003-redo-dialog-send"));
    await waitFor(() =>
      expect(gen).toHaveBeenCalledWith(
        "p1",
        "VP-003",
        "",
        expect.objectContaining({ id: expect.any(String) }),
      ),
    );
  });

  it("rewrites a beat on the system's AI, one turn per unit", async () => {
    const gen = vi.spyOn(apiClient, "generateSequenceBeat").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("seq-beat-b1-generate"));
    expect(await screen.findByTestId("seq-beat-b1-dialog")).toHaveTextContent("mỗi unit một lượt");
    fireEvent.click(screen.getByTestId("seq-beat-b1-confirm-run"));
    await waitFor(() =>
      expect(gen).toHaveBeenCalledWith(
        "p1",
        "b1",
        "",
        expect.objectContaining({ id: expect.any(String) }),
      ),
    );
  });
});
