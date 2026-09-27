import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { IllustrationsStepPage } from "../../src/pages/IllustrationsStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

function renderPage(engine: "remotion" | "manim", chain: apiClient.AuthoringChainState) {
  // The draft itself is not kept across reloads; the engine comes from the last-used settings.
  window.localStorage.setItem("conceptflow.lastUsedSettings.v1", JSON.stringify({ renderEngine: engine }));
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({ enabled: true, provider: "hive" });
  vi.spyOn(apiClient, "getAuthoringChain").mockResolvedValue(chain);
  vi.spyOn(apiClient, "saveAuthoringMode").mockResolvedValue(undefined);
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ mode: "ai", topic: "Răng", story: "s", storyboard: "{}", code: "" });
  vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [], ready: true });
  vi.spyOn(apiClient, "listIllustrationFolders").mockResolvedValue([]);
  return render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/create/script/illustrations"]}>
        <ProjectDraftProvider>
          <AuthoringRunProvider>
            <Routes>
              <Route path="/create/script/illustrations" element={<IllustrationsStepPage />} />
              <Route path="/create/script/code" element={<p>trang code</p>} />
            </Routes>
          </AuthoringRunProvider>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

const NONE: apiClient.AuthoringChainState = { running: false, steps: [], current_index: 0, finished: false };

describe("IllustrationsStepPage (CR-045)", () => {
  beforeEach(() => window.localStorage.clear());
  afterEach(() => {
    vi.restoreAllMocks();
    window.localStorage.clear();
  });

  it("is its own tab for a Remotion video: run bar for this step, then the drawing list", async () => {
    renderPage("remotion", NONE);
    expect(await screen.findByTestId("illustrations-step-page")).toBeInTheDocument();
    expect(await screen.findByTestId("run-with-ai-illustrations")).toHaveTextContent("Chạy hình minh hoạ bằng AI");
    expect(await screen.findByTestId("project-illustrations")).toBeInTheDocument();
    expect(screen.getByTestId("illustrations-step-next")).toHaveTextContent("Sang bước Code");
  });

  it("a Manim video has no drawings step: it goes on to Code", async () => {
    renderPage("manim", NONE);
    expect(await screen.findByText("trang code")).toBeInTheDocument();
  });

  it("the running chain lists four steps and counts the drawings as they are drawn", async () => {
    vi.spyOn(apiClient, "getAuthoringProgress").mockResolvedValue({
      running: true, phase: "draw", reasoning_chars: 0, content_chars: 0, elapsed_seconds: 75,
      drawings_total: 5, drawings_done: 2, drawings_failed: 1, drawings_reused: 3, drawings_planned: 8,
    });
    renderPage("remotion", { running: true, steps: ["story", "storyboard", "illustrations", "code"], current_index: 2, finished: false });
    const step = await screen.findByTestId("authoring-run-panel-illustrations");
    expect(screen.getAllByRole("listitem").filter((li) => li.getAttribute("data-testid")?.startsWith("authoring-run-panel-"))).toHaveLength(4);
    await waitFor(() => expect(step).toHaveTextContent("Đang vẽ 2/5 hình · 1 lỗi · 3 hình dùng lại · 1m15s"));
    expect(step).toHaveAttribute("aria-current", "step");
    const bar = step.querySelector('[role="progressbar"]');
    expect(bar).toHaveAttribute("aria-valuenow", "40");
  });

  it("a chain that stopped for review says so plainly — not as an error", async () => {
    renderPage("remotion", {
      running: false, steps: ["story", "storyboard", "illustrations", "code"], current_index: 2, finished: true,
      waiting: "Đã vẽ xong. Còn 2 hình chờ bạn duyệt hoặc bỏ qua (Motorbike, Candy) — duyệt xong thì chạy bước Code.",
      waiting_step: "illustrations", finished_at: new Date().toISOString(),
    });
    // The bar here runs this step alone; the chain above was started from the outline tab.
    const waiting = await screen.findByTestId("run-with-ai-waiting");
    expect(waiting).toHaveTextContent("Còn 2 hình chờ bạn duyệt");
    expect(screen.queryByTestId("run-with-ai-error")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-with-ai-done")).not.toBeInTheDocument();
    expect(screen.queryByTestId("run-with-ai-open-review")).not.toBeInTheDocument();
  });
});
