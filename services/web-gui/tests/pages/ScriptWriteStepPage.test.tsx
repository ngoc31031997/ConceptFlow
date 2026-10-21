import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";
import { ScriptWriteStepPage } from "../../src/pages/ScriptWriteStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { ScriptBeat, ScriptLine, ScriptState } from "../../src/api/client";
import type { Project } from "../../src/types";

function Where() {
  return <span data-testid="where">{useLocation().pathname}</span>;
}

function line(id: string, text: string, over: Partial<ScriptLine> = {}): ScriptLine {
  return { id, text, facts: [], seconds: 3, estimated: true, flag: null, audio: { state: "none" }, ...over };
}

function beat(key: string, number: number, over: Partial<ScriptBeat> = {}): ScriptBeat {
  return {
    key,
    number,
    fn: "Giải thích cơ chế",
    title: `Beat ${key}`,
    goal: `Mục tiêu ${key}`,
    targetSeconds: 30,
    written: true,
    status: "pending",
    mine: [],
    stale: false,
    lines: [line(`${key}.1`, `Câu một của ${key}.`), line(`${key}.2`, `Câu hai của ${key}.`)],
    seconds: 6,
    join: null,
    ...over,
  };
}

function baseState(over: Partial<ScriptState> = {}): ScriptState {
  return {
    status: "pending",
    beatsApproved: true,
    beats: [beat("b1", 1, { fn: "Hook" }), beat("b2", 2, { join: { ok: true, msg: "nối mượt với beat trước" } }), beat("b3", 3, { fn: "Payoff", join: { ok: true, msg: "nối mượt với beat trước" } })],
    checks: [
      { ok: true, text: "Câu mở (b1.1) đủ ngắn: 3 giây" },
      { ok: false, text: "1 câu dài hoặc khó nghe khi đọc" },
    ],
    stats: { lines: 6, beats: 3, written: 3, estimated: 6, totalSeconds: 18, openFlags: 1, openJoins: 0 },
    targetLabel: "4–5 phút",
    inRange: false,
    voiceId: "vi-VN-HoaiMyNeural",
    ...over,
  };
}

let state: ScriptState;
let project: Project;

class FakeAudio {
  static instances: FakeAudio[] = [];
  onended: (() => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public src: string) {
    FakeAudio.instances.push(this);
  }
  play() {
    return Promise.resolve();
  }
  pause() {}
}

