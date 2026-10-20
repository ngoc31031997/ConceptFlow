import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  STORY_UNIT,
  approveUnit,
  editUnit,
  generateStory,
  generateStoryApproaches,
  getStory,
  isRequestCancelled,
  redoUnit,
  saveApproachAsArchetype,
  selectStoryApproach,
  setStoryFactUse,
  storyFieldUnit,
  unapproveUnit,
  type OperationCall,
  type StoryFieldKey,
  type StoryState,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kinds the server records for the AI runs of the story step. */
const STORY_RUN_KINDS = ["story_approach", "story", "story_field"];

/** What the AI is writing, for the card that follows the run. */
export type StoryRunning = "approaches" | "story" | { field: StoryFieldKey } | null;

/** The story step as the screen works with it. */
export interface StoryControls {
  /** null until the server answers. */
  state: StoryState | null;
  loadError: string | null;
  /** What the AI is writing now; a run found again after leaving the page counts as "story". */
  running: StoryRunning;
  run: OperationRun;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** The note of the AI run that failed, so its dialog starts from it. */
  failedNote: string;
  generateApproaches: (note: string) => Promise<boolean>;
  /** Chooses the approach; with `rewrite` the AI then writes the fields the Creator has not written. */
  selectApproach: (key: string, rewrite: boolean) => Promise<boolean>;
  saveArchetype: (key: string) => Promise<boolean>;
  generateStory: (note: string) => Promise<boolean>;
  redoField: (key: StoryFieldKey, note: string) => Promise<boolean>;
  /** Saves a hand edit of one field; throws so the edit dialog can show why it failed. */
  editField: (key: StoryFieldKey, text: string) => Promise<void>;
  setFactUse: (id: string, use: boolean) => Promise<boolean>;
  approve: () => Promise<boolean>;
  unapprove: () => Promise<boolean>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's story step; nothing loads until `enabled`.
 * Every change is followed by a reload of the whole state, since one change
 * (a field, a fact turned off, a new approach) moves the approval and the
 * facts the story uses.
 */
export function useStory(projectId: string, enabled: boolean): StoryControls {
  const [state, setState] = useState<StoryState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<StoryRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getStory(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      setLoadError(messageOf(err));
    }
  }, [projectId]);

  // A run that outlived the page is followed from here; what it stored is loaded when it ends.
  const run = useOperationRun({ projectId: enabled ? projectId : undefined, kinds: STORY_RUN_KINDS, onSettled: () => void load() });

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
        return false;
      } finally {
        setBusy(false);
      }
    },
    [load],
  );

  // Runs an AI call under an operation id, then reloads; a failure keeps the note it ran with.
  const runAI = useCallback(
    async (what: Exclude<StoryRunning, null>, note: string, call: (operation: OperationCall) => Promise<unknown>) => {
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
        // The card stays with the failure and the time already spent.
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

  const generateApproaches = useCallback(
    (note: string) => runAI("approaches", note, (op) => generateStoryApproaches(projectId, note, op)),
    [runAI, projectId],
  );
  const writeStory = useCallback(
    (note: string) => runAI("story", note, (op) => generateStory(projectId, note, op)),
    [runAI, projectId],
  );
  const redoField = useCallback(
    (key: StoryFieldKey, note: string) => runAI({ field: key }, note, (op) => redoUnit(projectId, storyFieldUnit(key), note, op)),
    [runAI, projectId],
  );

  const selectApproach = useCallback(
    async (key: string, rewrite: boolean) => {
      const saved = await act(() => selectStoryApproach(projectId, key));
      return saved && rewrite ? writeStory("") : saved;
    },
    [act, projectId, writeStory],
  );

  const editField = useCallback(
    async (key: StoryFieldKey, text: string) => {
      setActionError(null);
      try {
        await editUnit(projectId, storyFieldUnit(key), { text });
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    [projectId, load],
  );

  return {
    state,
    loadError,
    running: running ?? (run.resumed ? "story" : null),
    run,
    busy,
    actionError,
    failedNote,
    generateApproaches,
    selectApproach,
    saveArchetype: (key) => act(() => saveApproachAsArchetype(projectId, key)),
    generateStory: writeStory,
    redoField,
    editField,
    setFactUse: (id, use) => act(() => setStoryFactUse(projectId, id, use)),
    approve: () => act(() => approveUnit(projectId, STORY_UNIT)),
    unapprove: () => act(() => unapproveUnit(projectId, STORY_UNIT)),
    reload: () => void load(),
  };
}
