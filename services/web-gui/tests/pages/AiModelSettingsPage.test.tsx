import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { AiModelSettingsPage } from "../../src/pages/AiModelSettingsPage";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

function renderPage() {
  vi.spyOn(apiClient, "listProjects").mockResolvedValue([]);
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    default_model: "deepseek-ai/deepseek-v4.1-flash",
    models: [
      { id: "deepseek-ai/deepseek-v4.1-flash", label: "DeepSeek V4.1 Flash", code_ok: true },
      { id: "zai-org/glm-5.3-flash", label: "GLM-5.3-Flash", code_ok: true },
      { id: "ollama", label: "Ollama (local AI)", code_ok: false },
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

  it("shows the saved default, offers only models that serve every step, and saves a change", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue({ defaultModel: "", serverModel: "deepseek-ai/deepseek-v4.1-flash" });
    const save = vi.spyOn(apiClient, "setAiModelDefault").mockResolvedValue(undefined);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("ai-default-model")).not.toBeDisabled());
    expect(screen.getByTestId("ai-default-model-save")).toBeDisabled();
    fireEvent.click(screen.getByTestId("ai-default-model"));
    expect(await screen.findByTestId("ai-default-model-option-zai-org/glm-5.3-flash")).toBeInTheDocument();
    expect(screen.queryByTestId("ai-default-model-option-ollama")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("ai-default-model-option-zai-org/glm-5.3-flash"));
    fireEvent.click(screen.getByTestId("ai-default-model-save"));
    await waitFor(() => expect(save).toHaveBeenCalledWith("zai-org/glm-5.3-flash"));
    expect(await screen.findByTestId("ai-default-model-saved")).toBeInTheDocument();
  });

  it("says why when the server refuses the model", async () => {
    vi.spyOn(apiClient, "getAiModelSetting").mockResolvedValue({ defaultModel: "", serverModel: "m" });
    vi.spyOn(apiClient, "setAiModelDefault").mockRejectedValue(new apiClient.ApiError("model không hợp lệ"));
    renderPage();
    await waitFor(() => expect(screen.getByTestId("ai-default-model")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("ai-default-model"));
    fireEvent.click(await screen.findByTestId("ai-default-model-option-zai-org/glm-5.3-flash"));
    fireEvent.click(screen.getByTestId("ai-default-model-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("model không hợp lệ");
  });
});
