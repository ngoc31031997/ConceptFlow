import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { RenderPage } from "../../src/pages/RenderPage";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  close() {
    this.closed = true;
  }
}

function renderRenderPage() {
  return render(
    <MemoryRouter initialEntries={["/projects/p1/render"]}>
      <Routes>
        <Route path="/projects/:id/render" element={<RenderPage />} />
        <Route path="/" element={<div data-testid="new-project-page-stub" />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("RenderPage error recovery", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    // @ts-expect-error test stub
    global.EventSource = FakeEventSource;
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a back-to-edit button when the failure is an input-related step", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ project_id: "p1", status: "failed_at_parse_script", scenes: [], error_message: "Kịch bản không hợp lệ" }),
    }) as unknown as typeof fetch;

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("error-banner-back-button")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("error-banner-back-button"));
    expect(screen.getByTestId("new-project-page-stub")).toBeInTheDocument();
  });

  it("does not show a back-to-edit button when the failure is a system/processing step", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ project_id: "p1", status: "failed_at_render_scenes", scenes: [], error_message: "Render timeout" }),
    }) as unknown as typeof fetch;

    renderRenderPage();

    await waitFor(() => expect(screen.getByTestId("error-banner-retry-button")).toBeInTheDocument());
    expect(screen.queryByTestId("error-banner-back-button")).not.toBeInTheDocument();
  });
});
