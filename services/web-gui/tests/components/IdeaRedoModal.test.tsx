import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ExternalPromptReady, ExternalRequestDescribed, type ExternalAi } from "../../src/api/client";
import { IdeaRedoModal } from "../../src/components/idea/IdeaRedoModal";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let bridge: FakeAiBridge | null = null;

afterEach(() => {
  bridge?.remove();
  bridge = null;
});

const REQUEST = { gateway: "http://localhost:8080", method: "POST", path: "/v1/projects/p1/idea/redo", body: '{"note":"ghi chú"}' };

function renderModal() {
  const run = vi.fn().mockImplementation((_note: string, external: ExternalAi) => {
    if (external.mode === "prompt") return Promise.reject(new ExternalPromptReady("PROMPT LÀM LẠI"));
    if (external.mode === "describe") return Promise.reject(new ExternalRequestDescribed(REQUEST));
    return Promise.resolve("xong");
  });
  const onDone = vi.fn();
  const onClose = vi.fn();
  render(
    <IdeaRedoModal open onClose={onClose} approved={false} initialNote="" external={{ run, onDone }} />,
  );
  return { run, onDone, onClose };
}

describe("IdeaRedoModal · Thêm vào hàng đợi AI", () => {
  it("needs a note, then queues the redo with that note and closes after reading", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const { run, onDone, onClose } = renderModal();

    const button = await screen.findByTestId("idea-redo-queue");
    expect(button).toBeDisabled();
    fireEvent.change(screen.getByTestId("idea-redo-note"), { target: { value: "  nhấn mạnh ngộ nhận " } });
    expect(button).toBeEnabled();
    fireEvent.click(button);

    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    expect(run).toHaveBeenCalledWith("nhấn mạnh ngộ nhận", { mode: "describe" });
    expect(bridge!.queued[0].item).toMatchObject({ label: "Làm lại kèm ghi chú · Ý tưởng", request: REQUEST });
    act(() => bridge!.announce("q1", { status: "done", provider: "gemini" }));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(onClose).toHaveBeenCalled();
  });

  it("hands over to the outside-AI dialog with the answer when the step cannot read it", async () => {
    bridge = installFakeAiBridge({ queue: true });
    const { onDone, onClose } = renderModal();

    fireEvent.change(await screen.findByTestId("idea-redo-note"), { target: { value: "ghi chú" } });
    fireEvent.click(await screen.findByTestId("idea-redo-queue"));
    await waitFor(() => expect(bridge!.queued).toHaveLength(1));
    act(() => bridge!.announce("q1", { status: "failed", provider: "grok", error: "Kết quả AI không dùng được", reply: "trả lời hỏng" }));

    expect(await screen.findByTestId("idea-redo-external-dialog-read-error")).toHaveTextContent("không dùng được");
    expect(screen.getByTestId("idea-redo-external-dialog-reply")).toHaveValue("trả lời hỏng");
    expect(onDone).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });
});
