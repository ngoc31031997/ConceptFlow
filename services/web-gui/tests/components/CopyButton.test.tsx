import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CopyBox } from "../../src/components/ui";

describe("CopyBox quick copy", () => {
  it("copies the text in one click and confirms", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(
      <CopyBox text="nội dung prompt" what="prompt" copyTestId="qc">
        <pre>nội dung prompt</pre>
      </CopyBox>,
    );
    fireEvent.click(screen.getByTestId("qc"));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith("nội dung prompt"));
    await waitFor(() => expect(screen.getByTestId("qc")).toHaveTextContent("Đã chép"));
  });

  it("says so when the browser refuses", async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) } });
    render(
      <CopyBox text="x" copyTestId="qc">
        <pre>x</pre>
      </CopyBox>,
    );
    fireEvent.click(screen.getByTestId("qc"));
    await waitFor(() => expect(screen.getByTestId("qc")).toHaveTextContent("Không chép được"));
  });

  it("is disabled when there is nothing to copy", () => {
    render(
      <CopyBox text="" copyTestId="qc">
        <pre />
      </CopyBox>,
    );
    expect(screen.getByTestId("qc")).toBeDisabled();
  });
});
