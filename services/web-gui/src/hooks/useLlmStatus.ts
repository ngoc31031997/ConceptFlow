import { useEffect, useState } from "react";
import { getLlmStatus, type LlmStatus } from "../api/client";

/**
 * CR-027 FR79.4 — whether the "Gọi API trực tiếp" mode exists on this
 * deployment at all.
 *
 * `null` means "not known yet", which callers must treat as neither: drawing
 * the AI option and then taking it away a tick later is worse than waiting a
 * tick. A failed request counts as disabled — if the GUI cannot even ask, the
 * copy-out path is the honest thing to offer.
 */
export function useLlmStatus(): LlmStatus | null {
  const [status, setStatus] = useState<LlmStatus | null>(null);

  useEffect(() => {
    let cancelled = false;
    getLlmStatus()
      .then((s) => {
        if (!cancelled) setStatus(s);
      })
      .catch(() => {
        if (!cancelled) setStatus({ enabled: false, provider: "", reason: "" });
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return status;
}
