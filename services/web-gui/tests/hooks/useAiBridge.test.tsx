import { describe, it, expect, afterEach, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useAiBridge, type AiBridgeOutcome } from "../../src/hooks/useAiBridge";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let fake: FakeAiBridge | null = null;

afterEach(() => {
  fake?.remove();
  fake = null;
  vi.useRealTimers();
});

async function startRun(result: { current: ReturnType<typeof useAiBridge> }) {
  let outcome: Promise<AiBridgeOutcome> = Promise.resolve({
    ok: false,
    cancelled: false,
    message: "",
  });
  act(() => {
    outcome = result.current.run("deepseek", "PROMPT", { deepThink: true, search: false });
  });
  await waitFor(() => expect(fake?.runs).toHaveLength(1));
  // Wrapped, so awaiting this helper does not also wait for the run to end.
  return { outcome };
}

describe("useAiBridge", () => {
  it("stays unavailable when no extension answers the ping", async () => {
    fake = installFakeAiBridge({ answersPing: false });
    const { result } = renderHook(() => useAiBridge());
    await new Promise((r) => setTimeout(r, 20));
    expect(result.current.available).toBe(false);
  });

  it("becomes available with the providers the extension reports", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    expect(result.current.providers).toEqual(["deepseek"]);
  });

  it("follows the run's stages and resolves with the answer", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    const { outcome } = await startRun(result);
    const requestId = fake.runs[0].requestId;
    expect(fake.runs[0]).toMatchObject({
      provider: "deepseek",
      prompt: "PROMPT",
      modes: { deep_think: true, search: false },
    });
    expect(result.current.stage).toBe("opening");

    act(() => fake?.send({ type: "progress", request_id: requestId, stage: "answering" }));
    await waitFor(() => expect(result.current.stage).toBe("answering"));
    act(() =>
      fake?.send({ type: "result", request_id: "someone-else", text: "không phải của trang này" }),
    );
    act(() => fake?.send({ type: "result", request_id: requestId, text: "```json\n{}\n```" }));
    await expect(outcome).resolves.toEqual({ ok: true, text: "```json\n{}\n```" });
    expect(result.current.stage).toBeNull();
  });

  it("resolves with the extension's error text", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    const { outcome } = await startRun(result);
    act(() =>
      fake?.send({
        type: "error",
        request_id: fake.runs[0].requestId,
        code: "not_logged_in",
        message: "Chưa đăng nhập DeepSeek.",
      }),
    );
    await expect(outcome).resolves.toEqual({
      ok: false,
      cancelled: false,
      message: "Chưa đăng nhập DeepSeek.",
    });
  });

  it("tells the extension when the run is cancelled and resolves as cancelled", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    const { outcome } = await startRun(result);
    act(() => result.current.cancel());
    await expect(outcome).resolves.toMatchObject({ ok: false, cancelled: true });
    await waitFor(() => expect(fake?.cancels).toEqual([fake?.runs[0].requestId]));
  });

  it("fails the run when the extension never acknowledges it", async () => {
    fake = installFakeAiBridge();
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    vi.useFakeTimers();
    let outcome: Promise<AiBridgeOutcome> = Promise.resolve({ ok: true, text: "" });
    act(() => {
      outcome = result.current.run("deepseek", "PROMPT", { deepThink: true, search: false });
    });
    await act(async () => {
      vi.advanceTimersByTime(5000);
    });
    await expect(outcome).resolves.toMatchObject({
      ok: false,
      cancelled: false,
      message: expect.stringContaining("Extension không phản hồi"),
    });
  });
});
