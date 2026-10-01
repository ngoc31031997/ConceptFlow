import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MakeShortButton } from "../../src/components/MakeShortButton";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", () => ({ useNavigate: () => mockNavigate }));

const getAuthoringState = vi.fn();
const createProjectDraft = vi.fn();
vi.mock("../../src/api/client", () => ({
  ApiError: class ApiError extends Error {},
  getAuthoringState: (...args: unknown[]) => getAuthoringState(...args),
  createProjectDraft: (...args: unknown[]) => createProjectDraft(...args),
}));

describe("MakeShortButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getAuthoringState.mockResolvedValue({ topic: "Vì sao gọi là tiểu đường" });
    createProjectDraft.mockResolvedValue({ similarProjects: [] });
  });

  it("creates a Remotion short of this video on the same topic and opens its step 2", async () => {
    render(<MakeShortButton projectId="long-1" contentLanguage="vi" />);
    fireEvent.click(screen.getByTestId("make-short"));

    await waitFor(() => expect(mockNavigate).toHaveBeenCalled());
    const [shortId, topic, language, engine, shortOf] = createProjectDraft.mock.calls[0];
    expect([topic, language, engine, shortOf]).toEqual(["Vì sao gọi là tiểu đường", "vi", "remotion", "long-1"]);
    expect(mockNavigate).toHaveBeenCalledWith(`/projects/${shortId}/resume?step=2`);
  });

  it("says so when the short cannot be created", async () => {
    createProjectDraft.mockRejectedValue(new Error("boom"));
    render(<MakeShortButton projectId="long-1" contentLanguage="vi" />);
    fireEvent.click(screen.getByTestId("make-short"));

    expect(await screen.findByRole("alert")).toHaveTextContent("Không tạo được bản short: Error: boom");
    expect(mockNavigate).not.toHaveBeenCalled();
  });
});
