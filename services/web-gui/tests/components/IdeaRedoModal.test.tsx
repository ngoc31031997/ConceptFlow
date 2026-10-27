import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, ExternalPromptReady, type ExternalAi } from "../../src/api/client";
import { IdeaRedoModal } from "../../src/components/idea/IdeaRedoModal";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let bridge: FakeAiBridge | null = null;

afterEach(() => {
  bridge?.remove();
  bridge = null;
});

function renderModal(pasteResult: () => Promise<unknown>) {
  const run = vi.fn().mockImplementation((_note: string, external: ExternalAi) =>
    external.mode === "prompt" ? Promise.reject(new ExternalPromptReady("PROMPT LÀM LẠI")) : pasteResult(),
  );
  const onDone = vi.fn();
  const onClose = vi.fn();
  render(
    <IdeaRedoModal open onClose={onClose} approved={false} initialNote="" external={{ run, onDone }} />,
  );
  return { run, onDone, onClose };
}

describe("IdeaRedoModal · Chạy bằng DeepSeek", () => {
  it("needs a note, then runs the redo with that note on DeepSeek and closes after reading", async () => {
    bridge = installFakeAiBridge();
    const { run, onDone, onClose } = renderModal(() => Promise.resolve("xong"));

    const button = await screen.findByTestId("idea-redo-deepseek");
    expect(button).toBeDisabled();
    fireEvent.change(screen.getByTestId("idea-redo-note"), { target: { value: "  nhấn mạnh ngộ nhận " } });
    expect(button).toBeEnabled();
    fireEvent.click(button);

    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    expect(run).toHaveBeenCalledWith("nhấn mạnh ngộ nhận", { mode: "prompt" });
    act(() => bridge!.send({ type: "result", request_id: bridge!.runs[0].requestId, text: "ý tưởng mới" }));

    await waitFor(() => expect(onDone).toHaveBeenCalledTimes(1));
    expect(run).toHaveBeenLastCalledWith("nhấn mạnh ngộ nhận", { mode: "paste", reply: "ý tưởng mới" });
    expect(onClose).toHaveBeenCalled();
  });

  it("hands over to the outside-AI dialog with the answer when the step cannot read it", async () => {
    bridge = installFakeAiBridge();
    const { onDone, onClose } = renderModal(() => Promise.reject(new ApiError("Kết quả AI không dùng được")));

    fireEvent.change(await screen.findByTestId("idea-redo-note"), { target: { value: "ghi chú" } });
    fireEvent.click(await screen.findByTestId("idea-redo-deepseek"));
    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    act(() => bridge!.send({ type: "result", request_id: bridge!.runs[0].requestId, text: "trả lời hỏng" }));

    expect(await screen.findByTestId("idea-redo-external-dialog-read-error")).toHaveTextContent("không dùng được");
    expect(screen.getByTestId("idea-redo-external-dialog-reply")).toHaveValue("trả lời hỏng");
    expect(onDone).not.toHaveBeenCalled();
    expect(onClose).not.toHaveBeenCalled();
  });
});
