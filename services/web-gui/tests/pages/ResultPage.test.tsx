import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { ResultPage } from "../../src/pages/ResultPage";

function renderResultPage() {
  return render(
    <MemoryRouter initialEntries={["/projects/p1/result"]}>
      <Routes>
        <Route path="/projects/:id/result" element={<ResultPage />} />
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
