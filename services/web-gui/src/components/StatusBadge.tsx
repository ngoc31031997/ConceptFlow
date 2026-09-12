import { statusLabel } from "../utils/pipelineLabels";
import glass from "../styles/glass.module.css";

interface StatusBadgeProps {
  status: string;
}

/**
 * The one status-color vocabulary in the app (UX review #3) — VideoListPage
 * used to be the only page with this badge; RenderPage/ResultPage each
 * re-invented their own status visual (spinner text, colored card border)
 * instead of reusing it, so the same underlying status ("failed",
 * "processing", "done") looked different depending which page you were on.
 */
function badgeClassFor(status: string): string {
  if (status.startsWith("failed_at_")) return glass.badgeFailed;
  if (status === "published" || status === "ready_to_publish") return glass.badgeSuccess;
  if (status === "draft") return glass.badgeNeutral;
  return glass.badgeProgress;
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span className={`${glass.badge} ${badgeClassFor(status)}`} data-testid="status-badge">
      <span className={glass.badgeDot} aria-hidden="true" />
      {statusLabel(status)}
    </span>
  );
}
