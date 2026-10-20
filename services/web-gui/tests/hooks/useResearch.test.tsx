import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { RESEARCH_POLL_MS, useResearch } from "../../src/hooks/useResearch";
import * as client from "../../src/api/client";
import type { ResearchFact, ResearchState } from "../../src/api/client";

function fact(over: Partial<ResearchFact> = {}): ResearchFact {
  return {
    id: "F01",
    question: "Q1",
    claimVi: "Dữ kiện",
    type: "fact",
    sourceTitle: "",
    publisher: "",
    year: "",
    url: "https://a.org/",
    quote: "đoạn trích",
    confidence: "",
    note: "",
    must: false,
    keep: true,
    link: { state: "ok", why: "", nearest: "", checkedAt: "" },
    mine: [],
    ...over,
  };
}

function stateWith(facts: ResearchFact[], over: Partial<ResearchState> = {}): ResearchState {
  return {
    gate: { status: "pending", skipped: false, resolved: false },
    questions: [{ id: "Q1", text: "Một?", mine: false, stale: false }],
    facts,
    pastes: [],
    coverage: [],
    briefApproved: true,
    questionsStale: false,
    ...over,
  };
}

describe("useResearch", () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("loads the step once the project exists, and not before", async () => {
    const get = vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([fact()]));
    const { result, rerender } = renderHook(({ enabled }) => useResearch("p1", enabled), { initialProps: { enabled: false } });
    expect(get).not.toHaveBeenCalled();
    expect(result.current.state).toBeNull();

    rerender({ enabled: true });
    await waitFor(() => expect(result.current.state?.facts).toHaveLength(1));
    expect(get).toHaveBeenCalledWith("p1");
  });

  it("reports why the load failed in the server's words", async () => {
    vi.spyOn(client, "getResearch").mockRejectedValue(new client.ApiError("không đọc được", undefined, undefined, 500));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.loadError).toBe("không đọc được"));
  });

  it("asks again every two seconds while a link is being checked, and stops when none is", async () => {
    const get = vi
      .spyOn(client, "getResearch")
      .mockResolvedValueOnce(stateWith([fact({ link: { state: "run", why: "", nearest: "", checkedAt: "" } })]))
      .mockResolvedValue(stateWith([fact()]));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state?.facts[0].link.state).toBe("run"));
    expect(get).toHaveBeenCalledTimes(1);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RESEARCH_POLL_MS + 50);
    });
    await waitFor(() => expect(result.current.state?.facts[0].link.state).toBe("ok"));
    const callsWhenDone = get.mock.calls.length;
    expect(callsWhenDone).toBeGreaterThanOrEqual(2);

    await act(async () => {
      await vi.advanceTimersByTimeAsync(RESEARCH_POLL_MS * 3);
    });
    expect(get.mock.calls.length).toBe(callsWhenDone);
  });

  it("does not poll while nothing is being checked", async () => {
    const get = vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([fact()]));
    renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(get).toHaveBeenCalledTimes(1));
    await act(async () => {
      await vi.advanceTimersByTimeAsync(RESEARCH_POLL_MS * 3);
    });
    expect(get).toHaveBeenCalledTimes(1);
  });

  it("lets only the newest load land", async () => {
    let releaseFirst: (s: ResearchState) => void = () => {};
    const first = new Promise<ResearchState>((resolve) => {
      releaseFirst = resolve;
    });
    const get = vi
      .spyOn(client, "getResearch")
      .mockReturnValueOnce(first)
      .mockResolvedValue(stateWith([fact({ id: "F09" })]));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(get).toHaveBeenCalledTimes(1));
    act(() => result.current.reload());
    await waitFor(() => expect(result.current.state?.facts[0]?.id).toBe("F09"));

    await act(async () => {
      releaseFirst(stateWith([fact({ id: "F01" })]));
    });
    expect(result.current.state?.facts[0].id).toBe("F09");
  });

  it("reloads after a change and keeps the server's words when it fails", async () => {
    const get = vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([fact()]));
    const flags = vi.spyOn(client, "setFactFlags").mockResolvedValue(fact({ must: true }));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    let ok = false;
    await act(async () => {
      ok = await result.current.setFlags("F01", { must: true });
    });
    expect(ok).toBe(true);
    expect(flags).toHaveBeenCalledWith("p1", "F01", { must: true });
    expect(get).toHaveBeenCalledTimes(2);

    vi.spyOn(client, "approveResearch").mockRejectedValue(new client.ApiError("chưa có dữ kiện nào để duyệt", undefined, undefined, 400));
    await act(async () => {
      ok = await result.current.approve();
    });
    expect(ok).toBe(false);
    expect(result.current.actionError).toBe("chưa có dữ kiện nào để duyệt");
    expect(result.current.busy).toBe(false);
  });

  it("throws a refused fact edit for the dialog and leaves the page's error alone", async () => {
    vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([fact()]));
    vi.spyOn(client, "editFact").mockRejectedValue(new client.ApiError("link không dùng được", undefined, undefined, 400));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    await expect(result.current.saveFact("F01", { url: "ftp://x" })).rejects.toThrow("link không dùng được");
    expect(result.current.actionError).toBeNull();
  });

  it("follows the AI run that writes the questions, with an operation id", async () => {
    vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([]));
    const generate = vi.spyOn(client, "generateResearchQuestions").mockResolvedValue([]);
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    let ok = false;
    await act(async () => {
      ok = await result.current.generateQuestions(true, "cơ chế");
    });
    expect(ok).toBe(true);
    expect(generate).toHaveBeenCalledWith("p1", { replaceEdited: true, note: "cơ chế" }, expect.objectContaining({ id: expect.any(String), signal: expect.any(AbortSignal) }));
    expect(result.current.running).toBe(false);
  });

  it("treats a cancelled AI run as no failure: no error on the page, the questions as they were", async () => {
    vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([]));
    vi.spyOn(client, "generateResearchQuestions").mockRejectedValue(new client.RequestCancelledError());
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    let ok = true;
    await act(async () => {
      ok = await result.current.generateQuestions(false);
    });
    expect(ok).toBe(false);
    expect(result.current.actionError).toBeNull();
    expect(result.current.running).toBe(false);
  });

  it("keeps the failure of the AI run for the card and the page", async () => {
    vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([]));
    vi.spyOn(client, "generateResearchQuestions").mockRejectedValue(new client.ApiError("Chưa cấu hình HIVE_API_KEY", undefined, undefined, 503));
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    let ok = true;
    await act(async () => {
      ok = await result.current.generateQuestions(false);
    });
    expect(ok).toBe(false);
    expect(result.current.actionError).toBe("Chưa cấu hình HIVE_API_KEY");
    expect(result.current.running).toBe(false);
  });

  it("tells what a paste read and left out, and warns when it read nothing", async () => {
    vi.spyOn(client, "getResearch").mockResolvedValue(stateWith([]));
    const paste = vi.spyOn(client, "pasteResearch").mockResolvedValueOnce({
      added: ["F01", "F02"],
      updated: ["F03"],
      rejected: [{ row: 4, reason: "thiếu type" }],
      log: { n: 1, at: "", added: [], updated: [], rejected: [] },
    });
    const { result } = renderHook(() => useResearch("p1", true));
    await waitFor(() => expect(result.current.state).not.toBeNull());

    await act(async () => {
      await result.current.pasteResults("câu trả lời");
    });
    expect(paste).toHaveBeenCalledWith("p1", "câu trả lời");
    expect(result.current.pasteMessage).toEqual({
      tone: "ok",
      text: "Đã đọc 3 dữ kiện, bỏ 1 dòng (lý do ở nhật ký). Đang tải từng link để kiểm đoạn trích.",
    });

    paste.mockRejectedValueOnce(new client.ApiError("không tìm thấy khối JSON", undefined, undefined, 400));
    let ok = true;
    await act(async () => {
      ok = await result.current.pasteResults("không có gì");
    });
    expect(ok).toBe(false);
    expect(result.current.pasteMessage).toEqual({ tone: "warn", text: "không tìm thấy khối JSON" });
  });
});
