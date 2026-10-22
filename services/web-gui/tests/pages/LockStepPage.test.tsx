import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { LockStepPage } from "../../src/pages/LockStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { LockBeat, ReviewFinding, ScriptLockState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  const l = useLocation();
  return <span data-testid="where">{l.pathname + l.search}</span>;
}

const SECTIONS = [
  { key: "must", label: "Thông tin bắt buộc truyền đạt", roles: ["storyboard", "illustration"] },
  { key: "terms", label: "Thuật ngữ cần giải thích, tránh hiểu sai", roles: ["art_world", "storyboard", "voice"] },
  { key: "exact", label: "Phải giữ chính xác", roles: ["art_world", "illustration", "validate"] },
  { key: "free", label: "Được phép sáng tạo", roles: ["storyboard", "code"] },
  { key: "unsure", label: "Điểm chưa chắc, xử lý cẩn thận", roles: ["storyboard", "illustration", "validate"] },
];

function finding(id: string, over: Partial<ReviewFinding> = {}): ReviewFinding {
  return {
    id,
    source: "ai",
    group: "logic",
    groupLabel: "Logic và nhịp",
    state: "open",
    title: `Mục ${id}`,
    detail: `Chi tiết ${id}`,
    line: "",
    fix: "",
    handoff: null,
    research: false,
    why: "",
    ...over,
  };
}

function beat(key: string, number: number, over: Partial<LockBeat> = {}): LockBeat {
  return {
    key,
    number,
    fn: "Giải thích cơ chế",
    title: `Beat ${key}`,
    goal: `Mục tiêu ${key}`,
    written: true,
    status: "pending",
    mine: [],
    stale: false,
    lines: 2,
    note: { must: [`Bắt buộc ${key}`], terms: [], exact: [], free: [], unsure: [] },
    ...over,
  };
}

function baseState(over: Partial<ScriptLockState> = {}): ScriptLockState {
  return {
    scriptReady: true,
    reviewed: false,
    findings: [
      finding("m-review", { source: "machine", group: "final", groupLabel: "Nhất quán cuối", title: "AI chưa rà soát kịch bản này" }),
      finding("r1", { line: "b2.1", fix: "Câu thay thế." }),
      finding("r2", { handoff: { beat: "b2", section: "terms", text: "đáy quả là đầu đối diện cuống" } }),
      finding("m-link:F03", { source: "machine", group: "research", groupLabel: "Nhất quán với nghiên cứu", research: true, line: "b3.1" }),
      finding("m-must", { source: "machine", state: "ok", title: "Dữ kiện bắt buộc đều có trong lời thoại" }),
    ],
    openCount: 4,
    aiNotes: ["Nhịp ổn."],
    myNotes: [],
    glossary: [{ term: "ficin", say: "phi-xin", draw: "quầng sáng", avoid: "cảnh tiêu hoá", mine: false }],
    beats: [beat("b1", 1, { fn: "Hook", status: "approved" }), beat("b2", 2), beat("b3", 3, { fn: "Payoff", written: false })],
    conditions: [
      { ok: false, text: "Rà soát: không còn mục cần xử lý (còn 4)" },
      { ok: true, text: "Lời thoại: mọi beat đã duyệt" },
      { ok: false, text: "Ghi chú bàn giao: mọi beat đã duyệt (còn 2 beat)" },
    ],
    canLock: false,
    package: {
      topic: "Vì sao ong chết?",
      audience: "Người xem phổ thông",
      targetLabel: "4–5 phút",
      totalSeconds: 268,
      lines: 22,
      beats: 3,
      chain: [{ fn: "Hook", title: "Mở", next: "Sang hoa" }],
      facts: [{ id: "F01", type: "fact", link: "ok" }],
      hypotheses: 0,
      unmatched: 0,
      noSource: false,
    },
    handoff: { beats: 3, sections: { must: 2 }, glossary: 1 },
    sections: SECTIONS,
    versions: [],
    lockedVersion: 0,
    nextVersion: 1,
    draft: [],
    skippedResearch: false,
    ...over,
  };
}

let state: ScriptLockState;
let project: Project;

beforeEach(() => {
  state = baseState();
  project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 3, run_state: "idle", video_output_mode: "long" } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive", reason: "", models: [{ id: "m1", label: "Model 1" }] });
  vi.spyOn(apiClient, "getProject").mockImplementation(async () => project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "getResearch").mockRejectedValue(new Error("no"));
  vi.spyOn(apiClient, "getScriptLock").mockImplementation(async () => state);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/lock"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/lock" element={<LockStepPage />} />
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
  await screen.findByTestId("lock-review-card");
}

