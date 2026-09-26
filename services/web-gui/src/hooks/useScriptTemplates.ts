import { useEffect, useState } from "react";
import { getScriptTemplates, type ScriptTemplates } from "../api/client";

// Static text: once fetched it is kept for the page's lifetime. Only a success
// is cached — a failed load is retried by the next mount.
let loaded: ScriptTemplates | null = null;

/** The starter scripts and hook/end-screen snippets (CR-040 FR113); null until loaded. */
export function useScriptTemplates(): ScriptTemplates | null {
  const [templates, setTemplates] = useState<ScriptTemplates | null>(loaded);
  useEffect(() => {
    if (loaded) return;
    let cancelled = false;
    getScriptTemplates()
      .then((t) => {
        loaded = t;
        if (!cancelled) setTemplates(t);
      })
      .catch(() => {
        /* buttons that need the text simply stay disabled */
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return templates;
}
