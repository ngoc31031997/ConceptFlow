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
});
