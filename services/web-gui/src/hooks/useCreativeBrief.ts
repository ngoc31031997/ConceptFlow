import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  BRIEF_UNIT,
  GENERIC_CONNECTION_ERROR,
  approveUnit,
  editUnit,
  generateBrief,
  getBriefState,
  redoUnit,
  unapproveUnit,
  type BriefState,
  type CreativeBrief,
  type ReviewUnit,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The idea step's brief as the screen works with it. */
export interface CreativeBriefControls {
  /** null until the server answers (or while the project does not exist). */
  state: BriefState | null;
  loadError: string | null;
  /** The AI is writing the brief: a first run or a redo. */
  running: boolean;
  run: OperationRun;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** The note of the redo that failed, so the redo dialog starts from it. */
  failedNote: string;
  /** Has the AI write the brief from the topic; resolves false when the run failed. */
  generate: () => Promise<boolean>;
  redo: (note: string) => Promise<boolean>;
  /** Saves a hand edit; throws so the edit dialog can show why it failed. */
  edit: (fields: Record<string, string>) => Promise<void>;
  approve: () => Promise<void>;
  unapprove: () => Promise<void>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's creative brief; nothing loads until
 * `enabled` (the project exists on the server). Every change answers with the
 * new unit, shown at once, and is followed by a reload of the whole state,
 * since whether the story outline was written from this brief can change with it.
 */
export function useCreativeBrief(projectId: string, enabled: boolean): CreativeBriefControls {
  const [state, setState] = useState<BriefState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
  const run = useOperationRun();
  // Only the newest load may land: a slow first load must not overwrite the
  // brief a run that finished after it has just returned.
  const loadSeq = useRef(0);

  // Not gated on `enabled`: a run started in the same click that created the
  // project still holds the closure from before, and its reload must load.
  const reload = useCallback(() => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    getBriefState(projectId)
      .then((next) => {
        if (seq !== loadSeq.current) return;
        setState(next);
        setLoadError(null);
      })
      .catch((err) => {
        if (seq !== loadSeq.current) return;
        setLoadError(messageOf(err));
      });
  }, [projectId]);

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) reload();
  }, [enabled, reload]);

  const showUnit = useCallback(
    (unit: ReviewUnit<CreativeBrief>) => {
      setState((prev) => (prev ? { ...prev, unit } : prev));
      reload();
    },
    [reload],
  );

  const runAI = useCallback(
    async (call: (operationId: string) => Promise<ReviewUnit<CreativeBrief>>, note: string) => {
      setRunning(true);
      setActionError(null);
      try {
        showUnit(await call(run.begin()));
        setFailedNote("");
        run.end();
        return true;
      } catch (err) {
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

  const generate = useCallback(() => runAI((op) => generateBrief(projectId, op), ""), [runAI, projectId]);
  const redo = useCallback(
    (note: string) => runAI((op) => redoUnit(projectId, BRIEF_UNIT, note, op), note),
    [runAI, projectId],
  );

  const edit = useCallback(
    async (fields: Record<string, string>) => {
      setActionError(null);
      try {
        showUnit(await editUnit(projectId, BRIEF_UNIT, fields));
      } catch (err) {
        throw new Error(messageOf(err));
      }
    },
    [projectId, showUnit],
  );

  const changeApproval = useCallback(
    async (call: () => Promise<ReviewUnit<CreativeBrief>>) => {
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
  const approve = useCallback(() => changeApproval(() => approveUnit(projectId, BRIEF_UNIT)), [changeApproval, projectId]);
  const unapprove = useCallback(
    () => changeApproval(() => unapproveUnit(projectId, BRIEF_UNIT)),
    [changeApproval, projectId],
  );

  return { state, loadError, running, run, actionError, busy, failedNote, generate, redo, edit, approve, unapprove, reload };
}
