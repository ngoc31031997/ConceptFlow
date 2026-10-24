import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  approveProjectBrief,
  editProjectBrief,
  generateProjectBrief,
  getBriefScreen,
  isRequestCancelled,
  reviewProjectBrief,
  unapproveProjectBrief,
  type BriefScreen,
  type ProjectBriefPatch,
} from "../api/client";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kinds the server records for the AI runs of the brief step. */
const WRITE_KIND = "brief";
const REVIEW_KIND = "brief_review";

export interface ProjectBriefControls {
  /** null until the server answers (or while the project does not exist). */
  screen: BriefScreen | null;
  loadError: string | null;
  /** The AI is writing the brief. */
  writing: boolean;
  writeRun: OperationRun;
  /** The AI is reading the brief for its review. */
  reviewing: boolean;
  reviewRun: OperationRun;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  /** An approve, unapprove or hand edit is in flight. */
  busy: boolean;
  /** The note of the redo that failed, so the redo dialog starts from it. */
  failedNote: string;
  /** Has the AI write the brief, then review it; resolves false when the writing failed. */
  generate: (note?: string) => Promise<boolean>;
  /** Has the AI read the brief again. */
  review: () => Promise<boolean>;
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
  const [writing, setWriting] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [failedNote, setFailedNote] = useState("");
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

  const writeRun = useOperationRun({
    projectId: enabled ? projectId : undefined,
    kinds: [WRITE_KIND],
    onSettled: reload,
  });
  const reviewRun = useOperationRun({
    projectId: enabled ? projectId : undefined,
    kinds: [REVIEW_KIND],
    onSettled: reload,
  });

  useEffect(() => {
    setScreen(null);
    setLoadError(null);
    setActionError(null);
    if (enabled) reload();
  }, [enabled, reload]);

  const review = useCallback(async () => {
    setReviewing(true);
    setActionError(null);
    try {
      await reviewProjectBrief(projectId, reviewRun.begin());
      reviewRun.end();
      reload();
      return true;
    } catch (err) {
      if (!isRequestCancelled(err)) setActionError(messageOf(err));
      return false;
    } finally {
      setReviewing(false);
    }
  }, [projectId, reviewRun, reload]);

  const generate = useCallback(
    async (note = "") => {
      setWriting(true);
      setActionError(null);
      try {
        await generateProjectBrief(projectId, note, writeRun.begin());
        setFailedNote("");
        writeRun.end();
        reload();
      } catch (err) {
        if (!isRequestCancelled(err)) {
          setActionError(messageOf(err));
          setFailedNote(note);
        }
        // The card stays with the failure and the time already spent.
        return false;
      } finally {
        setWriting(false);
      }
      // The review is advice for the Creator's decision; a failed review leaves the brief as written.
      await review();
      return true;
    },
    [projectId, writeRun, reload, review],
  );

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
    writing: writing || writeRun.resumed,
    writeRun,
    reviewing: reviewing || reviewRun.resumed,
    reviewRun,
    actionError,
    busy,
    failedNote,
    generate,
    review,
    edit,
    approve,
    unapprove,
    reload,
  };
}
