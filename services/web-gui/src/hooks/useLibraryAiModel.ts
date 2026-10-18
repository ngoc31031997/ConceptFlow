import { useState } from "react";

const KEY = "conceptflow.libraryAiModel";

/**
 * The model a drawing in the illustration library runs on. It belongs to no
 * project, so the choice is remembered in this browser ("" = server default);
 * a blocked storage just means the default each time.
 */
export function useLibraryAiModel(): [string, (model: string) => void] {
  const [model, setModel] = useState<string>(() => {
    try {
      return window.localStorage.getItem(KEY) ?? "";
    } catch {
      return "";
    }
  });
  const choose = (next: string) => {
    setModel(next);
    try {
      window.localStorage.setItem(KEY, next);
    } catch {
      /* per-viewer convenience only */
    }
  };
  return [model, choose];
}
