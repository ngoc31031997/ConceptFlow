import { useEffect, useState } from "react";
import { getUnitImpact, type UnitAddress } from "../api/client";

/**
 * What changing a unit sends back in later steps, as the server lists it.
 * `items` is null while the list loads.
 */
export interface UnitImpact {
  items: string[] | null;
  failed: boolean;
}

/**
 * True when the action reaches work a later step already did, so its button
 * takes the orange fill (docs/ux-ui-design-rules.md, rule 6). A list that could
 * not be read counts as reaching: the colour must not promise less than the
 * action may do.
 */
export function reachesLaterSteps(impact: UnitImpact): boolean {
  return impact.failed || (impact.items?.length ?? 0) > 0;
}

/** Loads the impact list each time `open` turns on, so the dialog shows today's dependents. */
export function useUnitImpact(projectId: string, ref: UnitAddress, open: boolean): UnitImpact {
  const [impact, setImpact] = useState<UnitImpact>({ items: null, failed: false });
  const { kind, key } = ref;

  useEffect(() => {
    if (!open || !projectId) return;
    let cancelled = false;
    setImpact({ items: null, failed: false });
    getUnitImpact(projectId, { kind, key })
      .then((items) => {
        if (!cancelled) setImpact({ items, failed: false });
      })
      .catch(() => {
        if (!cancelled) setImpact({ items: [], failed: true });
      });
    return () => {
      cancelled = true;
    };
  }, [open, projectId, kind, key]);

  return impact;
}
