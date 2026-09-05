import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { PluginSelector } from "../../src/components/PluginSelector";

describe("PluginSelector", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads plugins on mount and renders options", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [{ plugin_id: "coding", name: "Lập trình" }],
    }) as unknown as typeof fetch;

    render(<PluginSelector value={null} onChange={vi.fn()} />);

    await waitFor(() =>
      expect(screen.getByRole("option", { name: "Lập trình" })).toBeInTheDocument(),
    );
  });
});
