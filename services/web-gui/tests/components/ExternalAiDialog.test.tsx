import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ApiError, ExternalPromptReady, type ExternalAi } from "../../src/api/client";
import { ExternalAiDialog } from "../../src/components/ExternalAiDialog";

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

  it("reads the pasted reply, hands the result on and closes", async () => {
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
    await waitFor(() => expect(onDone).toHaveBeenCalledWith("xong"));
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
});
