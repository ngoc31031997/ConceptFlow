import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  cancelOperation,
  listRunningOperations,
  newOperationId,
  type OperationCall,
  type OperationProgress,
} from "../api/client";
import { useOperationProgress } from "./useOperationProgress";

export interface OperationRun {
  /** Live progress; null until the first poll answers (or when not running). */
  progress: OperationProgress | null;
  /** Mint an id and start polling it; pass the returned call with the request. */
  begin: () => OperationCall;
  /** Stop polling and drop the card. */
  end: () => void;
  /** Stop the call (the server drops the model call) and drop the card. */
  cancel: () => Promise<void>;
  /** Why the server could not be told to stop the call; null otherwise. */
  cancelError: string | null;
  /** A call this page did not start is still running; the screen treats it as its own run. */
  resumed: boolean;
}

/** What `useOperationRun` needs to pick up a call that outlived the page that started it. */
export interface OperationRunOptions {
  /** With `kinds`: on mount, adopt a call of this project that is still running. */
  projectId?: string;
  /** The operation kinds this screen starts. */
  kinds?: string[];
  /** Called once an adopted call has ended, so the screen reloads what it stored. */
  onSettled?: () => void;
  /** Abort the request when the page goes away, for a call whose answer is not stored anywhere. */
  abortOnUnmount?: boolean;
}

/**
 * The GUI side of a long call: pick an operation id, send it
 * with the request, and poll `/v1/operations/{id}` while the request is open.
 * The request also carries an abort signal so the Creator can stop it. A call
 * keeps running on the server when the Creator leaves the page; with
 * `projectId` and `kinds` the page finds it again on return and follows it,
 * and `cancel` stops it through the server.
 */
export function useOperationRun(options: OperationRunOptions = {}): OperationRun {
  const { projectId, kinds, onSettled, abortOnUnmount } = options;
  const [operationId, setOperationId] = useState<string | null>(null);
  const [resumed, setResumed] = useState(false);
  const [cancelError, setCancelError] = useState<string | null>(null);
  const progress = useOperationProgress(operationId);
  const controller = useRef<AbortController | null>(null);
  const idRef = useRef<string | null>(null);
  const settled = useRef(onSettled);
  settled.current = onSettled;
  const kindsKey = kinds?.join(",") ?? "";

  const begin = useCallback((): OperationCall => {
    const id = newOperationId();
    const next = new AbortController();
    controller.current = next;
    idRef.current = id;
    setCancelError(null);
    setOperationId(id);
    return { id, signal: next.signal };
  }, []);
  const end = useCallback(() => {
    controller.current = null;
    idRef.current = null;
    setResumed(false);
    setOperationId(null);
  }, []);

  useEffect(() => {
    if (!abortOnUnmount) return;
    return () => controller.current?.abort();
  }, [abortOnUnmount]);

  // A call started by an earlier visit to this page is still running: follow it.
  useEffect(() => {
    if (!projectId || !kindsKey) return;
    let stale = false;
    listRunningOperations(projectId)
      .then((running) => {
        const found = running.find((op) => kindsKey.split(",").includes(op.kind));
        if (stale || !found || idRef.current) return;
        idRef.current = found.id;
        setResumed(true);
        setOperationId(found.id);
      })
      .catch(() => undefined); // best-effort, like the progress poll: no card is better than a false error
    return () => {
      stale = true;
    };
  }, [projectId, kindsKey]);

  const status = progress?.status;
  useEffect(() => {
    if (!resumed || !status || status === "running" || status === "pending") return;
    setResumed(false);
    if (status !== "failed") {
      idRef.current = null;
      setOperationId(null);
    }
    settled.current?.();
  }, [resumed, status]);

  const cancel = useCallback(async () => {
    const id = idRef.current;
    const local = controller.current;
    const told = id ? cancelOperation(id) : Promise.resolve();
    local?.abort();
    try {
      await told;
    } catch (err) {
      // 404: the call had already ended, which is what the Creator wanted.
      if (!(err instanceof ApiError && err.status === 404) && !local) {
        setCancelError(err instanceof Error ? err.message : String(err));
        return;
      }
    }
    end();
  }, [end]);

  return { progress, begin, end, cancel, cancelError, resumed };
}
