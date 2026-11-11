import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { VisualConceptStepPage } from "../../src/pages/VisualConceptStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type {
  VisualConceptState,
  VisualConceptUnit,
  VisualConceptUnitKey,
} from "../../src/api/client";
import type { Project } from "../../src/types";

const EMPTY_REF_DRAWING: apiClient.ReferenceDrawing = {
  medium: "",
  shape: "",
  outline: "",
  fill: "",
  shading: "",
  light: "",
  depth: "",
  perspective: "",
  texture: "",
};

function Where() {
  const l = useLocation();
  return <span data-testid="where">{l.pathname}</span>;
}

const TITLES: [VisualConceptUnitKey, string][] = [
  ["creative_direction", "Định hướng sáng tạo"],
  ["color_system", "Bảng màu và tương phản"],
  ["visual_language", "Hình dạng, nét, ánh sáng, chiều sâu, bề mặt, layout, bố cục"],
  ["representation_rules", "Nhân vật, vật thể, khái niệm vô hình, hiện tượng khoa học"],
  ["typography", "Chữ, nhãn, chú thích và công thức"],
  ["motion_language", "Nguyên tắc chuyển động"],
  ["do_not_use", "Cách thể hiện không được dùng"],
  ["feasibility", "Kiểm tra khả thi"],
];

function unit(i: number, over: Partial<VisualConceptUnit> = {}): VisualConceptUnit {
  const [key, title] = TITLES[i];
  return {
    key,
    number: i + 1,
    title,
    status: "pending",
    written: false,
    content: null,
    creator_fields: [],
    reads: ["Beat sheet"],
    stale: false,
    stale_sources: [],
    problems: [],
    redo_count: 0,
    ...over,
  };
}

const colour = {
  background: "#101010",
  palette: [
    { token: "ink", hex: "#ffffff", role: "chữ", name: "trắng" },
    { token: "accent", hex: "#e1ad01", role: "nhấn", name: "vàng" },
  ],
  journey: [{ beat_keys: ["b1", "b2"], mood: "ấm", accent_token: "accent", background_token: "" }],
  contrast: { text_min: 4.5, text_token: "ink", subject_vs_bg: "sáng hơn", accent_per_frame: 1 },
  exceptions: [],
  examples: [],
};

function baseState(over: Partial<VisualConceptState> = {}): VisualConceptState {
  return {
    input: {
      script_version: 2,
      facts_count: 3,
      research_skipped: false,
      beats: [
        { key: "b1", number: 1, fn: "Hook", title: "Mở", dur: 20 },
        { key: "b2", number: 2, fn: "Payoff", title: "Đóng", dur: 40 },
      ],
      terms_count: 2,
      refs_count: 1,
    },
    input_view: [{ key: "brief", label: "Creative brief", text: "Khán giả: phổ thông" }],
    refs: [
      {
        ref_id: "R01",
        style_name: "Sáng xuyên lớp",
        intent: "nền tối ấm",
        tags: ["ánh sáng"],
        source: { url: "", title: "", kind: "" },
        core_rules: [],
        drawing: EMPTY_REF_DRAWING,
        color_rules: { background: "", subject: "", accents: "", note: "" },
        composition: { structure: "", optional_devices: [], layering: "" },
        avoid: [],
        prompt_template: "",
        negative_prompt: "",
        picked: true,
      },
      {
        ref_id: "R02",
        style_name: "Sơ đồ",
        intent: "nét mảnh",
        tags: [],
        source: { url: "", title: "", kind: "" },
        core_rules: [],
        drawing: EMPTY_REF_DRAWING,
        color_rules: { background: "", subject: "", accents: "", note: "" },
        composition: { structure: "", optional_devices: [], layering: "" },
        avoid: [],
        prompt_template: "",
        negative_prompt: "",
        picked: false,
      },
    ],
    units: [
      unit(0, {
        status: "approved",
        written: true,
        content: { style_name: "Phẳng sáng", abstraction_level: "hybrid", tone: ["ấm"] },
      }),
      unit(1, {
        written: true,
        content: colour,
        reads: ["Beat sheet", "Mô tả tham chiếu"],
        creator_fields: ["background"],
      }),
      unit(2, {
        written: true,
        content: { shape_rules: [] },
        problems: ["Chưa có quy tắc đường nét"],
      }),
      ...[3, 4, 5, 6, 7].map((i) => unit(i)),
    ],
    approved_count: 1,
    checks: [
      {
        id: "m-contrast",
        label: "Chữ đạt 4.5:1 ở 1/1 đoạn hành trình màu",
        ok: true,
        detail: "",
        accepted: false,
        why: "",
      },
      {
        id: "m-terms",
        label: "0/2 thuật ngữ của kịch bản có cách vẽ ở phần 4",
        ok: false,
        detail: "Chưa có cách vẽ: enzyme.",
        accepted: false,
        why: "",
      },
    ],
    fonts_error: "rendering không trả được danh sách font: connection refused",
    fonts: [],
    review: {
      reviewed: true,
      rounds: 1,
      running: false,
      notes: ["Ổn"],
      open_count: 1,
      findings: [
        {
          id: "f1",
          group: "accuracy",
          group_label: "Độ chính xác khoa học",
          unit_key: "typography",
          unit_number: 5,
          unit_title: "Chữ",
          field: "captions",
          text: "Chú thích mơ hồ",
          detail: "d",
          fix: "Chỉ cho số liệu",
          state: "open",
          why: "",
        },
        {
          id: "f2",
          group: "feasibility",
          group_label: "Khả thi",
          unit_key: "feasibility",
          unit_number: 8,
          unit_title: "Khả thi",
          field: "",
          text: "Glow nặng",
          detail: "",
          fix: "",
          state: "fixed",
          why: "",
        },
      ],
    },
    criteria: [
      {
        label: "Quy tắc đủ rõ để các bước sau không tự tạo phong cách khác",
        ok: false,
        detail: "1/8 phần đã duyệt",
      },
      { label: "Các khái niệm quan trọng có cách biểu diễn phù hợp", ok: false, detail: "x" },
      { label: "Không có quy tắc hình dễ gây hiểu lầm khoa học", ok: false, detail: "x" },
      { label: "Phong cách khả thi với nguồn lực sản xuất", ok: false, detail: "x" },
      { label: "Được người phụ trách sáng tạo duyệt", ok: false, detail: "x" },
    ],
    blockers: ["Duyệt hết 8 phần, không phần nào cần làm lại."],
    can_lock: false,
    versions: [],
    locked_version: 0,
    next_version: 1,
    draft: [],
    model: "",
    default_model: "ollama",
    models: [{ id: "ollama", label: "Ollama", local: true }],
    ...over,
  };
}

