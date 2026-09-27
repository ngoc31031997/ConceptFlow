import { useEffect, useState } from "react";

/** Có nên chạy hiệu ứng thu lại không: tắt khi người dùng xin giảm chuyển động (và trong jsdom, không có matchMedia). */
function motionAllowed(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") return false;
  return !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/**
 * Giữ khối còn trong DOM thêm `exitMs` sau khi `open` tắt, để kịp chạy hiệu
 * ứng thu lại. `closing` = đang thu lại (gắn class revealOut).
 */
export function usePresence(open: boolean, exitMs = 180) {
  const [mounted, setMounted] = useState(open);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    if (open) {
      setMounted(true);
      setClosing(false);
      return;
    }
    if (!mounted) return;
    if (!motionAllowed()) {
      setMounted(false);
      return;
    }
    setClosing(true);
    const id = window.setTimeout(() => {
      setMounted(false);
      setClosing(false);
    }, exitMs);
    return () => window.clearTimeout(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  return { mounted: open || mounted, closing: !open && closing };
}
