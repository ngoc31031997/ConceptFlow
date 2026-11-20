import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { StoryboardStepPage } from "../../src/pages/StoryboardStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type {
  StoryboardAsset,
  StoryboardShot,
  StoryboardState,
  StoryboardUnit,
} from "../../src/api/client";
import type { Project } from "../../src/types";

const queueAdd = vi.fn();
let queueAvailable = true;

vi.mock("../../src/hooks/useAiQueue", () => ({
  useAiQueue: () => ({
    available: queueAvailable,
    counts: { waiting: 2, running: 1, done: 0, failed: 0, paused: false },
    add: queueAdd,
  }),
}));

function Where() {
  const l = useLocation();
  return <span data-testid="where">{l.pathname}</span>;
}

function asset(id: string, over: Partial<StoryboardAsset> = {}): StoryboardAsset {
  return {
    asset_id: id,
    concept_id: "characters.astronaut",
    name: id.split("-")[1].charAt(0) + id.split("-")[1].slice(1).toLowerCase(),
    kind: "figure",
    tier: "hero",
    portrayal: `Cảm giác ${id}`,
    must_not_imply: ["không gợi sự hoảng loạn"],
    palette_tokens: ["accent"],
    expressions: ["trôi"],
    base_illustration_id: "",
    units: ["VP-001"],
    has_detail: true,
    version: 1,
    preview_path: "storyboard/assets/x/v1.png",
    status: "approved",
    to_library: true,
    library_folder_id: "",
    library_illustration_id: "",
    warnings: [],
    list_status: "approved",
    creator_fields: [],
    used_by: [],
    ...over,
  };
}

function detailCounts(items: StoryboardAsset[]) {
  const withDetail = items.filter((a) => a.has_detail).length;
  return { with_detail: withDetail, without_detail: items.length - withDetail };
}

/** The assets part of a screen where every asset is approved. */
function allApprovedAssets(): StoryboardState["assets"] {
  const items = baseState().assets.items.map((a) => ({ ...a, status: "approved" as const }));
  return { approved: true, approved_count: items.length, items, ...detailCounts(items) };
}

function shot(id: string, unitId: string, over: Partial<StoryboardShot> = {}): StoryboardShot {
  return {
    shot_id: id,
    script_refs: ["b1.1"],
    shot_purpose: `Mục đích ${id}`,
    composition: {
      subject: "AST-HERO-001",
      subject_position: "lower_left",
      main_object: "",
      main_object_position: "",
      camera_view: "wide",
    },
    action_summary: `Hành động ${id}`,
    on_screen_text: [],
    asset_refs: ["AST-HERO-001"],
    notes: [],
    status: "pending",
    unit_id: unitId,
    beat_id: "b1",
    creator_fields: [],
    problems: [],
    missing_assets: [],
    frame: { drawn: true, stale: false, path: `storyboard/${id}.png`, error: "" },
    ...over,
  };
}

function unit(id: string, order: number, shots: StoryboardShot[], over: Partial<StoryboardUnit> = {}): StoryboardUnit {
  return {
    unit_id: id,
    order,
    beat_id: "b1",
    script_refs: ["b1.1", "b1.2"],
    lines: [
      { id: "b1.1", text: "Dòng một", facts: [] },
      { id: "b1.2", text: "Dòng hai", facts: [] },
    ],
    planned_s: 8,
    communication_goal: `Mục tiêu ${id}`,
    visual_approach: { method: "diagram", description: "d", concept_rule_refs: [] },
    planned_text: [],
    purpose_in_flow: `Vai trò ${id}`,
    written: shots.length > 0,
    approved: shots.length > 0 && shots.every((s) => s.status === "approved"),
    stale: false,
    stale_sources: [],
    shots,
    creator_fields: [],
    ...over,
  };
}

function baseState(over: Partial<StoryboardState> = {}): StoryboardState {
  const hero = asset("AST-HERO-001", { used_by: ["SH-001", "SH-002"], to_library: true });
  const observer = asset("AST-OBSERVER-001", { status: "drawn", name: "Observer", tier: "support" });
  const disk = asset("AST-DISK-001", {
    status: "planned",
    name: "Disk",
    tier: "support",
    preview_path: "",
    base_illustration_id: "",
  });
  const sh1 = shot("SH-001", "VP-001", { status: "approved" });
  const sh2 = shot("SH-002", "VP-002", { status: "approved" });
  const sh3 = shot("SH-003", "VP-002", {
    frame: { drawn: true, stale: true, path: "storyboard/SH-003.png", error: "" },
  });
  const sh4 = shot("SH-004", "VP-003", {
    missing_assets: ["AST-DISK-001"],
    frame: { drawn: false, stale: false, path: "", error: "" },
    problems: [{ check: "m-shot-assets", detail: "Tài sản AST-DISK-001 chưa duyệt" }],
  });
  const vp1 = unit("VP-001", 1, [sh1]);
  const vp2 = unit("VP-002", 2, [sh2, sh3]);
  const vp3 = unit("VP-003", 3, [sh4]);
  const vp4 = unit("VP-004", 4, [], { beat_id: "b2" });
  return {
    input: { sequence_version: 1, plan_version: 1, concept_version: 1, script_version: 2, units_count: 4, outdated: [] },
    kit: { ref_id: "R12", missing: false, message: "" },
    assets: { approved: false, items: [hero, observer, disk], approved_count: 1, ...detailCounts([hero, observer, disk]) },
    beats: [
      { key: "b1", number: 1, fn: "Hook", title: "Mở bài", written: true, approved: false, units: [vp1, vp2, vp3] },
      { key: "b2", number: 2, fn: "Payoff", title: "Kết", written: false, approved: false, units: [vp4] },
    ],
    shots_approved: 2,
    shots_total: 4,
    checks: [
      { id: "m-asset-unused", label: "Tài sản chưa shot nào dùng", ok: false, detail: "AST-DISK-001", accepted: false, why: "" },
      { id: "m-shot-frame", label: "Khung còn mới", ok: false, detail: "SH-003", accepted: false, why: "" },
    ],
    review: {
      reviewed: true,
      rounds: 1,
      notes: ["Nhịp ổn"],
      open_count: 2,
      findings: [
        { id: "f1", group: "consistency", group_label: "Nhất quán", unit_key: "SH-003", field: "action_summary", text: "Áo khác SH-002", detail: "", fix: "Dùng áo xanh", state: "open", why: "" },
        { id: "f2", group: "clarity", group_label: "Rõ ràng", unit_key: "VP-002", field: "", text: "Hai shot giống nhau", detail: "", fix: "Gộp", state: "open", why: "" },
      ],
    },
    blockers: ["1 tài sản chưa duyệt (AST-1)", "2 shot chưa duyệt (SH-1, SH-2)"],
    can_lock: false,
    locked_version: 0,
    locked_note: "",
    locked_at: null,
    next_version: 1,
    draft: [],
    pending_units: ["VP-002", "VP-003"],
    empty_units: ["VP-004"],
    model: "",
    default_model: "deepseek-chat",
    models: [{ id: "ollama", label: "Ollama", local: true }],
    ...over,
  };
}

