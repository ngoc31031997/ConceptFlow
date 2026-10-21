import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AiModelSettingsPage } from "../../src/pages/AiModelSettingsPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const STEPS = ["brief", "research", "story_frame", "beat_sheet", "story", "storyboard", "code", "publish", "illustrations"];

/** Every step on the first catalog model, as the server reports an unconfigured one. */
function setting(over: Record<string, string> = {}): apiClient.AiModelSetting {
  return { stepModels: { ...Object.fromEntries(STEPS.map((s) => [s, "ollama"])), ...over }, steps: STEPS };
}

function renderPage() {
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    models: [
      { id: "ollama", label: "Ollama (local AI)", local: true },
      { id: "deepseek-ai/deepseek-v4.1-flash", label: "DeepSeek V4.1 Flash" },
      { id: "zai-org/glm-5.3-flash", label: "GLM-5.3-Flash" },
    ],
  });
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <ProjectDraftProvider>
          <AiModelSettingsPage />
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("AiModelSettingsPage", () => {
  afterEach(() => vi.restoreAllMocks());

  it("has no app-wide model, one dropdown per step, and every step starts on Ollama", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue(setting());
    renderPage();

    await waitFor(() => expect(screen.getByTestId("ai-step-model-brief")).toBeInTheDocument());
    expect(screen.queryByTestId("ai-default-model")).not.toBeInTheDocument();
    for (const step of STEPS) expect(screen.getByTestId(`ai-step-model-${step}`)).toHaveTextContent("Ollama");
    expect(screen.getByTestId("ai-default-model-save")).toBeDisabled();
  });

  it("lists Ollama first, then the Hive models", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue(setting());
    renderPage();

    await waitFor(() => expect(screen.getByTestId("ai-step-model-brief")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("ai-step-model-brief"));
    const options = (await screen.findAllByRole("option")).map((o) => o.textContent ?? "");
    expect(options[0]).toContain("Ollama");
    expect(options).toHaveLength(3);
  });

  it("saves only the steps that changed", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue(setting({ research: "deepseek-ai/deepseek-v4.1-flash" }));
    const saveStep = vi.spyOn(apiClient, "setAiModelStepDefault").mockResolvedValue(undefined);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("ai-step-model-research")).toHaveTextContent("DeepSeek V4.1 Flash"));
    fireEvent.click(screen.getByTestId("ai-step-model-brief"));
    fireEvent.click(await screen.findByTestId("ai-step-model-brief-option-zai-org/glm-5.3-flash"));
    fireEvent.click(screen.getByTestId("ai-step-model-research"));
    fireEvent.click(await screen.findByTestId("ai-step-model-research-option-ollama"));
    fireEvent.click(screen.getByTestId("ai-default-model-save"));

    await waitFor(() => expect(saveStep).toHaveBeenCalledTimes(2));
    expect(saveStep).toHaveBeenCalledWith("brief", "zai-org/glm-5.3-flash");
    expect(saveStep).toHaveBeenCalledWith("research", "ollama");
    expect(await screen.findByTestId("ai-default-model-saved")).toBeInTheDocument();
    expect(screen.getByTestId("ai-default-model-save")).toBeDisabled();
  });

  it("warns about the local model on code and drawings only", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue(setting());
    renderPage();

    await waitFor(() => expect(screen.getByTestId("ai-step-model-code-local-warning")).toHaveTextContent("hết giờ"));
    expect(screen.getByTestId("ai-step-model-illustrations-local-warning")).toBeInTheDocument();
    expect(screen.queryByTestId("ai-step-model-brief-local-warning")).not.toBeInTheDocument();
  });

  it("says why when the server refuses the model", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue(setting());
    vi.spyOn(apiClient, "setAiModelStepDefault").mockRejectedValue(new apiClient.ApiError("model không hợp lệ"));
    renderPage();
    await waitFor(() => expect(screen.getByTestId("ai-step-model-brief")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("ai-step-model-brief"));
    fireEvent.click(await screen.findByTestId("ai-step-model-brief-option-zai-org/glm-5.3-flash"));
    fireEvent.click(screen.getByTestId("ai-default-model-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("model không hợp lệ");
  });
});
