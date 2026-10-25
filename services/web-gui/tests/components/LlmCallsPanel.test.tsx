import { afterEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { LlmCallsPanel } from "../../src/components/LlmCallsPanel";
import type { LlmCall, LlmCallDetail, LlmCallsPage } from "../../src/api/client";
import { ThemeProvider } from "../../src/context/ThemeContext";

function call(over: Partial<LlmCall>): LlmCall {
  return {
    call_id: "c1", created_at: "2026-10-25T10:00:00Z", provider: "hive", model: "m1", role: "story_frame",
    step: "story", phase: "", project_id: "aaaaaaaa-1111", response: '{"title":"T"}', ok: true, error_kind: "",
    error_message: "", prompt_tokens: 10, completion_tokens: 5, reasoning_tokens: 0, cached_tokens: 0,
    reasoning_chars: 0, usage_reported: true, duration_ms: 1500, prompt_chars: 4200, num_ctx: 8192, ...over,
  };
}

const rows = [
  call({ call_id: "c1" }),
  call({ call_id: "c2", ok: false, error_kind: "truncated", error_message: "cut off", response: "half", role: "beat_sheet" }),
];

/** Serves the log from `state`; the delete route removes what it is told to. */
function serve(state: { calls: LlmCall[]; next?: string | null }) {
  const requests: { url: string; body?: unknown }[] = [];
  global.fetch = vi.fn().mockImplementation(async (url: string, init?: RequestInit) => {
    const body = init?.body ? JSON.parse(String(init.body)) : undefined;
    requests.push({ url: String(url), body });
    if (String(url).includes("/v1/llm/calls/delete")) {
      const before = state.calls.length;
      if (body.ids) state.calls = state.calls.filter((c) => !body.ids.includes(c.call_id));
      else state.calls = [];
      return { ok: true, status: 200, json: async () => ({ deleted: before - state.calls.length }) };
    }
    const one = String(url).match(/\/v1\/llm\/calls\/([^/?]+)$/);
    if (one) {
      const row = state.calls.find((c) => c.call_id === decodeURIComponent(one[1]));
      if (!row) return { ok: false, status: 404, json: async () => ({ error: "no call" }) };
      const detail: LlmCallDetail = { ...row, system_prompt: `SYS of ${row.call_id}`, user_prompt: `USR of ${row.call_id}` };
      return { ok: true, status: 200, json: async () => detail };
    }
    const page: LlmCallsPage = { calls: state.calls, next_before: state.next ?? null };
    return { ok: true, status: 200, json: async () => page };
  }) as unknown as typeof fetch;
  return requests;
}

function renderPanel() {
  return render(
    <ThemeProvider>
      <LlmCallsPanel />
    </ThemeProvider>,
  );
}

describe("LlmCallsPanel", () => {
  afterEach(() => vi.restoreAllMocks());

  it("loads the raw rows and shows how each call ended", async () => {
    serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-call-c1")).toBeInTheDocument());
    expect(screen.getByTestId("llm-call-c1")).toHaveTextContent("story_frame");
    expect(screen.getByTestId("llm-call-c1")).toHaveTextContent("OK");
    expect(screen.getByTestId("llm-call-c1")).toHaveTextContent("10 / 5");
    expect(screen.getByTestId("llm-call-c2")).toHaveTextContent("truncated");
  });

  it("filters by project, role and result", async () => {
    const requests = serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-call-c1")).toBeInTheDocument());

    fireEvent.change(screen.getByTestId("llm-filter-project"), { target: { value: "aaaa" } });
    fireEvent.change(screen.getByTestId("llm-filter-role"), { target: { value: "beat_sheet" } });
    fireEvent.change(screen.getByTestId("llm-filter-status"), { target: { value: "failed" } });
    fireEvent.click(screen.getByTestId("llm-filter-apply"));

    await waitFor(() => {
      const last = requests[requests.length - 1].url;
      expect(last).toContain("project_id=aaaa");
      expect(last).toContain("role=beat_sheet");
      expect(last).toContain("ok=false");
    });
  });

  it("opens the stored response with its call id, ready to copy", async () => {
    serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-view-c2")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("llm-view-c2"));
    const dialog = await screen.findByTestId("llm-response-dialog");
    expect(within(dialog).getByTestId("llm-response-text")).toHaveTextContent("half");
    expect(dialog).toHaveTextContent("c2");
    expect(dialog).toHaveTextContent("cut off");
    expect(within(dialog).getByTestId("llm-response-copy")).toBeInTheDocument();
  });

  it("shows the system and user prompt that went out, each ready to copy, with the context window", async () => {
    const requests = serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-view-c1")).toBeInTheDocument());
    expect(screen.getByTestId("llm-call-c1")).toHaveTextContent("4.200");
    fireEvent.click(screen.getByTestId("llm-view-c1"));
    const dialog = await screen.findByTestId("llm-response-dialog");
    await waitFor(() => expect(within(dialog).getByTestId("llm-system-text")).toHaveTextContent("SYS of c1"));
    expect(within(dialog).getByTestId("llm-user-text")).toHaveTextContent("USR of c1");
    expect(within(dialog).getByTestId("llm-system-copy")).toBeInTheDocument();
    expect(within(dialog).getByTestId("llm-user-copy")).toBeInTheDocument();
    expect(dialog).toHaveTextContent("num_ctx 8192");
    expect(requests.some((r) => r.url.endsWith("/v1/llm/calls/c1"))).toBe(true);
  });

  it("still shows the response when the prompt cannot be read, and says why", async () => {
    serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-view-c1")).toBeInTheDocument());
    const ok = global.fetch as unknown as ReturnType<typeof vi.fn>;
    global.fetch = vi.fn().mockImplementation(async (url: string, init?: RequestInit) =>
      String(url).endsWith("/v1/llm/calls/c1")
        ? { ok: false, status: 404, json: async () => ({ error: "no call" }) }
        : ok(url, init),
    ) as unknown as typeof fetch;
    fireEvent.click(screen.getByTestId("llm-view-c1"));
    const dialog = await screen.findByTestId("llm-response-dialog");
    await waitFor(() => expect(within(dialog).getByTestId("llm-detail-error")).toBeInTheDocument());
    expect(within(dialog).getByTestId("llm-response-text")).toHaveTextContent("T");
  });

  it("selects every shown row at once and deletes them after a confirmation that says what is lost", async () => {
    const state = { calls: [...rows] };
    const requests = serve(state);
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-select-all")).toBeInTheDocument());
    expect(screen.getByTestId("llm-delete-selected")).toBeDisabled();

    fireEvent.click(screen.getByTestId("llm-select-all"));
    expect(screen.getByTestId("llm-delete-selected")).toHaveTextContent("Xoá 2 dòng đã chọn");
    fireEvent.click(screen.getByTestId("llm-delete-selected"));

    const modal = await screen.findByTestId("confirm-modal");
    expect(modal).toHaveTextContent("2 cuộc gọi đã chọn");
    expect(modal).toHaveTextContent("Không khôi phục được");
    expect(modal).toHaveTextContent("thống kê");
    // Nothing is deleted before the Creator confirms.
    expect(requests.some((r) => r.url.includes("/delete"))).toBe(false);

    fireEvent.click(within(modal).getByText("Xoá các dòng đã chọn"));
    await waitFor(() => expect(screen.getByTestId("llm-calls-notice")).toHaveTextContent("Đã xoá 2 dòng."));
    expect(requests.find((r) => r.url.includes("/delete"))?.body).toEqual({ ids: ["c1", "c2"] });
    await waitFor(() => expect(screen.getByText("Chưa có cuộc gọi nào khớp.")).toBeInTheDocument());
  });

  it("deletes the rows older than 30 days, including the ones not on the page", async () => {
    const requests = serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-delete-older")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("llm-delete-older"));
    const modal = await screen.findByTestId("confirm-modal");
    expect(modal).toHaveTextContent("cũ hơn 30 ngày");
    expect(modal).toHaveTextContent("kể cả các dòng chưa hiện trên trang");
    fireEvent.click(within(modal).getByText("Xoá dòng cũ hơn 30 ngày"));
    await waitFor(() => expect(requests.find((r) => r.url.includes("/delete"))?.body).toEqual({ older_than_days: 30 }));
  });

  it("cancelling the confirmation deletes nothing", async () => {
    const requests = serve({ calls: rows });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-delete-older")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("llm-delete-older"));
    fireEvent.click(await screen.findByTestId("confirm-modal-cancel"));
    expect(requests.some((r) => r.url.includes("/delete"))).toBe(false);
  });

  it("loads the next page from where the last one stopped", async () => {
    const requests = serve({ calls: [rows[0]], next: "2026-10-25T10:00:00Z" });
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-load-more")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("llm-load-more"));
    await waitFor(() => expect(requests[requests.length - 1].url).toContain("before=2026-10-25T10%3A00%3A00Z"));
  });

  it("says why the list could not be read instead of showing it empty", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("down")) as unknown as typeof fetch;
    renderPanel();
    await waitFor(() => expect(screen.getByTestId("llm-calls-error")).toBeInTheDocument());
  });
});
