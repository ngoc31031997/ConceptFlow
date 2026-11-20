import { describe, it, expect, afterEach, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useAiQueue, type QueueItem } from "../../src/hooks/useAiQueue";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let fake: FakeAiBridge | null = null;

afterEach(() => {
  fake?.remove();
  fake = null;
  vi.useRealTimers();
});

const item: QueueItem = {
  label: "B2 · Beat",
  project: "p1",
  request: { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/beats/b2/redo", body: "{}" },
  modes: { deepThink: true, search: false },
};

describe("useAiQueue", () => {
  it("is unavailable until the extension reports a queue", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiQueue());
    await new Promise((r) => setTimeout(r, 20));
    expect(result.current.available).toBe(false);
    expect(result.current.counts).toBeNull();
  });

  it("follows the counts and the default provider the extension reports", async () => {
    fake = installFakeAiBridge({ queue: true, defaultProvider: "grok" });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    expect(result.current.defaultProvider).toBe("grok");
    act(() => fake?.send({ type: "queue_state", waiting: 3, running: 1, done: 2, failed: 0, paused: true }));
    expect(result.current.counts).toEqual({ waiting: 3, running: 1, done: 2, failed: 0, paused: true });
  });

  it("says whether the item was queued or the extension already held it, with its id", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    let outcome: unknown;
    await act(async () => {
      outcome = await result.current.add(item);
    });
    expect(outcome).toEqual({ ok: true, added: true, itemId: "q1" });
    expect(fake.queued[0].item).toMatchObject({ label: "B2 · Beat", project: "p1", modes: { deepThink: true, search: false } });
  });

  it("sends the provider and the kind of an item that chose them", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.add({ ...item, provider: "gemini", kind: "prompt", prompt: "hỏi" });
    });
    expect(fake.queued[0].item).toMatchObject({ provider: "gemini", kind: "prompt", prompt: "hỏi" });
  });

  it("returns the extension's reason when it refuses an item", async () => {
    fake = installFakeAiBridge({ queue: true, refuseQueue: "Mục không hợp lệ." });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    let outcome: unknown;
    await act(async () => {
      outcome = await result.current.add(item);
    });
    expect(outcome).toEqual({ ok: false, message: "Mục không hợp lệ." });
  });

  it("gives up when the extension does not answer an add", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    // From here the extension no longer answers.
    fake.remove();
    vi.useFakeTimers();
    let outcome: unknown;
    const pending = act(async () => {
      outcome = await result.current.add(item);
    });
    await vi.advanceTimersByTimeAsync(5000);
    await pending;
    expect(outcome).toMatchObject({ ok: false, message: expect.stringContaining("Extension không phản hồi") });
  });

  it("follows only the items it was asked to watch", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.add(item, { watch: true });
      await result.current.add({ ...item, label: "B3", request: { ...item.request, path: "/v1/projects/p1/beats/b3/redo" } });
    });
    act(() => fake?.announce("q1", { status: "running", stage: "answering", provider: "gemini" }));
    act(() => fake?.announce("q2", { status: "running", stage: "opening" }));
    expect(result.current.items.q1).toMatchObject({ state: "running", stage: "answering", provider: "gemini", path: "/v1/projects/p1/beats/b2/redo" });
    expect(result.current.items.q2).toBeUndefined();
    act(() => result.current.watch("q2"));
    act(() => fake?.announce("q2", { status: "waiting" }));
    expect(result.current.items.q2?.state).toBe("waiting");
  });

  it("calls onItemDone once with the path of an item that finished, watched or not", async () => {
    fake = installFakeAiBridge({ queue: true });
    const done = vi.fn();
    const { result } = renderHook(() => useAiQueue(done));
    await waitFor(() => expect(result.current.available).toBe(true));
    act(() => fake?.send({ type: "queue_item", item_id: "q9", path: "/v1/projects/p1/beats/b2/redo", status: "done" }));
    act(() => fake?.send({ type: "queue_item", item_id: "q9", path: "/v1/projects/p1/beats/b2/redo", status: "done" }));
    act(() => fake?.send({ type: "queue_item", item_id: "q8", path: "/v1/projects/p1/beats/b3/redo", status: "failed", error: "lỗi" }));
    expect(done).toHaveBeenCalledTimes(1);
    expect(done).toHaveBeenCalledWith("/v1/projects/p1/beats/b2/redo");
  });

  it("asks the extension to take an item out of the queue", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    act(() => result.current.cancel("q1"));
    await waitFor(() => expect(fake?.cancels).toEqual(["q1"]));
  });

  it("flags an extension with the old protocol", async () => {
    fake = installFakeAiBridge({ legacy: true });
    const { result } = renderHook(() => useAiQueue());
    await waitFor(() => expect(result.current.outdated).toBe(true));
    expect(result.current.available).toBe(false);
  });
});
