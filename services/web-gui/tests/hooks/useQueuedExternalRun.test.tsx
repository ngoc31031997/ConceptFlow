import { describe, it, expect, afterEach, beforeEach, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { ApiError, ExternalRequestDescribed, type ExternalAi, type ExternalRequest } from "../../src/api/client";
import { useQueuedExternalRun } from "../../src/hooks/useQueuedExternalRun";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let fake: FakeAiBridge | null = null;

beforeEach(() => {
  describing.mockClear();
});

afterEach(() => {
  fake?.remove();
  fake = null;
});

const request: ExternalRequest = {
  gateway: "http://localhost:8080",
  method: "POST",
  path: "/v1/projects/p1/story/generate",
  body: '{"note":""}',
};

/** A step whose request can only be described. */
const describing = vi.fn(async (external: ExternalAi) => {
  if (external.mode === "describe") throw new ExternalRequestDescribed(request);
  throw new Error("the page must not make the request itself");
});

function setup(overrides: Partial<Parameters<typeof useQueuedExternalRun>[0]> = {}) {
  const onDone = vi.fn();
  const onUnreadable = vi.fn();
  const hook = renderHook(() => useQueuedExternalRun({ run: describing, label: "Viết câu chuyện", onDone, onUnreadable, ...overrides }));
  return { ...hook, onDone, onUnreadable };
}

describe("useQueuedExternalRun", () => {
  it("is not available without the extension's queue", async () => {
    fake = installFakeAiBridge({ answersPing: false });
    const { result } = setup();
    await new Promise((r) => setTimeout(r, 20));
    expect(result.current.available).toBe(false);
  });

  it("queues the described request with the step's label and project, then follows the item to done", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result, onDone } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    expect(fake.queued).toHaveLength(1);
    expect(fake.queued[0].item).toMatchObject({ label: "Viết câu chuyện", project: "p1", request, modes: { deepThink: true, search: false } });
    expect(result.current.busy).toBe(true);

    act(() => fake?.announce("q1", { status: "waiting", provider: "gemini" }));
    expect(result.current.item).toMatchObject({ state: "waiting", provider: "gemini" });
    act(() => fake?.announce("q1", { status: "running", stage: "answering", provider: "gemini" }));
    expect(result.current.item).toMatchObject({ state: "running", stage: "answering" });
    expect(onDone).not.toHaveBeenCalled();

    act(() => fake?.announce("q1", { status: "done", provider: "gemini" }));
    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(onDone).toHaveBeenCalledWith();
    expect(result.current.busy).toBe(false);
    expect(result.current.error).toBeNull();
  });

  it("hands over an answer the step could not read, with the reason", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result, onUnreadable, onDone } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    act(() => fake?.announce("q1", { status: "failed", provider: "grok", error: "AI trả thiếu ô goal", reply: "câu trả lời lạ" }));
    await waitFor(() => expect(onUnreadable).toHaveBeenCalledWith("câu trả lời lạ", "AI trả thiếu ô goal"));
    expect(onDone).not.toHaveBeenCalled();
    expect(result.current.busy).toBe(false);
  });

  it("shows the reason of a failure that has no answer", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result, onUnreadable } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    act(() => fake?.announce("q1", { status: "failed", provider: "gemini", error: "Chưa đăng nhập Gemini." }));
    await waitFor(() => expect(result.current.error).toBe("Chưa đăng nhập Gemini."));
    expect(onUnreadable).not.toHaveBeenCalled();
    expect(result.current.busy).toBe(false);
  });

  it("says why the request could not be described and queues nothing", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = setup({ run: async () => Promise.reject(new ApiError("Duyệt brief trước.")) });
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    expect(result.current.error).toBe("Duyệt brief trước.");
    expect(fake.queued).toHaveLength(0);
    expect(result.current.busy).toBe(false);
  });

  it("shows the extension's reason when it refuses the item", async () => {
    fake = installFakeAiBridge({ queue: true, refuseQueue: "Mục không hợp lệ." });
    const { result } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    expect(result.current.error).toBe("Mục không hợp lệ.");
    expect(result.current.busy).toBe(false);
  });

  it("takes the item out of the queue on cancel and stops following it", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result, onDone } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    act(() => fake?.announce("q1", { status: "running", stage: "sending" }));
    act(() => result.current.cancel());
    await waitFor(() => expect(fake?.cancels).toEqual(["q1"]));
    act(() => fake?.announce("q1", { status: "cancelled" }));
    await waitFor(() => expect(result.current.busy).toBe(false));
    expect(result.current.error).toBeNull();
    expect(onDone).not.toHaveBeenCalled();
  });

  it("queues a prompt the page already holds as an item of kind prompt, without describing anything", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result, onDone } = setup({ run: undefined, label: "Tra cứu dữ kiện", modes: { deepThink: true, search: true } });
    await waitFor(() => expect(result.current.available).toBe(true));
    const paste: ExternalRequest = { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/research/paste" };
    await act(async () => {
      await result.current.start({ prompt: "PROMPT NGHIÊN CỨU", request: paste });
    });
    expect(describing).not.toHaveBeenCalled();
    expect(fake.queued[0].item).toMatchObject({
      label: "Tra cứu dữ kiện",
      project: "p1",
      kind: "prompt",
      prompt: "PROMPT NGHIÊN CỨU",
      request: paste,
      modes: { deepThink: true, search: true },
    });
    act(() => fake?.announce("q1", { status: "done" }));
    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
  });

  it("counts the items ahead of a waiting one from the queue the extension reported", async () => {
    fake = installFakeAiBridge({ queue: true });
    const { result } = setup();
    await waitFor(() => expect(result.current.available).toBe(true));
    await act(async () => {
      await result.current.start();
    });
    act(() => fake?.send({ type: "queue_state", waiting: 3, running: 1, done: 0, failed: 0, paused: false }));
    act(() => fake?.announce("q1", { status: "waiting" }));
    expect(result.current.ahead).toBe(3);
  });
});
