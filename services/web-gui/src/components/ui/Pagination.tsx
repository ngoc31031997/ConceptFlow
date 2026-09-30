import { pageItems } from "../../utils/pagination";
import styles from "./Pagination.module.css";

interface PaginationProps {
  page: number;
  pageCount: number;
  total: number;
  pageSize: number;
  pageSizes: number[];
  /** Danh từ đếm được, ví dụ "video". */
  label: string;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

/**
 * Thanh phân trang. Không giữ state: trang hiện tại do nơi gọi quyết.
 * Trái → phải: đang hiện gì, số mỗi trang, rồi các nút chuyển trang.
 */
export function Pagination({
  page, pageCount, total, pageSize, pageSizes, label, onPageChange, onPageSizeChange,
}: PaginationProps) {
  const from = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);
  return (
    <nav className={styles.bar} aria-label="Phân trang" data-testid="pagination">
      <span className={styles.summary} data-testid="pagination-summary">
        Hiển thị {from}–{to} / {total} {label}
      </span>
      <label className={styles.size}>
        Mỗi trang
        <select
          value={pageSize}
          onChange={(e) => onPageSizeChange(Number(e.target.value))}
          data-testid="pagination-size"
        >
          {pageSizes.map((s) => (
            <option key={s} value={s}>{s}</option>
          ))}
        </select>
      </label>
      <div className={styles.pages}>
        <button
          type="button"
          className={styles.btn}
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
          data-testid="pagination-prev"
        >
          ‹ Trước
        </button>
        {pageItems(page, pageCount).map((item, i) =>
          item === "…" ? (
            <span key={`gap-${i}`} className={styles.gap} aria-hidden="true">…</span>
          ) : (
            <button
              key={item}
              type="button"
              className={`${styles.btn} ${item === page ? styles.current : ""}`}
              aria-current={item === page ? "page" : undefined}
              aria-label={`Trang ${item}`}
              onClick={() => item !== page && onPageChange(item)}
              data-testid={`pagination-page-${item}`}
            >
              {item}
            </button>
          ),
        )}
        <button
          type="button"
          className={styles.btn}
          disabled={page >= pageCount}
          onClick={() => onPageChange(page + 1)}
          data-testid="pagination-next"
        >
          Sau ›
        </button>
      </div>
    </nav>
  );
}
