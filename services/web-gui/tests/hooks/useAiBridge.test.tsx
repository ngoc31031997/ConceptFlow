import { describe, it, expect, afterEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { providerName, useAiBridge } from "../../src/hooks/useAiBridge";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let fake: FakeAiBridge | null = null;

afterEach(() => {
  fake?.remove();
  fake = null;
});

describe("useAiBridge", () => {
  it("stays unavailable when no extension answers the ping", async () => {
    fake = installFakeAiBridge({ answersPing: false });
    const { result } = renderHook(() => useAiBridge());
    await new Promise((r) => setTimeout(r, 20));
    expect(result.current.available).toBe(false);
    expect(result.current.outdated).toBe(false);
  });

  it("becomes available with the providers and the default the extension reports", async () => {
    fake = installFakeAiBridge({ defaultProvider: "gemini" });
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.available).toBe(true));
    expect(result.current.providers).toEqual(["deepseek", "gemini", "grok"]);
    expect(result.current.defaultProvider).toBe("gemini");
    expect(result.current.outdated).toBe(false);
  });

  it("marks an extension that still speaks the old protocol as outdated", async () => {
    fake = installFakeAiBridge({ legacy: true });
    const { result } = renderHook(() => useAiBridge());
    await waitFor(() => expect(result.current.outdated).toBe(true));
    expect(result.current.available).toBe(false);
  });

  it("names the providers", () => {
    expect(providerName("deepseek")).toBe("DeepSeek");
    expect(providerName("gemini")).toBe("Gemini");
    expect(providerName("grok")).toBe("Grok");
    expect(providerName("khac")).toBe("khac");
  });
});
