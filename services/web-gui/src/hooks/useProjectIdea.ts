import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveProjectIdea,
  editProjectIdea,
  getIdeaScreen,
  unapproveProjectIdea,
  type IdeaScreen,
  type ProjectIdeaPatch,
} from "../api/client";

export interface ProjectIdeaControls {
  /** null until the server answers (or while the project does not exist). */
  screen: IdeaScreen | null;
  loadError: string | null;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** Saves a hand edit; throws so the edit dialog can show why it failed. */
  edit: (patch: ProjectIdeaPatch) => Promise<void>;
  approve: () => Promise<void>;
  unapprove: () => Promise<void>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's idea; nothing loads until `enabled` (the
 * project exists on the server). Every change is followed by a reload of the
 * whole screen, since the review and the changed-input flag depend on it.
 */
export function useProjectIdea(projectId: string, enabled: boolean): ProjectIdeaControls {
  const [screen, setScreen] = useState<IdeaScreen | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  // Only the newest load may land: a slow first load must not overwrite what a
  // run that finished after it has just stored.
  const loadSeq = useRef(0);

  // Not gated on `enabled`: a run started in the same click that created the
  // project still holds the closure from before, and its reload must load.
  const reload = useCallback(() => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    getIdeaScreen(projectId)
      .then((next) => {
        if (seq !== loadSeq.current) return;
        setScreen(next);
        setLoadError(null);
      })
      .catch((err) => {
        if (seq !== loadSeq.current) return;
        setLoadError(messageOf(err));
      });
  }, [projectId]);

  useEffect(() => {
    setScreen(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) reload();
  }, [enabled, reload]);

  const edit = useCallback(
    async (patch: ProjectIdeaPatch) => {
      setActionError(null);
      try {
        await editProjectIdea(projectId, patch);
        reload();
      } catch (err) {
        throw new Error(messageOf(err));
      }
    },
    [projectId, reload],
  );

  const changeApproval = useCallback(
    async (call: () => Promise<unknown>) => {
      setBusy(true);
      setActionError(null);
      try {
        await call();
        reload();
      } catch (err) {
        setActionError(messageOf(err));
      } finally {
        setBusy(false);
      }
    },
    [reload],
  );
  const approve = useCallback(
    () => changeApproval(() => approveProjectIdea(projectId)),
    [changeApproval, projectId],
  );
  const unapprove = useCallback(
    () => changeApproval(() => unapproveProjectIdea(projectId)),
    [changeApproval, projectId],
  );

  return {
    screen,
    loadError,
    actionError,
    busy,
    edit,
    approve,
    unapprove,
    reload,
  };
}
