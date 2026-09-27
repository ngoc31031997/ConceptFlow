import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ProductionSettingsPanel } from "../../src/components/ProductionSettingsPanel";
import * as client from "../../src/api/client";
import type { Project } from "../../src/types";

const PROJECT: Project = {
  project_id: "p1",
  status: "awaiting_review",
  voice_language: "vi",
  scenes: [],
  render_quality: "720p30",
  render_engine: "remotion",
  subtitle_mode: "track",
  tts_enabled: true,
};

describe("ProductionSettingsPanel", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.spyOn(client, "getMusicInfo").mockResolvedValue({ exists: false, background_music_path: null } as never);
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows the project's saved values and only the requested stages", () => {
    render(<ProductionSettingsPanel project={PROJECT} stages={["render"]} hint="h" />);

    expect(screen.getByTestId("production-render-quality")).toHaveTextContent("Nháp (720p30)");
    expect(screen.getByTestId("video-font-picker")).toBeInTheDocument();
    expect(screen.queryByTestId("production-subtitles")).not.toBeInTheDocument();
  });

  it("hides the video font for Manim, whose fonts come from its theme", () => {
    render(<ProductionSettingsPanel project={{ ...PROJECT, render_engine: "manim" }} stages={["render"]} hint="h" />);

    expect(screen.queryByTestId("video-font-picker")).not.toBeInTheDocument();
  });

  it("saves a change on the project and remembers it as the next default", async () => {
    const patch = vi.spyOn(client, "patchWizardSettings").mockResolvedValue();
    const onSaved = vi.fn();
    render(<ProductionSettingsPanel project={PROJECT} stages={["merge"]} hint="h" onSaved={onSaved} />);

    fireEvent.click(screen.getByTestId("production-subtitles-toggle"));
    fireEvent.click(screen.getByTestId("subtitle-mode-off"));

    await waitFor(() => expect(patch).toHaveBeenCalledWith("p1", { subtitleMode: "off" }));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    const remembered = JSON.parse(window.localStorage.getItem("conceptflow.lastUsedSettings.v1") ?? "{}");
    expect(remembered.subtitleMode).toBe("off");
  });

  it("says so when the step reading the setting has already run (409)", async () => {
    vi.spyOn(client, "patchWizardSettings").mockRejectedValue(
      new client.ApiError("project status does not allow this operation", undefined, undefined, 409),
    );
    render(<ProductionSettingsPanel project={PROJECT} stages={["merge"]} hint="h" />);

    fireEvent.click(screen.getByTestId("production-subtitles-toggle"));
    fireEvent.click(screen.getByTestId("subtitle-mode-off"));

    expect(await screen.findByTestId("production-settings-error")).toHaveTextContent("không đổi được nữa");
  });
});
