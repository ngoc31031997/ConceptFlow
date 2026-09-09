import { useContext, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { validateScript } from "../utils/scriptValidation";

/**
 * Guards the later wizard steps. Landing on step 2 or 3 directly — a stale
 * bookmark, a reload after the draft was cleared — would otherwise show
 * settings for a script that does not exist, and submit an empty render.
 *
 * Returns false while redirecting so the caller can render nothing.
 */
export function useRequireScript(): boolean {
  const draft = useContext(ProjectDraftContext);
  const navigate = useNavigate();
  const isReady = draft.scriptContent.trim().length > 0 && validateScript(draft.scriptContent).isValid;

  useEffect(() => {
    if (!isReady) navigate("/", { replace: true });
  }, [isReady, navigate]);

  return isReady;
}
