import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { PublishForm } from "../../src/components/PublishForm";
import { ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import * as apiClient from "../../src/api/client";

describe("PublishForm", () => {
  it("disables submit until title is filled", () => {
    render(<PublishForm projectId="p1" onSubmit={vi.fn()} isSubmitting={false} />);
    expect(screen.getByTestId("publish-form-submit-button")).toBeDisabled();

    fireEvent.change(screen.getByTestId("publish-form-title-input"), {
      target: { value: "My Video" },
    });
    expect(screen.getByTestId("publish-form-submit-button")).not.toBeDisabled();
  });

  it("submits metadata with default private visibility", () => {
    const onSubmit = vi.fn();
    render(<PublishForm projectId="p1" onSubmit={onSubmit} isSubmitting={false} />);

    fireEvent.change(screen.getByTestId("publish-form-title-input"), {
      target: { value: "My Video" },
    });
    fireEvent.submit(screen.getByTestId("publish-form-submit-button").closest("form")!);

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ youtube_title: "My Video", visibility: "private" }),
    );
  });
});

describe("PublishForm model choice", () => {
  it("offers the model for the suggestion next to its button, saved on the project", async () => {
    vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
      enabled: true,
      provider: "hive",
      default_model: "deepseek-ai/deepseek-v4.1-flash",
      models: [
        { id: "deepseek-ai/deepseek-v4.1-flash", label: "DeepSeek V4.1 Flash", code_ok: true },
        { id: "ollama", label: "Ollama (local AI)", code_ok: false },
      ],
    });
    vi.spyOn(apiClient, "getAuthoringState").mockResolvedValue({ topic: "", story: "", storyboard: "", code: "" });
    const save = vi.spyOn(apiClient, "saveAuthoringModels").mockResolvedValue(undefined);
    render(
      <ProjectDraftProvider>
        <PublishForm projectId="p1" onSubmit={vi.fn()} isSubmitting={false} />
      </ProjectDraftProvider>,
    );

    fireEvent.click(await screen.findByTestId("publish-model"));
    fireEvent.click(await screen.findByTestId("publish-model-option-ollama"));
    await waitFor(() => expect(save).toHaveBeenCalledWith("p1", expect.objectContaining({ publish: "ollama" })));
  });
});
