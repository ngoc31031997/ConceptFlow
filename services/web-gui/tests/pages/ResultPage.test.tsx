import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResultPage } from "../../src/pages/ResultPage";

function renderResultPage() {
  return render(
    <MemoryRouter initialEntries={["/projects/p1/result"]}>
      <Routes>
        <Route path="/projects/:id/result" element={<ResultPage />} />
        <Route path="/projects/:id/render" element={<div data-testid="render-page-stub" />} />
        <Route path="/videos" element={<div data-testid="video-list-page-stub" />} />
      </Routes>
    </MemoryRouter>,
  );
}

function mockFetch(overrides: { onDelete?: () => { ok: boolean; status: number } } = {}) {
  return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
    if (init?.method === "DELETE") {
      return Promise.resolve(overrides.onDelete ? overrides.onDelete() : { ok: true, status: 204 });
    }
    if (url.includes("/v1/auth/youtube/status")) {
      return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false, accounts: [] }) });
    }
    // CR-012: the channel list and the OAuth app catalogue both return arrays.
    if (url.includes("/v1/auth/youtube/accounts") || url.includes("/v1/auth/youtube/apps")) {
      return Promise.resolve({ ok: true, status: 200, json: async () => [] });
    }
    return Promise.resolve({
      ok: true,
      status: 200,
      json: async () => ({ project_id: "p1", status: "ready_to_publish", scenes: [], video_path: "/shared/p1/video/final.mp4" }),
    });
  }) as unknown as typeof fetch;
}

describe("ResultPage delete button", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("deletes the project and navigates to the video list on confirm", async () => {
    global.fetch = mockFetch();
    vi.spyOn(window, "confirm").mockReturnValue(true);

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("result-delete-button"));

    await waitFor(() => expect(screen.getByTestId("video-list-page-stub")).toBeInTheDocument());
    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });

  it("does not delete or navigate when the user cancels", async () => {
    global.fetch = mockFetch();
    vi.spyOn(window, "confirm").mockReturnValue(false);

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-delete-button")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("result-delete-button"));

    expect(screen.getByTestId("result-page")).toBeInTheDocument();
    expect(global.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/v1/projects/p1"),
      expect.objectContaining({ method: "DELETE" }),
    );
  });
});

