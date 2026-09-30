import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CodeChunkShotsField, CodeSegmentsPanel } from "../../src/components/CodeSegmentsPanel";
import { ThemeProvider } from "../../src/context/ThemeContext";
import { AuthoringRunProvider } from "../../src/context/AuthoringRunContext";
import * as apiClient from "../../src/api/client";

const seg = (over: Partial<apiClient.CodeSegment>): apiClient.CodeSegment => ({
  key: "frame", kind: "frame", position: 0, shots: [], status: "pending", source: "",
  error_kind: "", error_message: "", duration_ms: 0, updated_at: "2026-09-30T00:00:00Z", ...over,
});

const FRAME = seg({ status: "done", source: "ai", content: { code: "const LAYOUT = {};" }, duration_ms: 42000 });
const DONE = seg({
  key: "1.1-1.3", kind: "shots", position: 1, shots: ["1.1", "1.2", "1.3"], status: "done", source: "external",
  content: { shots: { "1.1": "function Shot1_1() {}", "1.2": "function Shot1_2() {}", "1.3": "function Shot1_3() {}" } },
});
const FAILED = seg({
  key: "1.4-1.6", kind: "shots", position: 2, shots: ["1.4", "1.5", "1.6"], status: "failed",
  error_kind: "timeout", error_message: "AI không trả lời kịp",
});
const PENDING = seg({ key: "2.1-2.1", kind: "shots", position: 3, shots: ["2.1"] });

function view(segments: apiClient.CodeSegment[], over: Partial<apiClient.CodeSegmentsView> = {}): apiClient.CodeSegmentsView {
  return { chunk_shots: 3, running: false, segments, ...over };
}

function renderPanel(isRemotion = true) {
  return render(
    <ThemeProvider>
      <AuthoringRunProvider>
        <CodeSegmentsPanel projectId="p1" isRemotion={isRemotion} reloadKey={0} />
      </AuthoringRunProvider>
    </ThemeProvider>,
  );
}

