import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { VideoFontPicker } from "../../src/components/VideoFontPicker";
import * as client from "../../src/api/client";

describe("VideoFontPicker", () => {
  afterEach(() => vi.restoreAllMocks());

  it("offers the fonts rendering lists and marks the current one", async () => {
    vi.spyOn(client, "listVideoFonts").mockResolvedValue({
      fonts: ["Be Vietnam Pro", "Montserrat"],
      default: "Be Vietnam Pro",
    });
    render(<VideoFontPicker value="Montserrat" onChange={() => {}} />);

    expect(screen.getByTestId("video-font-loading")).toBeInTheDocument();
    expect(await screen.findByTestId("video-font-Be Vietnam Pro")).toBeInTheDocument();
    expect(screen.getByTestId("video-font-Montserrat")).toBeInTheDocument();
    expect(screen.queryByTestId("video-font-error")).not.toBeInTheDocument();
  });

  it("says the list could not be fetched and offers no font to pick", async () => {
    vi.spyOn(client, "listVideoFonts").mockRejectedValue(new Error("down"));
    render(<VideoFontPicker value="Montserrat" onChange={() => {}} />);

    expect(await screen.findByTestId("video-font-error")).toHaveTextContent(
      "Không lấy được danh sách font từ rendering",
    );
    expect(screen.queryByTestId("video-font-Montserrat")).not.toBeInTheDocument();
  });

  it("keeps a saved font that rendering no longer lists, marked as missing", async () => {
    vi.spyOn(client, "listVideoFonts").mockResolvedValue({ fonts: ["Be Vietnam Pro"], default: "Be Vietnam Pro" });
    render(<VideoFontPicker value="Old Font" onChange={() => {}} />);

    expect(await screen.findByTestId("video-font-Old Font")).toHaveTextContent("Old Font (không có trong rendering)");
  });
});
