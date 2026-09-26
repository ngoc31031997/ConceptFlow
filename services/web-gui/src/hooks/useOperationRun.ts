import { useCallback, useState } from "react";
import { newOperationId, type OperationProgress } from "../api/client";
import { useOperationProgress } from "./useOperationProgress";

export interface OperationRun {
  /** Live progress; null until the first poll answers (or when not running). */
  progress: OperationProgress | null;
  /** Mint an id and start polling it; pass the id with the request. */
  begin: () => string;
  /** Stop polling and drop the card. */
  end: () => void;
}

/**
 * The GUI side of a long call (CR-040 FR116.3): pick an operation id, send it
 * with the request, and poll `/v1/operations/{id}` while the request is open.
 */
export function useOperationRun(): OperationRun {
  const [operationId, setOperationId] = useState<string | null>(null);
  const progress = useOperationProgress(operationId);
  const begin = useCallback(() => {
    const id = newOperationId();
    setOperationId(id);
    return id;
  }, []);
  const end = useCallback(() => setOperationId(null), []);
  return { progress, begin, end };
}
