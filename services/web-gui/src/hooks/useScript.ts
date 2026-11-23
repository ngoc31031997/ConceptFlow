import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveUnit,
  editUnit,
  generateScript,
  getScript,
  isRequestCancelled,
  scriptBeatUnit,
  speakScriptLines,
  unapproveUnit,
  type OperationCall,
  type ScriptState,
} from "../api/client";
import { useLinePlayer, type LinePlayer } from "./useLinePlayer";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kind the server records for the AI runs of the script step. */
const SCRIPT_RUN_KINDS = ["script_beat"];

/** How often the screen asks again while voice files are being made. */
const READING_POLL_MS = 1500;

/** What the AI is writing, for the cards that follow the run: the beats without dialogue, one after another. */
export type ScriptRunning = "write" | null;

/** The script step as the screen works with it. */
export interface ScriptControls {
  /** null until the server answers. */
  state: ScriptState | null;
  loadError: string | null;
  running: ScriptRunning;
  run: OperationRun;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  player: LinePlayer;
  /** Writes the dialogue of every beat that has none, each stored as it is written. */
  generate: (note: string) => Promise<boolean>;
  /** Saves a hand edit of the lines of one beat; throws so the edit dialog can show why it failed. */
  edit: (beat: string, fields: Record<string, string>) => Promise<void>;
  /** Replaces the text of one line with a new sentence; the old spoken wording no longer fits it and is cleared. */
  setLine: (beat: string, lineId: string, text: string) => Promise<boolean>;
  approve: (beat: string) => Promise<boolean>;
  unapprove: (beat: string) => Promise<boolean>;
  /** Approves every beat that has dialogue and is still waiting. */
  approveAll: () => Promise<boolean>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's script; nothing loads until `enabled`.
 * Every change is followed by a reload of the whole state, since one change
 * (a line, an approval) moves the flags, the joins and the length. While voice
 * files are being made the state is reloaded until they are ready.
 */
export function useScript(projectId: string, enabled: boolean): ScriptControls {
  const [state, setState] = useState<ScriptState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<ScriptRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getScript(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      setLoadError(messageOf(err));
    }
  }, [projectId]);

  const run = useOperationRun({
    projectId: enabled ? projectId : undefined,
    kinds: SCRIPT_RUN_KINDS,
    onSettled: () => void load(),
  });

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) void load();
  }, [enabled, load]);

  // Voice files are made in the background: ask again until every line asked for has its answer.
  const reading =
    state?.beats.some((b) =>
      b.lines.some((l) => l.audio.state === "pending" || l.audio.state === "running"),
    ) ?? false;
  useEffect(() => {
    if (!reading) return;
    const timer = window.setTimeout(() => void load(), READING_POLL_MS);
    return () => window.clearTimeout(timer);
  }, [reading, state, load]);

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
    async (call: (operation: OperationCall) => Promise<unknown>) => {
      setRunning("write");
      setActionError(null);
      try {
        await call(run.begin());
        run.end();
        await load();
        return true;
      } catch (err) {
        if (isRequestCancelled(err)) return false;
        setActionError(messageOf(err));
        await load();
        return false;
      } finally {
        setRunning(null);
      }
    },
    [run, load],
  );

  const editFields = useCallback(
    (beat: string, fields: Record<string, string>) =>
      editUnit(projectId, scriptBeatUnit(beat), fields),
    [projectId],
  );

  const edit = useCallback(
    async (beat: string, fields: Record<string, string>) => {
      setActionError(null);
      try {
        await editFields(beat, fields);
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    [editFields, load],
  );

  const speak = useCallback(
    async (lineIds: string[]) => {
      try {
        const next = await speakScriptLines(projectId, lineIds);
        setState(next);
        return true;
      } catch (err) {
        setActionError(messageOf(err));
        await load();
        return false;
      }
    },
    [projectId, load],
  );

  const player = useLinePlayer(projectId, state, speak);

  return {
    state,
    loadError,
    running: running ?? (run.resumed ? "write" : null),
    run,
    busy,
    actionError,
    player,
    generate: (note) => runAI((op) => generateScript(projectId, "", note, op)),
    edit,
    setLine: (beat, lineId, text) =>
      act(() => editFields(beat, { [`line:${lineId}`]: text, [`spoken:${lineId}`]: "" })),
    approve: (beat) => act(() => approveUnit(projectId, scriptBeatUnit(beat))),
    unapprove: (beat) => act(() => unapproveUnit(projectId, scriptBeatUnit(beat))),
    approveAll: () =>
      act(async () => {
        for (const beat of state?.beats ?? []) {
          if (beat.written && beat.status === "pending")
            await approveUnit(projectId, scriptBeatUnit(beat.key));
        }
      }),
    reload: () => void load(),
  };
}