describe("CodeSegmentsPanel (CR-050 FR-9)", () => {
  afterEach(() => vi.restoreAllMocks());

  it("lists every segment with its state, source, time and why it failed", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE, FAILED, PENDING]));
    renderPanel();
    expect(await screen.findByTestId("code-segments-summary")).toHaveTextContent("2/4 đoạn xong · 1 lỗi · 1 chưa chạy");
    expect(screen.getByTestId("code-segment-frame")).toHaveTextContent("Khung (LAYOUT)");
    expect(screen.getByTestId("code-segment-frame")).toHaveTextContent("42 giây");
    expect(screen.getByTestId("code-segment-1.1-1.3")).toHaveTextContent("Shot 1.1–1.3");
    expect(screen.getByTestId("code-segment-1.1-1.3")).toHaveTextContent("AI ngoài");
    expect(screen.getByTestId("code-segment-1.4-1.6-status")).toHaveTextContent("Lỗi");
    expect(screen.getByTestId("code-segment-1.4-1.6-error")).toHaveTextContent("timeout: AI không trả lời kịp");
    expect(screen.getByTestId("code-segment-2.1-2.1")).toHaveTextContent("Shot 2.1");
    expect(screen.getByTestId("code-segments-run-missing")).toHaveTextContent("Chạy các đoạn còn thiếu (2)");
  });

  it("names the Manim frame cast, and a storyboard-given LAYOUT has nothing to run", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([seg({ status: "done", source: "storyboard" }), PENDING]));
    renderPanel();
    expect(await screen.findByTestId("code-segment-frame")).toHaveTextContent("Từ storyboard");
    expect(screen.queryByTestId("code-segment-frame-run")).toBeNull();
  });

  it("re-runs one segment through the chain and locks the panel while it runs", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE, FAILED]));
    const start = vi.spyOn(apiClient, "startAuthoringChain").mockResolvedValue();
    renderPanel();
    fireEvent.click(await screen.findByTestId("code-segment-1.4-1.6-run"));
    await waitFor(() => expect(start).toHaveBeenCalledWith("p1", ["code"], { segment: "1.4-1.6" }));
    // the chain it started locks the other write buttons
    await waitFor(() => expect(screen.getByTestId("code-segments-run-missing")).toBeDisabled());
  });

  it("runs the missing segments through the chain, and asks before a fresh run", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE, FAILED]));
    const start = vi.spyOn(apiClient, "startAuthoringChain").mockResolvedValue();
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    renderPanel();
    fireEvent.click(await screen.findByTestId("code-segments-regenerate"));
    expect(confirm).toHaveBeenCalled();
    expect(start).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("code-segments-run-missing"));
    await waitFor(() => expect(start).toHaveBeenCalledWith("p1", ["code"], undefined));
  });

  it("starts a fresh run once confirmed", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE]));
    const start = vi.spyOn(apiClient, "startAuthoringChain").mockResolvedValue();
    vi.spyOn(window, "confirm").mockReturnValue(true);
    renderPanel();
    // nothing missing: the same run only merges and checks (FR-8)
    expect(await screen.findByTestId("code-segments-run-missing")).toHaveTextContent("Ghép và kiểm lại code");
    fireEvent.click(screen.getByTestId("code-segments-regenerate"));
    await waitFor(() => expect(start).toHaveBeenCalledWith("p1", ["code"], { fresh: true }));
  });

  it("copies a segment's prompt for an outside AI", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, FAILED]));
    vi.spyOn(apiClient, "getCodeSegmentPrompt").mockResolvedValue({ system: "SYS", user: "USER" });
    const write = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText: write } });
    renderPanel();
    fireEvent.click(await screen.findByTestId("code-segment-1.4-1.6-copy"));
    await waitFor(() => expect(write).toHaveBeenCalledWith("SYS\n\n---\n\nUSER"));
    expect(await screen.findByRole("status")).toHaveTextContent("Đã sao chép prompt của Shot 1.4–1.6");
  });

  it("pastes an outside AI's reply, shows why the server refused it, and saves a fixed one", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, FAILED]));
    const put = vi.spyOn(apiClient, "putCodeSegment")
      .mockRejectedValueOnce(new apiClient.ApiError("kết quả dán vào không dùng được: missing shot function(s): 1.6"))
      .mockResolvedValue({ ...FAILED, status: "done", source: "external" });
    renderPanel();
    fireEvent.click(await screen.findByTestId("code-segment-1.4-1.6-paste"));
    fireEvent.change(await screen.findByTestId("code-segment-1.4-1.6-editor-text"), { target: { value: "REPLY" } });
    fireEvent.click(screen.getByTestId("code-segment-1.4-1.6-editor-save"));
    expect(await screen.findByText(/missing shot function\(s\): 1.6/)).toBeInTheDocument();
    expect(put).toHaveBeenCalledWith("p1", "1.4-1.6", "REPLY", "external");
    fireEvent.click(screen.getByTestId("code-segment-1.4-1.6-editor-save"));
    await waitFor(() => expect(put).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Đã lưu Shot 1.4–1.6.")).toBeInTheDocument();
  });

  it("edits a finished segment by hand, starting from its code", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE]));
    const put = vi.spyOn(apiClient, "putCodeSegment").mockResolvedValue({ ...DONE, source: "manual" });
    renderPanel();
    fireEvent.click(await screen.findByTestId("code-segment-1.1-1.3-edit"));
    const text = await screen.findByTestId("code-segment-1.1-1.3-editor-text");
    expect((text as HTMLTextAreaElement).value).toBe("function Shot1_1() {}\n\nfunction Shot1_2() {}\n\nfunction Shot1_3() {}");
    fireEvent.click(screen.getByTestId("code-segment-1.1-1.3-editor-save"));
    await waitFor(() => expect(put).toHaveBeenCalledWith("p1", "1.1-1.3", (text as HTMLTextAreaElement).value, "manual"));
  });

  it("locks every write while the code step runs on the server", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, { ...FAILED, status: "running" }], { running: true }));
    renderPanel();
    expect(await screen.findByTestId("code-segment-1.4-1.6-status")).toHaveTextContent("Đang chạy");
    expect(screen.getByTestId("code-segment-1.4-1.6-run")).toBeDisabled();
    expect(screen.getByTestId("code-segment-1.4-1.6-paste")).toBeDisabled();
    expect(screen.getByTestId("code-segments-run-missing")).toBeDisabled();
    expect(screen.getByTestId("code-segments-regenerate")).toBeDisabled();
  });
});

describe("CodeChunkShotsField (CR-050 FR-7)", () => {
  afterEach(() => vi.restoreAllMocks());

  function renderField(onSaved = vi.fn()) {
    render(
      <ThemeProvider>
        <AuthoringRunProvider>
          <CodeChunkShotsField projectId="p1" onSaved={onSaved} />
        </AuthoringRunProvider>
      </ThemeProvider>,
    );
    return onSaved;
  }

  it("saves a new size, warning first when finished segments will be re-cut", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME, DONE]));
    const put = vi.spyOn(apiClient, "putCodeChunkShots").mockResolvedValue();
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValue(true);
    const onSaved = renderField();
    const input = await screen.findByTestId("code-chunk-shots-input");
    expect(input).toHaveValue(3);
    expect(screen.getByTestId("code-chunk-shots-save")).toBeDisabled();
    fireEvent.change(input, { target: { value: "2" } });
    fireEvent.click(screen.getByTestId("code-chunk-shots-save"));
    expect(confirm).toHaveBeenCalledTimes(1);
    expect(put).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("code-chunk-shots-save"));
    await waitFor(() => expect(put).toHaveBeenCalledWith("p1", 2));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
  });

  it("refuses a size out of range", async () => {
    vi.spyOn(apiClient, "getCodeSegments").mockResolvedValue(view([FRAME]));
    renderField();
    fireEvent.change(await screen.findByTestId("code-chunk-shots-input"), { target: { value: "11" } });
    expect(screen.getByTestId("code-chunk-shots-save")).toBeDisabled();
    expect(screen.getByTestId("code-chunk-shots")).toHaveTextContent("Nhập từ 1 đến 10.");
  });
});
