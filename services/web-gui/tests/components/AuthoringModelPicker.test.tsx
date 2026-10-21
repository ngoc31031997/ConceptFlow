import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, within } from "@testing-library/react";
import { AuthoringModelPicker, codeStatsLine } from "../../src/components/AuthoringModelPicker";
import type { AuthoringModelOption, ModelUsageStats } from "../../src/api/client";

const OPTIONS: AuthoringModelOption[] = [
  { id: "ollama", label: "Ollama (local AI)", local: true },
  { id: "deepseek-ai/deepseek-v4.1-flash", label: "DeepSeek V4.1 Flash" },
  { id: "zai-org/glm-5.3-flash", label: "GLM-5.3-Flash" },
];
const DEEPSEEK: ModelUsageStats = {
  model: "deepseek-ai/deepseek-v4.1-flash", calls: 169, ok: 107, failures: { empty: 45, server: 17 },
  avg_completion_tokens: 27962, avg_duration_ms: 200761,
};
const GLM: ModelUsageStats = {
  model: "zai-org/glm-5.3-flash", calls: 22, ok: 18, failures: { budget: 2, truncated: 1, balance: 1 },
  avg_completion_tokens: 97870, avg_duration_ms: 527338,
};

function openOptions(testId: string): string[] {
  fireEvent.click(screen.getByTestId(testId));
  return within(screen.getByRole("listbox")).getAllByRole("option").map((o) => o.textContent ?? "");
}

describe("AuthoringModelPicker", () => {
  it("offers Ollama for every step, the code step included", () => {
    render(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "" }} onChange={vi.fn()}
      options={OPTIONS} />);
    expect(openOptions("authoring-model-code").some((t) => t.includes("Ollama"))).toBe(true);
  });

  it("warns on the code step when the chosen model is the local one", () => {
    const { rerender } = render(<AuthoringModelPicker models={{ story: "", storyboard: "", code: GLM.model }} onChange={vi.fn()}
      options={OPTIONS} />);
    expect(screen.queryByTestId("authoring-model-code-local-warning")).not.toBeInTheDocument();

    rerender(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "ollama" }} onChange={vi.fn()}
      options={OPTIONS} />);
    expect(screen.getByTestId("authoring-model-code-local-warning")).toHaveTextContent("hết giờ");

    rerender(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "" }} onChange={vi.fn()}
      options={OPTIONS} stepDefaults={{ code: "ollama" }} />);
    expect(screen.getByTestId("authoring-model-code-local-warning")).toBeInTheDocument();
  });

  it("pre-selects the default saved for each step, and the first model when a step has none", () => {
    render(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "" }} onChange={vi.fn()}
      options={OPTIONS}
      stepDefaults={{ story: "deepseek-ai/deepseek-v4.1-flash", code: "zai-org/glm-5.3-flash" }} />);
    expect(screen.getByTestId("authoring-model-story")).toHaveTextContent("DeepSeek V4.1 Flash");
    expect(screen.getByTestId("authoring-model-storyboard")).toHaveTextContent("Ollama");
    expect(screen.getByTestId("authoring-model-code")).toHaveTextContent("GLM-5.3-Flash");
  });

  it("shows the measured cost of the chosen code model, as a warning when it ran out of budget", () => {
    const { rerender } = render(<AuthoringModelPicker models={{ story: "", storyboard: "", code: DEEPSEEK.model }} onChange={vi.fn()}
      options={OPTIONS}
      codeStats={{ [DEEPSEEK.model]: DEEPSEEK, [GLM.model]: GLM }} />);
    const line = screen.getByTestId("authoring-model-code-stats");
    expect(line).toHaveTextContent("107/169 đoạn code đạt");
    expect(line).toHaveTextContent("TB 28k token, 3,3 phút mỗi đoạn");
    expect(line).toHaveTextContent("trả về rỗng 45");

    rerender(<AuthoringModelPicker models={{ story: "", storyboard: "", code: GLM.model }} onChange={vi.fn()}
      options={OPTIONS}
      codeStats={{ [DEEPSEEK.model]: DEEPSEEK, [GLM.model]: GLM }} />);
    expect(screen.getByTestId("authoring-model-code-stats")).toHaveTextContent("TB 98k token, 8,8 phút mỗi đoạn");
    expect(codeStatsLine(GLM).warn).toBe(true);
    expect(codeStatsLine(DEEPSEEK).warn).toBe(false);
  });

  it("says when there is no data, and says so differently when the numbers could not be read", () => {
    const { rerender } = render(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "" }} onChange={vi.fn()}
      options={OPTIONS} codeStats={{}} />);
    expect(screen.getByTestId("authoring-model-code-stats")).toHaveTextContent("Chưa có số liệu");

    rerender(<AuthoringModelPicker models={{ story: "", storyboard: "", code: "" }} onChange={vi.fn()}
      options={OPTIONS} codeStatsError="không đọc được số liệu đã đo" />);
    expect(screen.getByTestId("authoring-model-code-stats")).toHaveTextContent("không đọc được số liệu đã đo");
  });
});
