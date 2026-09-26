import { useContext, useEffect } from "react";
import { createProjectDraft, getAuthoringState, saveAuthoringMode, type AuthoringMode } from "../api/client";
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
// A save can still be in flight (or start) while the next tab mounts and
// immediately re-fetches (the Creator toggled the mode on step 1, then hit
// "Tiếp tục" before the PUT settled). A presence-check on an in-flight-save
// map is not enough: the PUT's promise can settle (removing itself from the
// map) before the racing GET — which read stale data — resolves, so the
// stale value still wins. A monotonic per-project write counter fixes this:
// the read only applies if no write happened between when it started and
// when it resolved, regardless of which network request finishes first.
const writeVersion = new Map<string, number>();

export function useAuthoringMode(projectId: string): {
  mode: AuthoringMode;
  setMode: (mode: AuthoringMode) => void;
} {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    const versionAtStart = writeVersion.get(projectId) ?? 0;
    getAuthoringState(projectId)
      .then((state) => {
        // A server that does not know this field yet sends nothing; keep the
        // draft's value rather than resetting the Creator to manual. Same if
        // a setMode happened for this project since this read started — this
        // read may have been resolved by a stale value that predates it.
        if (!cancelled && state.mode && (writeVersion.get(projectId) ?? 0) === versionAtStart) {
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
      writeVersion.set(projectId, (writeVersion.get(projectId) ?? 0) + 1);
      // The mode picker is reachable before the project row exists server-side
      // (step 1, before a topic/engine ever triggered createProjectDraft), and
      // project_authoring has an FK to projects — saving the mode first would
      // fail against a row that isn't there yet, and that failure used to be
      // swallowed, silently reverting the Creator's choice on next load.
      // Ensuring the row exists first (idempotent upsert, same call used on
      // engine change) makes the save land instead of quietly disappearing.
      void createProjectDraft(projectId, "", draft.voiceLanguage, draft.renderEngine)
        .catch(() => {
          /* best-effort — the mode PUT below still tries even if this raced
             with another creator of the same row */
        })
        .then(() => saveAuthoringMode(projectId, mode))
        .catch((err) => {
          // eslint-disable-next-line no-console
          console.error("Không lưu được cách làm đã chọn lên máy chủ:", err);
        });
    },
  };
}
