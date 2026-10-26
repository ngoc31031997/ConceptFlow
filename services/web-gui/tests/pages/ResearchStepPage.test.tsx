import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { ResearchStepPage } from "../../src/pages/ResearchStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { QuestionCoverage, ResearchFact, ResearchQuestion, ResearchState } from "../../src/api/client";
import type { Project } from "../../src/types";
import { installFakeAiBridge } from "../helpers/fakeAiBridge";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function fact(id: string, over: Partial<ResearchFact> = {}): ResearchFact {
  const n = Number(id.slice(1));
  return {
    id,
    question: "Q1",
    claimVi: `Dữ kiện số ${n}`,
    type: "fact",
    sourceTitle: `Nguồn ${n}`,
    publisher: "Wikipedia",
    year: "2025",
    url: `https://example.org/${n}`,
    quote: `Đoạn trích số ${n}`,
    confidence: "high",
    note: "",
    must: false,
    keep: true,
    link: { state: "ok", why: "", nearest: "", checkedAt: "" },
    mine: [],
    ...over,
  };
}

function question(id: string, text: string, over: Partial<ResearchQuestion> = {}): ResearchQuestion {
  return { id, text, mine: false, stale: false, ...over };
}

function coverageOf(id: string, sources: number, weak: boolean, badFacts: string[] = []): QuestionCoverage {
  return { question: id, sources, badFacts, weak };
}

function twelveFacts(): ResearchFact[] {
  const facts: ResearchFact[] = [];
  for (let i = 1; i <= 12; i += 1) facts.push(fact(`F${String(i).padStart(2, "0")}`));
  facts[7] = fact("F08", { link: { state: "bad", why: "trang tải được nhưng không có đoạn trích này", nearest: "Females leave the fig carrying pollen", checkedAt: "" } });
  facts[1] = fact("F02", { link: { state: "na", why: "trang trả phí hoặc cần đăng nhập (HTTP 403)", nearest: "", checkedAt: "" } });
  facts[11] = fact("F12", { url: "", quote: "", sourceTitle: "", publisher: "", year: "", type: "unknown", link: { state: "none", why: "", nearest: "", checkedAt: "" } });
  return facts;
}

function baseState(over: Partial<ResearchState> = {}): ResearchState {
  return {
    gate: { status: "pending", skipped: false, resolved: false },
    questions: [question("Q1", "Quả sung có phải là hoa không?"), question("Q2", "Ong nào thụ phấn cho sung?")],
    facts: twelveFacts(),
    pastes: [{ n: 1, at: "2026-10-18T07:05:00Z", added: ["F01", "F02"], updated: [], rejected: [{ row: 13, reason: "dữ kiện F13 thiếu trường type" }] }],
    coverage: [coverageOf("Q1", 2, false), coverageOf("Q2", 1, true, ["F08"])],
    briefApproved: true,
    questionsStale: false,
    idea: { centralQuestion: "Vì sao ong chết trong quả sung?", curiosityPoints: ["Ong vào quả bằng đường nào?"], scope: "Sung và ong thụ phấn" },
    brief: {
      audience: "Người xem phổ thông",
      durationSeconds: 270,
      tone: "Tò mò",
      learningObjectives: ["Hiểu vì sao ong chết trong quả"],
      avoid: ["Nói ong bị ăn thịt"],
    },
    ...over,
  };
}

let state: ResearchState;
let project: Project;
let impacts: Record<string, string[]>;

beforeEach(() => {
  state = baseState();
  project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 2, run_state: "idle", video_output_mode: "long" } as Project;
  impacts = {};
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
  vi.spyOn(apiClient, "getResearch").mockImplementation(async () => state);
  vi.spyOn(apiClient, "getResearchPrompt").mockResolvedValue("PROMPT NGHIÊN CỨU\nQ1. Quả sung có phải là hoa không?");
  vi.spyOn(apiClient, "getFollowUpPrompt").mockResolvedValue("PROMPT BỔ SUNG\nQ2. Ong nào thụ phấn cho sung?");
  vi.spyOn(apiClient, "getFactImpact").mockImplementation(async (_p, id) => impacts[id] ?? []);
  vi.spyOn(apiClient, "getUnitImpact").mockImplementation(async (_p, ref) => impacts[ref.key] ?? []);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/research"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/research" element={<ResearchStepPage />} />
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
  await screen.findByTestId("research-facts-table");
}

