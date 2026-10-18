import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { VisualLanguagePage } from "../../src/pages/VisualLanguagePage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const RULES = { illustration_rules: "[S1] luật hình", backdrop_rules: "[B1] luật nền", scene_rules: "[C1] luật cảnh" };

const SYSTEM: apiClient.VisualLanguage = {
  id: "phatsang", name: "Khoa học phát sáng", note: "Tham khảo: phim khoa học", description: "Ánh sáng kể chuyện.",
  kit_key: "phatsang", is_system: true, is_active: true, version: 1, rules: RULES, created_at: "2026-10-18T00:00:00Z",
};
const MINE: apiClient.VisualLanguage = {
  ...SYSTEM, id: "mine", name: "Khoa học phát sáng (bản sao)", note: "", is_system: false, is_active: false,
};
const SAMPLES: apiClient.LanguageSample[] = (["fig", "space"] as const).flatMap((scene) =>
  (["wide", "tall"] as const).map((frame) => ({ kit_key: "phatsang", kit_version: 1, scene, frame, file: `${scene}-${frame}.png` })),
);

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <VisualLanguagePage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("VisualLanguagePage", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listVisualLanguages").mockResolvedValue([SYSTEM, MINE]);
    vi.spyOn(apiClient, "listLanguageSamples").mockResolvedValue(SAMPLES);
  });
  afterEach(() => vi.restoreAllMocks());

  it("hiện ngôn ngữ đang bật, bốn ảnh mẫu và ba luật chỉ đọc của ngôn ngữ hệ thống", async () => {
    renderPage();
    expect(await screen.findByTestId("language-active-strip")).toHaveTextContent("Đang bật: Khoa học phát sáng v1");
    const samples = await screen.findByTestId("language-samples");
    await waitFor(() => expect(samples.querySelectorAll("img")).toHaveLength(4));
    expect(samples.querySelector("img")?.getAttribute("src")).toContain("/v1/stills/fig-wide.png");
    for (const key of ["illustration_rules", "backdrop_rules", "scene_rules"]) {
      expect(screen.getByTestId(`language-rule-${key}`)).toHaveAttribute("readonly");
    }
    expect(screen.getByTestId("language-copy")).toBeEnabled();
    expect(screen.queryByTestId("language-edit")).not.toBeInTheDocument();
    expect(screen.queryByTestId("language-delete")).not.toBeInTheDocument();
    expect(screen.getByTestId("language-activate")).toBeDisabled();
  });

  it("sao chép tạo bản của bạn và chọn nó", async () => {
    const copy = vi.spyOn(apiClient, "copyVisualLanguage").mockResolvedValue(MINE);
    renderPage();
    fireEvent.click(await screen.findByTestId("language-copy"));
    await waitFor(() => expect(copy).toHaveBeenCalledWith("phatsang"));
    expect(await screen.findByTestId("language-status")).toHaveTextContent("Đã sao chép");
  });

  it("bật ngôn ngữ của bạn", async () => {
    const activate = vi.spyOn(apiClient, "activateVisualLanguage").mockResolvedValue(undefined);
    renderPage();
    fireEvent.click(await screen.findByTestId("language-row-mine"));
    expect(screen.getByTestId("language-activate-note")).toHaveTextContent("video đã tạo giữ ngôn ngữ đã ghim");
    fireEvent.click(screen.getByTestId("language-activate"));
    await waitFor(() => expect(activate).toHaveBeenCalledWith("mine"));
  });

  it("sửa luật mở hộp thoại, lưu tạo phiên bản mới và nói video đã ghim giữ bản cũ", async () => {
    const update = vi.spyOn(apiClient, "updateVisualLanguage").mockResolvedValue({ ...MINE, version: 2 });
    renderPage();
    fireEvent.click(await screen.findByTestId("language-row-mine"));
    fireEvent.click(screen.getByTestId("language-edit"));
    fireEvent.change(await screen.findByTestId("language-edit-scene_rules"), { target: { value: "[C1] luật cảnh mới" } });
    fireEvent.click(screen.getByTestId("language-save"));
    await waitFor(() =>
      expect(update).toHaveBeenCalledWith("mine", {
        name: MINE.name, note: "", description: MINE.description,
        rules: { ...RULES, scene_rules: "[C1] luật cảnh mới" },
      }),
    );
    expect(await screen.findByTestId("language-status")).toHaveTextContent("Video đã ghim giữ phiên bản cũ");
  });

  it("không lưu được khi một luật để trống", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("language-row-mine"));
    fireEvent.click(screen.getByTestId("language-edit"));
    fireEvent.change(await screen.findByTestId("language-edit-backdrop_rules"), { target: { value: "  " } });
    expect(screen.getByTestId("language-save")).toBeDisabled();
  });

  it("xoá hỏi trước, nói không khôi phục được; huỷ thì không xoá", async () => {
    const del = vi.spyOn(apiClient, "deleteVisualLanguage").mockResolvedValue(undefined);
    renderPage();
    fireEvent.click(await screen.findByTestId("language-row-mine"));
    fireEvent.click(screen.getByTestId("language-delete"));
    const modal = await screen.findByTestId("confirm-modal");
    expect(modal).toHaveTextContent("Không khôi phục lại được");
    fireEvent.click(screen.getByTestId("confirm-modal-cancel"));
    expect(del).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("language-delete"));
    fireEvent.click(await screen.findByTestId("confirm-modal-confirm"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("mine"));
  });

  it("ngôn ngữ đang bật không xoá được", async () => {
    vi.spyOn(apiClient, "listVisualLanguages").mockResolvedValue([SYSTEM, { ...MINE, is_active: true }]);
    renderPage();
    fireEvent.click(await screen.findByTestId("language-row-mine"));
    expect(screen.getByTestId("language-delete")).toBeDisabled();
  });

  it("dựng lại ảnh mẫu và báo khi rendering lỗi", async () => {
    const rebuild = vi.spyOn(apiClient, "rebuildLanguageSamples").mockResolvedValueOnce(SAMPLES);
    renderPage();
    fireEvent.click(await screen.findByTestId("language-rebuild-samples"));
    await waitFor(() => expect(rebuild).toHaveBeenCalledWith("phatsang"));
    expect(await screen.findByTestId("language-status")).toHaveTextContent("Đã dựng lại ảnh mẫu");

    rebuild.mockRejectedValueOnce(new apiClient.ApiError("rendering unreachable"));
    fireEvent.click(screen.getByTestId("language-rebuild-samples"));
    expect(await screen.findByTestId("language-status")).toHaveTextContent("rendering unreachable");
  });

  it("chưa có ảnh mẫu thì nói rõ, không để khung trống", async () => {
    vi.spyOn(apiClient, "listLanguageSamples").mockResolvedValue([]);
    renderPage();
    expect(await screen.findByTestId("language-no-samples")).toHaveTextContent("Chưa có ảnh mẫu");
  });
});
