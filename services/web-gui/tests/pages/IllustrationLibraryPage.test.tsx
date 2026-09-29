import { describe, it, expect, vi, beforeEach, afterEach, onTestFinished } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { IllustrationLibraryPage } from "../../src/pages/IllustrationLibraryPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const FOLDERS: apiClient.IllustrationFolder[] = [
  // CR-052: first in the list, yet never offered as a place to save or draw into.
  { id: "hinh-mau", name: "Hình mẫu", description: "", position: 0, is_system: true },
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
  id: "exemplar-Cat", name: "Cat", title: "Con mèo", folder_id: "hinh-mau", home_folder_id: "dong-vat", tags: [], description: "Hình mẫu",
  usage: "<Cat />", code: "export function Cat() {}", builtin: false, exemplar: true, warnings: [], status: "approved", version: 1, has_preview: true,
};
const VAN: apiClient.Illustration = {
  ...BUS, id: "v1", name: "Van", title: "Xe tải", code: "export function Van() {}", warnings: [], status: "approved", version: 1,
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
    vi.spyOn(apiClient, "getIllustrationStyle").mockResolvedValue({ rules: "[S1] PHẲNG: không gradient", exemplar_ids: ["exemplar-Cat"], max_exemplars: 5 });
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

  it("exports the whole library as a dated ZIP download", async () => {
    const blob = new Blob(["PK"], { type: "application/zip" });
    const exportIt = vi.spyOn(apiClient, "exportIllustrationLibrary").mockResolvedValue(blob);
    const createUrl = vi.fn(() => "blob:backup");
    const revokeUrl = vi.fn();
    const original = { createObjectURL: URL.createObjectURL, revokeObjectURL: URL.revokeObjectURL };
    Object.assign(URL, { createObjectURL: createUrl, revokeObjectURL: revokeUrl });
    onTestFinished(() => {
      Object.assign(URL, original);
    });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
    renderPage();
    fireEvent.click(await screen.findByTestId("illustration-export-button"));
    await waitFor(() => expect(click).toHaveBeenCalledTimes(1));
    expect(exportIt).toHaveBeenCalledTimes(1);
    expect(createUrl).toHaveBeenCalledWith(blob);
    const link = click.mock.contexts[0] as HTMLAnchorElement;
    expect(link.download).toMatch(/^conceptflow-thu-vien-hinh-\d{8}-\d{4}\.zip$/);
    expect(revokeUrl).toHaveBeenCalledWith("blob:backup");
  });

  it("restores a backup with the chosen conflict mode, reports each drawing and reloads the library", async () => {
    const importIt = vi.spyOn(apiClient, "importIllustrationLibrary").mockResolvedValue({
      exported_at: "2026-09-27T08:30:00Z",
      folders_created: ["do-choi"],
      folder_errors: [],
      items: [
        { name: "Ball", folder_id: "do-choi", result: "created" },
        { name: "Bus", folder_id: "phuong-tien", result: "replaced" },
        { name: "Kite", folder_id: "do-vat", result: "failed", reason: "dòng 3: TS2304" },
      ],
      not_processed: 0,
    });
    renderPage();
    await screen.findByTestId("illustration-tile-Bus");
    const list = vi.mocked(apiClient.listIllustrations);
    const loads = list.mock.calls.length;
    expect(screen.getByTestId("illustration-import-button")).toBeDisabled();

    const file = new File(["PK"], "thu-vien.zip", { type: "application/zip" });
    fireEvent.change(screen.getByTestId("illustration-import-input"), { target: { files: [file] } });
    expect(screen.getByTestId("illustration-import-file")).toHaveTextContent("thu-vien.zip");
    fireEvent.click(screen.getByTestId("illustration-import-replace"));
    fireEvent.click(screen.getByTestId("illustration-import-button"));

    const report = await screen.findByTestId("illustration-import-report");
    expect(importIt).toHaveBeenCalledWith(file, true);
    expect(report).toHaveTextContent("Đã thêm 1 · Ghi đè 1 · Bỏ qua 0 · Lỗi 1");
    expect(report).toHaveTextContent("do-choi");
    expect(report).toHaveTextContent("Kite — Lỗi: dòng 3: TS2304");
    await waitFor(() => expect(list.mock.calls.length).toBeGreaterThan(loads));
  });

  it("says how many drawings were left when a restore stops midway", async () => {
    vi.spyOn(apiClient, "importIllustrationLibrary").mockResolvedValue({
      exported_at: "", folders_created: [], folder_errors: [], items: [],
      aborted: "dừng ở hình Ball: rendering unreachable", not_processed: 5,
    });
    renderPage();
    const file = new File(["PK"], "b.zip", { type: "application/zip" });
    fireEvent.change(await screen.findByTestId("illustration-import-input"), { target: { files: [file] } });
    fireEvent.click(screen.getByTestId("illustration-import-button"));
    expect(await screen.findByTestId("illustration-import-aborted")).toHaveTextContent("còn 5 hình chưa nhập");
  });

  it("shows why a file that is not a backup was refused", async () => {
    vi.spyOn(apiClient, "importIllustrationLibrary").mockRejectedValue(
      new apiClient.ApiError("illustration library backup is not valid: thiếu manifest.json"),
    );
    renderPage();
    const file = new File(["x"], "anh.zip", { type: "application/zip" });
    fireEvent.change(await screen.findByTestId("illustration-import-input"), { target: { files: [file] } });
    fireEvent.click(screen.getByTestId("illustration-import-button"));
    expect(await screen.findByRole("alert")).toHaveTextContent("thiếu manifest.json");
  });

  // CR-052 — delete from the tile, and the Hình mẫu the Creator picks.
  it("deletes a drawing from its tile after confirming", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const del = vi.spyOn(apiClient, "deleteIllustration").mockResolvedValue(undefined);
    renderPage();
    fireEvent.click(await screen.findByTestId("delete-Bus"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("b1"));
    await waitFor(() => expect(screen.queryByTestId("illustration-tile-Bus")).toBeNull());
  });

  it("keeps the drawing and says which project still uses it when the delete is refused", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    vi.spyOn(apiClient, "deleteIllustration").mockRejectedValue(
      new apiClient.ApiError('Đang được dùng trong dự án "Sâu răng" — xoá được sau khi dự án tới bước Kết quả.', undefined, undefined, 409),
    );
    renderPage();
    fireEvent.click(await screen.findByTestId("delete-Bus"));
    expect(await screen.findByRole("status")).toHaveTextContent('dự án "Sâu răng"');
    expect(screen.getByTestId("illustration-tile-Bus")).toBeInTheDocument();
  });

  it("offers no delete, edit or review on the kit and the Hình mẫu", async () => {
    renderPage();
    await screen.findByTestId("illustration-tile-Bus");
    for (const name of ["Tooth", "Cat"]) {
      expect(screen.queryByTestId(`delete-${name}`)).toBeNull();
      expect(screen.queryByTestId(`edit-${name}`)).toBeNull();
      expect(screen.queryByTestId(`approve-${name}`)).toBeNull();
    }
    fireEvent.click(screen.getAllByLabelText("Mở Con mèo")[0]);
    expect(screen.getByTestId("illustration-editor")).toHaveTextContent("(Hình mẫu)");
    expect(screen.getByTestId("illustration-code-input")).toHaveAttribute("readonly");
    expect(screen.queryByTestId("illustration-save-button")).toBeNull();
    expect(screen.queryByTestId("illustration-delete-button")).toBeNull();
  });

  it("makes an approved drawing a Hình mẫu and takes one out again", async () => {
    vi.spyOn(apiClient, "listIllustrations").mockResolvedValue([TOOTH, BUS, CAT, VAN]);
    const copy: apiClient.Illustration = { ...VAN, id: "v2", name: "VanMau", folder_id: "hinh-mau", exemplar: true, source_id: "v1" };
    const make = vi.spyOn(apiClient, "makeExemplar").mockResolvedValue(copy);
    const unmake = vi.spyOn(apiClient, "unmakeExemplar").mockResolvedValue(undefined);
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage();
    expect(screen.queryByTestId("make-exemplar-Bus")).toBeNull(); // a draft cannot be one
    fireEvent.click(await screen.findByTestId("make-exemplar-Van"));
    await waitFor(() => expect(make).toHaveBeenCalledWith("v1"));
    fireEvent.click(screen.getByTestId("illustration-style-toggle"));
    expect(await screen.findByTestId("illustration-exemplar-count")).toHaveTextContent("Hình mẫu (2/5)");
    expect(screen.queryByTestId("make-exemplar-Van")).toBeNull(); // already has its copy

    fireEvent.click(screen.getByTestId("unmake-exemplar-VanMau"));
    await waitFor(() => expect(unmake).toHaveBeenCalledWith("v2"));
    await waitFor(() => expect(screen.getByTestId("illustration-exemplar-count")).toHaveTextContent("Hình mẫu (1/5)"));
    expect(screen.getByTestId("make-exemplar-Van")).toBeInTheDocument();
  });

  it("files an original Hình mẫu back into its folder when taken out", async () => {
    const back: apiClient.Illustration = { ...CAT, exemplar: false, folder_id: "dong-vat", home_folder_id: undefined };
    vi.spyOn(apiClient, "unmakeExemplar").mockResolvedValue(back);
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPage();
    fireEvent.click(await screen.findByTestId("unmake-exemplar-tile-Cat"));
    await waitFor(() => expect(screen.getByTestId("illustration-status-Cat")).toHaveTextContent("Đã duyệt"));
    expect(screen.getByTestId("delete-Cat")).toBeInTheDocument();
  });

  it("cannot add a sixth Hình mẫu", async () => {
    vi.spyOn(apiClient, "listIllustrations").mockResolvedValue([TOOTH, BUS, CAT, VAN]);
    vi.spyOn(apiClient, "getIllustrationStyle").mockResolvedValue({ rules: "", exemplar_ids: ["exemplar-Cat"], max_exemplars: 1 });
    renderPage();
    expect(await screen.findByTestId("make-exemplar-Van")).toBeDisabled();
  });
});
