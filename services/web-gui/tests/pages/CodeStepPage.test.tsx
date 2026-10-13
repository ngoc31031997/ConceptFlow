import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { CodeStepPage } from "../../src/pages/CodeStepPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import { mockRenderPrompt } from "../helpers/renderPromptMock";

const VALID_CODE = [
  "import {registerRoot, Composition} from 'remotion';",
  "import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';",
  "",
  'export const narrations: string[] = ["Xin chao"];',
  "",
  "function CreatorComposition({segments = []}) {",
  "  return <Segments segments={segments}>{(index) => <p>{narrations[index]}</p>}</Segments>;",
  "}",
  "",
  "registerRoot(() => (",
  '  <Composition id="creator" component={CreatorComposition} width={1920} height={1080} fps={30} durationInFrames={150} calculateMetadata={calculateMetadataFromSegments} />',
  "));",
].join("\n");

// The Code step: fetches the remotion_engineer template and rehydrates saved
// authoring state from the server — stub both so these tests don't need a
// live backend, mirroring VisualDirectorStepPage.test.tsx's stub.
beforeEach(() => {
  mockRenderPrompt();
  vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({
    topic: "",
    story: "",
    storyboard: "",
    code: "",
  });
  vi.spyOn(apiClient, "saveAuthoringCode").mockResolvedValue(undefined);
  vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [], ready: true });
});

describe("CodeStepPage", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("blocks the step until valid Remotion code is pasted, then saves and advances", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(screen.getByTestId("code-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("code-step-code-input"), {
      target: { value: "```tsx\nnot valid tsx at all" },
    });
    expect(screen.getByTestId("code-step-next")).toBeDisabled();

    fireEvent.change(screen.getByTestId("code-step-code-input"), {
      target: { value: VALID_CODE },
    });
    expect(screen.getByTestId("code-step-next")).not.toBeDisabled();

    fireEvent.click(screen.getByTestId("code-step-next"));

    await waitFor(() => {
      expect(apiClient.saveAuthoringCode).toHaveBeenCalledWith(expect.any(String), VALID_CODE);
    });
  });

  it("shows an error and stays put when saving fails", async () => {
    vi.spyOn(apiClient, "saveAuthoringCode").mockRejectedValue(new Error("boom"));

    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("code-step-code-input"), {
      target: { value: VALID_CODE },
    });
    fireEvent.click(screen.getByTestId("code-step-next"));

    await waitFor(() => {
      expect(screen.getByText("Không lưu được code. Vui lòng thử lại.")).toBeInTheDocument();
    });
  });

  it("strips a pasted markdown code fence instead of saving the ``` markers as part of the script", () => {
    // The engineer prompt asks the AI to answer with exactly one ```tsx
    // fenced block. Pasting that whole block, fence included, is
    // the single most common way this round trip fails: esbuild
    // chokes on line 1 with an opaque syntax error that says nothing about
    // the real cause.
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.change(screen.getByTestId("code-step-code-input"), {
      target: { value: "```tsx\n" + VALID_CODE + "\n```" },
    });

    expect(screen.getByTestId("code-step-code-input")).toHaveValue(VALID_CODE);
  });

  // The {{topic}} substitution lives on the server now (and is held
  // to the old TypeScript output by its golden tests). This proves the wiring: the
  // page sends the draft's values and shows what the server answers.
  it("asks the server to render the engineer prompt and shows its answer", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    await waitFor(() => {
      expect((screen.getByTestId("code-step-prompt") as HTMLTextAreaElement).value).toContain(
        "RENDERED[remotion_engineer]",
      );
    });
    expect(apiClient.renderPrompt).toHaveBeenCalledWith(
      expect.objectContaining({ role: "remotion_engineer", language: expect.stringMatching(/^(vi|en)$/) }),
    );
  });

  it("has no render engine picker", () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    fireEvent.click(screen.getByTestId("pipeline-settings-toggle"));
    // Screen titles name the step as the rail does.
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bước 6 — Code");
    expect(screen.queryByTestId("render-engine-picker")).not.toBeInTheDocument();
  });

  // The drawings moved to their own step; the Code tab only says where they stand.
  it("the Code tab shows how many drawings are ready and leads back to the drawings step", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({
      illustrations: [
        { id: "r1", position: 1, name: "Candy", description: "", folder_id: "do-vat", shots: [], state: "skipped" },
        { id: "r2", position: 2, name: "Cat", description: "", folder_id: "dong-vat", shots: [], state: "planned" },
      ],
      ready: false,
    });
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );
    expect(await screen.findByTestId("illustrations-gate-note")).toHaveTextContent("1/2 hình sẵn sàng");
    expect(screen.queryByTestId("project-illustrations")).not.toBeInTheDocument();
    expect(screen.getByText("Quay lại Hình minh hoạ")).toBeInTheDocument();
  });
  it("nói rõ cần dán code gì ở chế độ tự làm, và quay lại đúng bước trước", async () => {
    render(
      <ThemeProvider>
        <MemoryRouter>
          <ProjectDraftProvider>
            <CodeStepPage />
          </ProjectDraftProvider>
        </MemoryRouter>
      </ThemeProvider>,
    );

    expect(await screen.findByText("Dán code Remotion từ AI để tiếp tục.")).toBeInTheDocument();
    expect(screen.getByTestId("wizard-back")).toHaveTextContent("Quay lại Hình minh hoạ");
    // The "already have code" helper is gone from this step.
    expect(screen.queryByTestId("existing-code-assistant")).not.toBeInTheDocument();
  });
});
