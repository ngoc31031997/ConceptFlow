import { useEffect, useId, useRef, useState, type KeyboardEvent } from "react";
import { useReadOnly } from "../../context/ReadOnlyContext";
import { ReadOnlyValue } from "./ReadOnlyValue";
import styles from "./Dropdown.module.css";

export interface DropdownOption {
  value: string;
  label: string;
  /** Dòng phụ nhỏ dưới nhãn — giải thích lựa chọn này là gì. */
  hint?: string;
  /** Nhãn ngắn bên phải (vd "Hệ thống", "Mặc định"). */
  badge?: string;
}

interface DropdownProps {
  value: string;
  options: DropdownOption[];
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  "aria-label"?: string;
  "data-testid"?: string;
  className?: string;
}

/**
 * Dropdown dùng chung thay cho <select> gốc của trình duyệt. Danh sách của
 * <select> gốc không style được (mỗi hệ điều hành vẽ một kiểu, không có dòng
 * phụ giải thích), nên nó lạc hẳn khỏi phong cách viền đậm + bóng khối của app.
 *
 * Bàn phím theo mẫu listbox của WAI-ARIA: mũi tên lên/xuống để di chuyển,
 * Enter/Space để chọn, Esc để đóng, Home/End để nhảy về đầu/cuối.
 */
export function Dropdown({
  value,
  options,
  onChange,
  placeholder = "Chọn...",
  disabled,
  id,
  className,
  ...rest
}: DropdownProps) {
  const readOnly = useReadOnly();
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLUListElement>(null);
  const listId = useId();
  const testId = rest["data-testid"];
  const selectedIndex = options.findIndex((o) => o.value === value);
  const selected = selectedIndex >= 0 ? options[selectedIndex] : null;

  useEffect(() => {
    if (!open) return;
    function onDown(event: MouseEvent) {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const item = listRef.current?.children[active] as HTMLElement | undefined;
    item?.scrollIntoView?.({ block: "nearest" });
  }, [open, active]);

  if (readOnly) {
    return <ReadOnlyValue value={selected?.label ?? ""} id={id} testId={testId} />;
  }

  function openList() {
    setActive(Math.max(0, selectedIndex));
    setOpen(true);
  }

  function choose(index: number) {
    const option = options[index];
    if (option) onChange(option.value);
    setOpen(false);
  }

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    if (disabled) return;
    const last = options.length - 1;
    switch (event.key) {
      case "ArrowDown":
      case "ArrowUp": {
        event.preventDefault();
        if (!open) return openList();
        const step = event.key === "ArrowDown" ? 1 : -1;
        setActive((i) => Math.min(last, Math.max(0, i + step)));
        return;
      }
      case "Home":
      case "End":
        if (!open) return;
        event.preventDefault();
        setActive(event.key === "Home" ? 0 : last);
        return;
      case "Enter":
      case " ":
        event.preventDefault();
        if (open) choose(active);
        else openList();
        return;
      case "Escape":
      case "Tab":
        setOpen(false);
        return;
    }
  }

  return (
    <div ref={rootRef} className={className ? `${styles.root} ${className}` : styles.root}>
      <button
        type="button"
        id={id}
        className={`${styles.trigger} ${open ? styles.triggerOpen : ""}`}
        onClick={() => (open ? setOpen(false) : openList())}
        onKeyDown={onKeyDown}
        disabled={disabled}
        role="combobox"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open ? `${listId}-${active}` : undefined}
        aria-label={rest["aria-label"]}
        data-testid={testId}
      >
        <span className={styles.triggerText}>
          <span className={selected ? styles.value : styles.placeholder}>{selected?.label ?? placeholder}</span>
          {selected?.hint && <span className={styles.triggerHint}>{selected.hint}</span>}
        </span>
        <svg className={styles.chevron} width="14" height="14" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M6 9l6 6 6-6" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      </button>

      {open && (
        <ul ref={listRef} id={listId} role="listbox" className={styles.list} aria-label={rest["aria-label"]}>
          {options.map((option, index) => {
            const isSelected = option.value === value;
            return (
              // Bàn phím do nút trigger xử lý (focus không rời nút), theo mẫu
              // listbox của WAI-ARIA — option chỉ cần nhận chuột.
              // eslint-disable-next-line jsx-a11y/click-events-have-key-events
              <li
                key={option.value}
                id={`${listId}-${index}`}
                role="option"
                aria-selected={isSelected}
                className={`${styles.option} ${index === active ? styles.optionActive : ""} ${isSelected ? styles.optionSelected : ""}`}
                onMouseEnter={() => setActive(index)}
                onMouseDown={(event) => event.preventDefault()}
                onClick={(event) => {
                  // Dropdown hay nằm trong <label> (FormField): không chặn thì
                  // label chuyển cú click sang nút trigger và mở lại danh sách.
                  event.preventDefault();
                  choose(index);
                }}
                data-testid={testId ? `${testId}-option-${option.value || "none"}` : undefined}
              >
                <span className={styles.check} aria-hidden="true">
                  {isSelected ? "✓" : ""}
                </span>
                <span className={styles.optionText}>
                  <span className={styles.optionLabel}>{option.label}</span>
                  {option.hint && <span className={styles.optionHint}>{option.hint}</span>}
                </span>
                {option.badge && <span className={styles.badge}>{option.badge}</span>}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
