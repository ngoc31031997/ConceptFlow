import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, ExternalPromptReady, type ExternalAi } from "../../src/api/client";
import { ExternalAiButton } from "../../src/components/ExternalAiButton";
import { installFakeAiBridge, type FakeAiBridge } from "../helpers/fakeAiBridge";

let bridge: FakeAiBridge | null = null;

afterEach(() => {
  bridge?.remove();
  bridge = null;
});

function stepRun(pasteResult: () => Promise<unknown>) {
  return vi.fn().mockImplementation((external: ExternalAi) =>
    external.mode === "prompt" ? Promise.reject(new ExternalPromptReady("PROMPT")) : pasteResult(),
  );
}

function renderButton(run: ReturnType<typeof stepRun>, onDone = vi.fn()) {
  render(<ExternalAiButton title="Câu chuyện · AI ngoài" run={run} onDone={onDone} testId="ext" />);
  return { onDone };
}

describe("ExternalAiButton · Chạy bằng DeepSeek", () => {
  it("shows no DeepSeek button without the extension", async () => {
    bridge = installFakeAiBridge({ answersPing: false });
    renderButton(stepRun(() => Promise.resolve("xong")));
    expect(screen.getByTestId("ext")).toBeInTheDocument();
    expect(screen.queryByTestId("ext-deepseek")).not.toBeInTheDocument();
  });

  it("explains how to install the extension in the dialog when it is missing", async () => {
    bridge = installFakeAiBridge({ answersPing: false });
    renderButton(stepRun(() => Promise.resolve("xong")));
    fireEvent.click(screen.getByTestId("ext"));
    expect(await screen.findByTestId("ext-dialog-deepseek-missing")).toHaveTextContent("ConceptFlow AI Bridge");
  });

  it("runs the prompt on DeepSeek, has the step read the answer and hands the result on", async () => {
    bridge = installFakeAiBridge();
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-deepseek"));
    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    expect(bridge!.runs[0]).toMatchObject({ provider: "deepseek", prompt: "PROMPT" });
    expect(screen.queryByTestId("ext-dialog")).not.toBeInTheDocument();

    act(() => bridge!.send({ type: "result", request_id: bridge!.runs[0].requestId, text: "câu trả lời" }));
    await waitFor(() => expect(onDone).toHaveBeenCalledWith("xong"));
    expect(run).toHaveBeenLastCalledWith({ mode: "paste", reply: "câu trả lời" });
  });

  it("says why the extension gave no answer and calls nothing else", async () => {
    bridge = installFakeAiBridge();
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-deepseek"));
    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    act(() => bridge!.send({ type: "error", request_id: bridge!.runs[0].requestId, message: "Chưa đăng nhập DeepSeek" }));

    expect(await screen.findByTestId("ext-deepseek-error")).toHaveTextContent("Chưa đăng nhập DeepSeek");
    expect(onDone).not.toHaveBeenCalled();
    expect(run).toHaveBeenCalledTimes(1);
  });

  it("says why the prompt could not be built and does not run DeepSeek", async () => {
    bridge = installFakeAiBridge();
    const run = vi.fn().mockRejectedValue(new ApiError("Duyệt brief ở bước Creative brief trước."));
    renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-deepseek"));
    expect(await screen.findByTestId("ext-deepseek-error")).toHaveTextContent("Duyệt brief");
    expect(bridge.runs).toHaveLength(0);
  });

  it("opens the dialog with the answer and the reason when the step cannot read it", async () => {
    bridge = installFakeAiBridge();
    const run = stepRun(() => Promise.reject(new ApiError("Kết quả AI không dùng được: khối JSON không đọc được")));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-deepseek"));
    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    act(() => bridge!.send({ type: "result", request_id: bridge!.runs[0].requestId, text: "xin chào" }));

    expect(await screen.findByTestId("ext-dialog-read-error")).toHaveTextContent("khối JSON không đọc được");
    expect(screen.getByTestId("ext-dialog-reply")).toHaveValue("xin chào");
    expect(onDone).not.toHaveBeenCalled();
  });

  it("stops waiting when cancelled and reads nothing", async () => {
    bridge = installFakeAiBridge();
    const run = stepRun(() => Promise.resolve("xong"));
    const { onDone } = renderButton(run);

    fireEvent.click(await screen.findByTestId("ext-deepseek"));
    await waitFor(() => expect(bridge!.runs).toHaveLength(1));
    act(() => bridge!.send({ type: "progress", request_id: bridge!.runs[0].requestId, stage: "answering" }));
    fireEvent.click(await screen.findByTestId("ext-deepseek-cancel"));

    await waitFor(() => expect(bridge!.cancels).toEqual([bridge!.runs[0].requestId]));
    expect(await screen.findByTestId("ext-deepseek")).toBeEnabled();
    expect(screen.queryByTestId("ext-deepseek-error")).not.toBeInTheDocument();
    expect(onDone).not.toHaveBeenCalled();
    expect(run).toHaveBeenCalledTimes(1);
  });
});