let state: VisualConceptState;

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
  vi.spyOn(apiClient, "getVisualConcept").mockImplementation(async () => state);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/visual-concept"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/visual-concept" element={<VisualConceptStepPage />} />
              <Route path="*" element={<Where />} />
            </Routes>
          </ProjectFlowProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("VisualConceptStepPage", () => {
  it("says to lock the script first when step 7 has no locked version", async () => {
    vi.spyOn(apiClient, "getVisualConcept").mockRejectedValue(
      new apiClient.ApiError("Chốt kịch bản ở bước 7 trước", "script_not_locked", undefined, 409),
    );
    renderPage();
    expect(await screen.findByTestId("concept-not-open")).toHaveTextContent(
      "Chốt kịch bản ở bước 7 trước",
    );
    fireEvent.click(screen.getByTestId("concept-go-lock"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/lock");
  });

  it("is step 8 of the visual phase, with the head card, the references and the eight parts", async () => {
    renderPage();
    expect(await screen.findByTestId("concept-head-card")).toBeInTheDocument();
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Hình ảnh · Bước 8/23");
    expect(screen.getByTestId("rail-step-8")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("concept-head-count")).toHaveTextContent(
      "1/8 phần đã duyệt · 1 vấn đề cần xử lý",
    );
    expect(screen.getByTestId("concept-check-m-terms")).toHaveTextContent("! 0/2 thuật ngữ");
    expect(screen.getByTestId("concept-fonts-error")).toHaveTextContent("rendering");
    expect(within(screen.getByTestId("concept-timeline")).getByText("B1 Hook")).toBeInTheDocument();
    for (const [key] of TITLES)
      expect(screen.getByTestId(`concept-unit-${key}`)).toBeInTheDocument();
    // An approved part is folded to one line.
    expect(screen.getByTestId("concept-unit-creative_direction")).toHaveTextContent(
      "Phẳng sáng · mức trừu tượng: hybrid · ấm",
    );
    expect(screen.getByTestId("concept-unit-creative_direction-open")).toBeInTheDocument();
    // A part with problems shows them and cannot be approved.
    expect(screen.getByTestId("concept-unit-visual_language-problems")).toHaveTextContent(
      "Chưa có quy tắc đường nét",
    );
    expect(screen.getByTestId("concept-unit-visual_language-approve")).toBeDisabled();
    expect(screen.getByTestId("concept-unit-color_system")).toHaveTextContent("· bạn sửa");
    expect(screen.getByTestId("concept-generate")).toHaveTextContent("AI viết 5 phần còn trống");
  });

  it("approves a part and writes the empty parts with the system's AI", async () => {
    const approve = vi
      .spyOn(apiClient, "approveVisualConceptUnit")
      .mockResolvedValue(baseState({ approved_count: 2 }));
    const generate = vi.spyOn(apiClient, "generateVisualConcept").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("concept-unit-color_system-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("p1", "color_system"));
    fireEvent.click(screen.getByTestId("concept-generate"));
    await waitFor(() =>
      expect(generate).toHaveBeenCalledWith(
        "p1",
        "",
        false,
        expect.objectContaining({ signal: expect.anything() }),
      ),
    );
  });

  it("edits a part through a form of its fields and sends only what changed", async () => {
    const edit = vi.spyOn(apiClient, "editVisualConceptUnit").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("concept-unit-color_system-edit"));
    const dialog = await screen.findByTestId("concept-unit-color_system-edit-dialog");
    fireEvent.change(
      within(dialog).getByTestId("concept-unit-color_system-edit-dialog-background"),
      { target: { value: "#000000" } },
    );
    fireEvent.click(screen.getByTestId("concept-unit-color_system-edit-dialog-save"));
    await waitFor(() =>
      expect(edit).toHaveBeenCalledWith("p1", "color_system", { background: "#000000" }),
    );
  });

  it("asks before changing the picks once a part that reads them is written", async () => {
    const setRefs = vi.spyOn(apiClient, "setVisualConceptRefs").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("concept-ref-R02"));
    const confirm = await screen.findByTestId("concept-refs-confirm");
    expect(confirm).toHaveTextContent("Phần 2 · Bảng màu và tương phản");
    expect(setRefs).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("concept-refs-confirm-save"));
    await waitFor(() => expect(setRefs).toHaveBeenCalledWith("p1", ["R01", "R02"]));
  });

  it("handles the review: open findings, a failing check, the criteria and the lock", async () => {
    const apply = vi.spyOn(apiClient, "applyVisualConceptFinding").mockResolvedValue(baseState());
    const accept = vi.spyOn(apiClient, "acceptVisualConceptFinding").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("concept-tab-review"));
    expect(screen.getByTestId("concept-finding-f1")).toHaveTextContent(
      "Độ chính xác khoa học · phần 5",
    );
    expect(screen.getByTestId("concept-finding-state-f2")).toHaveTextContent("Đã sửa");
    expect(screen.getByTestId("concept-lock")).toBeDisabled();
    expect(screen.getByTestId("concept-blockers")).toHaveTextContent("Duyệt hết 8 phần");

    fireEvent.click(screen.getByTestId("concept-finding-apply-f1"));
    await waitFor(() => expect(apply).toHaveBeenCalledWith("p1", "f1"));

    fireEvent.click(screen.getByTestId("concept-accept-m-terms"));
    fireEvent.change(await screen.findByTestId("concept-accept-why"), {
      target: { value: "thuật ngữ chỉ nói" },
    });
    fireEvent.click(screen.getByTestId("concept-accept-save"));
    await waitFor(() => expect(accept).toHaveBeenCalledWith("p1", "m-terms", "thuật ngữ chỉ nói"));

    fireEvent.click(screen.getByTestId("concept-finding-open-f1"));
    await waitFor(() => expect(screen.getByTestId("concept-unit-typography")).toBeInTheDocument());
  });

  it("locks v2 with the changes listed, and opens the next step once locked", async () => {
    state = baseState({
      can_lock: true,
      blockers: [],
      locked_version: 1,
      next_version: 2,
      draft: ["Phần 3 · Hình dạng"],
      versions: [
        { version: 1, script_version: 2, note: "Chốt lần đầu", locked_at: "2026-11-10T00:00:00Z" },
      ],
    });
    const lock = vi
      .spyOn(apiClient, "lockVisualConcept")
      .mockResolvedValue({ ...state, draft: [], locked_version: 2, next_version: 3 });
    renderPage();
    fireEvent.click(await screen.findByTestId("concept-tab-review"));
    fireEvent.click(screen.getByTestId("concept-lock"));
    expect(await screen.findByTestId("concept-lock-draft")).toHaveTextContent("Phần 3 · Hình dạng");
    fireEvent.click(screen.getByTestId("concept-lock-confirm"));
    await waitFor(() => expect(lock).toHaveBeenCalledWith("p1", ""));
    await waitFor(() => expect(screen.getByTestId("concept-locked")).toHaveTextContent("v2"));
    fireEvent.click(screen.getByTestId("concept-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/coming/9");
  });
});
