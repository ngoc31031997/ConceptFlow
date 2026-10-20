import { describe, it, expect, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useOperationRun } from "../../src/hooks/useOperationRun";
import * as client from "../../src/api/client";
import type { OperationProgress, RunningOperation } from "../../src/api/client";

function progress(status: OperationProgress["status"]): OperationProgress {
  return { kind: "research_question", phase: "writing", reasoning_chars: 0, content_chars: 5, done: null, total: null, elapsed_ms: 1000, status, error: null };
}

function running(id: string, kind: string): RunningOperation {
  return { id, ...progress("running"), kind };
}

describe("useOperationRun", () => {
  afterEach(() => vi.restoreAllMocks());

  it("follows a call of the project that is still running when the page opens again", async () => {
    vi.spyOn(client, "listRunningOperations").mockResolvedValue([running("op-other", "brief"), running("op-1", "research_question")]);
    const get = vi.spyOn(client, "getOperation").mockResolvedValue(progress("running"));

    const { result } = renderHook(() => useOperationRun({ projectId: "p1", kinds: ["research_question"] }));

    await waitFor(() => expect(result.current.resumed).toBe(true));
    await waitFor(() => expect(get).toHaveBeenCalledWith("op-1"));
  });

  it("reloads once when the followed call ends, then lets go of it", async () => {
    vi.spyOn(client, "listRunningOperations").mockResolvedValue([running("op-1", "research_question")]);
    vi.spyOn(client, "getOperation").mockResolvedValue(progress("succeeded"));
    const onSettled = vi.fn();

    const { result } = renderHook(() => useOperationRun({ projectId: "p1", kinds: ["research_question"], onSettled }));

    await waitFor(() => expect(onSettled).toHaveBeenCalledTimes(1));
    expect(result.current.resumed).toBe(false);
    expect(result.current.progress).toBeNull();
  });

  it("follows nothing when no call of this screen's kind is running", async () => {
    const list = vi.spyOn(client, "listRunningOperations").mockResolvedValue([running("op-other", "brief")]);

    const { result } = renderHook(() => useOperationRun({ projectId: "p1", kinds: ["research_question"] }));

    await waitFor(() => expect(list).toHaveBeenCalled());
    expect(result.current.resumed).toBe(false);
  });

  it("asks the server to stop a followed call and drops it", async () => {
    vi.spyOn(client, "listRunningOperations").mockResolvedValue([running("op-1", "research_question")]);
    vi.spyOn(client, "getOperation").mockResolvedValue(progress("running"));
    const stop = vi.spyOn(client, "cancelOperation").mockResolvedValue(undefined);
    const { result } = renderHook(() => useOperationRun({ projectId: "p1", kinds: ["research_question"] }));
    await waitFor(() => expect(result.current.resumed).toBe(true));

    await act(async () => {
      await result.current.cancel();
    });

    expect(stop).toHaveBeenCalledWith("op-1");
    expect(result.current.resumed).toBe(false);
    expect(result.current.cancelError).toBeNull();
  });

  it("aborts the request it started and tells the server, in one cancel", async () => {
    const stop = vi.spyOn(client, "cancelOperation").mockResolvedValue(undefined);
    vi.spyOn(client, "getOperation").mockResolvedValue(progress("running"));
    const { result } = renderHook(() => useOperationRun());
    let call!: client.OperationCall;
    act(() => {
      call = result.current.begin();
    });

    await act(async () => {
      await result.current.cancel();
    });

    expect(call.signal.aborted).toBe(true);
    expect(stop).toHaveBeenCalledWith(call.id);
  });

  it("keeps the card and says why when the server cannot be told to stop a call it did not start here", async () => {
    vi.spyOn(client, "listRunningOperations").mockResolvedValue([running("op-1", "research_question")]);
    vi.spyOn(client, "getOperation").mockResolvedValue(progress("running"));
    vi.spyOn(client, "cancelOperation").mockRejectedValue(new client.ApiError("máy chủ không trả lời", undefined, undefined, 502));
    const { result } = renderHook(() => useOperationRun({ projectId: "p1", kinds: ["research_question"] }));
    await waitFor(() => expect(result.current.resumed).toBe(true));

    await act(async () => {
      await result.current.cancel();
    });

    expect(result.current.cancelError).toBe("máy chủ không trả lời");
    expect(result.current.resumed).toBe(true);
  });

  it("aborts the request when the page goes away, if asked to", () => {
    const { result, unmount } = renderHook(() => useOperationRun({ abortOnUnmount: true }));
    let call!: client.OperationCall;
    act(() => {
      call = result.current.begin();
    });
    unmount();
    expect(call.signal.aborted).toBe(true);
  });

  it("lets the request run on when the page goes away by default", () => {
    const { result, unmount } = renderHook(() => useOperationRun());
    let call!: client.OperationCall;
    act(() => {
      call = result.current.begin();
    });
    unmount();
    expect(call.signal.aborted).toBe(false);
  });
});
