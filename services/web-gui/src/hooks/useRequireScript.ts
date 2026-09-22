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
 *
 * feature/remotion-engine: validateScript only understands Manim's
 * self.narrate/ConceptFlowScene conventions — calling it unconditionally
 * meant a project with a perfectly valid Remotion script would fail this
 * check and get silently bounced back to "/" every time Settings/Review
 * mounted (e.g. right after tab 1c navigated here). Remotion
 * has no client-side lint yet (same as ScriptEditor/ManimEngineerStepPage);
 * non-empty is enough here, same bar the "ready" source already trusts.
 */
export function useRequireScript(): boolean {
  const draft = useContext(ProjectDraftContext);
  const navigate = useNavigate();
  const hasContent = draft.scriptContent.trim().length > 0;
  const isReady =
    hasContent && (draft.renderEngine === "remotion" || validateScript(draft.scriptContent).isValid);

  useEffect(() => {
    if (!isReady) navigate("/", { replace: true });
  }, [isReady, navigate]);

  return isReady;
}
