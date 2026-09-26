import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VideoArchetypeSettingsPage } from "../../src/pages/VideoArchetypeSettingsPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const SYSTEM: apiClient.VideoArchetype = {
  id: "system-A", code: "A", name: "Nghịch lý", when_to_use: "lỗi tư duy", playbook: "PB-A", is_system: true,
};
const MINE: apiClient.VideoArchetype = {
  id: "u1", code: "E", name: "Của tôi", when_to_use: "hợp x", playbook: "PB-E", is_system: false,
};

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <VideoArchetypeSettingsPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("VideoArchetypeSettingsPage (CR-041)", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listVideoArchetypes").mockResolvedValue([SYSTEM, MINE]);
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows a system kind read-only, with Copy but no Save/Delete", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByTestId("archetype-name-input")).toHaveValue("Nghịch lý"));
    expect(screen.getByTestId("archetype-playbook-input")).toHaveAttribute("readonly");
    expect(screen.getByTestId("archetype-copy-button")).toBeInTheDocument();
    expect(screen.queryByTestId("archetype-save-button")).toBeNull();
    expect(screen.queryByTestId("archetype-delete-button")).toBeNull();
  });

  it("adds a new kind, leaving the code blank for the server to assign", async () => {
    const create = vi.spyOn(apiClient, "createVideoArchetype").mockResolvedValue({ ...MINE, id: "u2" });
    renderPage();
    await waitFor(() => screen.getByTestId("archetype-new-button"));
    fireEvent.click(screen.getByTestId("archetype-new-button"));
    expect(screen.getByTestId("archetype-save-button")).toBeDisabled();
    fireEvent.change(screen.getByTestId("archetype-name-input"), { target: { value: "Gỡ lỗi" } });
    fireEvent.change(screen.getByTestId("archetype-when-input"), { target: { value: "đoạn code sai" } });
    fireEvent.change(screen.getByTestId("archetype-playbook-input"), { target: { value: "hook = lỗi" } });
    fireEvent.click(screen.getByTestId("archetype-save-button"));
    await waitFor(() =>
      expect(create).toHaveBeenCalledWith({ code: "", name: "Gỡ lỗi", when_to_use: "đoạn code sai", playbook: "hook = lỗi" }),
    );
  });

  it("edits and deletes the creator's own kind", async () => {
    const update = vi.spyOn(apiClient, "updateVideoArchetype").mockResolvedValue(MINE);
    const del = vi.spyOn(apiClient, "deleteVideoArchetype").mockResolvedValue();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage();
    await waitFor(() => screen.getByTestId("archetype-row-u1"));
    fireEvent.click(screen.getByTestId("archetype-row-u1"));
    await waitFor(() => expect(screen.getByTestId("archetype-name-input")).toHaveValue("Của tôi"));
    fireEvent.change(screen.getByTestId("archetype-name-input"), { target: { value: "Đổi tên" } });
    fireEvent.click(screen.getByTestId("archetype-save-button"));
    await waitFor(() => expect(update).toHaveBeenCalledWith("u1", expect.objectContaining({ name: "Đổi tên", code: "E" })));
    fireEvent.click(screen.getByTestId("archetype-delete-button"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("u1"));
  });

  it("shows the server's reason when a save is refused", async () => {
    vi.spyOn(apiClient, "updateVideoArchetype").mockRejectedValue(new Error("code is already used"));
    renderPage();
    await waitFor(() => screen.getByTestId("archetype-row-u1"));
    fireEvent.click(screen.getByTestId("archetype-row-u1"));
    await waitFor(() => expect(screen.getByTestId("archetype-name-input")).toHaveValue("Của tôi"));
    fireEvent.click(screen.getByTestId("archetype-save-button"));
    await waitFor(() => expect(screen.getByTestId("archetype-status")).toHaveTextContent("code is already used"));
  });
});
