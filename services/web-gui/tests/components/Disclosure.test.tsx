import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Disclosure } from "../../src/components/Disclosure";

describe("Disclosure", () => {
  it("ẩn nội dung mặc định, hiện tiêu đề và hint", () => {
    render(
      <Disclosure title="Tiêu đề" hint="Gợi ý" testId="d1">
        <p>Nội dung ẩn</p>
      </Disclosure>,
    );

    expect(screen.getByText("Tiêu đề")).toBeInTheDocument();
    expect(screen.getByText("Gợi ý")).toBeInTheDocument();
    expect(screen.queryByText("Nội dung ẩn")).not.toBeInTheDocument();
  });

  it("bấm vào toggle thì hiện nội dung", () => {
    render(
      <Disclosure title="Tiêu đề" testId="d1">
        <p>Nội dung ẩn</p>
      </Disclosure>,
    );

    fireEvent.click(screen.getByTestId("d1-toggle"));
    expect(screen.getByText("Nội dung ẩn")).toBeInTheDocument();
    expect(screen.getByTestId("d1-toggle")).toHaveAttribute("aria-expanded", "true");
  });

  it("defaultOpen=true thì mở sẵn từ đầu", () => {
    render(
      <Disclosure title="Tiêu đề" defaultOpen testId="d1">
        <p>Nội dung</p>
      </Disclosure>,
    );

    expect(screen.getByText("Nội dung")).toBeInTheDocument();
  });

  it("bấm lần hai thì thu gọn lại", () => {
    render(
      <Disclosure title="Tiêu đề" testId="d1">
        <p onClick={vi.fn()}>Nội dung</p>
      </Disclosure>,
    );

    fireEvent.click(screen.getByTestId("d1-toggle"));
    fireEvent.click(screen.getByTestId("d1-toggle"));
    expect(screen.queryByText("Nội dung")).not.toBeInTheDocument();
  });
});
