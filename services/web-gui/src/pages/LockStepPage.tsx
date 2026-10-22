import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { ScriptLockState } from "../api/client";
import { AppShell } from "../components/AppShell";
import { LockHandoffTab } from "../components/lock/LockHandoffTab";
import { LockPackageTab } from "../components/lock/LockPackageTab";
import { LockReviewTab } from "../components/lock/LockReviewTab";
import lockStyles from "../components/lock/lock.module.css";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { Button } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useScriptLock } from "../hooks/useScriptLock";
import { VIEW_LOCK, comingRoute, readOnlyReason, scriptRoute } from "../utils/flow";

const SUBTITLE =
  "Máy và AI rà soát kịch bản; bạn xử lý từng mục. Viết ghi chú bàn giao cho các vai AI phía sau rồi chốt một phiên bản.";

const ROLE =
  "Rà soát nội dung, logic, thời lượng và khả năng sản xuất; viết ghi chú bàn giao cho các vai AI phía sau; chốt một phiên bản kịch bản.";

type Tab = "review" | "handoff" | "package";

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(s: ScriptLockState, running: boolean): StepStripProps {
  if (!s.scriptReady) {
    return { tone: "todo", label: "Chưa mở bước", message: "Duyệt lời thoại của mọi beat ở bước Viết kịch bản trước." };
  }
  if (running) return { tone: "run", label: "AI đang làm", message: "Ghi chú và lời thoại bạn đã viết giữ nguyên." };
  if (s.lockedVersion === 0) {
    return {
      tone: "wait",
      label: `Chưa chốt · ${s.openCount} mục rà soát cần xử lý`,
      message: "Chốt khi rà soát xong, lời thoại và ghi chú bàn giao đã duyệt.",
    };
  }
  if (s.draft.length > 0) {
    return {
      tone: "wait",
      label: `v${s.lockedVersion} đã chốt · bản nháp v${s.nextVersion} có ${s.draft.length} thay đổi`,
      message: `Các bước sau vẫn dùng v${s.lockedVersion} tới khi bạn chốt bản mới.`,
    };
  }
  return { tone: "ok", label: `Đã chốt v${s.lockedVersion}`, message: "Tiếp theo: Mỹ thuật lượt 1 đọc bộ bàn giao này." };
}

/**
 * Step 7 of the 21-step flow: reviewing and locking the script. Three tabs: the
 * review of the whole script (machine checks and the AI), the handoff note of
 * each beat with the shared glossary, and the package with the version history.
 * Locking freezes a version the steps after the script read.
 */
export function LockStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const projectId = params.id ?? draft.projectId;
  const lock = useScriptLock(projectId, !!projectId);
  const [tab, setTab] = useState<Tab>("review");

  if (!projectId) return <Navigate to="/" replace />;

  const state = lock.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;
  const handoffDone = state ? state.beats.filter((b) => b.written && b.status === "approved" && !b.stale).length : 0;

  const tabs: { key: Tab; label: string; count: string }[] = [
    { key: "review", label: "Rà soát", count: state ? (state.openCount > 0 ? `${state.openCount} cần xử lý` : "xong") : "" },
    { key: "handoff", label: "Ghi chú bàn giao", count: state ? `${handoffDone}/${state.beats.length}` : "" },
    { key: "package", label: "Bộ bàn giao và phiên bản", count: state ? (state.lockedVersion > 0 ? `v${state.lockedVersion}` : "nháp") : "" },
  ];

  return (
    <div data-testid="lock-step-page">
      <AppShell viewStep={VIEW_LOCK} wide title="Rà soát và chốt kịch bản" subtitle={SUBTITLE} role={ROLE} strip={state ? stripOf(state, lock.running !== null) : undefined}>
        {locked && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="lock-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {!state && lock.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="lock-load-error">
              Không tải được bước Rà soát và chốt: {lock.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={lock.reload} data-testid="lock-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !lock.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && !state.scriptReady && (
          <p className={researchStyles.hint} data-testid="lock-not-open">
            Duyệt lời thoại của mọi beat ở bước Viết kịch bản trước.
          </p>
        )}
        {state && state.scriptReady && (
          <div className={researchStyles.body}>
            <div className={lockStyles.tabs} role="tablist">
              {tabs.map((t) => (
                <button
                  key={t.key}
                  type="button"
                  role="tab"
                  className={lockStyles.tab}
                  aria-selected={tab === t.key}
                  onClick={() => setTab(t.key)}
                  data-testid={`lock-tab-${t.key}`}
                >
                  {t.label}
                  <span className={lockStyles.tabCount}>{t.count}</span>
                </button>
              ))}
            </div>
            {tab === "review" && <LockReviewTab projectId={projectId} lock={lock} aiAvailable={aiAvailable} locked={locked} />}
            {tab === "handoff" && <LockHandoffTab projectId={projectId} lock={lock} aiAvailable={aiAvailable} locked={locked} />}
            {tab === "package" && (
              <LockPackageTab lock={lock} locked={locked} onOpenHandoff={() => setTab("handoff")} onOpenScript={() => navigate(scriptRoute(projectId))} />
            )}
            {lock.actionError && (
              <p className={researchStyles.error} role="alert" data-testid="lock-error">
                {lock.actionError}
              </p>
            )}
          </div>
        )}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={state && state.lockedVersion > 0 ? `Đã chốt v${state.lockedVersion}. Sang Mỹ thuật lượt 1.` : "Chốt một phiên bản để sang Mỹ thuật lượt 1."}
        onBack={() => navigate(scriptRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_LOCK + 1, projectId))}
        nextDisabled={!state || state.lockedVersion === 0}
        nextLabel="Sang Mỹ thuật lượt 1"
        nextTestId="lock-next"
      />
    </div>
  );
}
