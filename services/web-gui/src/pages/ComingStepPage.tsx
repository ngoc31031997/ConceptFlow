import { Navigate, useNavigate, useParams } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Button } from "../components/ui";
import { useStepNav } from "../hooks/useStepNav";
import { FLOW_VIEW, comingRoute, isBuilt, previewRoute, screenRoute } from "../utils/flow";
import styles from "./ComingStepPage.module.css";

/**
 * The page of a step of the 21-step flow that the app does not have yet: it
 * says the step is not developed, what the step will be for (the "Bước này:"
 * line), and offers the step before it. A step the app does have opens its own
 * screen instead.
 *
 * Reached from the step menu, the flow map and the "next step" button of the
 * step before it.
 */
export function ComingStepPage() {
  const params = useParams<{ id?: string; n: string }>();
  const n = Number(params.n);
  const view = Number.isInteger(n) ? FLOW_VIEW.find((v) => v.n === n) : undefined;
  const navigate = useNavigate();
  const nav = useStepNav(undefined, { webScreen: true });
  const projectId = params.id ?? "";

  if (!view) return <Navigate to="/" replace />;
  if (isBuilt(view)) {
    if (view.screen) return <Navigate to={projectId ? screenRoute(view, projectId) : "/"} replace />;
    return <Navigate to={previewRoute(view.server[0], projectId)} replace />;
  }

  const before = FLOW_VIEW.find((v) => v.n === n - 1);
  const goBefore = () => {
    if (!before) return;
    if (before.screen) {
      if (projectId) navigate(screenRoute(before, projectId));
    } else if (!isBuilt(before)) navigate(comingRoute(before.n, projectId));
    else nav.open(before.server[0]);
  };

  return (
    <div data-testid="coming-step-page">
      <AppShell viewStep={n} title={view.title ?? view.label} subtitle="" role={view.note}>
        <p className={styles.notice} role="status" data-testid="coming-step-notice">
          <b>Bước này chưa phát triển.</b> Màn của bước này sẽ có ở một bản sau.
        </p>
        {before && (
          <div className={styles.actions}>
            <Button variant="ghost" onClick={goBefore} data-testid="coming-step-back">
              ← Bước trước: {before.label}
            </Button>
          </div>
        )}
      </AppShell>
    </div>
  );
}
