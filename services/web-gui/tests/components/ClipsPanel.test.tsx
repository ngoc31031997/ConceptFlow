import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ClipsPanel } from "../../src/components/ClipsPanel";
import type { Clip } from "../../src/types";

describe("ClipsPanel", () => {
  it("báo rõ vì sao trống thay vì im lặng, khi script chưa đánh dấu self.clip(...)", () => {
    render(<ClipsPanel projectId="proj-1" clips={[]} videoOutputMode="short" />);
    expect(screen.getByText(/self\.clip/)).toBeInTheDocument();
  });

  it("liệt kê clip cắt thành công kèm nút tải", () => {
    const clips: Clip[] = [
      { name: "vi du", preset: "short", status: "ok", output_path: "/shared/p/clips/vi-du_short.mp4", duration_seconds: 45 },
    ];
    render(<ClipsPanel projectId="proj-1" clips={clips} videoOutputMode="both" />);

    const row = screen.getByTestId("clip-vi du-short");
    expect(row).toHaveTextContent("vi du");
    expect(row).toHaveTextContent("45s");
    expect(screen.getByRole("link", { name: "Tải clip" })).toHaveAttribute(
      "href",
      expect.stringContaining("/v1/projects/proj-1/clips/vi%20du/short"),
    );
  });

  it("hiện riêng clip lỗi kèm lý do, không lẫn với clip tải được", () => {
    const clips: Clip[] = [
      { name: "vi du", preset: "long", status: "error", error_message: "too short for preset" },
    ];
    render(<ClipsPanel projectId="proj-1" clips={clips} videoOutputMode="both" />);

    expect(screen.getByTestId("clip-error-vi du-long")).toHaveTextContent("too short for preset");
    expect(screen.queryByRole("link", { name: "Tải clip" })).not.toBeInTheDocument();
  });
});