beforeEach(() => {
  FakeAudio.instances = [];
  vi.stubGlobal("Audio", FakeAudio);
  state = baseState();
  project = { project_id: "p1", status: "draft", voice_language: "vi", scenes: [], flow_step: 3, run_state: "idle", video_output_mode: "long" } as Project;
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive", reason: "", models: [{ id: "m1", label: "Model 1" }] });
  vi.spyOn(apiClient, "getProject").mockImplementation(async () => project);
  vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([]);
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
  vi.spyOn(apiClient, "getResearch").mockRejectedValue(new Error("no"));
  vi.spyOn(apiClient, "getScript").mockImplementation(async () => state);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/projects/p1/script"]}>
        <ProjectDraftProvider>
          <ProjectFlowProvider>
            <Routes>
              <Route path="/projects/:id/script" element={<ScriptWriteStepPage />} />
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
  await screen.findByTestId("script-overview-card");
}

describe("ScriptWriteStepPage", () => {
  it("shows the overview, its checks and one card per beat with its lines", async () => {
    await ready();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Viết kịch bản");
    expect(screen.getByTestId("phase-line")).toHaveTextContent("Tiền kỳ · Nội dung · Bước 6/21");
    expect(screen.getByTestId("rail-step-6")).toHaveAttribute("aria-current", "step");
    expect(screen.getByTestId("script-counts")).toHaveTextContent("6 câu · 3/3 beat có lời");
    expect(screen.getByTestId("script-total")).toHaveTextContent("0:18 · mục tiêu 4–5 phút");
    expect(screen.getByTestId("script-total")).toHaveTextContent("! ngoài khung");
    expect(screen.getByTestId("script-checks").querySelectorAll("[data-ok='false']")).toHaveLength(1);
    expect(screen.getByTestId("script-beat-b2")).toHaveTextContent("B2 · Giải thích cơ chế · Beat b2");
    expect(screen.getByTestId("script-line-text-b2.2")).toHaveTextContent("Câu hai của b2.");
    expect(screen.getByTestId("script-join-b2")).toHaveAttribute("data-ok", "true");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chờ bạn duyệt · 0/3 beat");
  });

  it("says what the step waits for while the beats are not approved", async () => {
    state = baseState({ beatsApproved: false, beats: [], stats: { ...baseState().stats, written: 0, lines: 0, beats: 0 } });
    renderPage();
    expect(await screen.findByTestId("script-not-open")).toHaveTextContent("Duyệt beat sheet");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Chưa mở bước");
    expect(screen.queryByTestId("script-generate")).not.toBeInTheDocument();
  });

  it("has the AI write the beats that have no dialogue, saying that what exists is kept", async () => {
    state = baseState({ beats: [beat("b1", 1, { written: false, lines: [], seconds: 0 }), beat("b2", 2)], stats: { ...baseState().stats, written: 1, lines: 2 } });
    const generate = vi.spyOn(apiClient, "generateScript").mockResolvedValue(baseState());
    renderPage();
    fireEvent.click(await screen.findByTestId("script-generate"));
    expect(screen.getByTestId("script-write-card")).toHaveTextContent("Lời thoại đã có giữ nguyên");
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p1", "", "", expect.anything()));
  });

  it("lets the Creator write a beat by hand when no model is configured", async () => {
    vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: false, provider: "hive", reason: "", models: [] });
    state = baseState({ beats: [beat("b1", 1, { written: false, lines: [], seconds: 0 })], stats: { ...baseState().stats, written: 0, lines: 0 } });
    renderPage();
    expect(await screen.findByTestId("script-empty-b1")).toHaveTextContent("Chưa cấu hình AI");
    expect(screen.queryByTestId("script-write-b1")).not.toBeInTheDocument();
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    fireEvent.click(screen.getByTestId("script-write-by-hand-b1"));
    const field = await screen.findByTestId("script-edit-b1-field-add:start");
    fireEvent.change(field, { target: { value: "Câu tôi tự viết." } });
    fireEvent.click(screen.getByTestId("script-edit-b1-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "script_beat", key: "b1" }), { "add:start": "Câu tôi tự viết." }));
  });

  it("flags a line, shows the AI's suggestion and applies it as an edit of that line", async () => {
    state = baseState({
      beats: [beat("b1", 1, { lines: [line("b1.1", "Chất CO2 rất nhiều.", { flag: { kind: "Khó nghe", msg: "“CO2” trộn chữ và số." } })] })],
    });
    const suggest = vi.spyOn(apiClient, "suggestScriptLine").mockResolvedValue("Chất các-bon đi-ô-xít rất nhiều.");
    const edit = vi.spyOn(apiClient, "editUnit").mockResolvedValue({} as never);
    renderPage();
    expect(await screen.findByTestId("script-flag-b1.1")).toHaveTextContent("Khó nghe");
    fireEvent.click(screen.getByTestId("script-ask-line-b1.1"));
    expect(await screen.findByTestId("script-suggestion-b1.1")).toHaveTextContent("các-bon đi-ô-xít");
    expect(suggest).toHaveBeenCalledWith("p1", "line", "b1.1", expect.anything());
    expect(edit).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("script-suggestion-apply-b1.1"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "script_beat", key: "b1" }), { "line:b1.1": "Chất các-bon đi-ô-xít rất nhiều." }));
  });

  it("offers a suggestion for a join that does not read well", async () => {
    state = baseState({
      beats: [beat("b1", 1), beat("b2", 2, { join: { ok: false, msg: "hai câu liền nhau cùng mở bằng “Ong”, nghe lặp." } })],
    });
    vi.spyOn(apiClient, "suggestScriptLine").mockResolvedValue("Rồi con ong mất cánh.");
    renderPage();
    expect(await screen.findByTestId("script-join-b2")).toHaveAttribute("data-ok", "false");
    fireEvent.click(screen.getByTestId("script-ask-join-b2.1"));
    expect(await screen.findByTestId("script-suggestion-b2.1")).toHaveTextContent("Rồi con ong mất cánh.");
  });

  it("sends a line to be read, waits for its file and plays it from the gateway", async () => {
    const speak = vi.spyOn(apiClient, "speakScriptLines").mockImplementation(async () => {
      state = baseState({
        beats: [beat("b1", 1, { lines: [line("b1.1", "Câu một.", { audio: { state: "running" } })] })],
      });
      return state;
    });
    state = baseState({ beats: [beat("b1", 1, { lines: [line("b1.1", "Câu một.")] })] });
    renderPage();
    fireEvent.click(await screen.findByTestId("script-play-b1.1"));
    await waitFor(() => expect(speak).toHaveBeenCalledWith("p1", ["b1.1"]));
    await waitFor(() => expect(screen.getByTestId("script-play-b1.1")).toHaveTextContent("Đang đọc"));
    expect(FakeAudio.instances).toHaveLength(0);
    state = baseState({
      beats: [beat("b1", 1, { lines: [line("b1.1", "Câu một.", { estimated: false, seconds: 4.2, audio: { state: "ready", file: "b1-1_0123456789ab.wav" } })] })],
    });
    await waitFor(() => expect(FakeAudio.instances).toHaveLength(1), { timeout: 4000 });
    expect(FakeAudio.instances[0].src.endsWith("/v1/projects/p1/audio/b1-1_0123456789ab.wav")).toBe(true);
    await waitFor(() => expect(screen.getByTestId("script-play-b1.1")).toHaveTextContent("Đang phát"));
    FakeAudio.instances[0].onended?.();
    await waitFor(() => expect(screen.getByTestId("script-play-b1.1")).toHaveTextContent("Nghe"));
  });

  it("plays a line that already has its file without asking the server to read it", async () => {
    const speak = vi.spyOn(apiClient, "speakScriptLines");
    state = baseState({
      beats: [beat("b1", 1, { lines: [line("b1.1", "Câu một.", { audio: { state: "ready", file: "b1-1_0123456789ab.wav" } })] })],
    });
    renderPage();
    fireEvent.click(await screen.findByTestId("script-play-b1.1"));
    await waitFor(() => expect(FakeAudio.instances).toHaveLength(1));
    expect(speak).not.toHaveBeenCalled();
  });

  it("stops with the reason when a line cannot be read, never skipping it silently", async () => {
    state = baseState({
      beats: [beat("b1", 1, { lines: [line("b1.1", "Câu một.", { audio: { state: "failed", error: "engine hỏng" } })] })],
    });
    renderPage();
    expect(await screen.findByTestId("script-audio-error-b1.1")).toHaveTextContent("engine hỏng");
    fireEvent.click(screen.getByTestId("script-play-b1.1"));
    expect(await screen.findByTestId("script-player-error", {}, { timeout: 3000 })).toHaveTextContent("engine hỏng");
    expect(FakeAudio.instances).toHaveLength(0);
  });

  it("approves every beat that has dialogue and waits, then lets the Creator move on", async () => {
    const approve = vi.spyOn(apiClient, "approveUnit").mockImplementation(async () => {
      state = baseState({ status: "approved", beats: state.beats.map((b) => ({ ...b, status: "approved" as const })) });
      return {} as never;
    });
    await ready();
    expect(screen.getByTestId("script-next")).toBeDisabled();
    fireEvent.click(screen.getByTestId("script-approve-all"));
    await waitFor(() => expect(screen.getByTestId("step-strip")).toHaveTextContent("Kịch bản đã duyệt"));
    expect(approve.mock.calls.map((c) => (c[1] as { key: string }).key)).toEqual(["b1", "b2", "b3"]);
    fireEvent.click(screen.getByTestId("script-next"));
    expect(screen.getByTestId("where")).toHaveTextContent("/projects/p1/coming/7");
  });

  it("redoes one beat's dialogue with a note", async () => {
    const redo = vi.spyOn(apiClient, "redoUnit").mockResolvedValue({} as never);
    await ready();
    fireEvent.click(screen.getByTestId("script-review-b2-redo"));
    fireEvent.change(await screen.findByTestId("script-redo-b2-note"), { target: { value: "ngắn hơn" } });
    fireEvent.click(screen.getByTestId("script-redo-b2-send"));
    await waitFor(() => expect(redo).toHaveBeenCalledWith("p1", expect.objectContaining({ kind: "script_beat", key: "b2" }), "ngắn hơn", expect.anything()));
  });

  it("shows why an action failed in the server's words", async () => {
    vi.spyOn(apiClient, "approveUnit").mockRejectedValue(new apiClient.ApiError("beat chưa có lời thoại"));
    await ready();
    fireEvent.click(screen.getByTestId("script-review-b1-approve"));
    expect(await screen.findByTestId("script-error")).toHaveTextContent("beat chưa có lời thoại");
  });
});
