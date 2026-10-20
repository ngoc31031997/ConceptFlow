import { useCallback, useEffect, useRef, useState } from "react";

export type CopyState = "idle" | "copied" | "failed";

const RESET_MS = 2000;

/**
 * Copies text to the clipboard and reports how it went for a couple of seconds,
 * so a button can switch its label. A browser that refuses (insecure context,
 * denied permission) ends in "failed" instead of looking as if it worked.
 */
export function useCopy(): { state: CopyState; copy: (text: string) => Promise<boolean> } {
  const [state, setState] = useState<CopyState>("idle");
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  const copy = useCallback(async (text: string) => {
    let ok = false;
    try {
      await navigator.clipboard.writeText(text);
      ok = true;
    } catch {
      ok = false;
    }
    setState(ok ? "copied" : "failed");
    if (timer.current) clearTimeout(timer.current);
    timer.current = setTimeout(() => setState("idle"), RESET_MS);
    return ok;
  }, []);

  return { state, copy };
}
