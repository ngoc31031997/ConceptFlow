import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { Pagination } from "../../../src/components/ui";

function renderPager(page: number, onPageChange = vi.fn(), onPageSizeChange = vi.fn()) {
  render(
    <Pagination
      page={page}
      pageCount={7}
      total={124}
      pageSize={20}
      pageSizes={[10, 20, 50]}
      label="video"
      onPageChange={onPageChange}
      onPageSizeChange={onPageSizeChange}
    />,
  );
  return { onPageChange, onPageSizeChange };
}

describe("Pagination", () => {
  it("says which rows are shown and marks the current page", () => {
    renderPager(2);
    expect(screen.getByTestId("pagination-summary")).toHaveTextContent("Hiển thị 21–40 / 124 video");
    expect(screen.getByTestId("pagination-page-2")).toHaveAttribute("aria-current", "page");
    expect(screen.getByTestId("pagination-page-3")).not.toHaveAttribute("aria-current");
  });

  it("counts the last page's rows up to the total", () => {
    renderPager(7);
    expect(screen.getByTestId("pagination-summary")).toHaveTextContent("Hiển thị 121–124 / 124 video");
  });

  it("disables Trước on the first page and Sau on the last", () => {
    renderPager(1);
    expect(screen.getByTestId("pagination-prev")).toBeDisabled();
    expect(screen.getByTestId("pagination-next")).toBeEnabled();
  });

  it("disables Sau on the last page", () => {
    renderPager(7);
    expect(screen.getByTestId("pagination-next")).toBeDisabled();
    expect(screen.getByTestId("pagination-prev")).toBeEnabled();
  });

  it("asks for the page the Creator picks", () => {
    const { onPageChange } = renderPager(4);
    fireEvent.click(screen.getByTestId("pagination-page-5"));
    fireEvent.click(screen.getByTestId("pagination-prev"));
    fireEvent.click(screen.getByTestId("pagination-next"));
    fireEvent.click(screen.getByTestId("pagination-page-4"));
    expect(onPageChange.mock.calls).toEqual([[5], [3], [5]]);
  });

  it("asks for a new page size", () => {
    const { onPageSizeChange } = renderPager(1);
    fireEvent.change(screen.getByTestId("pagination-size"), { target: { value: "50" } });
    expect(onPageSizeChange).toHaveBeenCalledWith(50);
  });
});
