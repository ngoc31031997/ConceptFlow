import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveUnit,
  editUnit,
  isRequestCancelled,
  redoUnit,
  unapproveUnit,
  type OperationCall,
  type ReviewUnit,
  type UnitRef,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** A step's screen state: the unit being reviewed, plus whatever else the server sends with it. */
export interface UnitDeskState<C> {
  unit: ReviewUnit<C> | null;
}

/** What a step that reviews one AI-written unit works with. */
export interface UnitDeskControls<S extends UnitDeskState<unknown>> {
  /** null until the server answers (or while the project does not exist). */
  state: S | null;
  loadError: string | null;
  /** The AI is writing the unit: a first run or a redo. */
  running: boolean;
  run: OperationRun;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** The note of the redo that failed, so the redo dialog starts from it. */
  failedNote: string;
  /** Has the AI write the unit; resolves false when the run failed. */
  generate: () => Promise<boolean>;
  redo: (note: string) => Promise<boolean>;
  /** Saves a hand edit; throws so the edit dialog can show why it failed. */
  edit: (fields: Record<string, string>) => Promise<void>;
  approve: () => Promise<void>;
  unapprove: () => Promise<void>;
  reload: () => void;
}

/** How a step loads and writes its unit. */
export interface UnitDeskConfig<C, S extends UnitDeskState<C>> {
  unit: UnitRef<C>;
  /** The operation kind the server records for the AI runs that write the unit. */
  runKind: string;
  load: (projectId: string) => Promise<S>;
  generate: (projectId: string, operation: OperationCall) => Promise<ReviewUnit<C>>;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes one review unit of a project; nothing loads until
 * `enabled` (the project exists on the server). Every change answers with the
 * new unit, shown at once, and is followed by a reload of the whole state,
 * since what depends on the unit can change with it.
 */
export function useUnitDesk<C, S extends UnitDeskState<C>>(
  config: UnitDeskConfig<C, S>,
  projectId: string,
  enabled: boolean,
): UnitDeskControls<S> {
  const { unit: unitRef, runKind, load, generate: generateUnit } = config;
  const [state, setState] = useState<S | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
  // Only the newest load may land: a slow first load must not overwrite the
  // unit a run that finished after it has just returned.
  const loadSeq = useRef(0);

  // Not gated on `enabled`: a run started in the same click that created the
  // project still holds the closure from before, and its reload must load.
  const reload = useCallback(() => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    load(projectId)
      .then((next) => {
        if (seq !== loadSeq.current) return;
        setState(next);
        setLoadError(null);
      })
      .catch((err) => {
        if (seq !== loadSeq.current) return;
        setLoadError(messageOf(err));
      });
  }, [projectId, load]);

  // A run that outlived the page is followed from here; the unit it stored is loaded when it ends.
  const run = useOperationRun({ projectId: enabled ? projectId : undefined, kinds: [runKind], onSettled: reload });

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) reload();
  }, [enabled, reload]);

  const showUnit = useCallback(
    (unit: ReviewUnit<C>) => {
      setState((prev) => (prev ? { ...prev, unit } : prev));
      reload();
    },
    [reload],
  );

  const runAI = useCallback(
    async (call: (operation: OperationCall) => Promise<ReviewUnit<C>>, note: string) => {
      setRunning(true);
      setActionError(null);
      try {
        showUnit(await call(run.begin()));
        setFailedNote("");
        run.end();
        return true;
      } catch (err) {
        if (isRequestCancelled(err)) return false;
        setActionError(messageOf(err));
        setFailedNote(note);
        // The card stays with the failure and the time already spent.
        return false;
      } finally {
        setRunning(false);
      }
    },
    [run, showUnit],
  );

  const generate = useCallback(() => runAI((op) => generateUnit(projectId, op), ""), [runAI, generateUnit, projectId]);
  const redo = useCallback(
    (note: string) => runAI((op) => redoUnit(projectId, unitRef, note, op), note),
    [runAI, projectId, unitRef],
  );

  const edit = useCallback(
    async (fields: Record<string, string>) => {
      setActionError(null);
      try {
        showUnit(await editUnit(projectId, unitRef, fields));
      } catch (err) {
        throw new Error(messageOf(err));
      }
    },
    [projectId, unitRef, showUnit],
  );

  const changeApproval = useCallback(
    async (call: () => Promise<ReviewUnit<C>>) => {
      setBusy(true);
      setActionError(null);
      try {
        showUnit(await call());
      } catch (err) {
        setActionError(messageOf(err));
      } finally {
        setBusy(false);
      }
    },
    [showUnit],
  );
  const approve = useCallback(() => changeApproval(() => approveUnit(projectId, unitRef)), [changeApproval, projectId, unitRef]);
  const unapprove = useCallback(
    () => changeApproval(() => unapproveUnit(projectId, unitRef)),
    [changeApproval, projectId, unitRef],
  );

  return { state, loadError, running: running || run.resumed, run, actionError, busy, failedNote, generate, redo, edit, approve, unapprove, reload };
}
