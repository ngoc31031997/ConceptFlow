import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { PromptSettingsPage } from "../../src/pages/PromptSettingsPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

beforeEach(() => {
  vi.spyOn(apiClient, "getPromptTemplate").mockResolvedValue({
    role: "story_architect",
    language: "vi",
    version: 3,
    template_text: "prompt tự sửa",
  });
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

describe("PromptSettingsPage — khôi phục prompt mặc định", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("thay nội dung trong ô soạn bằng bản mặc định server trả về", async () => {
    // Đây là đường duy nhất để prompt cải tiến trong source vào được một DB đã
    // bootstrap — seeding ở Orchestrator cố ý không đè lên bản sửa tay.
    const reset = vi
      .spyOn(apiClient, "resetPromptTemplate")
      .mockResolvedValue({
        role: "story_architect",
        language: "vi",
        version: 4,
        template_text: "prompt mặc định mới",
      });
    vi.stubGlobal("confirm", () => true);
    renderPage();

    await waitFor(() =>
      expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("prompt tự sửa"),
    );
    fireEvent.click(screen.getByTestId("prompt-reset-button"));

    await waitFor(() => expect(reset).toHaveBeenCalledWith("story_architect", "vi"));
    expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("prompt mặc định mới");
    expect(screen.getByTestId("prompt-settings-status")).toHaveTextContent("phiên bản 4");
  });

  it("không gọi server khi người dùng huỷ xác nhận", async () => {
    // Thao tác này xoá hẳn bản đã sửa và server không giữ lịch sử prompt, nên
    // huỷ phải thực sự là không có gì xảy ra.
    const reset = vi.spyOn(apiClient, "resetPromptTemplate");
    vi.stubGlobal("confirm", () => false);
    renderPage();

    await waitFor(() =>
      expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("prompt tự sửa"),
    );
    fireEvent.click(screen.getByTestId("prompt-reset-button"));

    expect(reset).not.toHaveBeenCalled();
    expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("prompt tự sửa");
  });
});
