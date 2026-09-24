import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { PromptSettingsPage } from "../../src/pages/PromptSettingsPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

function aPrompt(patch: Partial<apiClient.Prompt> = {}): apiClient.Prompt {
  return {
    id: "system-story_architect",
    role: "story_architect",
    name: "Mặc định của hệ thống",
    template_text: "PROMPT GỐC CỦA HỆ THỐNG",
    is_system: true,
    is_active: true,
    ...patch,
  };
}

const SYSTEM = aPrompt();
const MINE = aPrompt({
  id: "u1",
  name: "Bản của tôi",
  template_text: "BẢN CỦA TÔI",
  is_system: false,
  is_active: false,
});

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <PromptSettingsPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// CR-031 — mỗi vai trò một danh sách prompt, một dòng đang bật. Dòng hệ thống
// chỉ xem/copy; dòng của người dùng sửa/xoá/bật được.
describe("PromptSettingsPage — thư viện prompt", () => {
  beforeEach(() => {
    vi.stubGlobal("confirm", () => true);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("mở lên thì chọn dòng đang chạy; prompt hệ thống là chỉ-đọc và không có Xoá/Lưu", async () => {
    vi.spyOn(apiClient, "listPrompts").mockResolvedValue([SYSTEM, MINE]);
    renderPage();

    await waitFor(() =>
      expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("PROMPT GỐC CỦA HỆ THỐNG"),
    );
    expect(screen.getByTestId("prompt-template-textarea")).toHaveAttribute("readonly");
    expect(screen.getByTestId("prompt-name-input")).toHaveAttribute("readonly");
    expect(screen.queryByTestId("prompt-delete-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("prompt-save-button")).not.toBeInTheDocument();
    expect(screen.getByTestId("prompt-copy-button")).toBeInTheDocument();
    expect(screen.getByTestId("prompt-in-use")).toHaveTextContent("Mặc định của hệ thống (hệ thống)");
  });

  it("Copy dòng hệ thống tạo dòng mới rồi chọn nó", async () => {
    const list = vi.spyOn(apiClient, "listPrompts").mockResolvedValue([SYSTEM]);
    const copy = vi.spyOn(apiClient, "copyPrompt").mockResolvedValue(MINE);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-copy-button")).toBeInTheDocument());
    list.mockResolvedValue([SYSTEM, MINE]);
    fireEvent.click(screen.getByTestId("prompt-copy-button"));

    await waitFor(() => expect(copy).toHaveBeenCalledWith("system-story_architect"));
    await waitFor(() =>
      expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("BẢN CỦA TÔI"),
    );
    expect(screen.getByTestId("prompt-template-textarea")).not.toHaveAttribute("readonly");
  });

  it("dòng của người dùng sửa được, bật được và xoá được", async () => {
    vi.spyOn(apiClient, "listPrompts").mockResolvedValue([SYSTEM, MINE]);
    const update = vi.spyOn(apiClient, "updatePrompt").mockResolvedValue(MINE);
    const activate = vi.spyOn(apiClient, "activatePrompt").mockResolvedValue({ ...MINE, is_active: true });
    const del = vi.spyOn(apiClient, "deletePrompt").mockResolvedValue(undefined);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-row-u1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-row-u1"));
    await waitFor(() => expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("BẢN CỦA TÔI"));

    fireEvent.change(screen.getByTestId("prompt-template-textarea"), { target: { value: "chữ mới" } });
    fireEvent.click(screen.getByTestId("prompt-save-button"));
    await waitFor(() => expect(update).toHaveBeenCalledWith("u1", "Bản của tôi", "chữ mới"));

    fireEvent.click(screen.getByTestId("prompt-activate-button"));
    await waitFor(() => expect(activate).toHaveBeenCalledWith("u1"));

    fireEvent.click(screen.getByTestId("prompt-delete-button"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("u1"));
  });

  it("không xoá khi người dùng từ chối xác nhận", async () => {
    vi.stubGlobal("confirm", () => false);
    vi.spyOn(apiClient, "listPrompts").mockResolvedValue([SYSTEM, MINE]);
    const del = vi.spyOn(apiClient, "deletePrompt");
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-row-u1")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-row-u1"));
    await waitFor(() => expect(screen.getByTestId("prompt-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-delete-button"));
    expect(del).not.toHaveBeenCalled();
  });

  it("thêm prompt mới: cần tên và nội dung rồi tạo theo vai trò đang chọn", async () => {
    vi.spyOn(apiClient, "listPrompts").mockResolvedValue([SYSTEM]);
    const create = vi.spyOn(apiClient, "createPrompt").mockResolvedValue(MINE);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-new-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-new-button"));
    expect(screen.getByTestId("prompt-save-button")).toBeDisabled();

    fireEvent.change(screen.getByTestId("prompt-name-input"), { target: { value: "Bản mới" } });
    fireEvent.change(screen.getByTestId("prompt-template-textarea"), { target: { value: "nội dung" } });
    fireEvent.click(screen.getByTestId("prompt-save-button"));

    await waitFor(() => expect(create).toHaveBeenCalledWith("story_architect", "Bản mới", "nội dung"));
  });

  it("đổi vai trò thì chỉ hiện các dòng của vai trò đó", async () => {
    vi.spyOn(apiClient, "listPrompts").mockResolvedValue([
      SYSTEM,
      aPrompt({ id: "system-visual_director", role: "visual_director", template_text: "ĐẠO DIỄN" }),
    ]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-row-system-story_architect")).toBeInTheDocument());
    expect(screen.queryByTestId("prompt-row-system-visual_director")).not.toBeInTheDocument();

    fireEvent.change(screen.getByTestId("prompt-role-select"), { target: { value: "visual_director" } });
    await waitFor(() => expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("ĐẠO DIỄN"));
    expect(screen.queryByTestId("prompt-row-system-story_architect")).not.toBeInTheDocument();
  });
});
