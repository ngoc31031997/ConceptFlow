import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, ExternalPromptReady, ExternalRequestDescribed, type ExternalAi } from "../../src/api/client";
import { ExternalAiButton } from "../../src/components/ExternalAiButton";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let bridge: FakeAiBridge | null = null;

afterEach(() => {
  bridge?.remove();
  bridge = null;
});

const REQUEST = { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/story/generate", body: "{}" };

function stepRun(pasteResult: () => Promise<unknown>) {
  return vi.fn().mockImplementation((external: ExternalAi) => {
    if (external.mode === "prompt") return Promise.reject(new ExternalPromptReady("PROMPT"));
    if (external.mode === "describe") return Promise.reject(new ExternalRequestDescribed(REQUEST));
    return pasteResult();
  });
}

function renderButton(run: ReturnType<typeof stepRun>, onDone = vi.fn()) {
  render(<ExternalAiButton title="Câu chuyện · AI ngoài" run={run} onDone={onDone} testId="ext" />);
  return { onDone };
}

describe("ExternalAiButton · Thêm vào hàng đợi AI", () => {
  it("shows no queue button without the extension", async () => {
    bridge = installFakeAiBridge({ answersPing: false });
    renderButton(stepRun(() => Promise.resolve("xong")));
    expect(screen.getByTestId("ext")).toBeInTheDocument();
    expect(screen.queryByTestId("ext-queue")).not.toBeInTheDocument();
  });

  it("explains how to install the extension in the dialog when it is missing", async () => {
    bridge = installFakeAiBridge({ answersPing: false });
    renderButton(stepRun(() => Promise.resolve("xong")));
    fireEvent.click(screen.getByTestId("ext"));
    expect(await screen.findByTestId("ext-dialog-extension-missing")).toHaveTextContent("ConceptFlow AI Bridge");
  });

  it("says to update an extension of the old protocol", async () => {
    bridge = installFakeAiBridge({ legacy: true });
    renderButton(stepRun(() => Promise.resolve("xong")));
    expect(await screen.findByTestId("ext-queue-outdated")).toHaveTextContent("Cập nhật extension ConceptFlow AI Bridge");
  });

  it("queues the request, follows the item and reloads when it is done", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-queue"));
    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    expect(bridge!.queued[0].item).toMatchObject({ label: "Câu chuyện · AI ngoài", project: "p1", request: REQUEST });
    expect(screen.queryByTestId("ext-dialog")).not.toBeInTheDocument();

    act(() => bridge!.announce("q1", { status: "running", stage: "opening", provider: "grok" }));
    expect(await screen.findByTestId("ext-queue-stage")).toHaveTextContent("Đang mở Grok");
    act(() => bridge!.announce("q1", { status: "done", provider: "grok" }));
    await waitFor(() => expect(onDone).toHaveBeenCalledWith());
    expect(await screen.findByTestId("ext-queue")).toBeEnabled();
    expect(run).not.toHaveBeenCalledWith(expect.objectContaining({ mode: "paste" }));
  });

  it("says why the item failed and reloads nothing", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-queue"));
    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    act(() => bridge!.announce("q1", { status: "failed", provider: "gemini", error: "Chưa đăng nhập Gemini." }));

    expect(await screen.findByTestId("ext-queue-error")).toHaveTextContent("Chưa đăng nhập Gemini.");
    expect(onDone).not.toHaveBeenCalled();
  });

  it("says why the request could not be described and queues nothing", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const run = vi.fn().mockRejectedValue(new ApiError("Duyệt brief ở bước Creative brief trước."));
    renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-queue"));
    expect(await screen.findByTestId("ext-queue-error")).toHaveTextContent("Duyệt brief");
    expect(bridge.queued).toHaveLength(0);
  });

  it("opens the dialog with the answer and the reason when the step cannot read it", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-queue"));
    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    act(() =>
      bridge!.announce("q1", { status: "failed", provider: "deepseek", error: "khối JSON không đọc được", reply: "xin chào" }),
    );

    expect(await screen.findByTestId("ext-dialog-read-error")).toHaveTextContent("khối JSON không đọc được");
    expect(screen.getByTestId("ext-dialog-reply")).toHaveValue("xin chào");
    expect(onDone).not.toHaveBeenCalled();
  });

  it("takes the item out of the queue when cancelled and reads nothing", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-queue"));
    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    act(() => bridge!.announce("q1", { status: "running", stage: "answering" }));
    fireEvent.click(await screen.findByTestId("ext-queue-cancel"));

    await waitFor(() => expect(bridge!.cancels).toEqual(["q1"]));
    act(() => bridge!.announce("q1", { status: "cancelled" }));
    expect(await screen.findByTestId("ext-queue")).toBeEnabled();
    expect(screen.queryByTestId("ext-queue-error")).not.toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();
  });

  it("a step that shows its result instead of storing it has no queue button", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const onDone = vi.fn();
    render(<ExternalAiButton title="Gợi ý · AI ngoài" run={stepRun(() => Promise.resolve("gợi ý"))} queued={false} onDone={onDone} testId="ext" />);
    await new Promise((r) => setTimeout(r, 20));
    expect(screen.getByTestId("ext")).toBeInTheDocument();
    expect(screen.queryByTestId("ext-queue")).not.toBeInTheDocument();
  });
});
