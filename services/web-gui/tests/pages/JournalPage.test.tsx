import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { JournalPage, aggregateByStep } from "../../src/pages/JournalPage";
import type { ProjectEvent } from "../../src/api/client";
import { ThemeProvider } from "../../src/context/ThemeContext";

const events = [
  { id: 3, project_id: "bbbbbbbb-1", at: "2026-01-02T10:00:00Z", flow_step: 9, step_label: "Render hoạt hình", run_state: "failed", source: "saga", duration_ms: 5000, detail: "boom" },
  { id: 2, project_id: "aaaaaaaa-1", at: "2026-01-01T10:05:00Z", flow_step: 5, step_label: "Code", run_state: "done", source: "authoring", duration_ms: 90000, content_chars: 1200, prompt_tokens: 100, completion_tokens: 400 },
  { id: 1, project_id: "aaaaaaaa-1", at: "2026-01-01T10:00:00Z", flow_step: 5, step_label: "Code", run_state: "running", source: "authoring" },
];

describe("JournalPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("groups events by project and shows per-step time and tokens", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => events }) as unknown as typeof fetch;
    render(
      <ThemeProvider>
        <MemoryRouter>
          <JournalPage />
        </MemoryRouter>
      </ThemeProvider>,
    );

    // The per-project view is the second tab.
    await waitFor(() => expect(screen.getByTestId("journal-tab-project")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("journal-tab-project"));
    // Newest project first; its failure detail is visible.
    await waitFor(() => expect(screen.getByTestId("journal-project-bbbbbbbb-1")).toBeInTheDocument());
    expect(screen.getByText("boom")).toBeInTheDocument();

    fireEvent.click(screen.getByTestId("journal-project-aaaaaaaa-1"));
    await waitFor(() => expect(screen.getAllByTestId("journal-row")).toHaveLength(2));
    expect(screen.getByTestId("journal-summary")).toHaveTextContent("Code: 1m 30s, 500 token");
  });

  it("overview: attributes a saga move's time to the step it LEFT, and skips waiting on the Creator", async () => {
    const saga = (over: Partial<ProjectEvent>): ProjectEvent => ({
      id: 1, project_id: "p", at: "2026-01-01T00:00:00Z", flow_step: 9, step_label: "Render", run_state: "running", source: "saga", ...over,
    });
    const stats = aggregateByStep([
      // TTS ran 60s then the project entered render.
      saga({ id: 1, flow_step: 9, from_flow_step: 8, duration_ms: 60000 }),
      // Render took 4 minutes, then merge began.
      saga({ id: 2, flow_step: 10, from_flow_step: 9, duration_ms: 240000 }),
      // 20 minutes sitting at review is the Creator, not a step's cost.
      saga({ id: 3, flow_step: 8, from_flow_step: 7, duration_ms: 1200000 }),
      // A render failure counts against render (the step entered as failed).
      saga({ id: 4, flow_step: 9, from_flow_step: 9, run_state: "failed", duration_ms: 0 }),
    ]);
    const by = Object.fromEntries(stats.map((x) => [x.step, x]));
    expect(by[8].totalMs).toBe(60000);
    expect(by[9].totalMs).toBe(240000);
    expect(by[7]).toBeUndefined();
    expect(by[9].failures).toBe(1);
  });

  it("overview tab lists steps with average time and failures", async () => {
    global.fetch = vi.fn().mockResolvedValue({ ok: true, status: 200, json: async () => events }) as unknown as typeof fetch;
    render(
      <ThemeProvider>
        <MemoryRouter>
          <JournalPage />
        </MemoryRouter>
      </ThemeProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("journal-overview")).toBeInTheDocument());
    expect(screen.getByTestId("overview-row-5")).toHaveTextContent("1m 30s");
  });
});