let state: StoryboardState;

beforeEach(() => {
  state = baseState();
  queueAdd.mockReset();
  queueAvailable = true;
  const project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 3, run_state: "idle", video_output_mode: "long" } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive", reason: "", models: [] });
  vi.spyOn(apiClient, "getProject").mockResolvedValue(project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([
    { project_id: "p1", status: "draft", updated_at: "2026-01-01T00:00:00Z", topic: "Hố đen" },
  ] as Awaited<ReturnType<typeof apiClient.listProjects>>);
  vi.spyOn(apiClient, "getResearch").mockRejectedValue(new Error("no"));
  vi.spyOn(apiClient, "getStoryboard").mockImplementation(async () => state);
  vi.spyOn(apiClient, "listIllustrationFolders").mockResolvedValue([
    { id: "tu-du-an", name: "Từ dự án", description: "", position: 1, is_system: true },
    { id: "hinh-mau", name: "Hình mẫu", description: "", position: 2, is_system: true },
    { id: "nhan-vat", name: "Nhân vật", description: "", position: 3, is_system: false },
  ]);
  vi.spyOn(apiClient, "listIllustrations").mockResolvedValue([]);
  Element.prototype.scrollIntoView = vi.fn();
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/storyboard"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/storyboard" element={<StoryboardStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

async function openShotsTab() {
  fireEvent.click(await screen.findByTestId("sb-tab-shots"));
}

describe("StoryboardStepPage", () => {
  it("says to lock the sequence first when step 10 is not locked, and leads there", async () => {
    vi.spyOn(apiClient, "getStoryboard").mockRejectedValue(
      new apiClient.ApiError("Chốt Chuỗi hình ở bước 10 trước", "sequence_not_locked", undefined, 409),
    );
    renderPage();
    expect(await screen.findByTestId("sb-not-open")).toHaveTextContent("Chốt Chuỗi hình ở bước 10 trước");
    fireEvent.click(screen.getByTestId("sb-go-back"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/visual-sequence");
  });

  it("is step 11: inputs, run card and three tabs with their counts", async () => {
    renderPage();
    expect(await screen.findByTestId("sb-head-card")).toBeInTheDocument();
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Hình ảnh · Bước 11/23");
    expect(screen.getByTestId("rail-step-11")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("sb-input")).toHaveTextContent("Chuỗi hình v1 · 4 unit");
    expect(screen.getByTestId("sb-input")).toHaveTextContent("Kit R12");
    const order = ["sb-head-card", "sb-run"].map((id) => screen.getByTestId(id));
    expect(order[0].compareDocumentPosition(order[1]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByTestId("sb-tab-assets")).toHaveTextContent("1/3 đã duyệt");
    expect(screen.getByTestId("sb-tab-shots")).toHaveTextContent("2/4 đã duyệt");
    expect(screen.getByTestId("sb-tab-review")).toHaveTextContent("2 vấn đề");
    expect(screen.getByTestId("sb-blockers-banner")).toHaveTextContent("1 tài sản chưa duyệt (AST-1)");
  });

  it("holds the AI writing shots until every asset is approved", async () => {
    renderPage();
    expect(await screen.findByTestId("sb-generate")).toBeDisabled();
    expect(screen.getByTestId("sb-run-blocker")).toHaveTextContent("AST-OBSERVER-001");
    expect(screen.getByTestId("sb-queue")).toBeDisabled();
  });

  it("writes every empty unit with the Creator's note once the assets are approved", async () => {
    state = baseState({
      assets: allApprovedAssets(),
    });
    const gen = vi.spyOn(apiClient, "generateStoryboard").mockResolvedValue(state);
    renderPage();
    fireEvent.click(await screen.findByTestId("sb-generate"));
    expect(await screen.findByTestId("sb-generate-dialog")).toHaveTextContent("1 unit chưa có shot: VP-004");
    fireEvent.change(screen.getByTestId("sb-generate-note"), { target: { value: " nhanh " } });
    fireEvent.click(screen.getByTestId("sb-generate-confirm"));
    await waitFor(() =>
      expect(gen).toHaveBeenCalledWith("p1", "nhanh", expect.objectContaining({ id: expect.any(String) })),
    );
  });

  it("queues one AI item per unit not approved, in sequence order, and shows the queue counts", async () => {
    state = baseState({
      assets: allApprovedAssets(),
    });
    const request = { gateway: "http://gw", method: "POST", path: "/v1/projects/p1/storyboard/units/VP-002/generate", body: "{}" };
    const describe = vi
      .spyOn(apiClient, "generateStoryboardUnit")
      .mockRejectedValue(new apiClient.ExternalRequestDescribed(request));
    queueAdd.mockResolvedValue({ ok: true, added: true });
    renderPage();
    expect(await screen.findByTestId("sb-queue-counts")).toHaveTextContent("1 đang chạy · 2 chờ · 0 lỗi");
    const button = screen.getByTestId("sb-queue");
    expect(button).toHaveTextContent("Chạy tất cả qua hàng đợi AI (3)");
    fireEvent.click(button);
    await waitFor(() => expect(queueAdd).toHaveBeenCalledTimes(3));
    expect(queueAdd.mock.calls.map((c) => c[0].label)).toEqual([
      "Storyboard · VP-002",
      "Storyboard · VP-003",
      "Storyboard · VP-004",
    ]);
    expect(describe.mock.calls.map((c) => c[1])).toEqual(["VP-002", "VP-003", "VP-004"]);
    expect(describe.mock.calls.every((c) => (c[3] as { external: { mode: string } }).external.mode === "describe")).toBe(true);
    expect(await screen.findByTestId("sb-queue-note")).toHaveTextContent("Đã thêm 3 unit vào hàng đợi AI");
  });

  it("stops queueing at the unit the extension refuses and says which one", async () => {
    state = baseState({
      assets: allApprovedAssets(),
    });
    vi.spyOn(apiClient, "generateStoryboardUnit").mockRejectedValue(
      new apiClient.ExternalRequestDescribed({ gateway: "g", method: "POST", path: "/p" }),
    );
    queueAdd.mockResolvedValueOnce({ ok: true, added: true }).mockResolvedValueOnce({ ok: false, message: "Extension không phản hồi" });
    renderPage();
    fireEvent.click(await screen.findByTestId("sb-queue"));
    expect(await screen.findByTestId("sb-queue-note")).toHaveTextContent("Dừng ở unit 2/3: Extension không phản hồi");
    expect(queueAdd).toHaveBeenCalledTimes(2);
  });

  it("hides the AI queue button when the extension has no queue", async () => {
    queueAvailable = false;
    renderPage();
    await screen.findByTestId("sb-run");
    expect(screen.queryByTestId("sb-queue")).toBeNull();
  });

  describe("tab Tài sản", () => {
    it("folds an approved asset to one line and opens it on demand", async () => {
      renderPage();
      expect(await screen.findByTestId("sb-asset-AST-HERO-001-open")).toBeInTheDocument();
      expect(screen.queryByTestId("sb-asset-AST-HERO-001-library")).toBeNull();
      fireEvent.click(screen.getByTestId("sb-asset-AST-HERO-001-open"));
      expect(screen.getByTestId("sb-asset-AST-HERO-001-library")).toBeInTheDocument();
      expect(screen.getByTestId("sb-asset-AST-HERO-001-tokens")).toHaveTextContent("accent · trôi");
      fireEvent.click(screen.getByTestId("sb-asset-AST-HERO-001-fold"));
      await waitFor(() => expect(screen.queryByTestId("sb-asset-AST-HERO-001-library")).toBeNull());
    });

    it("approves a drawn asset, sends the video's title, and folds it at once", async () => {
      const approve = vi.spyOn(apiClient, "approveStoryboardAsset").mockResolvedValue(
        baseState({
          assets: {
            approved: false,
            approved_count: 2,
            items: baseState().assets.items.map((a) => (a.asset_id === "AST-OBSERVER-001" ? { ...a, status: "approved" as const } : a)),
            ...detailCounts(baseState().assets.items),
          },
        }),
      );
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-OBSERVER-001-approve"));
      await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", "AST-OBSERVER-001", "Hố đen"));
      expect(await screen.findByTestId("sb-asset-AST-OBSERVER-001-open")).toBeInTheDocument();
    });

    it("reopens an approved asset to edit it when the Creator takes the approval back", async () => {
      const unapprove = vi.spyOn(apiClient, "unapproveStoryboardAsset").mockResolvedValue(
        baseState({
          assets: {
            approved: false,
            approved_count: 0,
            items: baseState().assets.items.map((a) => (a.asset_id === "AST-HERO-001" ? { ...a, status: "drawn" as const } : a)),
            ...detailCounts(baseState().assets.items),
          },
        }),
      );
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-HERO-001-open"));
      fireEvent.click(screen.getByTestId("sb-asset-AST-HERO-001-unapprove"));
      await waitFor(() => expect(unapprove).toHaveBeenCalledWith("p1", "AST-HERO-001"));
      expect(await screen.findByTestId("sb-asset-AST-HERO-001-redraw")).toBeInTheDocument();
    });

    it("draws an asset not drawn yet and redraws a drawn one with a note", async () => {
      const draw = vi.spyOn(apiClient, "drawStoryboardAsset").mockResolvedValue(state);
      const redraw = vi.spyOn(apiClient, "redrawStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-DISK-001-draw"));
      await waitFor(() =>
        expect(draw).toHaveBeenCalledWith("p1", "AST-DISK-001", expect.objectContaining({ id: expect.any(String) })),
      );
      fireEvent.click(await screen.findByTestId("sb-asset-AST-OBSERVER-001-redraw"));
      fireEvent.change(await screen.findByTestId("sb-asset-AST-OBSERVER-001-redraw-note"), { target: { value: " tối hơn " } });
      fireEvent.click(screen.getByTestId("sb-asset-AST-OBSERVER-001-redraw-confirm"));
      await waitFor(() =>
        expect(redraw).toHaveBeenCalledWith("p1", "AST-OBSERVER-001", "tối hơn", expect.objectContaining({ id: expect.any(String) })),
      );
    });

    it("puts an asset in the library by default, and lets the Creator untick it or pick a folder", async () => {
      const edit = vi.spyOn(apiClient, "editStoryboardAsset").mockResolvedValue(state);
      renderPage();
      const box = await screen.findByTestId("sb-asset-AST-OBSERVER-001-to-library");
      expect(box).toBeChecked();
      const folder = screen.getByTestId("sb-asset-AST-OBSERVER-001-folder");
      expect(folder).toHaveValue("");
      expect(folder).toHaveTextContent("Từ dự án");
      expect(folder).toHaveTextContent("Nhân vật");
      expect(folder).not.toHaveTextContent("Hình mẫu");
      fireEvent.change(folder, { target: { value: "nhan-vat" } });
      await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "AST-OBSERVER-001", { library_folder_id: "nhan-vat" }));
      fireEvent.click(box);
      await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "AST-OBSERVER-001", { to_library: false }));
    });

    it("adds an asset by hand with the library tick on", async () => {
      const add = vi.spyOn(apiClient, "addStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-add"));
      await screen.findByTestId("sb-asset-dialog");
      expect(screen.getByTestId("sb-asset-dialog-confirm")).toBeDisabled();
      fireEvent.change(screen.getByTestId("sb-asset-name"), { target: { value: "BlackHole" } });
      fireEvent.change(screen.getByTestId("sb-asset-concept"), { target: { value: "scientific_concepts.bh" } });
      fireEvent.change(screen.getByTestId("sb-asset-portrayal"), { target: { value: "Vùng tối uy nghi" } });
      fireEvent.change(screen.getByTestId("sb-asset-must-not"), { target: { value: "không có miệng\nkhông có mắt" } });
      fireEvent.click(screen.getByTestId("sb-asset-dialog-confirm"));
      await waitFor(() =>
        expect(add).toHaveBeenCalledWith("p1", {
          name: "BlackHole",
          concept_id: "scientific_concepts.bh",
          kind: "figure",
          tier: "support",
          portrayal: "Vùng tối uy nghi",
          must_not_imply: ["không có miệng", "không có mắt"],
          palette_tokens: [],
          expressions: [],
          base_illustration_id: "",
          units: [],
          to_library: true,
          library_folder_id: "",
        }),
      );
    });

    it("sends only the changed fields of an asset and warns that shots using it get stale frames", async () => {
      const edit = vi.spyOn(apiClient, "editStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-HERO-001-open"));
      fireEvent.click(screen.getByTestId("sb-asset-AST-HERO-001-edit"));
      await screen.findByTestId("sb-asset-dialog");
      expect(screen.getByTestId("sb-asset-dialog-confirm")).toBeDisabled();
      fireEvent.change(screen.getByTestId("sb-asset-portrayal"), { target: { value: "Mới" } });
      expect(screen.getByTestId("sb-asset-impact")).toHaveTextContent("SH-001, SH-002");
      fireEvent.click(screen.getByTestId("sb-asset-dialog-confirm"));
      await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "AST-HERO-001", { portrayal: "Mới" }));
    });

    it("will not delete an asset a shot uses, and asks before deleting one nobody uses", async () => {
      const del = vi.spyOn(apiClient, "deleteStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-HERO-001-open"));
      expect(screen.getByTestId("sb-asset-AST-HERO-001-delete")).toBeDisabled();
      fireEvent.click(screen.getByTestId("sb-asset-AST-DISK-001-delete"));
      expect(await screen.findByTestId("sb-asset-delete-impact")).toHaveTextContent("Chưa shot nào dùng");
      expect(del).not.toHaveBeenCalled();
      fireEvent.click(screen.getByTestId("sb-asset-delete-dialog-confirm"));
      await waitFor(() => expect(del).toHaveBeenCalledWith("p1", "AST-DISK-001"));
    });

    it("lists the assets with the AI, on the system's AI or one outside", async () => {
      const gen = vi.spyOn(apiClient, "generateStoryboardAssets").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-assets-generate"));
      expect(await screen.findByTestId("sb-assets-dialog")).toHaveTextContent("Danh sách về “Chờ duyệt”");
      fireEvent.click(screen.getByTestId("sb-assets-dialog-send"));
      await waitFor(() => expect(gen).toHaveBeenCalledWith("p1", "", expect.objectContaining({ id: expect.any(String) })));
      fireEvent.click(await screen.findByTestId("sb-assets-generate"));
      gen.mockRejectedValue(new apiClient.ExternalPromptReady("PROMPT TÀI SẢN"));
      fireEvent.click(await screen.findByTestId("sb-assets-dialog-external"));
      expect(await screen.findByTestId("sb-assets-dialog-external-dialog-prompt")).toHaveTextContent("PROMPT TÀI SẢN");
      expect(gen).toHaveBeenLastCalledWith("p1", "", { external: { mode: "prompt" } });
    });

    it("shows the units and the detail state of each asset, and holds the draw until it has detail", async () => {
      const listed = baseState();
      listed.assets.items = listed.assets.items.map((a) =>
        a.asset_id === "AST-DISK-001" ? { ...a, has_detail: false, portrayal: "", units: ["VP-001", "VP-003"] } : a,
      );
      listed.assets = { ...listed.assets, ...detailCounts(listed.assets.items) };
      state = listed;
      const draw = vi.spyOn(apiClient, "drawStoryboardAsset").mockResolvedValue(state);
      renderPage();
      expect(await screen.findByTestId("sb-asset-AST-DISK-001-detail-state")).toHaveTextContent("Chưa có chi tiết");
      expect(screen.getByTestId("sb-asset-AST-DISK-001-units")).toHaveTextContent("VP-001");
      expect(screen.getByTestId("sb-asset-AST-DISK-001-units")).toHaveTextContent("VP-003");
      expect(screen.getByTestId("sb-asset-AST-DISK-001-portrayal")).toHaveTextContent("Chưa viết chi tiết");
      expect(screen.getByTestId("sb-asset-AST-OBSERVER-001-detail-state")).toHaveTextContent("Đã có chi tiết");
      const drawButton = screen.getByTestId("sb-asset-AST-DISK-001-draw");
      expect(drawButton).toBeDisabled();
      expect(drawButton).toHaveAttribute("title", "Viết chi tiết cho tài sản này trước khi vẽ");
      expect(screen.getByTestId("sb-asset-AST-DISK-001-draw-reason")).toHaveTextContent("Viết chi tiết cho tài sản này");
      fireEvent.click(drawButton);
      expect(draw).not.toHaveBeenCalled();
    });

    it("writes one asset's detail on the system's AI or through an AI outside, from its card", async () => {
      const gen = vi.spyOn(apiClient, "generateStoryboardAssetDetail").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-DISK-001-detail"));
      expect(await screen.findByTestId("sb-asset-AST-DISK-001-detail-dialog")).toHaveTextContent("chỉ đọc mục khái niệm");
      expect(screen.getByTestId("sb-asset-AST-DISK-001-detail-dialog-external")).toBeInTheDocument();
      fireEvent.change(screen.getByTestId("sb-asset-AST-DISK-001-detail-dialog-note"), { target: { value: " dịu hơn " } });
      fireEvent.click(screen.getByTestId("sb-asset-AST-DISK-001-detail-dialog-send"));
      await waitFor(() =>
        expect(gen).toHaveBeenCalledWith("p1", "AST-DISK-001", "dịu hơn", expect.objectContaining({ id: expect.any(String) })),
      );
      fireEvent.click(await screen.findByTestId("sb-asset-AST-DISK-001-detail"));
      gen.mockRejectedValue(new apiClient.ExternalPromptReady("PROMPT CHI TIẾT"));
      fireEvent.click(await screen.findByTestId("sb-asset-AST-DISK-001-detail-dialog-external"));
      expect(await screen.findByTestId("sb-asset-AST-DISK-001-detail-dialog-external-dialog-prompt")).toHaveTextContent("PROMPT CHI TIẾT");
      expect(gen).toHaveBeenLastCalledWith("p1", "AST-DISK-001", "", { external: { mode: "prompt" } });
    });

    describe("run card of the asset detail", () => {
      function withoutDetail(ids: string[]): StoryboardState {
        const s2 = baseState();
        s2.assets.items = s2.assets.items.map((a) => (ids.includes(a.asset_id) ? { ...a, has_detail: false, portrayal: "" } : a));
        s2.assets = { ...s2.assets, ...detailCounts(s2.assets.items) };
        return s2;
      }

      it("counts the assets with and without detail and writes every missing one on the system's AI", async () => {
        state = withoutDetail(["AST-OBSERVER-001", "AST-DISK-001"]);
        const gen = vi.spyOn(apiClient, "generateStoryboardAssetDetails").mockResolvedValue(state);
        renderPage();
        expect(await screen.findByTestId("sb-detail-counts")).toHaveTextContent("1/3 tài sản đã có chi tiết");
        expect(screen.getByTestId("sb-detail-generate")).toHaveTextContent("Viết chi tiết cho 2 tài sản");
        fireEvent.click(screen.getByTestId("sb-detail-generate"));
        expect(await screen.findByTestId("sb-detail-dialog")).toHaveTextContent("AST-OBSERVER-001, AST-DISK-001");
        fireEvent.change(screen.getByTestId("sb-detail-note"), { target: { value: " ngắn " } });
        fireEvent.click(screen.getByTestId("sb-detail-confirm"));
        await waitFor(() =>
          expect(gen).toHaveBeenCalledWith("p1", "ngắn", expect.objectContaining({ id: expect.any(String) })),
        );
      });

      it("has nothing to write once every asset has its detail", async () => {
        renderPage();
        expect(await screen.findByTestId("sb-detail-generate")).toBeDisabled();
        expect(screen.getByTestId("sb-detail-generate")).toHaveTextContent("Mọi tài sản đã có chi tiết");
        expect(screen.queryByTestId("sb-detail-queue")).toBeNull();
      });

      it("queues one AI item per asset without detail, in list order, and shows the queue counts", async () => {
        state = withoutDetail(["AST-OBSERVER-001", "AST-DISK-001"]);
        const request = { gateway: "http://gw", method: "POST", path: "/v1/projects/p1/storyboard/assets/AST-OBSERVER-001/detail", body: "{}" };
        const describe = vi
          .spyOn(apiClient, "generateStoryboardAssetDetail")
          .mockRejectedValue(new apiClient.ExternalRequestDescribed(request));
        queueAdd.mockResolvedValue({ ok: true, added: true });
        renderPage();
        expect(await screen.findByTestId("sb-detail-queue-counts")).toHaveTextContent("1 đang chạy · 2 chờ · 0 lỗi");
        const button = screen.getByTestId("sb-detail-queue");
        expect(button).toHaveTextContent("Chạy tất cả qua hàng đợi AI (2)");
        fireEvent.click(button);
        await waitFor(() => expect(queueAdd).toHaveBeenCalledTimes(2));
        expect(queueAdd.mock.calls.map((c) => c[0].label)).toEqual([
          "Chi tiết tài sản · AST-OBSERVER-001",
          "Chi tiết tài sản · AST-DISK-001",
        ]);
        expect(describe.mock.calls.map((c) => c[1])).toEqual(["AST-OBSERVER-001", "AST-DISK-001"]);
        expect(describe.mock.calls.every((c) => (c[3] as { external: { mode: string } }).external.mode === "describe")).toBe(true);
        expect(await screen.findByTestId("sb-detail-queue-note")).toHaveTextContent("Đã thêm 2 tài sản vào hàng đợi AI");
      });

      it("stops queueing at the asset the extension refuses and says which one", async () => {
        state = withoutDetail(["AST-OBSERVER-001", "AST-DISK-001"]);
        vi.spyOn(apiClient, "generateStoryboardAssetDetail").mockRejectedValue(
          new apiClient.ExternalRequestDescribed({ gateway: "g", method: "POST", path: "/p" }),
        );
        queueAdd.mockResolvedValueOnce({ ok: true, added: true }).mockResolvedValueOnce({ ok: false, message: "Extension không phản hồi" });
        renderPage();
        fireEvent.click(await screen.findByTestId("sb-detail-queue"));
        expect(await screen.findByTestId("sb-detail-queue-note")).toHaveTextContent("Dừng ở tài sản 2/2: Extension không phản hồi");
      });

      it("hides the AI button when the extension has no queue", async () => {
        state = withoutDetail(["AST-DISK-001"]);
        queueAvailable = false;
        renderPage();
        await screen.findByTestId("sb-detail-generate");
        expect(screen.queryByTestId("sb-detail-queue")).toBeNull();
      });
    });

    it("lets the Creator tick the units of an asset and sends them with the other changes", async () => {
      const edit = vi.spyOn(apiClient, "editStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-AST-HERO-001-open"));
      fireEvent.click(screen.getByTestId("sb-asset-AST-HERO-001-edit"));
      await screen.findByTestId("sb-asset-dialog");
      expect(screen.getByTestId("sb-asset-unit-VP-001")).toBeChecked();
      expect(screen.getByTestId("sb-asset-unit-VP-002")).not.toBeChecked();
      fireEvent.click(screen.getByTestId("sb-asset-unit-VP-003"));
      fireEvent.click(screen.getByTestId("sb-asset-unit-VP-002"));
      expect(screen.getByTestId("sb-asset-impact")).toHaveTextContent("Chưa vẽ");
      fireEvent.click(screen.getByTestId("sb-asset-dialog-confirm"));
      await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "AST-HERO-001", { units: ["VP-001", "VP-002", "VP-003"] }));
    });

    it("saves an asset with no detail yet and keeps its detail fields editable", async () => {
      const add = vi.spyOn(apiClient, "addStoryboardAsset").mockResolvedValue(state);
      renderPage();
      fireEvent.click(await screen.findByTestId("sb-asset-add"));
      await screen.findByTestId("sb-asset-dialog");
      fireEvent.change(screen.getByTestId("sb-asset-name"), { target: { value: "Probe" } });
      fireEvent.change(screen.getByTestId("sb-asset-concept"), { target: { value: "objects.probe" } });
      fireEvent.click(screen.getByTestId("sb-asset-unit-VP-002"));
      fireEvent.click(screen.getByTestId("sb-asset-dialog-confirm"));
      await waitFor(() =>
        expect(add).toHaveBeenCalledWith(
          "p1",
          expect.objectContaining({ name: "Probe", portrayal: "", units: ["VP-002"], must_not_imply: [] }),
        ),
      );
    });

    it("approves the list only when every asset is approved", async () => {
      renderPage();
      expect(await screen.findByTestId("sb-assets-approve")).toBeDisabled();
    });
  });

  describe("tab Shot", () => {
    it("folds approved shots, units and beats and opens them on demand", async () => {
      state = baseState();
      state.beats[0].units[0] = { ...state.beats[0].units[0] };
      renderPage();
      await openShotsTab();
      // VP-001 holds only an approved shot: the unit folds; VP-002 is half approved: its approved shot folds.
      expect(await screen.findByTestId("sb-unit-VP-001-open")).toBeInTheDocument();
      expect(screen.queryByTestId("sb-shot-SH-001")).toBeNull();
      expect(screen.getByTestId("sb-shot-SH-002-open")).toBeInTheDocument();
      expect(screen.getByTestId("sb-shot-SH-003")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-unit-VP-001-open"));
      expect(screen.getByTestId("sb-shot-SH-001-open")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-shot-SH-001-open"));
      expect(screen.getByTestId("sb-shot-SH-001-frame")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-shot-SH-001-fold"));
      await waitFor(() => expect(screen.queryByTestId("sb-shot-SH-001-frame")).toBeNull());
    });

    it("folds a beat once every unit is approved", async () => {
      const done = baseState();
      for (const u of done.beats[0].units) {
        u.shots = u.shots.map((s) => ({ ...s, status: "approved" as const, frame: { ...s.frame, stale: false }, missing_assets: [], problems: [] }));
        u.approved = true;
      }
      done.beats[0].approved = true;
      state = done;
      renderPage();
      await openShotsTab();
      expect(await screen.findByTestId("sb-beat-b1-open")).toHaveTextContent("Mở");
      expect(screen.queryByTestId("sb-unit-VP-001")).toBeNull();
      fireEvent.click(screen.getByTestId("sb-beat-b1-open"));
      expect(screen.getByTestId("sb-unit-VP-001-open")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-beat-b1-fold"));
      expect(screen.getByTestId("sb-beat-b1-open")).toBeInTheDocument();
    });

    it("shows a stale frame, missing assets and a frame error on the shot", async () => {
      const failing = baseState();
      failing.beats[0].units[2].shots[0] = {
        ...failing.beats[0].units[2].shots[0],
        frame: { drawn: false, stale: false, path: "", error: "Bộ dựng không chạy được" },
      };
      state = failing;
      renderPage();
      await openShotsTab();
      expect(await screen.findByTestId("sb-shot-SH-003-stale")).toHaveTextContent("Khung cũ");
      expect(screen.getByTestId("sb-shot-SH-003-frame")).toHaveAttribute(
        "src",
        expect.stringContaining("/v1/projects/p1/storyboard/frames/SH-003"),
      );
      expect(screen.getByTestId("sb-shot-SH-004-missing")).toHaveTextContent("AST-DISK-001");
      expect(screen.getByTestId("sb-shot-SH-004-missing-chip")).toHaveTextContent("Thiếu tài sản");
      expect(screen.getByTestId("sb-shot-SH-004-frame-error")).toHaveTextContent("Bộ dựng không chạy được");
      expect(screen.getByTestId("sb-shot-SH-004-frame-empty")).toHaveTextContent("Dựng khung lỗi");
      expect(screen.getByTestId("sb-shot-SH-004-approve")).toBeDisabled();
      expect(screen.getByTestId("sb-shot-SH-004-redraw")).toBeDisabled();
    });

    it("says so when the frame image cannot be loaded", async () => {
      renderPage();
      await openShotsTab();
      fireEvent.error(await screen.findByTestId("sb-shot-SH-003-frame"));
      expect(screen.getByTestId("sb-shot-SH-003-frame-empty")).toHaveTextContent("Không tải được ảnh khung");
    });

    it("blocks approving a shot whose frame is stale and approves one that can be", async () => {
      const approve = vi.spyOn(apiClient, "approveStoryboardShot").mockResolvedValue(state);
      const clean = baseState();
      clean.beats[0].units[1].shots[1] = { ...clean.beats[0].units[1].shots[1], frame: { drawn: true, stale: false, path: "storyboard/SH-003.png", error: "" } };
      state = clean;
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-shot-SH-003-approve"));
      await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", "VP-002", "SH-003"));
    });

    it("explains why a shot with a stale frame cannot be approved yet", async () => {
      renderPage();
      await openShotsTab();
      const approve = await screen.findByTestId("sb-shot-SH-003-approve");
      expect(approve).toBeDisabled();
      expect(approve).toHaveAttribute("title", expect.stringContaining("Khung đã cũ"));
    });

    it("takes an approval back and keeps the shot open to edit", async () => {
      const unapprove = vi.spyOn(apiClient, "unapproveStoryboardShot").mockResolvedValue(
        (() => {
          const next = baseState();
          next.beats[0].units[1].shots[0] = { ...next.beats[0].units[1].shots[0], status: "pending" };
          return next;
        })(),
      );
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-shot-SH-002-open"));
      fireEvent.click(screen.getByTestId("sb-shot-SH-002-unapprove"));
      await waitFor(() => expect(unapprove).toHaveBeenCalledWith("p1", "VP-002", "SH-002"));
      expect(await screen.findByTestId("sb-shot-SH-002-approve")).toBeInTheDocument();
    });

    it("approves every shot that can be approved", async () => {
      const all = vi.spyOn(apiClient, "approveAllStoryboard").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      // SH-003 has a stale frame and SH-004 misses an asset: nothing can be approved.
      expect(await screen.findByTestId("sb-approve-all")).toBeDisabled();
      expect(all).not.toHaveBeenCalled();
    });

    it("edits a shot with closed lists for positions and camera views and sends only what changed", async () => {
      const edit = vi.spyOn(apiClient, "editStoryboardShot").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-shot-SH-003-edit"));
      await screen.findByTestId("sb-shot-dialog");
      const positions = Array.from(
        screen.getByTestId("sb-shot-subject-position").querySelectorAll("option"),
      ).map((o) => (o as HTMLOptionElement).value);
      expect(positions).toEqual(["", ...apiClient.SHOT_POSITIONS]);
      const cameras = Array.from(screen.getByTestId("sb-shot-camera").querySelectorAll("option")).map(
        (o) => (o as HTMLOptionElement).value,
      );
      expect(cameras).toEqual(["", ...apiClient.SHOT_CAMERA_VIEWS]);
      expect(screen.getByTestId("sb-shot-impact")).toHaveTextContent("Chưa duyệt");
      expect(screen.getByTestId("sb-shot-dialog-confirm")).toBeDisabled();
      fireEvent.change(screen.getByTestId("sb-shot-camera"), { target: { value: "close" } });
      fireEvent.change(screen.getByTestId("sb-shot-text"), { target: { value: "Góc nhìn A\nGóc nhìn B" } });
      fireEvent.click(screen.getByTestId("sb-shot-dialog-confirm"));
      await waitFor(() =>
        expect(edit).toHaveBeenCalledWith("p1", "VP-002", "SH-003", {
          composition: {
            subject: "AST-HERO-001",
            subject_position: "lower_left",
            main_object: "",
            main_object_position: "",
            camera_view: "close",
          },
          on_screen_text: ["Góc nhìn A", "Góc nhìn B"],
        }),
      );
    });

    it("warns that editing an approved shot sends the unit back to review", async () => {
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-shot-SH-002-open"));
      fireEvent.click(screen.getByTestId("sb-shot-SH-002-edit"));
      expect(await screen.findByTestId("sb-shot-impact")).toHaveTextContent("unit VP-002 mất trạng thái đã duyệt");
    });

    it("adds a shot at the end of a unit", async () => {
      const add = vi.spyOn(apiClient, "addStoryboardShot").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-unit-VP-003-add-shot"));
      await screen.findByTestId("sb-shot-dialog");
      fireEvent.change(screen.getByTestId("sb-shot-purpose"), { target: { value: "Cho thấy vành" } });
      fireEvent.change(screen.getByTestId("sb-shot-subject"), { target: { value: "AST-HERO-001" } });
      fireEvent.change(screen.getByTestId("sb-shot-subject-position"), { target: { value: "center" } });
      fireEvent.change(screen.getByTestId("sb-shot-camera"), { target: { value: "medium" } });
      fireEvent.change(screen.getByTestId("sb-shot-action"), { target: { value: "Vành xoáy" } });
      fireEvent.click(screen.getByTestId("sb-shot-dialog-confirm"));
      await waitFor(() => expect(add).toHaveBeenCalled());
      expect(add.mock.calls[0].slice(0, 2)).toEqual(["p1", "VP-003"]);
      expect(add.mock.calls[0][2]).toMatchObject({
        shot_purpose: "Cho thấy vành",
        action_summary: "Vành xoáy",
        composition: { subject: "AST-HERO-001", subject_position: "center", main_object: "", camera_view: "medium" },
      });
    });

    it("asks before deleting a shot, and will not delete the last shot of a unit", async () => {
      const del = vi.spyOn(apiClient, "deleteStoryboardShot").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      expect(await screen.findByTestId("sb-shot-SH-004-delete")).toBeDisabled();
      fireEvent.click(screen.getByTestId("sb-shot-SH-003-delete"));
      expect(await screen.findByTestId("sb-shot-SH-003-delete-impact")).toHaveTextContent("Xoá shot SH-003");
      expect(del).not.toHaveBeenCalled();
      fireEvent.click(screen.getByTestId("sb-shot-SH-003-delete-dialog-confirm"));
      await waitFor(() => expect(del).toHaveBeenCalledWith("p1", "VP-002", "SH-003"));
    });

    it("redraws a frame on the system's drawer", async () => {
      const redraw = vi.spyOn(apiClient, "redrawStoryboardFrame").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-shot-SH-003-redraw"));
      await waitFor(() =>
        expect(redraw).toHaveBeenCalledWith("p1", "VP-002", "SH-003", expect.objectContaining({ id: expect.any(String) })),
      );
    });

    it("writes one unit's shots on the system's AI, or through an outside AI", async () => {
      state = baseState({
        assets: allApprovedAssets(),
      });
      const gen = vi.spyOn(apiClient, "generateStoryboardUnit").mockResolvedValue(state);
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-unit-VP-004-ai"));
      expect(await screen.findByTestId("sb-unit-VP-004-ai-dialog")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-unit-VP-004-ai-dialog-send"));
      await waitFor(() =>
        expect(gen).toHaveBeenCalledWith("p1", "VP-004", "", expect.objectContaining({ id: expect.any(String) })),
      );
      fireEvent.click(await screen.findByTestId("sb-unit-VP-004-ai"));
      gen.mockRejectedValue(new apiClient.ExternalPromptReady("PROMPT UNIT"));
      fireEvent.click(await screen.findByTestId("sb-unit-VP-004-ai-dialog-external"));
      expect(await screen.findByTestId("sb-unit-VP-004-ai-dialog-external-dialog-prompt")).toHaveTextContent("PROMPT UNIT");
      expect(gen).toHaveBeenLastCalledWith("p1", "VP-004", "", { external: { mode: "prompt" } });
    });

    it("warns in the dialog that rewriting a unit sends its approved shots back", async () => {
      state = baseState({
        assets: allApprovedAssets(),
      });
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-unit-VP-002-ai"));
      expect(await screen.findByTestId("sb-unit-VP-002-ai-impact")).toHaveTextContent("1 shot đã duyệt của unit về “Chưa duyệt”");
    });

    it("shows the whole film as a grid, read only, and opens a shot from its frame", async () => {
      renderPage();
      await openShotsTab();
      fireEvent.click(await screen.findByTestId("sb-view-toggle"));
      expect(screen.getByTestId("sb-view-chip")).toHaveTextContent("Xem dạng lưới");
      expect(screen.getByTestId("sb-sheet")).toBeInTheDocument();
      expect(screen.queryByTestId("sb-beats")).toBeNull();
      fireEvent.click(screen.getByTestId("sb-sheet-SH-001"));
      expect(screen.queryByTestId("sb-sheet")).toBeNull();
      expect(screen.getByTestId("sb-shot-SH-001-frame")).toBeInTheDocument();
    });
  });

  describe("many lock blockers", () => {
    const many = Array.from({ length: 67 }, (_, i) => `Điều kiện số ${i + 1} chưa đạt`);

    it("keeps the bottom bar compact, the banner to three lines and the review tab complete", async () => {
      state = baseState({ blockers: many });
      renderPage();
      const banner = await screen.findByTestId("sb-blockers-banner");
      expect(banner.querySelectorAll("li")).toHaveLength(3);
      expect(banner).toHaveTextContent("Điều kiện số 3 chưa đạt");
      expect(banner).not.toHaveTextContent("Điều kiện số 4 chưa đạt");
      expect(screen.getByTestId("sb-blockers-more")).toHaveTextContent("và 64 điều kiện khác");
      const hint = screen.getByTestId("sb-hint-blockers");
      expect(hint).toHaveTextContent("Chưa chốt được: 67 điều kiện chưa đạt");
      expect(hint).not.toHaveTextContent("Điều kiện số 1 ");
      expect(hint.getAttribute("title")?.split("\n")).toHaveLength(67);
      fireEvent.click(screen.getByTestId("sb-tab-review"));
      const items = (await screen.findByTestId("sb-blockers")).querySelectorAll("li");
      expect(items).toHaveLength(67);
      expect(items[66]).toHaveTextContent("Điều kiện số 67 chưa đạt");
    });
  });


  describe("tab Rà soát và chốt", () => {
    async function openReview() {
      fireEvent.click(await screen.findByTestId("sb-tab-review"));
    }

    it("keeps the lock closed and says why", async () => {
      renderPage();
      await openReview();
      const lock = await screen.findByTestId("sb-lock");
      expect(lock).toBeDisabled();
      expect(lock).toHaveAttribute("title", expect.stringContaining("1 tài sản chưa duyệt (AST-1)"));
      expect(screen.getByTestId("sb-blockers")).toHaveTextContent("2 shot chưa duyệt (SH-1, SH-2)");
      expect(screen.getByTestId("sb-criteria")).toHaveTextContent("✘Mọi shot đã duyệt");
      expect(screen.getByTestId("sb-criteria")).toHaveTextContent("2 khung cần vẽ lại");
      expect(screen.getByTestId("sb-pending-units")).toHaveTextContent("VP-002, VP-003");
    });

    it("accepts a machine warning in one click but not a blocking check", async () => {
      const accept = vi.spyOn(apiClient, "acceptStoryboardFinding").mockResolvedValue(state);
      renderPage();
      await openReview();
      expect(await screen.findByTestId("sb-check-row-m-shot-frame")).not.toHaveTextContent("Chấp nhận");
      fireEvent.click(screen.getByTestId("sb-accept-m-asset-unused"));
      await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "m-asset-unused", ""));
    });

    it("applies a fix on a shot finding, never on a unit finding, and opens the shot", async () => {
      const apply = vi.spyOn(apiClient, "applyStoryboardFinding").mockResolvedValue(state);
      renderPage();
      await openReview();
      expect(screen.queryByTestId("sb-finding-apply-f2")).toBeNull();
      fireEvent.click(await screen.findByTestId("sb-finding-apply-f1"));
      await waitFor(() => expect(apply).toHaveBeenCalledWith("p1", "f1"));
      fireEvent.click(screen.getByTestId("sb-finding-open-f1"));
      expect(await screen.findByTestId("sb-shot-SH-003")).toBeInTheDocument();
    });

    it("runs the AI review, on the system's AI or through an outside AI", async () => {
      const review = vi.spyOn(apiClient, "reviewStoryboard").mockResolvedValue(state);
      renderPage();
      await openReview();
      fireEvent.click(await screen.findByTestId("sb-review-run"));
      fireEvent.click(await screen.findByTestId("sb-review-dialog-send"));
      await waitFor(() => expect(review).toHaveBeenCalledWith("p1", "", expect.objectContaining({ id: expect.any(String) })));
    });

    it("locks v1 and tells what the lock writes", async () => {
      state = baseState({ can_lock: true, blockers: [], shots_approved: 4 });
      const lock = vi.spyOn(apiClient, "lockStoryboard").mockResolvedValue(baseState({ locked_version: 1 }));
      renderPage();
      await openReview();
      fireEvent.click(await screen.findByTestId("sb-lock"));
      expect(await screen.findByTestId("sb-lock-first")).toBeInTheDocument();
      fireEvent.click(screen.getByTestId("sb-lock-confirm"));
      await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", ""));
    });

    it("locks v2 with the orange button and lists what changed", async () => {
      state = baseState({ locked_version: 1, next_version: 2, draft: ["SH-003"], can_lock: true, blockers: [] });
      const lock = vi.spyOn(apiClient, "lockStoryboard").mockResolvedValue(baseState({ locked_version: 2 }));
      renderPage();
      await openReview();
      fireEvent.click(await screen.findByTestId("sb-lock"));
      expect(await screen.findByTestId("sb-lock-draft")).toHaveTextContent("SH-003");
      fireEvent.click(screen.getByTestId("sb-lock-confirm"));
      await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", ""));
    });

    it("shows the locked state and opens a locked version", async () => {
      state = baseState({ locked_version: 2, next_version: 3, locked_at: "2026-02-01T10:00:00Z", locked_note: "ổn", draft: [] });
      const version = vi.spyOn(apiClient, "getStoryboardVersion").mockResolvedValue({
        version: 1,
        sequence_version: 1,
        note: "",
        locked_at: "2026-01-01T00:00:00Z",
        document: { shots: [{ shot_id: "SH-001" }] },
      });
      renderPage();
      await openReview();
      expect(await screen.findByTestId("sb-locked")).toHaveTextContent("Đã chốt storyboard v2");
      fireEvent.click(screen.getByTestId("sb-version-1"));
      await waitFor(() => expect(version).toHaveBeenCalledWith("p1", 1));
      expect(await screen.findByTestId("sb-version-dialog")).toHaveTextContent("SH-001");
    });

    it("shows why a locked version could not be read", async () => {
      state = baseState({ locked_version: 1, next_version: 2 });
      vi.spyOn(apiClient, "getStoryboardVersion").mockRejectedValue(new apiClient.ApiError("Không có bản chốt", undefined, undefined, 404));
      renderPage();
      await openReview();
      fireEvent.click(await screen.findByTestId("sb-version-1"));
      expect(await screen.findByTestId("sb-version-error")).toHaveTextContent("Không có bản chốt");
    });
  });

  it("shows the run in flight at the top with a stop button", async () => {
    state = baseState({
      assets: allApprovedAssets(),
    });
    let signal: AbortSignal | undefined;
    vi.spyOn(apiClient, "generateStoryboard").mockImplementation(
      (_p, _n, call) =>
        new Promise((_, reject) => {
          signal = call?.signal;
          signal?.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
        }),
    );
    renderPage();
    fireEvent.click(await screen.findByTestId("sb-generate"));
    fireEvent.click(await screen.findByTestId("sb-generate-confirm"));
    expect(await screen.findByTestId("sb-running")).toHaveTextContent("AI đang viết shot các unit còn trống");
    fireEvent.click(screen.getByTestId("sb-stop"));
    expect(signal?.aborted).toBe(true);
    await waitFor(() => expect(screen.queryByTestId("sb-running")).toBeNull());
  });

  it("shows the server's reason when an action fails", async () => {
    vi.spyOn(apiClient, "approveStoryboardAsset").mockRejectedValue(new apiClient.ApiError("Tài sản chưa vẽ", "x", undefined, 409));
    renderPage();
    fireEvent.click(await screen.findByTestId("sb-asset-AST-OBSERVER-001-approve"));
    expect(await screen.findByTestId("sb-error")).toHaveTextContent("Tài sản chưa vẽ");
  });
});
