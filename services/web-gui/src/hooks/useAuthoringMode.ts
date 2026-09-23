import { useContext, useEffect } from "react";
import { getAuthoringState, saveAuthoringMode, type AuthoringMode } from "../api/client";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";

/**
 * CR-027 FR79 — the step-1 working mode, read from and written to the project
 * rather than only the browser.
 *
 * All four tabs of step 1 use this, so all four agree and none of them has to
 * know how the value is stored. On mount it takes whatever the server has
 * (which is what makes the choice survive a reload, another browser, or a
 * restart of the stack, whichever tab the project is sitting on); setMode
 * updates the draft immediately and writes to the server behind it.
 *
 * Both directions are best-effort by design. A failed read leaves the draft's
 * value in place, and a failed write leaves the UI on the mode the Creator just
 * picked: this is a preference about how to work, and refusing to switch
 * because a bookkeeping request did not land would block the actual work over
 * nothing. The mode is re-sent on the next toggle anyway.
 */
// One save can still be in flight when the next tab mounts and immediately
// re-fetches (the Creator toggled the mode on step 1, then hit "Tiếp tục"
// before the PUT settled). Tracking it here — module scope, keyed by
// project — lets that tab's read notice the race and trust the draft's own
// value instead of a GET that can resolve before the PUT it raced against
// has committed.
const pendingSaves = new Map<string, Promise<unknown>>();

export function useAuthoringMode(projectId: string): {
  mode: AuthoringMode;
  setMode: (mode: AuthoringMode) => void;
} {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    getAuthoringState(projectId)
      .then((state) => {
        // A server that does not know this field yet sends nothing; keep the
        // draft's value rather than resetting the Creator to manual. Same if
        // a save for this project is still in flight — this read may have
        // been resolved by a stale value that predates it.
        if (!cancelled && state.mode && !pendingSaves.has(projectId)) {
          dispatch({ type: "SET_AUTHORING_MODE", payload: state.mode });
        }
      })
      .catch(() => {
        /* best-effort — the draft's own value still drives the UI */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  return {
    mode: draft.authoringMode,
    setMode: (mode) => {
      dispatch({ type: "SET_AUTHORING_MODE", payload: mode });
      if (!projectId) return;
      const save = saveAuthoringMode(projectId, mode).catch(() => {
        /* best-effort — see the docstring */
      });
      pendingSaves.set(projectId, save);
      void save.finally(() => {
        if (pendingSaves.get(projectId) === save) pendingSaves.delete(projectId);
      });
    },
  };
}
