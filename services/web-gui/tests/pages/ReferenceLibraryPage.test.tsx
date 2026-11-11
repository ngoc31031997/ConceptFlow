import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ReferenceLibraryPage } from "../../src/pages/ReferenceLibraryPage";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { ProjectSummary } from "../../src/types";

const EMPTY_DRAWING: apiClient.ReferenceDrawing = {
  medium: "",
  shape: "",
  outline: "",
  fill: "",
  shading: "",
  light: "",
  depth: "",
  perspective: "",
  texture: "",
};
const R01: apiClient.ReferenceStyle = {
  id: "R01",
  name: "Ánh sáng xuyên lớp mỏng",
  summary: "Nền tối ấm; mép chủ thể phát sáng.",
  tags: ["ánh sáng", "màu"],
  source: { url: "https://example.com/v", title: "Video mẫu", kind: "video" },
  drawing: { ...EMPTY_DRAWING, shape: "tròn mềm", texture: "giấy" },
  layout: { structure: "một chủ thể giữa", layering: "", persistent_elements: "logo góc" },
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};
const R02: apiClient.ReferenceStyle = {
  ...R01,
  id: "R02",
  name: "Sơ đồ tối giản",
  tags: [],
  source: { url: "", title: "", kind: "" },
};
const NO_SOURCE = { url: "", title: "", kind: "" };
const NO_LAYOUT = { structure: "", layering: "", persistent_elements: "" };

