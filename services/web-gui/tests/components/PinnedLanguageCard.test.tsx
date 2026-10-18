import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { PinnedLanguageCard } from "../../src/components/PinnedLanguageCard";
import * as apiClient from "../../src/api/client";

function renderCard(projectId: string | null) {
  return render(
    <MemoryRouter initialEntries={["/create"]}>
      <Routes>
        <Route path="/create" element={<PinnedLanguageCard projectId={projectId} />} />
        <Route path="/settings/visual-languages" element={<p>trang ngôn ngữ hình</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("PinnedLanguageCard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("hiện tên và phiên bản đã ghim, và mở trang Ngôn ngữ hình", async () => {
    vi.spyOn(apiClient, "getProjectVisualLanguage").mockResolvedValue({
      visual_language_id: "phatsang", visual_language_version: 2, kit_key: "phatsang", name: "Khoa học phát sáng",
      rules: { illustration_rules: "a", backdrop_rules: "b", scene_rules: "c" },
    });
    renderCard("p1");
    expect(await screen.findByTestId("pinned-language-name")).toHaveTextContent("Khoa học phát sáng v2");
    fireEvent.click(screen.getByTestId("pinned-language-open"));
    expect(await screen.findByText("trang ngôn ngữ hình")).toBeInTheDocument();
  });

  it("project chưa ghim thì nói rõ là chưa vẽ hay render được", async () => {
    vi.spyOn(apiClient, "getProjectVisualLanguage").mockRejectedValue(new apiClient.ApiError("not pinned", undefined, undefined, 404));
    renderCard("p1");
    expect(await screen.findByTestId("pinned-language-error")).toHaveTextContent("chưa ghim ngôn ngữ hình");
  });

  it("lỗi khác thì báo không đọc được, không giả vờ là chưa ghim", async () => {
    vi.spyOn(apiClient, "getProjectVisualLanguage").mockRejectedValue(new apiClient.ApiError("boom", undefined, undefined, 502));
    renderCard("p1");
    expect(await screen.findByTestId("pinned-language-error")).toHaveTextContent("Không đọc được");
  });

  it("chưa có project thì không gọi máy chủ", () => {
    const get = vi.spyOn(apiClient, "getProjectVisualLanguage");
    renderCard(null);
    expect(get).not.toHaveBeenCalled();
  });
});
