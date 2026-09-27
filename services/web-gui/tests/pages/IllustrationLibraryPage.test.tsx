import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { IllustrationLibraryPage } from "../../src/pages/IllustrationLibraryPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const FOLDERS: apiClient.IllustrationFolder[] = [
  { id: "co-the-suc-khoe", name: "Cơ thể & sức khoẻ", description: "", position: 1, is_system: true },
  { id: "phuong-tien", name: "Phương tiện", description: "", position: 2, is_system: true },
];
const TOOTH: apiClient.Illustration = {
  id: "builtin-Tooth", name: "Tooth", title: "Chiếc răng", folder_id: "co-the-suc-khoe", tags: ["sâu răng"],
  description: "Răng hàm", usage: "<Tooth decay />", builtin: true, status: "approved", version: 1, has_preview: true,
};
const BUS: apiClient.Illustration = {
  id: "b1", name: "Bus", title: "Xe buýt", folder_id: "phuong-tien", tags: ["xe"], description: "Xe buýt vàng",
  usage: "<Bus />", code: "export function Bus() {}", builtin: false, status: "draft", version: 2, has_preview: true,
};

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <IllustrationLibraryPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("IllustrationLibraryPage (CR-044)", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listIllustrationFolders").mockResolvedValue(FOLDERS);
    vi.spyOn(apiClient, "listIllustrations").mockResolvedValue([TOOTH, BUS]);
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows every drawing as its own tile with its status, and filters by folder and search", async () => {
    renderPage();
    await screen.findByTestId("illustration-tile-Bus");
    expect(screen.getByTestId("illustration-status-Tooth")).toHaveTextContent("Có sẵn");
    expect(screen.getByTestId("illustration-status-Bus")).toHaveTextContent("Chờ duyệt");
    // The tile image is the versioned preview URL.
    const img = within(screen.getByTestId("illustration-tile-Bus")).getByRole("img");
    expect(img.getAttribute("src")).toContain("/v1/illustrations/b1/preview.png?v=2");

    fireEvent.click(screen.getByTestId("illustration-folder-phuong-tien"));
    expect(screen.queryByTestId("illustration-tile-Tooth")).toBeNull();
    fireEvent.click(screen.getByText("Tất cả"));
    fireEvent.change(screen.getByTestId("illustration-search"), { target: { value: "sâu" } });
    expect(screen.getByTestId("illustration-tile-Tooth")).toBeInTheDocument();
    expect(screen.queryByTestId("illustration-tile-Bus")).toBeNull();
  });

  it("approves a draft from its tile", async () => {
    const approve = vi.spyOn(apiClient, "setIllustrationStatus").mockResolvedValue({ ...BUS, status: "approved" });
    renderPage();
    fireEvent.click(await screen.findByTestId("approve-Bus"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("b1", "approved"));
    await waitFor(() => expect(screen.getByTestId("illustration-status-Bus")).toHaveTextContent("Đã duyệt"));
  });

  it("re-renders one tile only", async () => {
    const rerender = vi.spyOn(apiClient, "rerenderIllustration").mockResolvedValue({ png: "", gif: "" });
    renderPage();
    fireEvent.click(await screen.findByTestId("rerender-Tooth"));
    await waitFor(() => expect(rerender).toHaveBeenCalledWith("builtin-Tooth"));
    expect(rerender).toHaveBeenCalledTimes(1);
  });

  it("opens a built-in read-only, without code or save", async () => {
    renderPage();
    fireEvent.click(await screen.findByLabelText("Mở Chiếc răng"));
    expect(screen.getByTestId("illustration-editor")).toHaveTextContent("(có sẵn)");
    expect(screen.queryByTestId("illustration-code-input")).toBeNull();
    expect(screen.queryByTestId("illustration-save-button")).toBeNull();
  });

  it("lists the renderer's line-numbered errors when new code does not pass", async () => {
    vi.spyOn(apiClient, "createIllustration").mockRejectedValue(
      new apiClient.ApiError("code hình minh hoạ chưa qua kiểm tra", undefined, [
        { message: "TS2304: Cannot find name 'colr'", line: 11 },
      ]),
    );
    renderPage();
    fireEvent.click(await screen.findByTestId("illustration-new-button"));
    fireEvent.change(screen.getByTestId("illustration-name-input"), { target: { value: "Truck" } });
    fireEvent.click(screen.getByTestId("illustration-save-button"));
    const list = await screen.findByTestId("illustration-diagnostics");
    expect(list).toHaveTextContent("dòng 11");
    expect(list).toHaveTextContent("colr");
  });

  it("previews unsaved code without saving it", async () => {
    const tryIt = vi.spyOn(apiClient, "tryIllustration").mockResolvedValue({ png: "AAAA", gif: "BBBB" });
    const create = vi.spyOn(apiClient, "createIllustration");
    renderPage();
    fireEvent.click(await screen.findByLabelText("Mở Xe buýt"));
    fireEvent.click(screen.getByTestId("illustration-try-button"));
    await screen.findByTestId("illustration-tried");
    expect(tryIt).toHaveBeenCalledWith("Bus", "export function Bus() {}");
    expect(create).not.toHaveBeenCalled();
  });
});
