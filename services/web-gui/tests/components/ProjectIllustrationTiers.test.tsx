import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ProjectIllustrationsPanel } from "../../src/components/ProjectIllustrationsPanel";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";

const ill = (over: Partial<apiClient.Illustration>): apiClient.Illustration => ({
  id: "i", name: "X", title: "X", folder_id: "dong-vat", tags: [], description: "", usage: "", builtin: false,
  exemplar: false, warnings: [], status: "approved", version: 1, has_preview: true, ...over,
});

const BEE: apiClient.ProjectIllustration = {
  id: "r1", position: 1, name: "Bee", description: "Con ong", folder_id: "dong-vat", shots: ["1.1"], state: "drawn",
  tier: "support", tier_reason: "ong chỉ có vai nhỏ",
  illustration_id: "bee", illustration: ill({ id: "bee", name: "Bee", title: "Con ong", detail_level: "medium" }),
};
const PLANNED: apiClient.ProjectIllustration = {
  id: "r2", position: 2, name: "Hive", description: "Tổ ong", folder_id: "do-vat", shots: [], state: "planned",
  tier: "hero", tier_reason: "phim nói về tổ ong",
};

function renderPanel() {
  return render(
    <ThemeProvider>
      <ProjectIllustrationsPanel projectId="p1" />
    </ThemeProvider>,
  );
}

describe("ProjectIllustrationsPanel — hạng của hình", () => {
  beforeEach(() => {
    vi.spyOn(apiClient, "listIllustrationFolders").mockResolvedValue([]);
  });
  afterEach(() => vi.restoreAllMocks());

  it("hiện hạng AI đề xuất cùng lý do của từng hình, và giải thích ba hạng", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [BEE, PLANNED], ready: false });
    renderPanel();
    expect(await screen.findByTestId("pi-tier-guide")).toHaveTextContent("Chính");
    expect(screen.getByTestId("pi-tier-select-Bee")).toHaveValue("support");
    expect(screen.getByTestId("pi-tier-Bee")).toHaveTextContent("AI: ong chỉ có vai nhỏ");
    expect(screen.getByTestId("pi-tier-select-Hive")).toHaveValue("hero");
    expect(screen.getByTestId("pi-tier-Hive")).toHaveTextContent("AI: phim nói về tổ ong");
  });

  it("đổi hạng hình chưa vẽ: lưu ngay, không hỏi", async () => {
    const list = vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [BEE, PLANNED], ready: false });
    const setTier = vi.spyOn(apiClient, "setProjectIllustrationTier").mockResolvedValue({ ...PLANNED, tier: "background" });
    renderPanel();
    fireEvent.change(await screen.findByTestId("pi-tier-select-Hive"), { target: { value: "background" } });
    await waitFor(() => expect(setTier).toHaveBeenCalledWith("p1", "r2", "background"));
    expect(screen.queryByTestId("confirm-modal")).not.toBeInTheDocument();
    await waitFor(() => expect(list.mock.calls.length).toBeGreaterThanOrEqual(2));
  });

  it("hạ hạng hình đã vẽ đủ chi tiết: lưu ngay, không hỏi", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [BEE], ready: true });
    const setTier = vi.spyOn(apiClient, "setProjectIllustrationTier").mockResolvedValue({ ...BEE, tier: "background" });
    renderPanel();
    fireEvent.change(await screen.findByTestId("pi-tier-select-Bee"), { target: { value: "background" } });
    await waitFor(() => expect(setTier).toHaveBeenCalledWith("p1", "r1", "background"));
    expect(screen.queryByTestId("confirm-modal")).not.toBeInTheDocument();
  });

  it("nâng hạng hình đã vẽ không đủ chi tiết: hỏi trước, nói rõ hậu quả, huỷ thì không đổi", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [BEE], ready: true });
    const setTier = vi.spyOn(apiClient, "setProjectIllustrationTier").mockResolvedValue({ ...BEE, tier: "hero", state: "planned" });
    renderPanel();
    fireEvent.change(await screen.findByTestId("pi-tier-select-Bee"), { target: { value: "hero" } });
    const modal = await screen.findByTestId("confirm-modal");
    expect(modal).toHaveTextContent("Đổi hạng của Bee?");
    expect(modal).toHaveTextContent("Chính");
    expect(modal).toHaveTextContent("Vừa");
    expect(modal).toHaveTextContent("AI vẽ một bản mới");
    expect(modal).toHaveTextContent("Code của shot dùng hình này phải chạy lại");
    expect(setTier).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("confirm-modal-cancel"));
    expect(setTier).not.toHaveBeenCalled();
    expect(screen.getByTestId("pi-tier-select-Bee")).toHaveValue("support");
  });

  it("xác nhận thì mới đổi hạng", async () => {
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [BEE], ready: true });
    const setTier = vi.spyOn(apiClient, "setProjectIllustrationTier").mockResolvedValue({ ...BEE, tier: "hero", state: "planned" });
    renderPanel();
    fireEvent.change(await screen.findByTestId("pi-tier-select-Bee"), { target: { value: "hero" } });
    fireEvent.click(await screen.findByTestId("confirm-modal-confirm"));
    await waitFor(() => expect(setTier).toHaveBeenCalledWith("p1", "r1", "hero"));
  });

  it("nền của cảnh không có hạng", async () => {
    const backdrop: apiClient.ProjectIllustration = {
      id: "r9", position: 9, name: "MeadowBackdrop", description: "Đồng cỏ", folder_id: "boi-canh", shots: ["1.1"],
      state: "reused", kind: "backdrop", illustration_id: "b", illustration: ill({ id: "b", name: "MeadowBackdrop", kind: "backdrop" }),
    };
    vi.spyOn(apiClient, "listProjectIllustrations").mockResolvedValue({ illustrations: [backdrop], ready: true });
    renderPanel();
    await screen.findByTestId("pi-backdrops-grid");
    expect(screen.queryByTestId("pi-tier-MeadowBackdrop")).not.toBeInTheDocument();
  });
});

describe("detailEnough", () => {
  it.each([
    ["rich", "hero", true],
    ["rich", "background", true],
    ["medium", "hero", false],
    ["medium", "support", true],
    ["medium", "background", true],
    ["silhouette", "support", false],
    ["silhouette", "background", true],
    [undefined, "background", false],
  ] as const)("%s cho hạng %s: %s", (detail, tier, expected) => {
    expect(apiClient.detailEnough(detail, tier)).toBe(expected);
  });
});