describe("LockStepPage", () => {
  it("opens on the review tab with the findings as the server says", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Rà soát và chốt kịch bản");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 7/21");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa chốt · 4 mục rà soát cần xử lý");
    expect(screen.getByTestId("finding-m-must")).toHaveAttribute("data-state", "ok");
    expect(screen.getByTestId("finding-r1")).toHaveAttribute("data-state", "open");
    expect(screen.getByTestId("lock-tab-review")).toHaveTextContent("4 cần xử lý");
    expect(screen.getByTestId("lock-review-run")).toHaveTextContent("AI rà soát");
  });

  it("says the step is closed until every beat's dialogue is approved", async () => {
    state = baseState({ scriptReady: false, findings: [] });
    renderPage();
    expect(await screen.findByTestId("lock-not-open")).toHaveTextContent("Duyệt lời thoại");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.queryByTestId("lock-tab-review")).not.toBeInTheDocument();
  });

  it("offers on each finding only what fits it", async () => {
    await ready();
    expect(screen.getByTestId("finding-apply-r1")).toBeInTheDocument();
    expect(screen.getByTestId("finding-open-line-r1")).toBeInTheDocument();
    expect(screen.queryByTestId("finding-handoff-r1")).not.toBeInTheDocument();
    expect(screen.getByTestId("finding-handoff-r2")).toBeInTheDocument();
    expect(screen.queryByTestId("finding-apply-r2")).not.toBeInTheDocument();
    expect(screen.getByTestId("finding-open-research-m-link:F03")).toBeInTheDocument();
    // A passed check offers nothing.
    expect(screen.queryByTestId("finding-accept-m-must")).not.toBeInTheDocument();
  });

  it("applies the suggestion of a finding through the server and shows what comes back", async () => {
    const apply = vi.spyOn(apiClient, "applyFinding").mockImplementation(async () => {
      state = baseState({ findings: state.findings.map((f) => (f.id === "r1" ? { ...f, state: "fixed" as const } : f)), openCount: 3 });
      return state;
    });
    await ready();
    fireEvent.click(screen.getByTestId("finding-apply-r1"));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("p1", "r1"));
    await waitFor(() => expect(screen.getByTestId("finding-r1")).toHaveAttribute("data-state", "fixed"));
  });

  it("accepts a finding with the reason typed in the dialog", async () => {
    const accept = vi.spyOn(apiClient, "acceptFinding").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("finding-accept-r2"));
    fireEvent.change(await screen.findByTestId("lock-accept-why"), { target: { value: "đã kiểm ở nguồn khác" } });
    fireEvent.click(screen.getByTestId("lock-accept-save"));
    await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "r2", "đã kiểm ở nguồn khác"));
  });

  it("starts the AI review, then calls it a re-review once the script was reviewed", async () => {
    const review = vi.spyOn(apiClient, "reviewScript").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("lock-review-run"));
    await waitFor(() => expect(review).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByTestId("lock-review-run")).toBeEnabled());
    state = baseState({ reviewed: true, findings: [finding("m-review", { source: "machine", state: "ok", title: "AI đã rà soát" }), finding("r1")] });
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    fireEvent.click(screen.getByTestId("lock-tab-review"));
    // The tab switch does not reload; a reload from the server brings the new state in.
    fireEvent.click(screen.getByTestId("lock-review-run"));
    await waitFor(() => expect(screen.getByTestId("lock-review-run")).toHaveTextContent("Rà soát lại"));
  });

  it("links 'Mở câu đó' to the script screen at that line", async () => {
    await ready();
    fireEvent.click(within(screen.getByTestId("finding-open-line-r1")).getByRole("button"));
    expect(await screen.findByTestId("where")).toHaveTextContent("/projects/p1/script?line=b2.1");
  });

  it("shows each beat's five handoff sections with the roles that read them", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    expect(await screen.findByTestId("handoff-beat-b2")).toHaveTextContent("B2 · Giải thích cơ chế · Beat b2");
    expect(screen.getByTestId("handoff-b2-must")).toHaveTextContent("Bắt buộc b2");
    expect(screen.getByTestId("handoff-beat-b2")).toHaveTextContent("Điểm chưa chắc, xử lý cẩn thận");
    expect(screen.getByTestId("handoff-beat-b2")).toHaveTextContent("Storyboard");
    expect(screen.getByTestId("glossary-ficin")).toHaveTextContent("phi-xin");
    // A beat with no note offers to write one; a written one has the review bar.
    expect(screen.getByTestId("handoff-write-b3")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-review-b2-approve")).toBeInTheDocument();
    expect(screen.getByTestId("handoff-review-b1-unapprove")).toBeInTheDocument();
  });

  it("has the AI write the missing notes and offers to approve the ones waiting", async () => {
    const gen = vi.spyOn(apiClient, "generateHandoff").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    fireEvent.click(await screen.findByTestId("handoff-generate"));
    await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.anything()));
    expect(screen.getByTestId("handoff-approve-all")).toBeEnabled();
  });

  it("asks the Creator to keep or redo a note whose dialogue changed", async () => {
    state = baseState({ beats: [beat("b1", 1, { stale: true, status: "approved" }), beat("b2", 2, { status: "approved" }), beat("b3", 3, { status: "approved" })] });
    const confirm = vi.spyOn(apiClient, "confirmHandoff").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    expect(await screen.findByTestId("handoff-stale-b1")).toHaveTextContent("lời thoại đã đổi");
    expect(screen.queryByTestId("handoff-review-b1-approve")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("handoff-confirm-b1"));
    await waitFor(() => expect(confirm).toHaveBeenCalledWith("p1", "b1"));
  });

  it("saves a hand edit of a note's sections, one line per item", async () => {
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    fireEvent.click(await screen.findByTestId("handoff-review-b2-edit"));
    fireEvent.change(await screen.findByTestId("handoff-edit-b2-field-terms"), { target: { value: "một\nhai" } });
    fireEvent.click(screen.getByTestId("handoff-edit-b2-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "handoff_beat", key: "b2" }), { terms: "một\nhai" }));
  });

  it("adds a glossary term from the dialog", async () => {
    const add = vi.spyOn(apiClient, "addGlossaryTerm").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-handoff"));
    fireEvent.click(await screen.findByTestId("glossary-add"));
    expect(screen.getByTestId("glossary-save")).toBeDisabled();
    fireEvent.change(await screen.findByTestId("glossary-term"), { target: { value: "ostiole" } });
    fireEvent.change(screen.getByTestId("glossary-say"), { target: { value: "cửa quả" } });
    fireEvent.click(screen.getByTestId("glossary-save"));
    await waitFor(() => expect(add).toHaveBeenCalledWith("p1", { term: "ostiole", say: "cửa quả", draw: "", avoid: "" }));
  });

  it("keeps the lock button off, and says what is missing, until every condition holds", async () => {
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-package"));
    expect(await screen.findByTestId("lock-conditions-card")).toHaveTextContent("còn 4");
    expect(screen.getByTestId("lock-conditions-card")).toHaveTextContent("còn 2 beat");
    expect(screen.getByTestId("lock-run")).toBeDisabled();
    expect(screen.getByTestId("lock-run")).toHaveTextContent("Chốt phiên bản v1");
    expect(screen.getByTestId("lock-package-a")).toHaveTextContent("mục tiêu 4–5 phút · ước tính 4:28");
    expect(screen.getByTestId("lock-package-b")).toHaveTextContent("1 thuật ngữ chung");
    expect(screen.getByTestId("lock-next")).toBeDisabled();
  });

  it("locks the first version straight away with the note typed", async () => {
    state = baseState({ canLock: true, openCount: 0 });
    const lock = vi.spyOn(apiClient, "lockScript").mockImplementation(async () => state);
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-package"));
    fireEvent.change(await screen.findByTestId("lock-note"), { target: { value: "bản đầu" } });
    fireEvent.click(screen.getByTestId("lock-run"));
    await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", "bản đầu"));
  });

  it("lists the draft against the locked version and confirms before locking the next one", async () => {
    state = baseState({
      canLock: true,
      openCount: 0,
      lockedVersion: 1,
      nextVersion: 2,
      draft: ["Lời thoại · B2"],
      versions: [{ version: 1, lockedAt: "2026-10-22T10:00:00Z", note: "Chốt lần đầu" }],
    });
    const lock = vi.spyOn(apiClient, "lockScript").mockImplementation(async () => state);
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("v1 đã chốt · bản nháp v2 có 1 thay đổi");
    fireEvent.click(screen.getByTestId("lock-tab-package"));
    expect(await screen.findByTestId("lock-draft")).toHaveTextContent("Lời thoại · B2");
    expect(screen.getByTestId("lock-history")).toHaveTextContent("Chốt lần đầu");
    fireEvent.click(screen.getByTestId("lock-run"));
    expect(lock).not.toHaveBeenCalled();
    fireEvent.click(await screen.findByTestId("confirm-modal-confirm"));
    await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", ""));
  });

  it("hides the lock controls once locked and nothing changed", async () => {
    state = baseState({ canLock: true, openCount: 0, lockedVersion: 1, nextVersion: 2, versions: [{ version: 1, lockedAt: "2026-10-22T10:00:00Z", note: "x" }] });
    await ready();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Đã chốt v1");
    fireEvent.click(screen.getByTestId("lock-tab-package"));
    expect(await screen.findByTestId("lock-version-chip")).toHaveTextContent("v1 đã chốt");
    expect(screen.queryByTestId("lock-run")).not.toBeInTheDocument();
    expect(screen.getByTestId("lock-next")).toBeEnabled();
  });

  it("shows the reason an action failed", async () => {
    vi.spyOn(apiClient, "lockScript").mockRejectedValue(new apiClient.ApiError("chưa đủ điều kiện chốt", undefined, undefined, 409));
    state = baseState({ canLock: true, openCount: 0 });
    await ready();
    fireEvent.click(screen.getByTestId("lock-tab-package"));
    fireEvent.click(await screen.findByTestId("lock-run"));
    expect(await screen.findByTestId("lock-error")).toHaveTextContent("chưa đủ điều kiện chốt");
  });
});
