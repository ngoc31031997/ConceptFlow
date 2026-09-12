import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { CompanionProjectCard } from "../../src/components/CompanionProjectCard";

function renderCard() {
  return render(
    <MemoryRouter>
      <CompanionProjectCard companionProjectId="proj-short" />
    </MemoryRouter>,
  );
}

describe("CompanionProjectCard", () => {
  afterEach(() => vi.restoreAllMocks());

  it("hiện trạng thái và link xem đầy đủ của project liên kết", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        project_id: "proj-short",
        status: "ready_to_publish",
        voice_language: "vi",
        scenes: [],
        video_output_mode: "short",
        video_path: "/shared/proj-short/video/final.mp4",
        clips: [{ name: "short", preset: "short", status: "ok", output_path: "x", duration_seconds: 40 }],
      }),
    }) as never;

    renderCard();

    await waitFor(() => expect(screen.getByText(/Bản Shorts\/TikTok riêng/)).toBeInTheDocument());
    expect(screen.getByText(/Sẵn sàng đăng/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Xem đầy đủ" })).toHaveAttribute(
      "href",
      "/projects/proj-short/result",
    );
    // ClipsPanel dùng lại nguyên vẹn — clip đã cắt phải hiện ra ở đây luôn.
    expect(screen.getByTestId("clip-short-short")).toBeInTheDocument();
  });

  it("báo lỗi rõ ràng khi không tải được project liên kết", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      json: async () => ({ error: "project not found" }),
    }) as never;

    renderCard();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("project not found"));
  });
});
