import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QCReportPanel } from "../../src/components/QCReportPanel";
import type { QCReport } from "../../src/api/client";

const REPORT: QCReport = {
  project_id: "proj-1",
  status: "has_findings",
  reason: null,
  findings: [
    {
      rule: "text_too_small",
      severity: "warning",
      message: "cỡ chữ 18px dưới ngưỡng đọc được",
      timestamp_seconds: 95,
    },
    {
      rule: "frame_overflow",
      severity: "blocking",
      message: "chữ vượt ra ngoài safe margin",
      timestamp_seconds: 12.5,
    },
  ],
  created_at: "2026-09-11T00:00:00Z",
};

function mockReport(report: unknown) {
  global.fetch = vi.fn().mockResolvedValue({ ok: true, json: async () => report }) as never;
}

beforeEach(() => {
  mockReport(REPORT);
});

describe("QCReportPanel", () => {
  it("nhóm phát hiện theo mức độ, lỗi nghiêm trọng lên trước", async () => {
    render(<QCReportPanel projectId="proj-1" />);

    const blocking = await screen.findByTestId("qc-blocking-group");
    const warnings = screen.getByTestId("qc-warning-group");
    expect(blocking).toHaveTextContent("Tràn khung");
    expect(warnings).toHaveTextContent("Chữ quá nhỏ");
    // Lỗi chặn phải đứng trước cảnh báo trong DOM, không chỉ tồn tại đâu đó.
    expect(blocking.compareDocumentPosition(warnings)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  it("bấm mốc thời gian thì tua tới đúng giây của phát hiện", async () => {
    // FR61.2: mốc bấm được là lý do báo cáo này dùng được — không có nó thì
    // Creator phải tự dò trong video mười phút.
    const onSeek = vi.fn();
    render(<QCReportPanel projectId="proj-1" onSeek={onSeek} />);

    fireEvent.click(await screen.findByTestId("qc-seek-frame_overflow"));

    expect(onSeek).toHaveBeenCalledWith(12.5);
  });

  it("hiện mốc thời gian dạng phút:giây", async () => {
    render(<QCReportPanel projectId="proj-1" onSeek={vi.fn()} />);
    expect(await screen.findByTestId("qc-seek-text_too_small")).toHaveTextContent("1:35");
  });

  it("nói rõ là không chấm được, và không làm nó trông như lỗi chặn", async () => {
    // FR61.4: một cổng hỏng không được biến thành cổng khoá — kể cả về mặt
    // giao diện, nên trạng thái này không có nhóm "lỗi nghiêm trọng" nào.
    mockReport({
      project_id: "proj-1",
      status: "not_scored",
      reason: "lệnh qc_video không mang layout_marks",
      findings: [],
      created_at: "2026-09-11T00:00:00Z",
    });
    render(<QCReportPanel projectId="proj-1" />);

    expect(await screen.findByTestId("qc-not-scored")).toHaveTextContent("layout_marks");
    expect(screen.queryByTestId("qc-blocking-group")).not.toBeInTheDocument();
  });

  it("một dòng gọn khi đã chấm và không có gì", async () => {
    mockReport({ ...REPORT, status: "passed", findings: [] });
    render(<QCReportPanel projectId="proj-1" />);

    expect(await screen.findByTestId("qc-passed")).toBeInTheDocument();
    expect(screen.queryByTestId("qc-warning-group")).not.toBeInTheDocument();
  });

  it("đánh dấu báo cáo đã từng bị bỏ qua", async () => {
    mockReport({ ...REPORT, overridden_at: "2026-09-11T01:00:00Z" });
    render(<QCReportPanel projectId="proj-1" />);

    expect(await screen.findByTestId("qc-overridden")).toBeInTheDocument();
  });

  it("không hiện gì khi không tải được báo cáo, thay vì chặn đường đăng", async () => {
    global.fetch = vi.fn().mockRejectedValue(new Error("network down")) as never;
    render(<QCReportPanel projectId="proj-1" />);

    await waitFor(() => {
      expect(screen.queryByTestId("qc-report")).not.toBeInTheDocument();
    });
  });
});
