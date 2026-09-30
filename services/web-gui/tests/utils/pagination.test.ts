import { describe, it, expect } from "vitest";
import { pageCount, pageItems } from "../../src/utils/pagination";

describe("pagination", () => {
  it("counts pages, at least one", () => {
    expect(pageCount(0, 20)).toBe(1);
    expect(pageCount(40, 20)).toBe(2);
    expect(pageCount(124, 20)).toBe(7);
  });

  it("shows first, last and the pages around the current one", () => {
    expect(pageItems(1, 1)).toEqual([1]);
    expect(pageItems(1, 3)).toEqual([1, 2, 3]);
    expect(pageItems(5, 9)).toEqual([1, "…", 4, 5, 6, "…", 9]);
    expect(pageItems(1, 7)).toEqual([1, 2, "…", 7]);
    expect(pageItems(7, 7)).toEqual([1, "…", 6, 7]);
  });

  it("shows a single skipped page as its number, not an ellipsis", () => {
    expect(pageItems(3, 7)).toEqual([1, 2, 3, 4, "…", 7]);
    expect(pageItems(4, 7)).toEqual([1, 2, 3, 4, 5, 6, 7]);
  });
});
