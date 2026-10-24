import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveStory,
  editStory,
  generateStory,
  getStory,
  isRequestCancelled,
  unapproveStory,
  type StoryPatch,
  type StoryState,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kind the server records for the AI run of the story step. */
const STORY_RUN_KINDS = ["story"];

/** The story step as the screen works with it. */
export interface StoryControls {
  /** null until the server answers. */
  state: StoryState | null;
  loadError: string | null;
  /** The AI is writing the story; a run found again after leaving the page counts. */
  writing: boolean;
  run: OperationRun;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** The note of the AI run that failed, so its dialog starts from it. */
  failedNote: string;
  generate: (note: string) => Promise<boolean>;
  /** Saves a hand edit; throws so the edit dialog can show why it failed. */
  edit: (patch: StoryPatch) => Promise<void>;
  approve: () => Promise<boolean>;
  unapprove: () => Promise<boolean>;
  /** Reads the state again; resolves when the new state has landed. */
  reload: () => Promise<void>;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's story step; nothing loads until `enabled`.
 * Every change is followed by a reload of the whole state, since a change
 * moves the approval and which facts the arc cites.
 */
export function useStory(projectId: string, enabled: boolean): StoryControls {
  const [state, setState] = useState<StoryState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [writing, setWriting] = useState(false);
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

  const generate = useCallback(
    async (note: string) => {
      setWriting(true);
      setActionError(null);
      try {
        await generateStory(projectId, note, run.begin());
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
        setWriting(false);
      }
    },
    [projectId, run, load],
  );

  const edit = useCallback(
    async (patch: StoryPatch) => {
      setActionError(null);
      try {
        await editStory(projectId, patch);
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    [projectId, load],
  );

  const changeApproval = useCallback(
    async (call: () => Promise<unknown>): Promise<boolean> => {
      setBusy(true);
      setActionError(null);
      try {
        await call();
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

  return {
    state,
    loadError,
    writing: writing || run.resumed,
    run,
    busy,
    actionError,
    failedNote,
    generate,
    edit,
    approve: () => changeApproval(() => approveStory(projectId)),
    unapprove: () => changeApproval(() => unapproveStory(projectId)),
    reload: load,
  };
}
