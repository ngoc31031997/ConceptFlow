import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { StyleWarnings } from "../../src/components/StyleWarnings";
import { resetStyleRuleNamesCache } from "../../src/hooks/useStyleRuleNames";
import * as apiClient from "../../src/api/client";

const WARNINGS: apiClient.CodeDiagnostic[] = [
  { message: '[S3] nét thiếu strokeLinecap="round"', line: 14 },
  { message: "[S19] hình có mặt nhưng không có chuyển động tự thân nào", line: null },
];

describe("StyleWarnings", () => {
  beforeEach(() => {
    resetStyleRuleNamesCache();
    vi.spyOn(apiClient, "getIllustrationStyle").mockResolvedValue({
      rules: "- [S3] BO TRÒN: góc bo.\n- [S19] Vật sống hoặc có năng lượng CÓ chuyển động tự thân theo frame: thở, nhún.",
      exemplar_ids: [],
      max_exemplars: 5,
    });
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows the count; hovering shows the list at once, with rule names", async () => {
    render(<StyleWarnings warnings={WARNINGS} name="Cat" />);
    expect(screen.getByTestId("illustration-warnings-Cat")).toHaveTextContent("⚠ 2 cảnh báo style");
    const tip = screen.getByTestId("illustration-warnings-tip-Cat");
    expect(tip).toHaveAttribute("role", "tooltip");
    await waitFor(() => expect(tip).toHaveTextContent('S3 · Bo tròn — dòng 14: nét thiếu strokeLinecap="round"'));
    expect(tip).toHaveTextContent("S19 · Vật sống hoặc có năng lượng có chuyển động tự thân theo frame");
  });

  it("a click opens the list under the tile; copy puts every warning on the clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<StyleWarnings warnings={WARNINGS} name="Cat" />);
    expect(screen.queryByTestId("illustration-warnings-panel-Cat")).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId("illustration-warnings-Cat"));
    expect(screen.getByTestId("illustration-warnings-Cat")).toHaveAttribute("aria-expanded", "true");
    const panel = screen.getByTestId("illustration-warnings-panel-Cat");
    await waitFor(() => expect(panel).toHaveTextContent("S3 · Bo tròn"));
    fireEvent.click(screen.getByTestId("illustration-warnings-copy-Cat"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(
        'S3 · Bo tròn — dòng 14: nét thiếu strokeLinecap="round"\n' +
          "S19 · Vật sống hoặc có năng lượng có chuyển động tự thân theo frame — hình có mặt nhưng không có chuyển động tự thân nào",
      ),
    );
    expect(await screen.findByText("Đã sao chép")).toBeInTheDocument();
  });

  it("'Nhờ AI sửa' hands over a redraw note built from the warnings", async () => {
    const onFix = vi.fn();
    render(<StyleWarnings warnings={WARNINGS} name="Cat" onFix={onFix} />);
    fireEvent.click(screen.getByTestId("illustration-warnings-Cat"));
    await waitFor(() => expect(screen.getByTestId("illustration-warnings-panel-Cat")).toHaveTextContent("Bo tròn"));
    fireEvent.click(screen.getByTestId("illustration-warnings-fix-Cat"));
    expect(onFix).toHaveBeenCalledTimes(1);
    expect(onFix.mock.calls[0][0]).toContain('- S3 · Bo tròn — dòng 14: nét thiếu strokeLinecap="round"');
  });

  it("no fix button for a drawing AI cannot redraw", () => {
    render(<StyleWarnings warnings={WARNINGS} name="Cat" />);
    fireEvent.click(screen.getByTestId("illustration-warnings-Cat"));
    expect(screen.queryByTestId("illustration-warnings-fix-Cat")).not.toBeInTheDocument();
  });
});
