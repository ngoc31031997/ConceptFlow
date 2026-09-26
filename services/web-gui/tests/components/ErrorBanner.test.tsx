import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ErrorBanner, shortError } from "../../src/components/ErrorBanner";
import * as apiClient from "../../src/api/client";

describe("ErrorBanner", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows a short message and triggers retry", () => {
    const onRetry = vi.fn();
    render(<ErrorBanner errorMessage="Lỗi render scene 3" onRetry={onRetry} isRetrying={false} />);

    expect(screen.getByTestId("error-banner-message")).toHaveTextContent("Lỗi render scene 3");
    fireEvent.click(screen.getByTestId("error-banner-retry-button"));
    expect(onRetry).toHaveBeenCalled();
  });

  it("disables retry button while retrying", () => {
    render(<ErrorBanner errorMessage="err" onRetry={vi.fn()} isRetrying={true} />);
    expect(screen.getByTestId("error-banner-retry-button")).toBeDisabled();
  });

  it("offers retry only: there is no back-to-edit button", () => {
    render(<ErrorBanner errorMessage="err" onRetry={vi.fn()} isRetrying={false} />);
    expect(screen.queryByTestId("error-banner-back-button")).not.toBeInTheDocument();
  });

  it("cuts a long message to one line and keeps the full text behind the ? button", () => {
    const long = "dòng đầu rất dài ".repeat(30) + "\nchi tiết dòng hai";
    render(<ErrorBanner errorMessage={long} onRetry={vi.fn()} isRetrying={false} />);

    expect(screen.getByTestId("error-banner-message").textContent).toContain("…");
    expect(screen.queryByTestId("error-banner-detail")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("error-banner-detail-toggle"));
    expect(screen.getByTestId("error-banner-detail")).toHaveTextContent("chi tiết dòng hai");
  });

  it("adds the technical trace from the project's error log when the ? is opened", async () => {
    vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([
      { at: "t0", source: "saga", step: "render_scenes", message: "cũ", detail: "traceback cũ" },
      { at: "t1", source: "saga", step: "assemble_video", message: "khác", detail: "không liên quan" },
      { at: "t2", source: "saga", step: "render_scenes", message: "mới", detail: "Manim: exit code 1" },
    ]);
    render(
      <ErrorBanner errorMessage="Render lỗi" onRetry={vi.fn()} isRetrying={false} projectId="p1" step="render_scenes" />,
    );

    fireEvent.click(screen.getByTestId("error-banner-detail-toggle"));
    await waitFor(() => expect(screen.getByTestId("error-banner-detail")).toHaveTextContent("Manim: exit code 1"));
    expect(screen.getByTestId("error-banner-detail")).not.toHaveTextContent("không liên quan");
  });
});

describe("ErrorBanner — what the Creator needs to decide on a retry", () => {
  afterEach(() => vi.restoreAllMocks());

  it("names the step on the retry button and lists step, time, duration and kind behind the ?", async () => {
    vi.spyOn(apiClient, "listProjectErrors").mockResolvedValue([
      { at: "2026-09-26T11:02:14Z", source: "saga", step: "assemble_video", kind: "input", elapsed_seconds: 41, message: "m", detail: "ffmpeg exited 1" },
    ]);
    render(<ErrorBanner errorMessage="Merge lỗi" onRetry={vi.fn()} isRetrying={false} projectId="p1" step="assemble_video" />);

    expect(screen.getByTestId("error-banner-retry-button")).toHaveTextContent("Thử lại bước Ghép video hoàn chỉnh");
    fireEvent.click(screen.getByTestId("error-banner-detail-toggle"));
    await waitFor(() => expect(screen.getByTestId("error-banner-meta")).toHaveTextContent("đã chạy: 41s"));
    expect(screen.getByTestId("error-banner-meta")).toHaveTextContent("loại: input");
    expect(screen.getByTestId("error-banner-meta")).toHaveTextContent("bước: Ghép video hoàn chỉnh");
  });

  it("copies the full detail", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(<ErrorBanner errorMessage={"dòng 1\ndòng 2"} onRetry={vi.fn()} isRetrying={false} />);

    fireEvent.click(screen.getByTestId("error-banner-detail-toggle"));
    fireEvent.click(screen.getByTestId("error-banner-copy"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("dòng 1\ndòng 2"));
  });
});

describe("shortError", () => {
  it("keeps only the first line", () => {
    expect(shortError("một\nhai")).toBe("một");
  });
});