/*
  The publish button used to re-enable as soon as POST /v1/sagas/publish
  returned, even though the upload had only just been queued — so a second
  click hit a 409 and the page showed nothing in between.
*/
describe("ResultPage publish state", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  function mockProjectFetch(project: Record<string, unknown>, onPublish?: () => unknown) {
    return vi.fn().mockImplementation((url: string) => {
      if (url.includes("/v1/auth/youtube/status")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false, accounts: [] }) });
      }
      if (url.includes("/v1/auth/youtube/accounts") || url.includes("/v1/auth/youtube/apps")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => [] });
      }
      if (url.includes("/v1/sagas/publish") || url.includes("/retry")) {
        onPublish?.();
        return Promise.resolve({ ok: true, status: 201, json: async () => ({ saga_id: "s1", status: "publishing" }) });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ project_id: "p1", scenes: [], video_path: "/shared/p1/video/final.mp4", ...project }),
      });
    }) as unknown as typeof fetch;
  }

  it("shows an in-progress card and hides the form while the project is publishing", async () => {
    global.fetch = mockProjectFetch({ status: "publishing" });

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-publishing-status")).toBeInTheDocument());
    expect(screen.queryByTestId("publish-form-submit-button")).not.toBeInTheDocument();
  });

  it("sends only one publish request when the button is clicked twice in a row", async () => {
    let publishCalls = 0;
    global.fetch = mockProjectFetch({ status: "ready_to_publish" }, () => {
      publishCalls += 1;
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("publish-form-title-input")).toBeInTheDocument());
    fireEvent.change(screen.getByTestId("publish-form-title-input"), { target: { value: "Video" } });
    const button = screen.getByTestId("publish-form-submit-button");
    fireEvent.click(button);
    fireEvent.click(button);

    await waitFor(() => expect(publishCalls).toBe(1));
    expect(publishCalls).toBe(1);
  });

  it("offers a retry after a failed publish instead of the publish form", async () => {
    let retried = false;
    global.fetch = mockProjectFetch({ status: "failed_at_publish_video", error_message: "quota exceeded" }, () => {
      retried = true;
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByTestId("result-publish-failed")).toBeInTheDocument());
    expect(screen.getByText(/quota exceeded/)).toBeInTheDocument();
    expect(screen.queryByTestId("publish-form-submit-button")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("result-retry-publish-button"));
    await waitFor(() => expect(retried).toBe(true));
  });

  it("shows the success card when the project reports published", async () => {
    global.fetch = mockProjectFetch({ status: "published", youtube_video_url: "https://youtu.be/abc" });

    renderResultPage();

    await waitFor(() => expect(screen.getByText("https://youtu.be/abc")).toBeInTheDocument());
  });

  it("warns when the caption track was skipped for lacking scope", async () => {
    global.fetch = mockProjectFetch({
      status: "published",
      youtube_video_url: "https://youtu.be/abc",
      caption_status: "skipped_no_scope",
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("cần được nối lại"));
  });

  it("warns when the caption upload failed", async () => {
    global.fetch = mockProjectFetch({
      status: "published",
      youtube_video_url: "https://youtu.be/abc",
      caption_status: "failed",
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("tải phụ đề lên YouTube thất bại"));
  });

  it("shows no caption warning when the caption uploaded successfully", async () => {
    global.fetch = mockProjectFetch({
      status: "published",
      youtube_video_url: "https://youtu.be/abc",
      caption_status: "uploaded",
    });

    renderResultPage();

    await waitFor(() => expect(screen.getByText("https://youtu.be/abc")).toBeInTheDocument());
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});

describe("ResultPage QC gate (CR-021 FR61.3)", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  /**
   * Như mockProjectFetch nhưng chặn lần publish ĐẦU bằng 409 code=qc_blocked,
   * và cho lần thứ hai đi qua — đúng hình dạng Orchestrator trả về khi
   * QC_ENFORCE=true và báo cáo có lỗi chặn.
   */
  function mockQCBlockedFetch(publishBodies: Array<Record<string, unknown>>) {
    return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/v1/auth/youtube/status")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false, accounts: [] }) });
      }
      if (url.includes("/v1/auth/youtube/accounts") || url.includes("/v1/auth/youtube/apps")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => [] });
      }
      if (url.includes("/v1/sagas/publish")) {
        publishBodies.push(JSON.parse(String(init?.body)));
        if (publishBodies.length === 1) {
          return Promise.resolve({
            ok: false,
            status: 409,
            json: async () => ({
              error: "quality check found blocking issues; re-submit with acknowledge_qc to publish anyway",
              code: "qc_blocked",
            }),
          });
        }
        return Promise.resolve({ ok: true, status: 201, json: async () => ({ saga_id: "s1", status: "publishing" }) });
      }
      if (url.includes("/qc-report")) {
        return Promise.resolve({
          ok: true,
          status: 200,
          json: async () => ({
            project_id: "p1",
            status: "has_findings",
            reason: null,
            findings: [
              { rule: "frame_overflow", severity: "blocking", message: "chữ vượt safe margin", timestamp_seconds: 12.5 },
            ],
            created_at: "2026-09-11T00:00:00Z",
          }),
        });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          project_id: "p1",
          status: "ready_to_publish",
          scenes: [],
          video_path: "/shared/p1/video/final.mp4",
        }),
      });
    }) as unknown as typeof fetch;
  }

  async function clickPublish() {
    await waitFor(() => expect(screen.getByTestId("publish-form-title-input")).toBeInTheDocument());
    fireEvent.change(screen.getByTestId("publish-form-title-input"), { target: { value: "Video" } });
    fireEvent.click(screen.getByTestId("publish-form-submit-button"));
  }

  it("never sends acknowledge_qc on the first attempt", async () => {
    // FR61.3: bỏ qua phải là hành động có ý thức. Gửi cờ ngay lần đầu là biến
    // cổng chặn thành thứ trang tự mở hộ.
    const bodies: Array<Record<string, unknown>> = [];
    global.fetch = mockQCBlockedFetch(bodies);
    vi.spyOn(window, "confirm").mockReturnValue(false);

    renderResultPage();
    await clickPublish();

    await waitFor(() => expect(bodies.length).toBe(1));
    expect(bodies[0].acknowledge_qc).toBeUndefined();
  });

  it("asks for confirmation, then re-sends with acknowledge_qc", async () => {
    const bodies: Array<Record<string, unknown>> = [];
    global.fetch = mockQCBlockedFetch(bodies);
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);

    renderResultPage();
    await clickPublish();

    await waitFor(() => expect(bodies.length).toBe(2));
    expect(confirm).toHaveBeenCalled();
    expect(bodies[1].acknowledge_qc).toBe(true);
  });

  it("does not publish when the Creator declines the override", async () => {
    const bodies: Array<Record<string, unknown>> = [];
    global.fetch = mockQCBlockedFetch(bodies);
    vi.spyOn(window, "confirm").mockReturnValue(false);

    renderResultPage();
    await clickPublish();

    await waitFor(() => expect(bodies.length).toBe(1));
    // Không có lần gửi thứ hai, và cũng không kẹt ở trạng thái đang gửi.
    expect(screen.queryByTestId("result-publishing-status")).not.toBeInTheDocument();
  });

  it("shows the blocking finding before the publish button", async () => {
    global.fetch = mockQCBlockedFetch([]);

    renderResultPage();

    const report = await screen.findByTestId("qc-report");
    const form = screen.getByTestId("publish-form-submit-button");
    expect(report).toHaveTextContent("Tràn khung");
    expect(report.compareDocumentPosition(form)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });
});

