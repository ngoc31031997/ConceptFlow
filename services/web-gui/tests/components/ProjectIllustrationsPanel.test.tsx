import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ProjectIllustrationsPanel } from "../../src/components/ProjectIllustrationsPanel";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const ill = (over: Partial<apiClient.Illustration>): apiClient.Illustration => ({
  id: "i", name: "X", title: "X", folder_id: "do-vat", tags: [], description: "", usage: "", builtin: false,
  exemplar: false, warnings: [], status: "draft", version: 1, has_preview: true, ...over,
});

const TOOTH: apiClient.ProjectIllustration = {
  id: "r1", position: 1, name: "Tooth", description: "Răng", folder_id: "co-the-suc-khoe", shots: [], state: "reused",
  illustration_id: "builtin-Tooth", illustration: ill({ id: "builtin-Tooth", name: "Tooth", title: "Chiếc răng", builtin: true, status: "approved" }),
};
const MOTO: apiClient.ProjectIllustration = {
  id: "r2", position: 2, name: "Motorbike", description: "Xe máy đỏ", folder_id: "phuong-tien", shots: ["1.1"], state: "drawn",
  illustration_id: "m1", illustration: ill({ id: "m1", name: "Motorbike", title: "Xe máy" }),
};
const DENTIST: apiClient.ProjectIllustration = {
  id: "r3", position: 3, name: "Dentist", description: "Nha sĩ", folder_id: "con-nguoi", shots: ["2.1", "2.2"], state: "failed",
  error: "AI vẽ 3 lần vẫn chưa qua kiểm tra",
};

function renderPanel() {
  return render(
    <ThemeProvider>
      <ProjectIllustrationsPanel projectId="p1" />
    </ThemeProvider>,
  );
}

