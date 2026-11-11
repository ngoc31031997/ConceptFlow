import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { YoutubePromptCopy } from "../../src/components/reference/YoutubePromptCopy";
import * as apiClient from "../../src/api/client";

const LINK = "https://www.youtube.com/watch?v=abc123";
const TEMPLATE = "Xem {{url}} rồi mô tả. Nhắc lại: {{url}}.";

function stubClipboard(writeText: (t: string) => Promise<void>) {
  Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
}

function mockPrompt() {
  return vi
    .spyOn(apiClient, "getPromptTemplate")
    .mockResolvedValue({
      id: "p",
      role: "reference_style_youtube",
      name: "n",
      template_text: TEMPLATE,
      is_system: true,
      is_active: true,
    });
}

function typeLink(value: string) {
  fireEvent.change(screen.getByTestId("youtube-url-input"), { target: { value } });
}

describe("YoutubePromptCopy", () => {
  afterEach(() => vi.restoreAllMocks());

  it("locks the button with a reminder while the link is empty or not http(s)", () => {
    const get = mockPrompt();
    render(<YoutubePromptCopy />);
    expect(screen.getByTestId("youtube-copy")).toBeDisabled();
    expect(screen.getByTestId("youtube-url-hint")).toHaveTextContent("Nhập link YouTube");
    typeLink("youtube.com/watch?v=1");
    expect(screen.getByTestId("youtube-copy")).toBeDisabled();
    expect(screen.getByTestId("youtube-url-hint")).toHaveTextContent("http hoặc https");
    typeLink("ftp://youtube.com/x");
    expect(screen.getByTestId("youtube-copy")).toBeDisabled();
    expect(get).not.toHaveBeenCalled();
  });

  it("copies the library prompt with every {{url}} replaced by the link", async () => {
    const write = vi.fn().mockResolvedValue(undefined);
    stubClipboard(write);
    const get = mockPrompt();
    render(<YoutubePromptCopy />);
    typeLink(` ${LINK} `);
    fireEvent.click(screen.getByTestId("youtube-copy"));
    await waitFor(() => expect(write).toHaveBeenCalledTimes(1));
    expect(get).toHaveBeenCalledWith("reference_style_youtube");
    expect(write).toHaveBeenCalledWith(`Xem ${LINK} rồi mô tả. Nhắc lại: ${LINK}.`);
    expect(await screen.findByText("Đã chép")).toBeInTheDocument();
  });

  it("shows an error and copies nothing when the prompt is not in the library", async () => {
    const write = vi.fn().mockResolvedValue(undefined);
    stubClipboard(write);
    vi.spyOn(apiClient, "getPromptTemplate").mockRejectedValue(
      new apiClient.ApiError("not found", undefined, undefined, 404),
    );
    render(<YoutubePromptCopy />);
    typeLink(LINK);
    fireEvent.click(screen.getByTestId("youtube-copy"));
    expect(await screen.findByTestId("youtube-error")).toHaveTextContent(
      "Không lấy được prompt từ thư viện prompt",
    );
    expect(write).not.toHaveBeenCalled();
    expect(screen.queryByTestId("youtube-manual")).toBeNull();
    await waitFor(() => expect(screen.getByTestId("youtube-copy")).not.toBeDisabled());
  });

  it("shows the prompt selected for hand copying when the clipboard refuses", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("denied")));
    mockPrompt();
    render(<YoutubePromptCopy />);
    typeLink(LINK);
    fireEvent.click(screen.getByTestId("youtube-copy"));
    const box = (await screen.findByTestId("youtube-manual")) as HTMLTextAreaElement;
    expect(box.value).toBe(`Xem ${LINK} rồi mô tả. Nhắc lại: ${LINK}.`);
    await waitFor(() => expect(box.selectionEnd - box.selectionStart).toBe(box.value.length));
    expect(screen.queryByText("Đã chép")).toBeNull();
  });
});