function renderPage() {
  return render(
    <ThemeProvider>
      <MemoryRouter>
        <ReferenceLibraryPage />
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("ReferenceLibraryPage", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listReferenceStyles").mockResolvedValue([R01, R02]);
    vi.spyOn(apiClient, "getReferenceStyleUsage").mockResolvedValue([]);
  });
  afterEach(() => vi.restoreAllMocks());

  it("lists every description with its id, name, summary, tags and source link", async () => {
    renderPage();
    const row = await screen.findByTestId("reference-row-R01");
    expect(within(row).getByText("R01")).toBeInTheDocument();
    expect(within(row).getByText(R01.name)).toBeInTheDocument();
    expect(within(row).getByText(R01.summary)).toBeInTheDocument();
    expect(within(row).getByText("ánh sáng")).toBeInTheDocument();
    expect(screen.getByTestId("reference-row-R02")).toBeInTheDocument();
    const link = within(row).getByRole("link", { name: "Video mẫu" });
    expect(link).toHaveAttribute("href", "https://example.com/v");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("opens every drawing and layout field when a row is clicked", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-row-R01"));
    const dialog = await screen.findByTestId("reference-detail-dialog");
    expect(within(dialog).getByTestId("reference-detail-drawing")).toHaveTextContent("tròn mềm");
    expect(within(dialog).getByTestId("reference-detail-drawing")).toHaveTextContent("giấy");
    expect(within(dialog).getByTestId("reference-detail-layout")).toHaveTextContent("logo góc");
    expect(within(dialog).getAllByText("Cách vẽ").length).toBeGreaterThan(0);
  });

  it("the row's buttons do not also open the detail view", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-edit-R01"));
    await screen.findByTestId("reference-edit-dialog");
    expect(screen.queryByTestId("reference-detail-dialog")).toBeNull();
  });

  it("the edit form carries every field of the description", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-edit-R01"));
    expect(await screen.findByTestId("reference-summary-input")).toHaveValue(R01.summary);
    expect(screen.getByTestId("reference-source-url-input")).toHaveValue("https://example.com/v");
    expect(screen.getByTestId("reference-source-kind-select")).toHaveValue("video");
    expect(screen.getByTestId("reference-drawing-shape-input")).toHaveValue("tròn mềm");
    expect(screen.getByTestId("reference-layout-persistent_elements-input")).toHaveValue(
      "logo góc",
    );
    expect(screen.getByText("Cách vẽ")).toBeInTheDocument();
    expect(screen.getByText("Layout")).toBeInTheDocument();
  });

  it("fills the form from pasted JSON for review without saving", async () => {
    const imported = { ...R02, name: "Đã dán", tags: ["x"] };
    const importFn = vi.spyOn(apiClient, "importReferenceStyle").mockResolvedValue(imported);
    const create = vi.spyOn(apiClient, "createReferenceStyle");
    renderPage();
    await waitFor(() => expect(screen.getByTestId("reference-add-button")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("reference-add-button"));
    fireEvent.click(await screen.findByTestId("reference-paste-toggle"));
    fireEvent.change(screen.getByTestId("reference-paste-input"), {
      target: { value: '{"name":"Đã dán"}' },
    });
    fireEvent.click(screen.getByTestId("reference-paste-read"));
    await waitFor(() => expect(screen.getByTestId("reference-name-input")).toHaveValue("Đã dán"));
    expect(importFn).toHaveBeenCalledWith('{"name":"Đã dán"}');
    expect(screen.getByTestId("reference-tags-input")).toHaveValue("x");
    expect(screen.getByTestId("reference-drawing-shape-input")).toHaveValue("tròn mềm");
    expect(create).not.toHaveBeenCalled();
    expect(screen.queryByTestId("reference-paste-box")).toBeNull();
  });

  it("shows the field at fault when pasted JSON is refused and keeps the form", async () => {
    vi.spyOn(apiClient, "importReferenceStyle").mockRejectedValue(
      new Error("drawing.shape: dài quá 300 ký tự"),
    );
    renderPage();
    await waitFor(() => expect(screen.getByTestId("reference-add-button")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("reference-add-button"));
    fireEvent.change(await screen.findByTestId("reference-name-input"), {
      target: { value: "giữ lại" },
    });
    fireEvent.click(screen.getByTestId("reference-paste-toggle"));
    fireEvent.change(screen.getByTestId("reference-paste-input"), { target: { value: "{}" } });
    fireEvent.click(screen.getByTestId("reference-paste-read"));
    expect(await screen.findByTestId("reference-paste-error")).toHaveTextContent(
      "drawing.shape: dài quá 300 ký tự",
    );
    expect(screen.getByTestId("reference-name-input")).toHaveValue("giữ lại");
  });

  it("shows an empty library with a hint", async () => {
    vi.mocked(apiClient.listReferenceStyles).mockResolvedValue([]);
    renderPage();
    expect(await screen.findByTestId("reference-empty")).toBeInTheDocument();
  });

  it("adds a description, splitting the tags on commas", async () => {
    const create = vi
      .spyOn(apiClient, "createReferenceStyle")
      .mockResolvedValue({ ...R01, id: "R03" });
    renderPage();
    await waitFor(() => expect(screen.getByTestId("reference-add-button")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("reference-add-button"));
    const dialog = await screen.findByTestId("reference-edit-dialog");
    expect(within(dialog).getByText("Thêm mô tả tham chiếu")).toBeInTheDocument();
    expect(screen.getByTestId("reference-edit-save")).toBeDisabled();
    fireEvent.change(screen.getByTestId("reference-name-input"), { target: { value: "Giấy cắt" } });
    fireEvent.change(screen.getByTestId("reference-summary-input"), {
      target: { value: "Nhiều lớp giấy" },
    });
    fireEvent.change(screen.getByTestId("reference-tags-input"), {
      target: { value: " chiều sâu, , bối cảnh " },
    });
    fireEvent.click(screen.getByTestId("reference-edit-save"));
    await waitFor(() =>
      expect(create).toHaveBeenCalledWith({
        name: "Giấy cắt",
        summary: "Nhiều lớp giấy",
        tags: ["chiều sâu", "bối cảnh"],
        source: NO_SOURCE,
        drawing: EMPTY_DRAWING,
        layout: NO_LAYOUT,
      }),
    );
    expect(await screen.findByTestId("reference-status")).toHaveTextContent("Đã thêm R03.");
    expect(apiClient.getReferenceStyleUsage).not.toHaveBeenCalled();
  });

  it("keeps the dialog open with the server's reason when saving fails", async () => {
    vi.spyOn(apiClient, "createReferenceStyle").mockRejectedValue(
      new Error("mô tả tham chiếu không hợp lệ: tối đa 8 nhãn"),
    );
    renderPage();
    await waitFor(() => expect(screen.getByTestId("reference-add-button")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("reference-add-button"));
    fireEvent.change(await screen.findByTestId("reference-name-input"), { target: { value: "n" } });
    fireEvent.change(screen.getByTestId("reference-summary-input"), { target: { value: "d" } });
    fireEvent.click(screen.getByTestId("reference-edit-save"));
    expect(await screen.findByTestId("reference-edit-error")).toHaveTextContent("tối đa 8 nhãn");
    expect(screen.getByTestId("reference-edit-dialog")).toBeInTheDocument();
  });

  it("edits an unused description with a plain save", async () => {
    const update = vi.spyOn(apiClient, "updateReferenceStyle").mockResolvedValue(R01);
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-edit-R01"));
    expect(await screen.findByTestId("reference-name-input")).toHaveValue(R01.name);
    expect(screen.getByTestId("reference-tags-input")).toHaveValue("ánh sáng, màu");
    await waitFor(() =>
      expect(screen.getByTestId("reference-edit-save")).toHaveTextContent(/^Lưu$/),
    );
    expect(screen.queryByTestId("reference-usage")).toBeNull();
    fireEvent.change(screen.getByTestId("reference-name-input"), { target: { value: "Đổi tên" } });
    fireEvent.click(screen.getByTestId("reference-edit-save"));
    await waitFor(() => expect(update).toHaveBeenCalledWith("R01", { ...R01, name: "Đổi tên" }));
  });

  it("lists, by topic, the videos that picked a description before saving an edit", async () => {
    vi.mocked(apiClient.getReferenceStyleUsage).mockResolvedValue(["p-1", "p-2"]);
    vi.spyOn(apiClient, "listProjects").mockResolvedValue([
      {
        project_id: "p-1",
        status: "draft",
        updated_at: "",
        topic: "Vì sao trong quả sung có xác ong?",
      },
      { project_id: "p-2", status: "draft", updated_at: "", topic: "" },
    ] as ProjectSummary[]);
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-edit-R01"));
    const usage = await screen.findByTestId("reference-usage");
    expect(usage).toHaveTextContent("Vì sao trong quả sung có xác ong?");
    expect(usage).toHaveTextContent("p-2");
    expect(usage).toHaveTextContent(
      "các phần đọc tham chiếu ở bước Ngôn ngữ hình ảnh sẽ cần làm lại; bản đã chốt giữ nguyên.",
    );
    expect(screen.getByTestId("reference-edit-save")).toHaveTextContent("Lưu và làm lại");
  });

  it("asks before deleting and names the videos that picked it", async () => {
    vi.mocked(apiClient.getReferenceStyleUsage).mockResolvedValue(["p-1"]);
    vi.spyOn(apiClient, "listProjects").mockResolvedValue([
      { project_id: "p-1", status: "draft", updated_at: "", topic: "Quả sung" },
    ] as ProjectSummary[]);
    const del = vi.spyOn(apiClient, "deleteReferenceStyle").mockResolvedValue();
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-delete-R02"));
    const dialog = await screen.findByTestId("reference-delete-dialog");
    expect(within(dialog).getByText("Xoá mô tả R02?")).toBeInTheDocument();
    expect(await within(dialog).findByTestId("reference-usage")).toHaveTextContent("Quả sung");
    expect(del).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("reference-delete-confirm"));
    await waitFor(() => expect(del).toHaveBeenCalledWith("R02"));
    expect(await screen.findByTestId("reference-status")).toHaveTextContent("Đã xoá R02.");
  });

  it("cancelling the delete confirmation deletes nothing", async () => {
    const del = vi.spyOn(apiClient, "deleteReferenceStyle").mockResolvedValue();
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-delete-R01"));
    await screen.findByTestId("reference-delete-dialog");
    fireEvent.click(screen.getByTestId("reference-delete-cancel"));
    await waitFor(() => expect(screen.queryByTestId("reference-delete-dialog")).toBeNull());
    expect(del).not.toHaveBeenCalled();
  });

  it("does not let a delete go through while the usage could not be checked", async () => {
    vi.mocked(apiClient.getReferenceStyleUsage).mockRejectedValue(
      new Error("upstream_unavailable"),
    );
    renderPage();
    fireEvent.click(await screen.findByTestId("reference-delete-R01"));
    expect(await screen.findByTestId("reference-usage-error")).toBeInTheDocument();
    expect(screen.getByTestId("reference-delete-confirm")).toBeDisabled();
  });
});
