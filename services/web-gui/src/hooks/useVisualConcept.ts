import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  acceptVisualConceptFinding,
  applyVisualConceptFinding,
  approveAllVisualConcept,
  approveVisualConceptUnit,
  editVisualConceptUnit,
  generateVisualConcept,
  generateVisualConceptUnit,
  getVisualConcept,
  isRequestCancelled,
  lockVisualConcept,
  newOperationId,
  reviewVisualConcept,
  setVisualConceptModel,
  setVisualConceptRefs,
  unapproveVisualConceptUnit,
  type VisualConceptState,
  type VisualConceptUnitKey,
} from "../api/client";

/** What the system's AI is doing now: every empty part, one part, or the review. */
export type ConceptRunning = "all" | VisualConceptUnitKey | "review" | null;

/** What the step 8 screen gets from useVisualConcept. */
export interface VisualConceptControls {
  /** null until the server answers. */
  state: VisualConceptState | null;
  /** The script is not locked yet, so step 8 cannot open. */
  notOpen: boolean;
  loadError: string | null;
  running: ConceptRunning;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  reload: () => Promise<void>;
  /** Stops the AI run in flight; the server drops the call when the connection closes. */
  stop: () => void;
  generateAll: (note: string, rewriteAll: boolean) => Promise<boolean>;
  generateUnit: (key: VisualConceptUnitKey, note: string, findingId?: string) => Promise<boolean>;
  review: (note: string) => Promise<boolean>;
  /** Saves a hand edit; throws so the form can show why it failed. */
  edit: (key: VisualConceptUnitKey, fields: Record<string, unknown>) => Promise<void>;
  approve: (key: VisualConceptUnitKey) => Promise<boolean>;
  unapprove: (key: VisualConceptUnitKey) => Promise<boolean>;
  approveAll: () => Promise<boolean>;
  setRefs: (refIds: string[]) => Promise<boolean>;
  setModel: (model: string) => Promise<boolean>;
  applyFix: (findingId: string) => Promise<boolean>;
  accept: (findingId: string, why: string) => Promise<boolean>;
  lock: (note: string) => Promise<boolean>;
}

/** The server's reason, or the generic connection message. */
export function visualErrorText(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes step 8 "Ngôn ngữ hình ảnh". Every change answers with the
 * whole screen, which replaces the state. An AI run of the system's AI holds
 * the request open until the answer is stored; `stop` cancels it.
 */
export function useVisualConcept(projectId: string): VisualConceptControls {
  const [state, setState] = useState<VisualConceptState | null>(null);
  const [notOpen, setNotOpen] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<ConceptRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const abort = useRef<AbortController | null>(null);
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const reload = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getVisualConcept(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setNotOpen(false);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      if (err instanceof ApiError && err.code === "script_not_locked") {
        setNotOpen(true);
        setLoadError(null);
        return;
      }
      setLoadError(visualErrorText(err));
    }
  }, [projectId]);

  useEffect(() => {
    setState(null);
    void reload();
  }, [reload]);

  useEffect(() => () => abort.current?.abort(), []);

  const act = useCallback(async (call: () => Promise<VisualConceptState>): Promise<boolean> => {
    setBusy(true);
    setActionError(null);
    try {
      loadSeq.current += 1;
      setState(await call());
      return true;
    } catch (err) {
      setActionError(visualErrorText(err));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  const runAi = useCallback(
    async (
      what: Exclude<ConceptRunning, null>,
      call: (op: { id: string; signal: AbortSignal }) => Promise<VisualConceptState>,
    ) => {
      const controller = new AbortController();
      abort.current = controller;
      setRunning(what);
      setActionError(null);
      try {
        loadSeq.current += 1;
        setState(await call({ id: newOperationId(), signal: controller.signal }));
        return true;
      } catch (err) {
        if (!isRequestCancelled(err)) setActionError(visualErrorText(err));
        // A run stopped or failed half way may have stored the parts written before it.
        await reload();
        return false;
      } finally {
        abort.current = null;
        setRunning(null);
      }
    },
    [reload],
  );

  return {
    state,
    notOpen,
    loadError,
    running,
    busy,
    actionError,
    reload,
    stop: () => abort.current?.abort(),
    generateAll: (note, rewriteAll) =>
      runAi("all", (op) => generateVisualConcept(projectId, note, rewriteAll, op)),
    generateUnit: (key, note, findingId) =>
      runAi(key, (op) => generateVisualConceptUnit(projectId, key, note, op, findingId)),
    review: (note) => runAi("review", (op) => reviewVisualConcept(projectId, note, op)),
    edit: async (key, fields) => {
      loadSeq.current += 1;
      setState(await editVisualConceptUnit(projectId, key, fields));
    },
    approve: (key) => act(() => approveVisualConceptUnit(projectId, key)),
    unapprove: (key) => act(() => unapproveVisualConceptUnit(projectId, key)),
    approveAll: () => act(() => approveAllVisualConcept(projectId)),
    setRefs: (refIds) => act(() => setVisualConceptRefs(projectId, refIds)),
    setModel: (model) => act(() => setVisualConceptModel(projectId, model)),
    applyFix: (findingId) => act(() => applyVisualConceptFinding(projectId, findingId)),
    accept: (findingId, why) => act(() => acceptVisualConceptFinding(projectId, findingId, why)),
    lock: (note) => act(() => lockVisualConcept(projectId, note)),
  };
}
