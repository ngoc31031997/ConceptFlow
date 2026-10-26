import { describe, it, expect, vi } from "vitest";
import { useState } from "react";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { Button, Dialog } from "../../src/components/ui";

/** Opens a dialog from a button, the way every screen does. */
function Harness({ hasInput = false, onClose }: { hasInput?: boolean; onClose?: () => void }) {
  const [open, setOpen] = useState(false);
  const close = () => {
    onClose?.();
    setOpen(false);
  };
  return (
    <>
      <Button onClick={() => setOpen(true)} data-testid="opener">
        Sửa tay
      </Button>
      <Dialog
        open={open}
        onClose={close}
        title="Sửa tay · Creative brief"
        hasInput={hasInput}
        testId="dlg"
        backdropTestId="dlg-backdrop"
        footer={
          <>
            <Button variant="ghost" onClick={close} data-testid="dlg-cancel">
              Huỷ
            </Button>
            <Button data-testid="dlg-main">Lưu bản sửa</Button>
          </>
        }
      >
        {hasInput ? <textarea data-testid="dlg-field" /> : <p>Không có gì để gõ.</p>}
      </Dialog>
    </>
  );
}

describe("Dialog", () => {
  it("applies the width class of the requested size", () => {
    render(
      <Dialog open onClose={() => {}} title="Sửa tay · Creative brief" size="xl" testId="dlg-xl" footer={<Button>Lưu</Button>}>
        <p>nhiều dữ liệu</p>
      </Dialog>,
    );
    expect(screen.getByTestId("dlg-xl").className).toMatch(/xl/);
  });

  it("moves focus to the first field, and back to the opener when closed", async () => {
    render(<Harness hasInput />);
    const opener = screen.getByTestId("opener");
    opener.focus();
    fireEvent.click(opener);

    expect(screen.getByTestId("dlg-field")).toHaveFocus();
    expect(screen.getByRole("dialog")).toHaveAccessibleName("Sửa tay · Creative brief");

    fireEvent.click(screen.getByTestId("dlg-cancel"));
    await waitFor(() => expect(screen.queryByTestId("dlg")).not.toBeInTheDocument());
    expect(opener).toHaveFocus();
  });

  it("focuses the main action when there is no field", () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("opener"));
    expect(screen.getByTestId("dlg-main")).toHaveFocus();
  });

  it("closes on Esc and on ×", () => {
    const onClose = vi.fn();
    render(<Harness onClose={onClose} />);
    fireEvent.click(screen.getByTestId("opener"));
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("opener"));
    fireEvent.click(screen.getByTestId("dlg-close"));
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("keeps a dialog with a form open on a backdrop click, but closes one without", () => {
    const onClose = vi.fn();
    const { unmount } = render(<Harness hasInput onClose={onClose} />);
    fireEvent.click(screen.getByTestId("opener"));
    fireEvent.click(screen.getByTestId("dlg-backdrop"));
    expect(onClose).not.toHaveBeenCalled();
    unmount();

    render(<Harness onClose={onClose} />);
    fireEvent.click(screen.getByTestId("opener"));
    fireEvent.click(screen.getByTestId("dlg-backdrop"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("keeps Tab inside the dialog", () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("opener"));
    const main = screen.getByTestId("dlg-main");
    main.focus();
    fireEvent.keyDown(document, { key: "Tab" });
    expect(screen.getByTestId("dlg-close")).toHaveFocus();
  });

  it("locks the page behind from scrolling while open", async () => {
    render(<Harness />);
    fireEvent.click(screen.getByTestId("opener"));
    expect(document.body.style.overflow).toBe("hidden");
    fireEvent.click(screen.getByTestId("dlg-cancel"));
    await waitFor(() => expect(document.body.style.overflow).toBe(""));
  });

  it("puts the page back where it was scrolled when it closes", () => {
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation(() => {});
    Object.defineProperty(window, "scrollY", { value: 480, configurable: true });
    const { rerender } = render(
      <Dialog open onClose={() => {}} title="T" footer={<button>ok</button>}>
        <p>x</p>
      </Dialog>,
    );
    rerender(
      <Dialog open={false} onClose={() => {}} title="T" footer={<button>ok</button>}>
        <p>x</p>
      </Dialog>,
    );
    expect(scrollTo).toHaveBeenCalledWith(0, 480);
    scrollTo.mockRestore();
  });
});
