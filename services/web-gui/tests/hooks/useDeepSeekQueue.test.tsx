import { describe, it, expect, afterEach, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useDeepSeekQueue, type QueueItem } from "../../src/hooks/useDeepSeekQueue";
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

describe("useDeepSeekQueue", () => {
  it("is unavailable until the extension reports a queue", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useDeepSeekQueue());
    await new Promise((r) => setTimeout(r, 20));
    expect(result.current.available).toBe(false);
    expect(result.current.counts).toBeNull();
  });

  it("follows the counts the extension reports", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useDeepSeekQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    act(() => fake?.send({ type: "queue_state", waiting: 3, running: 1, done: 2, failed: 0, paused: true }));
    expect(result.current.counts).toEqual({ waiting: 3, running: 1, done: 2, failed: 0, paused: true });
  });

  it("says whether the item was queued or the extension already held it", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useDeepSeekQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    let outcome: unknown;
    await act(async () => {
      outcome = await result.current.add(item);
    });
    expect(outcome).toEqual({ ok: true, added: true });
    expect(fake.queued[0].item).toMatchObject({ label: "B2 · Beat", project: "p1", modes: { deepThink: true, search: false } });
  });

  it("returns the extension's reason when it refuses an item", async () => {
    fake = installFakeAiBridge({ queue: true, refuseQueue: "Mục không hợp lệ." });
    const { result } = renderHook(() => useDeepSeekQueue());
    await waitFor(() => expect(result.current.available).toBe(true));
    let outcome: unknown;
    await act(async () => {
      outcome = await result.current.add(item);
    });
    expect(outcome).toEqual({ ok: false, message: "Mục không hợp lệ." });
  });

  it("gives up when the extension does not answer an add", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = renderHook(() => useDeepSeekQueue());
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
});
