import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { VisualPlanState } from "../api/client";
import { AppShell } from "../components/AppShell";
import lockStyles from "../components/lock/lock.module.css";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { Button } from "../components/ui";
import { PlanBeatsCard } from "../components/visualplan/PlanBeatsCard";
import { PlanHeadCard } from "../components/visualplan/PlanHeadCard";
import { PlanReviewTab } from "../components/visualplan/PlanReviewTab";
import { beatOfLine } from "../components/visualplan/planModel";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useVisualPlan } from "../hooks/useVisualPlan";
import { VIEW_VISUAL_PLAN, comingRoute, lockRoute, readOnlyReason, visualConceptRoute } from "../utils/flow";

const SUBTITLE = "Chia kịch bản thành các đơn vị hình, mỗi đơn vị một mục tiêu truyền đạt, cách hiển thị và thời lượng.";

const ROLE =
  "Mỗi đoạn lời thoại cần cho người xem nhìn thấy gì, và hình giữ bao lâu? Không chọn shot, không làm storyboard, không vẽ hình.";

type Tab = "plan" | "review";

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(s: VisualPlanState, running: boolean): StepStripProps {
  if (running) {
    return { tone: "run", label: "AI đang làm", message: "Unit đã duyệt và trường bạn sửa tay giữ nguyên." };
  }
  if (s.locked_version > 0 && s.draft.length === 0) {
    return { tone: "ok", label: `Đã chốt v${s.locked_version}`, message: "Tiếp theo: diễn tiến hình ảnh." };
  }
  if (s.locked_version > 0) {
    return {
      tone: "wait",
      label: `v${s.locked_version} đã chốt · bản nháp v${s.next_version} có ${s.draft.length} thay đổi`,
      message: `Các bước sau vẫn dùng v${s.locked_version} tới khi bạn chốt bản mới.`,
    };
  }
  if (s.units_total === 0) {
    return { tone: "wait", label: "Chưa có kế hoạch", message: "Cho AI lập kế hoạch cho các beat." };
  }
  if (s.units_approved < s.units_total) {
    return {
      tone: "wait",
      label: `Chờ bạn duyệt · ${s.units_approved}/${s.units_total}`,
      message: "Unit đã duyệt giữ nguyên khi unit khác làm lại.",
    };
  }
  return {
    tone: "wait",
    label: `Đã duyệt ${s.units_total}/${s.units_total}`,
    message: "Xử lý các vấn đề ở tab Rà soát và chốt rồi chốt.",
  };
}

/**
 * Step 9 of the 23-step flow: the visual plan. The AI turns each beat's lines
 * into visual units (what the viewer must take away, how it is shown, how long
 * it is held); the Creator edits, splits, merges, answers the open questions,
 * approves, has the whole plan reviewed and locks a version the later visual
 * steps read.
 */
export function VisualPlanStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const projectId = params.id ?? draft.projectId;
  const plan = useVisualPlan(projectId);
  const [tab, setTab] = useState<Tab>("plan");

  if (!projectId) return <Navigate to="/" replace />;

  const state = plan.state;
  const readOnly = !flow.editable;

  /** Shows the plan tab and scrolls to `id` once it has rendered. */
  function scrollToOnPlanTab(id: string) {
    setTab("plan");
    window.setTimeout(() => document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" }), 50);
  }

  return (
    <div data-testid="visual-plan-page">
      <AppShell
        viewStep={VIEW_VISUAL_PLAN}
        wide
        title="Kế hoạch hình"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, plan.running !== null) : undefined}
      >
        {readOnly && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="plan-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {plan.notOpen === "script" && (
          <>
            <p className={researchStyles.hint} data-testid="plan-not-open">
              Chốt kịch bản ở bước 7 trước.
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={() => navigate(lockRoute(projectId))} data-testid="plan-go-lock">
                ← Rà soát và chốt
              </Button>
            </div>
          </>
        )}
        {plan.notOpen === "concept" && (
          <>
            <p className={researchStyles.hint} data-testid="plan-not-open">
              Chốt Ngôn ngữ hình ảnh ở bước 8 trước.
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={() => navigate(visualConceptRoute(projectId))} data-testid="plan-go-concept">
                ← Ngôn ngữ hình ảnh
              </Button>
            </div>
          </>
        )}
        {!state && !plan.notOpen && plan.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="plan-load-error">
              Không tải được bước Kế hoạch hình: {plan.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={() => void plan.reload()} data-testid="plan-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !plan.notOpen && !plan.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && !plan.notOpen && (
          <div className={researchStyles.body}>
            <PlanHeadCard projectId={projectId} state={state} plan={plan} readOnly={readOnly} />
            <div className={lockStyles.tabs} role="tablist">
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "plan"}
                onClick={() => setTab("plan")}
                data-testid="plan-tab-plan"
              >
                Kế hoạch hình
                <span className={lockStyles.tabCount}>
                  {state.units_approved}/{state.units_total}
                </span>
              </button>
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "review"}
                onClick={() => setTab("review")}
                data-testid="plan-tab-review"
              >
                Rà soát và chốt
                <span className={lockStyles.tabCount}>{state.review.open_count} vấn đề</span>
              </button>
            </div>
            {tab === "plan" && <PlanBeatsCard projectId={projectId} state={state} plan={plan} readOnly={readOnly} />}
            {tab === "review" && (
              <PlanReviewTab
                projectId={projectId}
                state={state}
                plan={plan}
                readOnly={readOnly}
                onOpenUnit={(unitId) => scrollToOnPlanTab(`plan-unit-${unitId}`)}
                onGoUncovered={(lineId) => scrollToOnPlanTab(`plan-beat-${beatOfLine(lineId)}`)}
              />
            )}
            {plan.actionError && (
              <p className={researchStyles.error} role="alert" data-testid="plan-error">
                {plan.actionError}
              </p>
            )}
          </div>
        )}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          state && state.locked_version > 0
            ? `Đã chốt v${state.locked_version}. Sang Diễn tiến hình ảnh.`
            : "Chốt kế hoạch hình để sang Diễn tiến hình ảnh."
        }
        onBack={() => navigate(visualConceptRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_VISUAL_PLAN + 1, projectId))}
        nextDisabled={!state || state.locked_version === 0}
        nextLabel="Sang Diễn tiến hình ảnh"
        nextTestId="plan-next"
      />
    </div>
  );
}