describe("render lại ở chất lượng khác", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  function mockRerenderFetch(bodies: Array<Record<string, unknown>>) {
    return vi.fn().mockImplementation((url: string, init?: RequestInit) => {
      if (init?.method === "POST" && url.includes("/v1/sagas/render")) {
        bodies.push(JSON.parse(init.body as string));
        return Promise.resolve({ ok: true, status: 201, json: async () => ({ saga_id: "s2", status: "draft" }) });
      }
      if (url.includes("/v1/auth/youtube") || url.includes("/qc-report")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false, accounts: [] }) });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({
          project_id: "p1",
          status: "ready_to_publish",
          scenes: [],
          video_path: "/shared/p1/video/final.mp4",
          script_content: "from conceptflow import *\n...",
          voice_language: "vi",
          tts_enabled: true,
          subtitle_mode: "track",
        }),
      });
    }) as unknown as typeof fetch;
  }

  it("hiện nút mở, ẩn form chọn chất lượng cho tới khi bấm", async () => {
    global.fetch = mockRerenderFetch([]);
    renderResultPage();

    await screen.findByTestId("rerender-toggle");
    expect(screen.queryByTestId("render-quality-picker")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("rerender-toggle"));

    expect(screen.getByTestId("render-quality-picker")).toBeInTheDocument();
  });

  it("gửi lại script gốc kèm chất lượng mới, rồi điều hướng sang trang render", async () => {
    const bodies: Array<Record<string, unknown>> = [];
    global.fetch = mockRerenderFetch(bodies);
    renderResultPage();

    fireEvent.click(await screen.findByTestId("rerender-toggle"));
    fireEvent.click(screen.getByTestId("render-quality-4k60"));
    fireEvent.click(screen.getByTestId("rerender-submit"));

    await waitFor(() => expect(bodies.length).toBe(1));
    expect(bodies[0]).toMatchObject({
      project_id: "p1",
      script_content: "from conceptflow import *\n...",
      render_quality: "4k60",
    });
  });

  it("báo lỗi rõ ràng thay vì gọi API khi project thiếu script_content", async () => {
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (url.includes("/v1/auth/youtube") || url.includes("/qc-report")) {
        return Promise.resolve({ ok: true, status: 200, json: async () => ({ connected: false, accounts: [] }) });
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        json: async () => ({ project_id: "p1", status: "ready_to_publish", scenes: [], video_path: "/shared/p1/video/final.mp4" }),
      });
    }) as unknown as typeof fetch;

    renderResultPage();
    fireEvent.click(await screen.findByTestId("rerender-toggle"));
    fireEvent.click(screen.getByTestId("rerender-submit"));

    await waitFor(() =>
      expect(screen.getByText(/Thiếu script gốc/)).toBeInTheDocument(),
    );
    expect(global.fetch).not.toHaveBeenCalledWith(
      expect.stringContaining("/v1/sagas/render"),
      expect.anything(),
    );
  });
});
