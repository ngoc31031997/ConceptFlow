import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  acceptFinding,
  addFindingToHandoff,
  addGlossaryTerm,
  addReviewNote,
  applyFinding,
  approveUnit,
  confirmHandoff,
  editUnit,
  generateHandoff,
  getScriptLock,
  handoffBeatUnit,
  isRequestCancelled,
  lockScript,
  redoUnit,
  reviewScript,
  unapproveUnit,
  type OperationCall,
  type ScriptLockState,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kinds the server records for the AI runs of the lock step. */
const LOCK_RUN_KINDS = ["script_review", "handoff_beat"];

/** What the AI is doing, for the card that follows the run. */
export type LockRunning = "review" | "handoff" | { beat: string } | null;

export interface ScriptLockControls {
  /** null until the server answers. */
  state: ScriptLockState | null;
  loadError: string | null;
  running: LockRunning;
  run: OperationRun;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  /** The note of the AI redo that failed, so its dialog starts from it. */
  failedNote: string;
  review: () => Promise<boolean>;
  generate: (beat: string) => Promise<boolean>;
  redo: (beat: string, note: string) => Promise<boolean>;
  /** Saves a hand edit of sections of a beat's note; throws so the dialog can show why it failed. */
  edit: (beat: string, fields: Record<string, string>) => Promise<void>;
  approve: (beat: string) => Promise<boolean>;
  unapprove: (beat: string) => Promise<boolean>;
  /** Keeps a stale note as it is and approves it. */
  confirm: (beat: string) => Promise<boolean>;
  approveAll: () => Promise<boolean>;
  applyFix: (findingId: string) => Promise<boolean>;
  toHandoff: (findingId: string) => Promise<boolean>;
  accept: (findingId: string, why: string) => Promise<boolean>;
  addNote: (text: string) => Promise<boolean>;
  addTerm: (term: { term: string; say: string; draw: string; avoid: string }) => Promise<void>;
  lock: (note: string) => Promise<boolean>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the lock step; nothing loads until `enabled`. Every change
 * is followed by a reload of the whole state, since one change (a note, a
 * decision, a line) moves the conditions and the draft.
 */
export function useScriptLock(projectId: string, enabled: boolean): ScriptLockControls {
  const [state, setState] = useState<ScriptLockState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<LockRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getScriptLock(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      setLoadError(messageOf(err));
    }
  }, [projectId]);

  const run = useOperationRun({ projectId: enabled ? projectId : undefined, kinds: LOCK_RUN_KINDS, onSettled: () => void load() });

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) void load();
  }, [enabled, load]);

  const act = useCallback(
    async (change: () => Promise<unknown>): Promise<boolean> => {
      setBusy(true);
      setActionError(null);
      try {
        await change();
        await load();
        return true;
      } catch (err) {
        setActionError(messageOf(err));
        await load();
        return false;
      } finally {
        setBusy(false);
      }
    },
    [load],
  );

  const runAI = useCallback(
    async (what: Exclude<LockRunning, null>, note: string, call: (operation: OperationCall) => Promise<unknown>) => {
      setRunning(what);
      setActionError(null);
      try {
        await call(run.begin());
        setFailedNote("");
        run.end();
        await load();
        return true;
      } catch (err) {
        if (isRequestCancelled(err)) return false;
        setActionError(messageOf(err));
        setFailedNote(note);
        await load();
        return false;
      } finally {
        setRunning(null);
      }
    },
    [run, load],
  );

  return {
    state,
    loadError,
    running: running ?? (run.resumed ? "review" : null),
    run,
    busy,
    actionError,
    failedNote,
    review: () => runAI("review", "", (op) => reviewScript(projectId, op)),
    generate: (beat) => runAI(beat ? { beat } : "handoff", "", (op) => generateHandoff(projectId, beat, op)),
    redo: (beat, note) => runAI({ beat }, note, (op) => redoUnit(projectId, handoffBeatUnit(beat), note, op)),
    edit: async (beat, fields) => {
      setActionError(null);
      try {
        await editUnit(projectId, handoffBeatUnit(beat), fields);
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    approve: (beat) => act(() => approveUnit(projectId, handoffBeatUnit(beat))),
    unapprove: (beat) => act(() => unapproveUnit(projectId, handoffBeatUnit(beat))),
    confirm: (beat) => act(() => confirmHandoff(projectId, beat)),
    approveAll: () =>
      act(async () => {
        for (const beat of state?.beats ?? []) {
          if (!beat.written) continue;
          if (beat.stale) await confirmHandoff(projectId, beat.key);
          else if (beat.status === "pending") await approveUnit(projectId, handoffBeatUnit(beat.key));
        }
      }),
    applyFix: (id) => act(() => applyFinding(projectId, id)),
    toHandoff: (id) => act(() => addFindingToHandoff(projectId, id)),
    accept: (id, why) => act(() => acceptFinding(projectId, id, why)),
    addNote: (text) => act(() => addReviewNote(projectId, text)),
    addTerm: async (term) => {
      try {
        await addGlossaryTerm(projectId, term);
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    lock: (note) => act(() => lockScript(projectId, note)),
    reload: () => void load(),
  };
}
