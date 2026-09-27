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
  description: "Răng hàm", usage: "<Tooth decay />", builtin: true, exemplar: false, warnings: [], status: "approved", version: 1, has_preview: true,
};
const BUS: apiClient.Illustration = {
  id: "b1", name: "Bus", title: "Xe buýt", folder_id: "phuong-tien", tags: ["xe"], description: "Xe buýt vàng",
  usage: "<Bus />", code: "export function Bus() {}", builtin: false, exemplar: false,
  warnings: [{ message: "[S9] màu #123456 không có trong bảng màu kênh", line: 4 }], status: "draft", version: 2, has_preview: true,
};
const CAT: apiClient.Illustration = {
  id: "exemplar-Cat", name: "Cat", title: "Con mèo", folder_id: "phuong-tien", tags: [], description: "Hình mẫu",
  usage: "<Cat />", code: "export function Cat() {}", builtin: true, exemplar: true, warnings: [], status: "approved", version: 1, has_preview: true,
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
    vi.spyOn(apiClient, "listIllustrations").mockResolvedValue([TOOTH, BUS, CAT]);
    vi.spyOn(apiClient, "getIllustrationStyle").mockResolvedValue({ rules: "[S1] PHẲNG: không gradient", exemplar_ids: ["exemplar-Cat"] });
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

  it("shows style warnings on the tile and in the editor, without blocking", async () => {
    renderPage();
    expect(await screen.findByTestId("illustration-warnings-Bus")).toHaveTextContent("1 cảnh báo style");
    fireEvent.click(screen.getByLabelText("Mở Xe buýt"));
    expect(screen.getByTestId("illustration-style-warnings")).toHaveTextContent("dòng 4");
    expect(screen.getByTestId("illustration-save-button")).not.toBeDisabled();
  });

  it("shows the style rules with the exemplar drawings", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("illustration-style-toggle"));
    expect(await screen.findByTestId("illustration-style-rules")).toHaveTextContent("[S1] PHẲNG");
    expect(within(screen.getByTestId("illustration-style")).getByTestId("illustration-tile-Cat")).toHaveTextContent("Hình mẫu");
  });

  it("turns an uploaded SVG into code in the editor, named after the file", async () => {
    renderPage();
    await screen.findByTestId("illustration-tile-Bus");
    const svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 50"><rect width="100" height="50" rx="8" fill="#FFC857"/></svg>';
    const file = new File([svg], "xe tải.svg", { type: "image/svg+xml" });
    Object.defineProperty(file, "text", { value: () => Promise.resolve(svg) });
    fireEvent.change(screen.getByTestId("illustration-upload-input"), { target: { files: [file] } });
    await waitFor(() => expect(screen.getByTestId("illustration-name-input")).toHaveValue("XeTai"));
    const code = (screen.getByTestId("illustration-code-input") as HTMLTextAreaElement).value;
    expect(code).toContain("export function XeTai(");
    expect(code).toContain("vw={100} vh={50}");
    expect(screen.getByTestId("illustration-usage-input")).toHaveValue("<XeTai /> — 100×50");
  });

  it("asks the AI to draw from a description into the chosen folder, then opens the result", async () => {
    const made = { ...BUS, id: "m1", name: "Motorbike", title: "Xe máy" };
    const draw = vi.spyOn(apiClient, "drawIllustration").mockResolvedValue(made);
    renderPage();
    fireEvent.click(await screen.findByTestId("illustration-draw-open"));
    expect(screen.getByTestId("illustration-draw-button")).toBeDisabled();
    fireEvent.change(screen.getByTestId("illustration-draw-text"), { target: { value: "xe máy đỏ" } });
    fireEvent.click(screen.getByTestId("illustration-draw-button"));
    await waitFor(() => expect(draw).toHaveBeenCalledWith({ description: "xe máy đỏ", folder_id: "co-the-suc-khoe" }));
    expect(await screen.findByTestId("illustration-tile-Motorbike")).toBeInTheDocument();
    expect(screen.getByTestId("illustration-editor")).toHaveTextContent("Sửa: Xe máy");
  });

  it("redraws one of my drawings with a note", async () => {
    const redraw = vi.spyOn(apiClient, "redrawIllustration").mockResolvedValue({ ...BUS, version: 3 });
    renderPage();
    fireEvent.click(await screen.findByTestId("edit-Bus"));
    fireEvent.change(screen.getByTestId("illustration-redraw-note"), { target: { value: "bánh to hơn" } });
    fireEvent.click(screen.getByTestId("illustration-redraw-button"));
    await waitFor(() => expect(redraw).toHaveBeenCalledWith("b1", "bánh to hơn"));
  });
});
