import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { PromptSettingsPage } from "../../src/pages/PromptSettingsPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const SEED = {
  role: "story_architect" as const,
  language: "vi" as const,
  version: 3,
  template_text: "PROMPT GỐC CỦA HỆ THỐNG",
};

function stubLayers(overrides: apiClient.PromptOverride[]) {
  vi.spyOn(apiClient, "listPromptTemplates").mockResolvedValue([SEED]);
  vi.spyOn(apiClient, "listPromptOverrides").mockResolvedValue(overrides);
}

function anOverride(patch: Partial<apiClient.PromptOverride> = {}): apiClient.PromptOverride {
  return {
    role: "story_architect",
    language: "vi",
    template_text: "BẢN TUỲ CHỈNH CỦA TÔI",
    is_active: true,
    based_on_version: 3,
    ...patch,
  };
}

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <PromptSettingsPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

// CR-027 FR84 — màn admin giờ hiện hai tầng: bản gốc chỉ-đọc và bản tuỳ
// chỉnh của Creator, bật/tắt được.
describe("PromptSettingsPage — prompt hai tầng", () => {
  beforeEach(() => {
    vi.stubGlobal("confirm", () => true);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("chưa có bản tuỳ chỉnh thì chạy bản gốc, và bản gốc là chỉ-đọc", async () => {
    stubLayers([]);
    renderPage();

    await waitFor(() =>
      expect(screen.getByTestId("prompt-seed-textarea")).toHaveValue("PROMPT GỐC CỦA HỆ THỐNG"),
    );
    expect(screen.getByTestId("prompt-seed-textarea")).toHaveAttribute("readonly");
    expect(screen.getByTestId("prompt-in-use")).toHaveTextContent("bản gốc của hệ thống");
    expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("");
  });

  it("có bản tuỳ chỉnh đang bật thì nó là cái đang chạy", async () => {
    stubLayers([anOverride()]);
    renderPage();

    await waitFor(() =>
      expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("BẢN TUỲ CHỈNH CỦA TÔI"),
    );
    expect(screen.getByTestId("prompt-in-use")).toHaveTextContent("BẢN TUỲ CHỈNH");
    // Bản gốc vẫn hiện bên cạnh để đối chiếu, không bị thay thế.
    expect(screen.getByTestId("prompt-seed-textarea")).toHaveValue("PROMPT GỐC CỦA HỆ THỐNG");
  });

  // Đây là điểm khác cốt lõi so với nút "Khôi phục mặc định" cũ: tắt không
  // phá huỷ gì, nên không cần hỏi xác nhận và bật lại là có nguyên.
  it("tắt bản tuỳ chỉnh mà không xoá nội dung", async () => {
    stubLayers([anOverride()]);
    const setActive = vi.spyOn(apiClient, "setPromptOverrideActive").mockResolvedValue(anOverride({ is_active: false }));
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-toggle-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-toggle-button"));

    await waitFor(() => expect(setActive).toHaveBeenCalledWith("story_architect", "vi", false));
    // Không hỏi xác nhận, và không đụng tới đường xoá: tắt là thao tác có thể
    // lùi lại được.
    expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("BẢN TUỲ CHỈNH CỦA TÔI");
  });

  it("'Chép từ bản gốc' cho điểm khởi đầu thay vì bắt chép tay", async () => {
    stubLayers([]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-copy-seed-button")).toBeEnabled());
    fireEvent.click(screen.getByTestId("prompt-copy-seed-button"));

    expect(screen.getByTestId("prompt-template-textarea")).toHaveValue("PROMPT GỐC CỦA HỆ THỐNG");
  });

  it("lưu bản tuỳ chỉnh gửi đúng nội dung đang soạn", async () => {
    stubLayers([]);
    const save = vi.spyOn(apiClient, "savePromptOverride").mockResolvedValue(anOverride({ template_text: "chữ mới" }));
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-copy-seed-button")).toBeEnabled());
    fireEvent.change(screen.getByTestId("prompt-template-textarea"), { target: { value: "chữ mới" } });
    fireEvent.click(screen.getByTestId("prompt-save-button"));

    await waitFor(() => expect(save).toHaveBeenCalledWith("story_architect", "vi", "chữ mới"));
  });

  // FR84.7 — không có cảnh báo này thì Creator cứ chạy một bản đã tách ra từ
  // hai đời prompt trước mà không biết.
  it("báo khi bản gốc đã đi tiếp kể từ lúc viết bản tuỳ chỉnh", async () => {
    stubLayers([anOverride({ based_on_version: 1 })]); // bản gốc đang ở version 3
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-seed-moved-on")).toBeInTheDocument());
    expect(screen.getByTestId("prompt-seed-moved-on")).toHaveTextContent("phiên bản 3");
  });

  it("không báo khi bản tuỳ chỉnh vẫn khớp đời bản gốc", async () => {
    stubLayers([anOverride({ based_on_version: 3 })]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-in-use")).toBeInTheDocument());
    expect(screen.queryByTestId("prompt-seed-moved-on")).not.toBeInTheDocument();
  });

  // based_on_version = 0 là hàng do di trú CR-027 tạo ra: không biết nó dựa
  // trên đời nào, nên không được đoán bừa là đã lỗi thời.
  it("không báo khi không rõ bản tuỳ chỉnh dựa trên đời nào", async () => {
    stubLayers([anOverride({ based_on_version: 0 })]);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-in-use")).toBeInTheDocument());
    expect(screen.queryByTestId("prompt-seed-moved-on")).not.toBeInTheDocument();
  });

  it("xoá hỏi xác nhận, huỷ thì không gọi server", async () => {
    stubLayers([anOverride()]);
    const del = vi.spyOn(apiClient, "deletePromptOverride");
    vi.stubGlobal("confirm", () => false);
    renderPage();

    await waitFor(() => expect(screen.getByTestId("prompt-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("prompt-delete-button"));

    expect(del).not.toHaveBeenCalled();
  });
});