describe("ProjectIllustrationsPanel (CR-044)", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listIllustrationFolders").mockResolvedValue([]);
  });
  afterEach(() => vi.restoreAllMocks());

  it("says how many drawings still hold the code step, and shows each as its own tile", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [TOOTH, MOTO, DENTIST], ready: false });
    renderPanel();
    expect(await screen.findByTestId("pi-summary")).toHaveTextContent("1/3 hình sẵn sàng");
    expect(screen.getByTestId("illustration-tile-Tooth")).toBeInTheDocument();
    expect(screen.getByTestId("illustration-tile-Motorbike")).toBeInTheDocument();
    expect(screen.getByTestId("pi-row-Dentist")).toHaveTextContent("AI vẽ 3 lần vẫn chưa qua kiểm tra");
    expect(screen.getByTestId("pi-draw-Dentist")).toHaveTextContent("Vẽ lại");
    expect(screen.getByTestId("pi-draw-all")).toHaveTextContent("Vẽ 1 hình còn thiếu");
  });

  it("approves a drawn drawing and skips a failed one, then reloads", async () => {
    const approvedMoto = { ...MOTO, illustration: ill({ id: "m1", name: "Motorbike", title: "Xe máy", status: "approved" as const }) };
    const list = vi.spyOn(apiClient, "listProjectIllustrations")
      .mockResolvedValueOnce({ illustrations: [TOOTH, MOTO, DENTIST], ready: false })
      .mockResolvedValueOnce({ illustrations: [TOOTH, approvedMoto, DENTIST], ready: false })
      .mockResolvedValue({ illustrations: [TOOTH, approvedMoto, { ...DENTIST, state: "skipped" }], ready: true });
    const approve = vi.spyOn(apiClient, "setIllustrationStatus").mockResolvedValue(ill({ status: "approved" }));
    const skip = vi.spyOn(apiClient, "skipProjectIllustration").mockResolvedValue({ ...DENTIST, state: "skipped" });
    renderPanel();
    fireEvent.click(await screen.findByTestId("pi-approve-Motorbike"));
    await waitFor(() => expect(approve).toHaveBeenCalledWith("m1", "approved"));
    await waitFor(() => expect(screen.getByTestId("pi-summary")).toHaveTextContent("2/3 hình sẵn sàng"));
    fireEvent.click(screen.getByTestId("pi-skip-Dentist"));
    await waitFor(() => expect(skip).toHaveBeenCalledWith("p1", "r3", true));
    await waitFor(() => expect(screen.getByTestId("pi-summary")).toHaveTextContent("Đủ 3 hình — bước Code chạy được."));
    expect(list.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it("plans from the storyboard when there is no list yet", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [], ready: true });
    const plan = vi.spyOn(apiClient, "planProjectIllustrations").mockResolvedValue({ illustrations: [{ ...DENTIST, state: "planned", error: undefined }], ready: false });
    renderPanel();
    expect(await screen.findByTestId("pi-summary")).toHaveTextContent("Chưa có danh sách");
    fireEvent.click(screen.getByTestId("pi-plan"));
    await waitFor(() => expect(plan).toHaveBeenCalledWith("p1"));
    expect(await screen.findByTestId("pi-draw-Dentist")).toHaveTextContent("Vẽ");
  });

  // CR-045 — each drawing in flight has its own progress bar.
  it("shows a progress bar with the drawer's attempt and phase on each drawing being drawn", async () => {
    const drawing: apiClient.ProjectIllustration = {
      ...DENTIST, state: "drawing", error: undefined,
      progress: { attempt: 2, max_attempts: 3, phase: "writing", reasoning_chars: 0, content_chars: 3200, elapsed_seconds: 42 },
    };
    const queued: apiClient.ProjectIllustration = { ...DENTIST, id: "r4", name: "Candy", state: "drawing", error: undefined };
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [TOOTH, drawing, queued], ready: false });
    renderPanel();
    const bar = await screen.findByTestId("pi-progress-Dentist");
    expect(bar).toHaveTextContent("Lần 2/3 · AI đang viết code · 3,2k ký tự · 42s");
    expect(bar.querySelector('[role="progressbar"]')).not.toBeNull();
    expect(screen.getByTestId("pi-progress-Candy")).toHaveTextContent("Đang bắt đầu…");
  });

  it("'Vẽ N hình còn thiếu' starts the illustrations step on the server", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [TOOTH, MOTO, DENTIST], ready: false });
    const start = vi.spyOn(apiClient, "startAuthoringChain").mockResolvedValue(undefined);
    renderPanel();
    fireEvent.click(await screen.findByTestId("pi-draw-all"));
    await waitFor(() => expect(start).toHaveBeenCalledWith("p1", ["illustrations"]));
  });

  it("deletes this video's AI draft from the library after confirming, and keeps library drawings", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations")
      .mockResolvedValueOnce({ illustrations: [TOOTH, MOTO], ready: false })
      .mockResolvedValue({ illustrations: [TOOTH, { ...MOTO, state: "skipped", illustration: undefined, illustration_id: undefined }], ready: true });
    const del = vi.spyOn(apiClient, "deleteProjectIllustrationDrawing").mockResolvedValue({ ...MOTO, state: "skipped" });
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPanel();
    fireEvent.click(await screen.findByTestId("pi-delete-Motorbike"));
    expect(confirm).toHaveBeenCalled();
    await waitFor(() => expect(del).toHaveBeenCalledWith("p1", "r2"));
    await waitFor(() => expect(screen.getByTestId("pi-summary")).toHaveTextContent("Đủ 2 hình"));
    expect(screen.queryByTestId("pi-delete-Tooth")).not.toBeInTheDocument();
  });

  it("'Nhờ AI sửa' on a tile opens the editor with the warnings in the redraw note", async () => {
    vi.spyOn(apiClient, "getIllustrationStyle").mockResolvedValue({ rules: "- [S3] BO TRÒN: góc bo.", exemplar_ids: [] });
    const warned = { ...MOTO, illustration: ill({ id: "m1", name: "Motorbike", title: "Xe máy", warnings: [{ message: "[S3] <rect> không bo góc (thêm rx)", line: 7 }] }) };
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [warned], ready: false });
    renderPanel();
    fireEvent.click(await screen.findByTestId("illustration-warnings-Motorbike"));
    await waitFor(() => expect(screen.getByTestId("illustration-warnings-panel-Motorbike")).toHaveTextContent("Bo tròn"));
    fireEvent.click(screen.getByTestId("illustration-warnings-fix-Motorbike"));
    const note = await screen.findByTestId("illustration-redraw-note");
    await waitFor(() => expect((note as HTMLTextAreaElement).value).toContain("- S3 · Bo tròn — dòng 7: <rect> không bo góc (thêm rx)"));
  });
});
