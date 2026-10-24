import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  BEAT_DURATION_STEP,
  GENERIC_CONNECTION_ERROR,
  addBeat,
  approveBeatSkeleton,
  approveUnit,
  buildBeatSkeleton,
  beatUnit,
  deleteBeat,
  editUnit,
  generateBeats,
  getBeats,
  isRequestCancelled,
  redoUnit,
  reorderBeats,
  unapproveUnit,
  type Beat,
  type BeatFn,
  type BeatState,
  type OperationCall,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kinds the server records for the AI runs of the beat sheet step. */
const BEAT_RUN_KINDS = ["beat_sheet", "beat"];

/** What the AI is writing, for the card that follows the run. */
export type BeatsRunning = "sheet" | "add" | { beat: string } | null;

/** The beat sheet step as the screen works with it. */
export interface BeatsControls {
  /** null until the server answers. */
  state: BeatState | null;
  loadError: string | null;
  /** What the AI is writing now; a run found again after leaving the page counts as "sheet". */
  running: BeatsRunning;
  run: OperationRun;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** The note of the AI run that failed, so its dialog starts from it. */
  failedNote: string;
  /** Builds the skeleton from the approved story (no AI). */
  buildSkeleton: () => Promise<boolean>;
  /** Approves the skeleton, so the AI may write the content. */
  approveSkeleton: () => Promise<boolean>;
  /**
   * The AI writes the content of the beats of the approved skeleton, one beat at a time:
   * only those without content with `onlyUnwritten`, every beat otherwise.
   */
  generate: (note: string, onlyUnwritten?: boolean) => Promise<boolean>;
  /** Adds a beat after `after` ("" on an empty sheet): the AI writes it, or with `blank` an empty one is added. */
  add: (after: string, blank: boolean, note: string) => Promise<boolean>;
  redo: (key: string, note: string) => Promise<boolean>;
  /** Saves a hand edit of the text fields of one beat; throws so the edit dialog can show why it failed. */
  edit: (key: string, fields: Record<string, string>) => Promise<void>;
  setFn: (beat: Beat, fn: BeatFn) => Promise<boolean>;
  /** Moves a beat's target length one step (BEAT_DURATION_STEP seconds) up or down. */
  shiftDuration: (beat: Beat, direction: 1 | -1) => Promise<boolean>;
  setFacts: (beat: Beat, facts: string[]) => Promise<boolean>;
  move: (key: string, direction: -1 | 1) => Promise<boolean>;
  remove: (key: string) => Promise<boolean>;
  approve: (key: string) => Promise<boolean>;
  unapprove: (key: string) => Promise<boolean>;
  /** Approves every beat still waiting that has a goal. */
  approveAll: () => Promise<boolean>;
  /** Reads the state again; resolves when it has landed. */
  reload: () => Promise<void>;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's beat sheet; nothing loads until `enabled`.
 * Every change is followed by a reload of the whole state, since one change
 * (a duration, a deleted beat, a new order) moves the checks and the approval.
 */
export function useBeats(projectId: string, enabled: boolean): BeatsControls {
  const [state, setState] = useState<BeatState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<BeatsRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getBeats(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      setLoadError(messageOf(err));
    }
  }, [projectId]);

  // A run that outlived the page is followed from here; what it stored is loaded when it ends.
  const run = useOperationRun({ projectId: enabled ? projectId : undefined, kinds: BEAT_RUN_KINDS, onSettled: () => void load() });

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) void load();
  }, [enabled, load]);

  // Runs a change, then reloads; reports whether it worked.
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

  // Runs an AI call under an operation id, then reloads; a failure keeps the note it ran with.
  const runAI = useCallback(
    async (what: Exclude<BeatsRunning, null>, note: string, call: (operation: OperationCall) => Promise<unknown>) => {
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

  const editFields = useCallback((key: string, fields: Record<string, string>) => editUnit(projectId, beatUnit(key), fields), [projectId]);

  const edit = useCallback(
    async (key: string, fields: Record<string, string>) => {
      setActionError(null);
      try {
        await editFields(key, fields);
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    [editFields, load],
  );

  const approve = useCallback((key: string) => act(() => approveUnit(projectId, beatUnit(key))), [act, projectId]);

  return {
    state,
    loadError,
    running: running ?? (run.resumed ? "sheet" : null),
    run,
    busy,
    actionError,
    failedNote,
    buildSkeleton: () => act(() => buildBeatSkeleton(projectId)),
    approveSkeleton: () => act(() => approveBeatSkeleton(projectId)),
    generate: (note, onlyUnwritten = false) => runAI("sheet", note, (op) => generateBeats(projectId, note, op, onlyUnwritten)),
    add: (after, blank, note) => (blank ? act(() => addBeat(projectId, after, true, "")) : runAI("add", note, (op) => addBeat(projectId, after, false, note, op))),
    redo: (key, note) => runAI({ beat: key }, note, (op) => redoUnit(projectId, beatUnit(key), note, op)),
    edit,
    setFn: (beat, fn) => act(() => editFields(beat.key, { fn })),
    shiftDuration: (beat, direction) => act(() => editFields(beat.key, { dur: String(beat.dur + direction * BEAT_DURATION_STEP) })),
    setFacts: (beat, facts) => act(() => editFields(beat.key, { facts: facts.join(", ") })),
    move: (key, direction) =>
      act(() => {
        const order = (state?.beats ?? []).map((b) => b.key);
        const at = order.indexOf(key);
        const to = at + direction;
        if (at < 0 || to < 0 || to >= order.length) return Promise.resolve();
        [order[at], order[to]] = [order[to], order[at]];
        return reorderBeats(projectId, order);
      }),
    remove: (key) => act(() => deleteBeat(projectId, key)),
    approve,
    unapprove: (key) => act(() => unapproveUnit(projectId, beatUnit(key))),
    approveAll: () =>
      act(async () => {
        for (const beat of state?.beats ?? []) {
          if (beat.status === "pending" && beat.goal.trim() !== "") await approveUnit(projectId, beatUnit(beat.key));
        }
      }),
    reload: load,
  };
}
