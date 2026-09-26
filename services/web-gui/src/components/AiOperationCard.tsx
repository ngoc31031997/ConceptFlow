import type { OperationRun } from "../hooks/useOperationRun";
import { operationErrorLabel, operationSubtitle } from "../lib/formatProgress";
import { OperationProgressCard } from "./OperationProgressCard";

interface AiOperationCardProps {
  run: OperationRun;
  /** The request is still open. */
  active: boolean;
  testId: string;
  title?: string;
}

/**
 * OperationProgressCard wired to a useOperationRun (CR-040 FR116.2): live while
 * the request is open, and — after a failure — kept on screen with the chars and
 * time already spent plus the classified error (FR116.5).
 */
export function AiOperationCard({ run, active, testId, title }: AiOperationCardProps) {
  const failed = run.progress?.status === "failed";
  if (!active && !failed) return null;
  return (
    <OperationProgressCard
      testId={testId}
      title={title}
      subtitle={operationSubtitle(run.progress)}
      done={run.progress?.done}
      total={run.progress?.total}
      error={failed ? operationErrorLabel(run.progress?.error ?? "server") : null}
    />
  );
}