describe("ResearchStepPage", () => {
  it("shows the five sections with the questions, the prompt, the paste log, the facts and the coverage", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Nghiên cứu và kiểm chứng");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 3/21");
    expect(screen.getByTestId("rail-step-3")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("question-input-Q1")).toHaveValue("Quả sung có phải là hoa không?");
    await waitFor(() => expect(screen.getByTestId("research-prompt-preview")).toHaveTextContent("PROMPT NGHIÊN CỨU"));
    expect(screen.getByTestId("research-paste-log-1")).toHaveTextContent("Đọc được 2 dữ kiện: thêm F01, F02. Bỏ 1 dòng: dòng 13 (dữ kiện F13 thiếu trường type).");
    for (let i = 1; i <= 12; i += 1) expect(screen.getByTestId(`fact-row-F${String(i).padStart(2, "0")}`)).toBeInTheDocument();
    expect(screen.getByTestId("coverage-chip-Q1")).toHaveTextContent("✓ 2 nguồn khớp");
    expect(screen.getByTestId("coverage-chip-Q2")).toHaveTextContent("1 nguồn khớp · cần thêm 1");
    expect(screen.getByTestId("coverage-bad-Q2")).toHaveTextContent("F08 không khớp");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt · 12 dữ kiện");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("9 khớp, 1 không khớp, 1 không tải được, 1 không có nguồn");
  });

  it("shows each link check with its reason and the nearest sentence of a page that lacks the quote", async () => {
    await ready();
    expect(screen.getByTestId("fact-link-F01")).toHaveAttribute("data-state", "ok");
    expect(screen.getByTestId("fact-link-F02")).toHaveTextContent("? không tải được");
    expect(screen.getByTestId("fact-row-F02")).toHaveTextContent("trang trả phí hoặc cần đăng nhập (HTTP 403)");
    expect(screen.getByTestId("fact-link-F08")).toHaveTextContent("✗ không khớp");
    expect(screen.getByTestId("fact-nearest-F08")).toHaveTextContent("Females leave the fig carrying pollen");
    expect(screen.getByTestId("fact-link-F12")).toHaveTextContent("– không có nguồn");
    expect(within(screen.getByTestId("fact-row-F01")).getByRole("link")).toHaveAttribute("rel", "noopener noreferrer");
  });

  it("filters the table by the result of the link check and by dropped facts", async () => {
    state = baseState({ facts: [...twelveFacts().slice(0, 11), fact("F12", { keep: false })] });
    await ready();
    fireEvent.click(screen.getByTestId("fact-filter-bad"));
    expect(screen.getByTestId("fact-row-F08")).toBeInTheDocument();
    expect(screen.queryByTestId("fact-row-F01")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("fact-filter-na"));
    expect(screen.getByTestId("fact-row-F02")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("fact-filter-drop"));
    expect(screen.getByTestId("fact-row-F12")).toBeInTheDocument();
    expect(screen.queryByTestId("fact-row-F08")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("fact-filter-all"));
    expect(screen.queryByTestId("fact-row-F12")).not.toBeInTheDocument();
    expect(screen.getByTestId("fact-filter-all")).toHaveTextContent("Tất cả · 11");
  });

  it("reads a pasted answer, clears the box and says what it read", async () => {
    const paste = vi.spyOn(apiClient, "pasteResearch").mockResolvedValue({
      added: ["F13", "F14"],
      updated: ["F03"],
      rejected: [{ row: 4, reason: "thiếu type" }],
      log: { n: 2, at: "", added: ["F13", "F14"], updated: ["F03"], rejected: [{ row: 4, reason: "thiếu type" }] },
    });
    await ready();
    fireEvent.change(screen.getByTestId("research-paste-input"), { target: { value: "```json\n{...}\n```" } });
    fireEvent.click(screen.getByTestId("research-paste-read"));
    await waitFor(() => expect(paste).toHaveBeenCalledWith("p1", "```json\n{...}\n```"));
    await waitFor(() => expect(screen.getByTestId("research-paste-input")).toHaveValue(""));
    expect(screen.getByTestId("research-paste-message")).toHaveTextContent("Đã đọc 3 dữ kiện, bỏ 1 dòng");
  });

  it("does not send an empty box, and shows the server's reason when no JSON block was found", async () => {
    const paste = vi.spyOn(apiClient, "pasteResearch").mockRejectedValue(new apiClient.ApiError("không tìm thấy khối JSON", undefined, undefined, 400));
    await ready();
    fireEvent.click(screen.getByTestId("research-paste-read"));
    expect(screen.getByTestId("research-paste-empty")).toBeInTheDocument();
    expect(paste).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId("research-paste-input"), { target: { value: "chỉ là văn bản" } });
    fireEvent.click(screen.getByTestId("research-paste-read"));
    await waitFor(() => expect(screen.getByTestId("research-paste-message")).toHaveTextContent("không tìm thấy khối JSON"));
    expect(screen.getByTestId("research-paste-input")).toHaveValue("chỉ là văn bản");
  });

  it("without the AI Bridge extension shows how to install it instead of the DeepSeek button", async () => {
    await ready();
    await screen.findByTestId("research-prompt-preview");
    expect(screen.getByTestId("research-bridge-missing")).toHaveTextContent("ConceptFlow AI Bridge");
    expect(screen.queryByTestId("research-bridge-run")).not.toBeInTheDocument();
  });

  it("runs the prompt on DeepSeek, puts the answer in section 3 and reads it", async () => {
    const fake = installFakeAiBridge();
    const paste = vi.spyOn(apiClient, "pasteResearch").mockResolvedValue({
      added: ["F13"],
      updated: [],
      rejected: [],
      log: { n: 2, at: "", added: ["F13"], updated: [], rejected: [] },
    });
    try {
      await ready();
      fireEvent.click(await screen.findByTestId("research-bridge-run"));
      await waitFor(() => expect(fake.runs).toHaveLength(1));
      expect(fake.runs[0]).toMatchObject({ provider: "deepseek", prompt: "PROMPT NGHIÊN CỨU\nQ1. Quả sung có phải là hoa không?" });
      fake.send({ type: "progress", request_id: fake.runs[0].requestId, stage: "answering" });
      await waitFor(() => expect(screen.getByTestId("research-bridge-stage")).toHaveTextContent("DeepSeek đang suy nghĩ"));
      fake.send({ type: "result", request_id: fake.runs[0].requestId, text: "```json\n{}\n```" });
      await waitFor(() => expect(paste).toHaveBeenCalledWith("p1", "```json\n{}\n```"));
      await waitFor(() => expect(screen.getByTestId("research-paste-input")).toHaveValue(""));
      expect(screen.getByTestId("research-paste-message")).toHaveTextContent("Đã đọc 1 dữ kiện");
      expect(screen.getByTestId("research-bridge-run")).toBeInTheDocument();
    } finally {
      fake.remove();
    }
  });

  it("keeps DeepSeek's answer in section 3 when the server cannot read it", async () => {
    const fake = installFakeAiBridge();
    vi.spyOn(apiClient, "pasteResearch").mockRejectedValue(new apiClient.ApiError("không tìm thấy khối JSON", undefined, undefined, 400));
    try {
      await ready();
      fireEvent.click(await screen.findByTestId("research-bridge-run"));
      await waitFor(() => expect(fake.runs).toHaveLength(1));
      fake.send({ type: "result", request_id: fake.runs[0].requestId, text: "chỉ là văn bản" });
      await waitFor(() => expect(screen.getByTestId("research-paste-message")).toHaveTextContent("không tìm thấy khối JSON"));
      expect(screen.getByTestId("research-paste-input")).toHaveValue("chỉ là văn bản");
    } finally {
      fake.remove();
    }
  });

  it("shows the extension's error and fills nothing", async () => {
    const fake = installFakeAiBridge();
    const paste = vi.spyOn(apiClient, "pasteResearch");
    try {
      await ready();
      fireEvent.click(await screen.findByTestId("research-bridge-run"));
      await waitFor(() => expect(fake.runs).toHaveLength(1));
      fake.send({ type: "error", request_id: fake.runs[0].requestId, code: "not_logged_in", message: "Chưa đăng nhập DeepSeek." });
      await waitFor(() => expect(screen.getByTestId("research-bridge-error")).toHaveTextContent("Chưa đăng nhập DeepSeek."));
      expect(screen.getByTestId("research-paste-input")).toHaveValue("");
      expect(paste).not.toHaveBeenCalled();
    } finally {
      fake.remove();
    }
  });

  it("cancels a DeepSeek run without an error", async () => {
    const fake = installFakeAiBridge();
    try {
      await ready();
      fireEvent.click(await screen.findByTestId("research-bridge-run"));
      await waitFor(() => expect(fake.runs).toHaveLength(1));
      fireEvent.click(screen.getByTestId("research-bridge-cancel"));
      await waitFor(() => expect(fake.cancels).toEqual([fake.runs[0].requestId]));
      expect(screen.getByTestId("research-bridge-run")).toBeInTheDocument();
      expect(screen.queryByTestId("research-bridge-error")).not.toBeInTheDocument();
    } finally {
      fake.remove();
    }
  });

  it("shows the whole prompt and copies it from the card", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    await ready();
    await waitFor(() => expect(screen.getByTestId("research-prompt-preview")).toHaveTextContent("PROMPT NGHIÊN CỨU"));
    fireEvent.click(screen.getByTestId("research-prompt-quick-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("PROMPT NGHIÊN CỨU\nQ1. Quả sung có phải là hoa không?"));
  });

  it("opens the follow-up prompt for the questions still short of sources", async () => {
    await ready();
    expect(screen.getByTestId("coverage-follow-up")).toHaveTextContent("Xem prompt bổ sung cho 1 câu hỏi");
    fireEvent.click(screen.getByTestId("coverage-follow-up"));
    expect(await screen.findByTestId("research-follow-up-modal-text")).toHaveTextContent("PROMPT BỔ SUNG");
  });

  it("says every question is sourced when none is weak", async () => {
    state = baseState({ coverage: [coverageOf("Q1", 2, false), coverageOf("Q2", 3, false)] });
    await ready();
    expect(screen.getByTestId("coverage-all-sourced")).toBeInTheDocument();
    expect(screen.queryByTestId("coverage-follow-up")).not.toBeInTheDocument();
  });

  describe("questions", () => {
    it("saves a changed question when the Creator leaves it, and ignores a blanked one", async () => {
      const save = vi.spyOn(apiClient, "editResearchQuestion").mockResolvedValue(undefined);
      await ready();
      const input = screen.getByTestId("question-input-Q1");
      fireEvent.change(input, { target: { value: "  Câu hỏi đã sửa?  " } });
      fireEvent.blur(input);
      await waitFor(() => expect(save).toHaveBeenCalledWith("p1", "Q1", "Câu hỏi đã sửa?"));

      save.mockClear();
      fireEvent.change(screen.getByTestId("question-input-Q2"), { target: { value: "   " } });
      fireEvent.blur(screen.getByTestId("question-input-Q2"));
      expect(save).not.toHaveBeenCalled();
      expect(screen.getByTestId("question-input-Q2")).toHaveValue("Ong nào thụ phấn cho sung?");
    });

    it("adds a question typed into a new row", async () => {
      const add = vi.spyOn(apiClient, "addResearchQuestion").mockResolvedValue(question("Q3", "Mới?", { mine: true }));
      await ready();
      fireEvent.click(screen.getByTestId("question-add"));
      const input = screen.getByTestId("question-new-input");
      await waitFor(() => expect(input).toHaveFocus());
      fireEvent.change(input, { target: { value: "Câu thêm tay?" } });
      fireEvent.blur(input);
      await waitFor(() => expect(add).toHaveBeenCalledWith("p1", "Câu thêm tay?"));
    });

    it("drops an empty new row without asking the server", async () => {
      const add = vi.spyOn(apiClient, "addResearchQuestion").mockResolvedValue(question("Q3", "x"));
      await ready();
      fireEvent.click(screen.getByTestId("question-add"));
      fireEvent.blur(screen.getByTestId("question-new-input"));
      expect(add).not.toHaveBeenCalled();
      expect(screen.queryByTestId("question-new-input")).not.toBeInTheDocument();
    });

    it("asks before deleting a question and says its facts stay", async () => {
      const del = vi.spyOn(apiClient, "deleteResearchQuestion").mockResolvedValue(undefined);
      await ready();
      fireEvent.click(screen.getByTestId("question-delete-Q2"));
      const dialog = await screen.findByTestId("question-delete-dialog");
      expect(dialog).toHaveTextContent("Xoá câu hỏi Q2?");
      expect(dialog).toHaveTextContent("Dữ kiện đã có của Q2 giữ nguyên");
      expect(del).not.toHaveBeenCalled();
      fireEvent.click(screen.getByTestId("question-delete-confirm"));
      await waitFor(() => expect(del).toHaveBeenCalledWith("p1", "Q2"));
    });

    it("regenerates straight away when the Creator wrote none of the questions", async () => {
      const generate = vi.spyOn(apiClient, "generateResearchQuestions").mockResolvedValue([]);
      await ready();
      expect(screen.getByTestId("question-generate")).toHaveTextContent("Sinh lại từ brief");
      fireEvent.click(screen.getByTestId("question-generate"));
      await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", { replaceEdited: false, note: undefined }, expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) })));
      expect(screen.queryByTestId("question-generate-dialog")).not.toBeInTheDocument();
    });

    it("asks what to do with the Creator's own questions before regenerating", async () => {
      const generate = vi.spyOn(apiClient, "generateResearchQuestions").mockResolvedValue([]);
      state = baseState({ questions: [question("Q1", "Của AI"), question("Q2", "Của tôi", { mine: true })] });
      await ready();
      fireEvent.click(screen.getByTestId("question-generate"));
      const dialog = await screen.findByTestId("question-generate-dialog");
      expect(dialog).toHaveTextContent("Câu hỏi bạn đã viết hoặc sửa: Q2.");
      expect(generate).not.toHaveBeenCalled();

      fireEvent.click(screen.getByTestId("question-generate-replace"));
      await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", { replaceEdited: true, note: undefined }, expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) })));

      fireEvent.click(screen.getByTestId("question-generate"));
      fireEvent.click(await screen.findByTestId("question-generate-keep"));
      await waitFor(() => expect(generate).toHaveBeenLastCalledWith("p1", { replaceEdited: false, note: undefined }, expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) })));
    });

    it("offers to generate when there are no questions, and has no prompt to show yet", async () => {
      state = baseState({ questions: [], coverage: [], facts: [] });
      await ready();
      expect(screen.getByTestId("question-generate")).toHaveTextContent("Sinh câu hỏi từ brief");
      expect(screen.getByTestId("question-generate")).not.toBeDisabled();
      expect(screen.getByTestId("research-questions-empty")).toHaveTextContent("AI sinh từ brief, hoặc bạn tự thêm");
      expect(screen.getByTestId("research-prompt-empty")).toBeInTheDocument();
      expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa có dữ kiện");
    });

    it("turns the generate button off without an AI and tells the Creator to add questions by hand", async () => {
      vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: false, provider: "", reason: "Chưa cấu hình" });
      state = baseState({ questions: [], coverage: [], facts: [] });
      await ready();
      await waitFor(() => expect(screen.getByTestId("research-questions-empty")).toHaveTextContent("Chưa cấu hình AI nên bạn tự thêm câu hỏi"));
      expect(screen.getByTestId("question-generate")).toBeDisabled();
      expect(screen.getByTestId("question-add")).not.toBeDisabled();
      expect(screen.queryByTestId("research-model-field")).not.toBeInTheDocument();
    });

    it("lets the Creator pick the model that writes the questions", async () => {
      await ready();
      expect(await screen.findByTestId("research-model-field")).toHaveTextContent("Model AI sinh câu hỏi");
    });

    it("shows that the AI's questions are out of date after the brief changed", async () => {
      state = baseState({ questions: [question("Q1", "Của AI", { stale: true }), question("Q2", "Của tôi", { mine: true })], questionsStale: true });
      await ready();
      expect(screen.getByTestId("research-questions-stale")).toHaveTextContent("Brief vừa đổi");
      expect(screen.getByTestId("question-stale-Q1")).toBeInTheDocument();
      expect(screen.queryByTestId("question-stale-Q2")).not.toBeInTheDocument();
      expect(screen.getByTestId("question-mine-Q2")).toHaveTextContent("Bạn viết");
    });

    it("shows the approved idea and brief, read again from the server, behind the ! button", async () => {
      await ready();
      const reads = vi.mocked(apiClient.getResearch).mock.calls.length;
      fireEvent.click(screen.getByTestId("research-input-open"));
      const dialog = await screen.findByTestId("research-input-dialog");
      await waitFor(() => expect(within(dialog).getByTestId("research-input-idea")).toBeInTheDocument());
      expect(vi.mocked(apiClient.getResearch).mock.calls.length).toBeGreaterThan(reads);
      expect(within(dialog).getByTestId("research-input-idea")).toHaveTextContent("Vì sao ong chết trong quả sung?");
      expect(within(dialog).getByTestId("research-input-idea")).toHaveTextContent("Ong vào quả bằng đường nào?");
      expect(within(dialog).getByTestId("research-input-brief")).toHaveTextContent("Người xem phổ thông");
      expect(within(dialog).getByTestId("research-input-brief")).toHaveTextContent("4:30");
      expect(within(dialog).getByTestId("research-input-brief")).toHaveTextContent("Nói ong bị ăn thịt");
    });

    it("calls the AI straight away, without asking again what it will read", async () => {
      const gen = vi.spyOn(apiClient, "generateResearchQuestions").mockResolvedValue([]);
      await ready();
      fireEvent.click(screen.getByTestId("question-generate"));
      await waitFor(() => expect(gen).toHaveBeenCalled());
      expect(screen.queryByTestId("research-input-dialog")).not.toBeInTheDocument();
    });

    it("says which of the idea and the brief is not approved yet", async () => {
      state = baseState({ idea: null, briefApproved: false });
      await ready();
      fireEvent.click(screen.getByTestId("research-input-open"));
      expect(await screen.findByTestId("research-input-no-idea")).toHaveTextContent("chưa được duyệt");
      expect(screen.getByTestId("research-input-brief")).toBeInTheDocument();
    });

    it("shows the failure of the AI and keeps the page usable", async () => {
      vi.spyOn(apiClient, "generateResearchQuestions").mockRejectedValue(new apiClient.ApiError("Chưa cấu hình HIVE_API_KEY", undefined, undefined, 503));
      await ready();
      fireEvent.click(screen.getByTestId("question-generate"));
      expect(await screen.findByTestId("research-error")).toHaveTextContent("Chưa cấu hình HIVE_API_KEY");
      expect(screen.getByTestId("question-generate")).not.toBeDisabled();
    });
  });

  describe("facts", () => {
    it("changes a type in the table when nothing uses the fact", async () => {
      const flags = vi.spyOn(apiClient, "setFactFlags").mockResolvedValue(fact("F03"));
      await ready();
      fireEvent.change(screen.getByTestId("fact-type-F03"), { target: { value: "hypothesis" } });
      await waitFor(() => expect(flags).toHaveBeenCalledWith("p1", "F03", { type: "hypothesis" }));
      expect(screen.queryByTestId("fact-change-dialog")).not.toBeInTheDocument();
    });

    it("asks first, listing the work it sends back, when later steps use the fact", async () => {
      impacts = { F03: ["Beat B3 dùng F03", "Câu 3.1 dựa vào F03"] };
      const flags = vi.spyOn(apiClient, "setFactFlags").mockResolvedValue(fact("F03"));
      await ready();
      fireEvent.click(screen.getByTestId("fact-keep-F03"));
      const dialog = await screen.findByTestId("fact-change-dialog");
      expect(dialog).toHaveTextContent("Bỏ dữ kiện F03?");
      expect(dialog).toHaveTextContent("Beat B3 dùng F03");
      expect(dialog).toHaveTextContent("Câu 3.1 dựa vào F03");
      expect(flags).not.toHaveBeenCalled();
      fireEvent.click(screen.getByTestId("fact-change-confirm"));
      await waitFor(() => expect(flags).toHaveBeenCalledWith("p1", "F03", { keep: false }));
    });

    it("drops nothing when the Creator cancels the confirmation", async () => {
      impacts = { F03: ["Beat B3 dùng F03"] };
      const flags = vi.spyOn(apiClient, "setFactFlags").mockResolvedValue(fact("F03"));
      await ready();
      fireEvent.click(screen.getByTestId("fact-must-F03"));
      fireEvent.click(await screen.findByTestId("fact-change-dialog-cancel"));
      await waitFor(() => expect(screen.queryByTestId("fact-change-dialog")).not.toBeInTheDocument());
      expect(flags).not.toHaveBeenCalled();
    });

    it("keeps a dropped fact back without asking", async () => {
      state = baseState({ facts: [fact("F01", { keep: false })] });
      impacts = { F01: ["Beat B1 dùng F01"] };
      const flags = vi.spyOn(apiClient, "setFactFlags").mockResolvedValue(fact("F01"));
      await ready();
      fireEvent.click(screen.getByTestId("fact-filter-drop"));
      fireEvent.click(screen.getByTestId("fact-keep-F01"));
      await waitFor(() => expect(flags).toHaveBeenCalledWith("p1", "F01", { keep: true }));
      expect(screen.queryByTestId("fact-change-dialog")).not.toBeInTheDocument();
    });

    it("checks a link again", async () => {
      const verify = vi.spyOn(apiClient, "verifyFacts").mockResolvedValue(undefined);
      await ready();
      expect(screen.queryByTestId("fact-recheck-F12")).not.toBeInTheDocument();
      fireEvent.click(screen.getByTestId("fact-recheck-F08"));
      await waitFor(() => expect(verify).toHaveBeenCalledWith("p1", ["F08"]));
    });

    it("cannot check again a link that is being checked", async () => {
      state = baseState({ facts: [fact("F01", { link: { state: "run", why: "", nearest: "", checkedAt: "" } })] });
      await ready();
      expect(screen.getByTestId("fact-link-F01")).toHaveTextContent("đang kiểm");
      expect(screen.getByTestId("fact-recheck-F01")).toBeDisabled();
    });

    it("shows that links are being checked in the fact table and holds the coverage count back", async () => {
      state = baseState({ facts: [fact("F01", { link: { state: "run", why: "", nearest: "", checkedAt: "" } }), fact("F02")] });
      await ready();
      expect(screen.getByTestId("facts-checking")).toHaveTextContent("Đang tải và kiểm 1 link");
      expect(screen.getByTestId("coverage-chip-Q1")).toHaveTextContent("đang tính…");
      expect(screen.getByTestId("coverage-chip-Q2")).toHaveTextContent("1 nguồn khớp · cần thêm 1");
      expect(screen.getByTestId("coverage-checking")).toBeInTheDocument();
      expect(screen.queryByTestId("coverage-follow-up")).not.toBeInTheDocument();
    });

    it("keeps a fact sent for checking shown as being checked even when the answer is already in", async () => {
      vi.spyOn(apiClient, "verifyFacts").mockResolvedValue(undefined);
      await ready();
      fireEvent.click(screen.getByTestId("fact-recheck-F08"));
      await waitFor(() => expect(screen.getByTestId("fact-link-F08")).toHaveTextContent("đang kiểm"));
      expect(screen.getByTestId("facts-checking")).toBeInTheDocument();
      expect(screen.getByTestId("coverage-chip-Q1")).toHaveTextContent("đang tính…");
      await waitFor(() => expect(screen.queryByTestId("facts-checking")).not.toBeInTheDocument(), { timeout: 3000 });
      expect(screen.getByTestId("coverage-chip-Q1")).toHaveTextContent("✓ 2 nguồn khớp");
    });

    it("shows no checking notice when every link is checked", async () => {
      await ready();
      expect(screen.queryByTestId("facts-checking")).not.toBeInTheDocument();
      expect(screen.queryByTestId("coverage-checking")).not.toBeInTheDocument();
    });

    it("edits a fact in a dialog and sends only what changed", async () => {
      const edit = vi.spyOn(apiClient, "editFact").mockResolvedValue(undefined);
      await ready();
      fireEvent.click(screen.getByTestId("fact-edit-F08"));
      const claim = await screen.findByTestId("fact-edit-claim");
      expect(claim).toHaveValue("Dữ kiện số 8");
      expect(screen.getByTestId("fact-edit-url")).toHaveValue("https://example.org/8");
      await waitFor(() => expect(screen.getByTestId("fact-edit-save")).not.toBeDisabled());
      fireEvent.change(screen.getByTestId("fact-edit-quote"), { target: { value: "Đoạn trích mới" } });
      fireEvent.click(screen.getByTestId("fact-edit-save"));
      await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", "F08", { quote: "Đoạn trích mới" }));
      await waitFor(() => expect(screen.queryByTestId("fact-edit")).not.toBeInTheDocument());
    });

    it("keeps the edit dialog open and shows why a save failed", async () => {
      vi.spyOn(apiClient, "editFact").mockRejectedValue(new apiClient.ApiError("link không dùng được", undefined, undefined, 400));
      await ready();
      fireEvent.click(screen.getByTestId("fact-edit-F08"));
      await waitFor(() => expect(screen.getByTestId("fact-edit-save")).not.toBeDisabled());
      fireEvent.change(screen.getByTestId("fact-edit-url"), { target: { value: "ftp://x" } });
      fireEvent.click(screen.getByTestId("fact-edit-save"));
      expect(await screen.findByTestId("fact-edit-error")).toHaveTextContent("link không dùng được");
      expect(screen.getByTestId("fact-edit")).toBeInTheDocument();
    });

    it("turns the save button orange when later work uses the fact", async () => {
      impacts = { F08: ["Beat B5 dùng F08"] };
      await ready();
      fireEvent.click(screen.getByTestId("fact-edit-F08"));
      expect(await screen.findByTestId("fact-edit-impact")).toHaveTextContent("Beat B5 dùng F08");
    });

    it("adds a fact by hand", async () => {
      const add = vi.spyOn(apiClient, "addFact").mockResolvedValue(fact("F13"));
      await ready();
      fireEvent.click(screen.getByTestId("fact-add-open"));
      await screen.findByTestId("fact-add");
      expect(screen.getByTestId("fact-add-save")).toBeDisabled();
      fireEvent.change(screen.getByTestId("fact-add-claim"), { target: { value: "Một dữ kiện tay" } });
      fireEvent.change(screen.getByTestId("fact-add-question"), { target: { value: "Q2" } });
      fireEvent.change(screen.getByTestId("fact-add-type"), { target: { value: "misconception" } });
      fireEvent.change(screen.getByTestId("fact-add-url"), { target: { value: "https://example.org/x" } });
      fireEvent.click(screen.getByTestId("fact-add-save"));
      await waitFor(() =>
        expect(add).toHaveBeenCalledWith("p1", {
          question: "Q2",
          claimVi: "Một dữ kiện tay",
          type: "misconception",
          sourceTitle: "",
          url: "https://example.org/x",
          quote: "",
        }),
      );
      await waitFor(() => expect(screen.queryByTestId("fact-add")).not.toBeInTheDocument());
    });
  });

  describe("approving and skipping", () => {
    it("blocks the next step until the table is approved, then moves on", async () => {
      const approve = vi.spyOn(apiClient, "approveResearch").mockImplementation(async () => {
        state = baseState({ gate: { status: "approved", skipped: false, resolved: true } });
      });
      await ready();
      expect(screen.getByTestId("research-status")).toHaveAttribute("data-state", "pending");
      expect(screen.getByTestId("research-next")).toBeDisabled();

      fireEvent.click(screen.getByTestId("research-approve"));
      await waitFor(() => expect(approve).toHaveBeenCalledWith("p1"));
      await waitFor(() => expect(screen.getByTestId("research-status")).toHaveAttribute("data-state", "approved"));
      expect(screen.getByTestId("step-strip")).toHaveTextContent("Bảng dữ kiện đã duyệt");
      expect(screen.getByTestId("research-unapprove")).toBeInTheDocument();
      expect(screen.getByTestId("research-next")).not.toBeDisabled();
      expect(screen.getByTestId("research-next")).toHaveTextContent("Lưu và sang câu chuyện");

      fireEvent.click(screen.getByTestId("research-next"));
      expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/story");
    });

    it("takes an approval back", async () => {
      state = baseState({ gate: { status: "approved", skipped: false, resolved: true } });
      const unapprove = vi.spyOn(apiClient, "unapproveResearch").mockResolvedValue(undefined);
      await ready();
      fireEvent.click(screen.getByTestId("research-unapprove"));
      await waitFor(() => expect(unapprove).toHaveBeenCalledWith("p1"));
    });

    it("shows the server's reason when the table cannot be approved", async () => {
      vi.spyOn(apiClient, "approveResearch").mockRejectedValue(new apiClient.ApiError("chưa có dữ kiện nào để duyệt", undefined, undefined, 400));
      state = baseState({ facts: [], coverage: [coverageOf("Q1", 0, true), coverageOf("Q2", 0, true)] });
      await ready();
      fireEvent.click(screen.getByTestId("research-approve"));
      expect(await screen.findByTestId("research-error")).toHaveTextContent("chưa có dữ kiện nào để duyệt");
      expect(screen.getByTestId("research-next")).toBeDisabled();
    });

    it("skips the step after a confirmation that lists what follows, and lets the Creator come back", async () => {
      const skip = vi.spyOn(apiClient, "setResearchSkipped").mockImplementation(async (_p, skipped) => {
        state = baseState({ gate: { status: "pending", skipped, resolved: skipped } });
        return state.gate;
      });
      await ready();
      expect(screen.getByTestId("research-optional")).toHaveTextContent("Tuỳ chọn");
      fireEvent.click(screen.getByTestId("research-skip"));
      const dialog = await screen.findByTestId("research-skip-dialog");
      expect(dialog).toHaveTextContent("Bỏ qua bước Nghiên cứu?");
      expect(dialog).toHaveTextContent("chưa có nguồn");
      expect(dialog).toHaveTextContent("Bấm “Làm nghiên cứu” để quay lại");
      expect(skip).not.toHaveBeenCalled();

      fireEvent.click(screen.getByTestId("research-skip-confirm"));
      await waitFor(() => expect(skip).toHaveBeenCalledWith("p1", true));
      expect(await screen.findByTestId("research-skipped-card")).toHaveTextContent("Khi bỏ qua bước này");
      expect(screen.getByTestId("step-strip")).toHaveTextContent("Đã bỏ qua");
      expect(screen.queryByTestId("research-questions-card")).not.toBeInTheDocument();
      expect(screen.getByTestId("research-next")).not.toBeDisabled();

      fireEvent.click(screen.getByTestId("research-unskip"));
      await waitFor(() => expect(skip).toHaveBeenLastCalledWith("p1", false));
      expect(await screen.findByTestId("research-questions-card")).toBeInTheDocument();
    });

    it("lets the Creator go on to the story from the skipped step", async () => {
      state = baseState({ gate: { status: "pending", skipped: true, resolved: true } });
      renderPage();
      await screen.findByTestId("research-skipped-card");
      fireEvent.click(screen.getByTestId("research-next"));
      expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/story");
    });
  });

  it("is read-only once the project is past editing", async () => {
    project = { ...project, status: "rendering", flow_step: 10, run_state: "running" } as Project;
    await ready();
    expect(await screen.findByTestId("research-read-only")).toBeInTheDocument();
    expect(screen.getByTestId("research-paste-read")).toBeDisabled();
    expect(screen.getByTestId("research-approve")).toBeDisabled();
    expect(screen.getByTestId("research-skip")).toBeDisabled();
    expect(screen.getByTestId("fact-add-open")).toBeDisabled();
    expect(screen.getByTestId("fact-edit-F01")).toBeDisabled();
    expect(screen.getByTestId("question-generate")).toBeDisabled();
  });

  it("explains a load failure and offers to try again", async () => {
    const get = vi.spyOn(apiClient, "getResearch").mockRejectedValueOnce(new apiClient.ApiError("máy chủ không trả lời", undefined, undefined, 500));
    get.mockImplementation(async () => state);
    renderPage();
    expect(await screen.findByTestId("research-load-error")).toHaveTextContent("máy chủ không trả lời");
    fireEvent.click(screen.getByTestId("research-reload"));
    await screen.findByTestId("research-facts-table");
  });
});
