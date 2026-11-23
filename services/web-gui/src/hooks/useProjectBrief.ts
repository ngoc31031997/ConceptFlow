import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveProjectBrief,
  editProjectBrief,
  getBriefScreen,
  unapproveProjectBrief,
  type BriefScreen,
  type ProjectBriefPatch,
} from "../api/client";

export interface ProjectBriefControls {
  /** null until the server answers (or while the project does not exist). */
  screen: BriefScreen | null;
  loadError: string | null;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** Saves a hand edit; throws so the edit dialog can show why it failed. */
  edit: (patch: ProjectBriefPatch) => Promise<void>;
  approve: () => Promise<void>;
  unapprove: () => Promise<void>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's brief; nothing loads until `enabled` (the
 * project exists on the server). Every change is followed by a reload of the
 * whole screen, since the review and the changed-idea flag depend on it.
 */
export function useProjectBrief(projectId: string, enabled: boolean): ProjectBriefControls {
  const [screen, setScreen] = useState<BriefScreen | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  // Only the newest load may land: a slow first load must not overwrite what a
  // run that finished after it has just stored.
  const loadSeq = useRef(0);

  const reload = useCallback(() => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    getBriefScreen(projectId)
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
    async (patch: ProjectBriefPatch) => {
      setActionError(null);
      try {
        await editProjectBrief(projectId, patch);
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
    () => changeApproval(() => approveProjectBrief(projectId)),
    [changeApproval, projectId],
  );
  const unapprove = useCallback(
    () => changeApproval(() => unapproveProjectBrief(projectId)),
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
