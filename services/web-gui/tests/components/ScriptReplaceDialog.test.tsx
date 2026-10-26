import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../src/api/client";
import { ScriptReplaceDialog } from "../../src/components/script/ScriptReplaceDialog";

const PREVIEW: client.ScriptReplaceResult = {
  matches: [
    { beat: "b1", line: "b1.1", before: "TRNG chậm.", after: "TI A EN GI chậm." },
    { beat: "b2", line: "b2.1", before: "Lại TRNG.", after: "Lại TI A EN GI." },
  ],
  skippedApproved: [{ beat: "b3", lines: 2 }],
  applied: false,
};

function renderDialog(onApplied = vi.fn(), onClose = vi.fn()) {
  render(<ScriptReplaceDialog open onClose={onClose} projectId="p1" onApplied={onApplied} />);
  return { onApplied, onClose };
}

function type(find: string, replace: string) {
  fireEvent.change(screen.getByTestId("script-replace-find"), { target: { value: find } });
  fireEvent.change(screen.getByTestId("script-replace-with"), { target: { value: replace } });
}

afterEach(() => vi.restoreAllMocks());

describe("ScriptReplaceDialog", () => {
  it("previews every line before it changes anything, and names the beats it leaves alone", async () => {
    const replace = vi.spyOn(client, "replaceInScript").mockResolvedValue(PREVIEW);
    renderDialog();
    expect(screen.getByTestId("script-replace-apply")).toBeDisabled();
    type("TRNG", "TI A EN GI");
    fireEvent.click(screen.getByTestId("script-replace-preview"));
    expect(await screen.findByTestId("script-replace-match-b1.1")).toHaveTextContent(
      "TI A EN GI chậm.",
    );
    expect(replace).toHaveBeenCalledWith("p1", {
      find: "TRNG",
      replace: "TI A EN GI",
      field: "spoken",
      matchCase: true,
      apply: false,
    });
    expect(screen.getByTestId("script-replace-summary")).toHaveTextContent("2 câu ở 2 beat");
    expect(screen.getByTestId("script-replace-skipped")).toHaveTextContent("b3 (2 câu)");
    expect(screen.getByTestId("script-replace-apply")).toHaveTextContent("Đổi 2 câu");
  });

  it("applies what was previewed, reloads the screen and closes", async () => {
    const replace = vi
      .spyOn(client, "replaceInScript")
      .mockImplementation(async (_id, o) => ({ ...PREVIEW, applied: o.apply }));
    const { onApplied, onClose } = renderDialog();
    type("TRNG", "TI A EN GI");
    fireEvent.click(screen.getByTestId("script-replace-field-text"));
    fireEvent.click(screen.getByTestId("script-replace-preview"));
    fireEvent.click(await screen.findByTestId("script-replace-apply").then((b) => b));
    await waitFor(() => expect(onApplied).toHaveBeenCalled());
    expect(replace).toHaveBeenLastCalledWith("p1", {
      find: "TRNG",
      replace: "TI A EN GI",
      field: "text",
      matchCase: true,
      apply: true,
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("clears the preview when the word changes, so only what was shown can be applied", async () => {
    vi.spyOn(client, "replaceInScript").mockResolvedValue(PREVIEW);
    renderDialog();
    type("TRNG", "X");
    fireEvent.click(screen.getByTestId("script-replace-preview"));
    await screen.findByTestId("script-replace-matches");
    fireEvent.change(screen.getByTestId("script-replace-with"), { target: { value: "Y" } });
    expect(screen.queryByTestId("script-replace-matches")).not.toBeInTheDocument();
    expect(screen.getByTestId("script-replace-apply")).toBeDisabled();
  });

  it("says when nothing matches, and shows the server's reason for a refusal", async () => {
    const replace = vi
      .spyOn(client, "replaceInScript")
      .mockResolvedValueOnce({ matches: [], skippedApproved: [], applied: false });
    renderDialog();
    type("ZZZ", "X");
    fireEvent.click(screen.getByTestId("script-replace-preview"));
    expect(await screen.findByTestId("script-replace-none")).toBeInTheDocument();
    replace.mockRejectedValueOnce(new client.ApiError("câu b2.1 sẽ trống sau khi thay"));
    fireEvent.click(screen.getByTestId("script-replace-preview"));
    expect(await screen.findByTestId("script-replace-error")).toHaveTextContent("sẽ trống");
  });
});
