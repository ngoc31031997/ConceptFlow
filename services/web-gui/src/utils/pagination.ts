/** CR-054 — phân trang: số video mỗi trang Creator chọn được. */
export const PAGE_SIZES = [10, 20, 50];

/** Số trang, ít nhất 1 (danh sách rỗng vẫn là một trang). */
export function pageCount(total: number, size: number): number {
  return Math.max(1, Math.ceil(total / size));
}

/**
 * Các ô số trang cần hiện: trang đầu, trang cuối, trang hiện tại ±1. Khoảng
 * trống từ 2 trang trở lên thành "…"; trống đúng 1 trang thì hiện luôn số đó.
 */
export function pageItems(current: number, count: number): (number | "…")[] {
  const shown = new Set([1, count, current - 1, current, current + 1]);
  const items: (number | "…")[] = [];
  let last = 0;
  for (let n = 1; n <= count; n++) {
    if (!shown.has(n)) continue;
    if (n - last === 2) items.push(n - 1);
    else if (n - last > 2) items.push("…");
    items.push(n);
    last = n;
  }
  return items;
}
