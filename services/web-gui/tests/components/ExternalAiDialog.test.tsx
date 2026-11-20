import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ApiError, ExternalPromptReady, ExternalRequestDescribed, type ExternalAi } from "../../src/api/client";
import { ExternalAiDialog } from "../../src/components/ExternalAiDialog";
import { installFakeAiBridge } from "../helpers/fakeAiBridge";

function renderDialog(
  run: (external: ExternalAi) => Promise<unknown>,
  onDone = vi.fn(),
  onClose = vi.fn(),
) {
  render(
    <ExternalAiDialog
      open
      onClose={onClose}
      title="Câu chuyện · AI ngoài"
      run={run}
      onDone={onDone}
      testId="ext"
    />,
  );
  return { onDone, onClose };
}

describe("ExternalAiDialog", () => {
  it("shows the prompt to copy and keeps the read button off until a reply is pasted", async () => {
    const run = vi.fn().mockRejectedValue(new ExternalPromptReady("PROMPT ĐẦY ĐỦ"));
    renderDialog(run);
    expect(await screen.findByTestId("ext-prompt")).toHaveTextContent("PROMPT ĐẦY ĐỦ");
    expect(run).toHaveBeenCalledWith({ mode: "prompt" });
    expect(screen.getByTestId("ext-read")).toBeDisabled();
    fireEvent.change(screen.getByTestId("ext-reply"), { target: { value: "  kết quả  " } });
    expect(screen.getByTestId("ext-read")).toBeEnabled();
  });

  it("reads the pasted reply, tells the screen to load what changed and closes", async () => {
    const run = vi
      .fn()
      .mockImplementation((external: ExternalAi) =>
        external.mode === "prompt"
          ? Promise.reject(new ExternalPromptReady("P"))
          : Promise.resolve("xong"),
      );
    const { onDone, onClose } = renderDialog(run);
    await screen.findByTestId("ext-prompt");
    fireEvent.change(screen.getByTestId("ext-reply"), { target: { value: '{"a":1}' } });
    fireEvent.click(screen.getByTestId("ext-read"));
    await waitFor(() => expect(onDone).toHaveBeenCalledWith());
    expect(run).toHaveBeenLastCalledWith({ mode: "paste", reply: '{"a":1}' });
    expect(onClose).toHaveBeenCalled();
  });

  it("explains a reply the step cannot use and keeps the pasted text", async () => {
    const run = vi
      .fn()
      .mockImplementation((external: ExternalAi) =>
        external.mode === "prompt"
          ? Promise.reject(new ExternalPromptReady("P"))
          : Promise.reject(new ApiError("Kết quả AI không dùng được: khối JSON không đọc được")),
      );
    const { onDone, onClose } = renderDialog(run);
    await screen.findByTestId("ext-prompt");
    fireEvent.change(screen.getByTestId("ext-reply"), { target: { value: "xin chào" } });
    fireEvent.click(screen.getByTestId("ext-read"));
    expect(await screen.findByTestId("ext-read-error")).toHaveTextContent(
      "khối JSON không đọc được",
    );
    expect(screen.getByTestId("ext-reply")).toHaveValue("xin chào");
    expect(onDone).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByTestId("ext-read")).toBeEnabled();
  });

  it("says why the prompt could not be built", async () => {
    renderDialog(
      vi.fn().mockRejectedValue(new ApiError("Duyệt brief ở bước Creative brief trước.")),
    );
    expect(await screen.findByTestId("ext-prompt-error")).toHaveTextContent("Duyệt brief");
    expect(screen.getByTestId("ext-read")).toBeDisabled();
  });

  it("without the AI Bridge extension says how to install it and has no queue button", async () => {
    renderDialog(vi.fn().mockRejectedValue(new ExternalPromptReady("P")));
    await screen.findByTestId("ext-prompt");
    expect(screen.queryByTestId("ext-queue")).not.toBeInTheDocument();
    expect(await screen.findByTestId("ext-extension-missing")).toHaveTextContent("ConceptFlow AI Bridge");
  });

  it("with an extension of the old protocol says to update it", async () => {
    const fake = installFakeAiBridge({ legacy: true });
    try {
      renderDialog(vi.fn().mockRejectedValue(new ExternalPromptReady("P")));
      expect(await screen.findByTestId("ext-queue-outdated")).toHaveTextContent("Cập nhật extension ConceptFlow AI Bridge");
      expect(screen.queryByTestId("ext-queue")).not.toBeInTheDocument();
    } finally {
      fake.remove();
    }
  });

  /** A step whose prompt can be copied and whose request can be described for the queue. */
  const queueable = () =>
    vi.fn().mockImplementation((external: ExternalAi) => {
      if (external.mode === "prompt") return Promise.reject(new ExternalPromptReady("PROMPT"));
      if (external.mode === "describe") return Promise.reject(new ExternalRequestDescribed(REQUEST));
      return Promise.resolve("xong");
    });

  const REQUEST = { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/story/generate", body: "{}" };

  it("queues the request, shows where the item stands and reloads when it is done", async () => {
    const fake = installFakeAiBridge({ queue: true });
    try {
      const { onDone, onClose } = renderDialog(queueable());
      fireEvent.click(await screen.findByTestId("ext-queue"));
      await waitFor(() => expect(fake.queued).toHaveLength(1));
      expect(fake.queued[0].item).toMatchObject({ label: "Câu chuyện · AI ngoài", project: "p1", request: REQUEST, modes: { deepThink: true, search: false } });
      act(() => fake.announce("q1", { status: "waiting", provider: "gemini" }));
      expect(await screen.findByTestId("ext-queue-stage")).toHaveTextContent("Đang chờ trong hàng đợi AI · sẽ chạy trên Gemini");
      act(() => fake.announce("q1", { status: "running", stage: "answering", provider: "gemini" }));
      expect(await screen.findByTestId("ext-queue-stage")).toHaveTextContent("Đang chạy trên Gemini");
      expect(onDone).not.toHaveBeenCalled();
      act(() => fake.announce("q1", { status: "done", provider: "gemini" }));
      await waitFor(() => expect(onDone).toHaveBeenCalledWith());
      expect(onClose).toHaveBeenCalled();
    } finally {
      fake.remove();
    }
  });

  it("keeps the answer in the box when the step cannot use it", async () => {
    const fake = installFakeAiBridge({ queue: true });
    try {
      const { onClose } = renderDialog(queueable());
      fireEvent.click(await screen.findByTestId("ext-queue"));
      await waitFor(() => expect(fake.queued).toHaveLength(1));
      act(() => fake.announce("q1", { status: "failed", provider: "grok", error: "khối JSON không đọc được", reply: "chỉ là văn bản" }));
      expect(await screen.findByTestId("ext-read-error")).toHaveTextContent("khối JSON không đọc được");
      expect(screen.getByTestId("ext-reply")).toHaveValue("chỉ là văn bản");
      expect(onClose).not.toHaveBeenCalled();
    } finally {
      fake.remove();
    }
  });

  it("shows why the item failed when there is no answer to keep", async () => {
    const fake = installFakeAiBridge({ queue: true });
    try {
      renderDialog(queueable());
      fireEvent.click(await screen.findByTestId("ext-queue"));
      await waitFor(() => expect(fake.queued).toHaveLength(1));
      act(() => fake.announce("q1", { status: "failed", provider: "gemini", error: "Chưa đăng nhập Gemini." }));
      expect(await screen.findByTestId("ext-queue-error")).toHaveTextContent("Chưa đăng nhập Gemini.");
      expect(screen.getByTestId("ext-reply")).toHaveValue("");
    } finally {
      fake.remove();
    }
  });

  it("closing the dialog does not take the item out of the queue", async () => {
    const fake = installFakeAiBridge({ queue: true });
    try {
      const onClose = vi.fn();
      renderDialog(queueable(), vi.fn(), onClose);
      fireEvent.click(await screen.findByTestId("ext-queue"));
      await waitFor(() => expect(fake.queued).toHaveLength(1));
      fireEvent.click(screen.getByTestId("ext-cancel"));
      expect(onClose).toHaveBeenCalled();
      expect(fake.cancels).toEqual([]);
    } finally {
      fake.remove();
    }
  });

  it("a step that shows its result instead of storing it has no queue button and gets the result of the pasted reply", async () => {
    const fake = installFakeAiBridge({ queue: true });
    try {
      const onDone = vi.fn();
      const run = queueable();
      render(<ExternalAiDialog open onClose={vi.fn()} title="Gợi ý" run={run} queued={false} onDone={onDone} testId="ext" />);
      await screen.findByTestId("ext-prompt");
      expect(screen.queryByTestId("ext-queue")).not.toBeInTheDocument();
      expect(screen.queryByTestId("ext-extension-missing")).not.toBeInTheDocument();
      fireEvent.change(screen.getByTestId("ext-reply"), { target: { value: "gợi ý" } });
      fireEvent.click(screen.getByTestId("ext-read"));
      await waitFor(() => expect(onDone).toHaveBeenCalledWith("xong"));
    } finally {
      fake.remove();
    }
  });
});
