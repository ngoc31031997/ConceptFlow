import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { VisualConceptState, VisualConceptUnitKey } from "../api/client";
import { AppShell } from "../components/AppShell";
import lockStyles from "../components/lock/lock.module.css";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { Button } from "../components/ui";
import { ConceptReviewTab } from "../components/visual/ConceptReviewTab";
import { ConceptUnitCard } from "../components/visual/ConceptUnitCard";
import { ReferencePicker } from "../components/visual/ReferencePicker";
import { VisualHeadCard } from "../components/visual/VisualHeadCard";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useVisualConcept } from "../hooks/useVisualConcept";
import { VIEW_VISUAL_CONCEPT, lockRoute, readOnlyReason, visualPlanRoute } from "../utils/flow";

const SUBTITLE =
  "Đặt quy tắc hình chung cho cả video trước khi quyết định cách minh hoạ từng đoạn.";

const ROLE =
  "Video sẽ trông như thế nào, và người xem tiếp nhận thông tin qua ngôn ngữ hình ảnh nào? Không chọn shot, không làm storyboard, không hoàn thiện tài nguyên.";

type Tab = "rules" | "review";

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(s: VisualConceptState, running: boolean): StepStripProps {
  const total = s.units.length;
  if (running)
    return {
      tone: "run",
      label: "AI đang làm",
      message: "Phần đã duyệt và trường bạn sửa tay giữ nguyên.",
    };
  if (s.locked_version > 0 && s.draft.length === 0) {
    return {
      tone: "ok",
      label: `Đã chốt v${s.locked_version}`,
      message: "Tiếp theo: kế hoạch hình cho từng beat.",
    };
  }
  if (s.locked_version > 0) {
    return {
      tone: "wait",
      label: `v${s.locked_version} đã chốt · bản nháp v${s.next_version} có ${s.draft.length} thay đổi`,
      message: `Các bước sau vẫn dùng v${s.locked_version} tới khi bạn chốt bản mới.`,
    };
  }
  if (s.approved_count < total) {
    return {
      tone: "wait",
      label: `Chờ bạn duyệt · ${s.approved_count}/${total}`,
      message: "Phần đã duyệt giữ nguyên khi phần khác làm lại.",
    };
  }
  return {
    tone: "wait",
    label: `Đã duyệt ${total}/${total}`,
    message: "Xử lý các vấn đề ở tab Rà soát rồi chốt.",
  };
}

/**
 * Step 8 of the 23-step flow: the visual concept. The AI writes eight parts
 * (direction, colour, shape and light, how things are drawn, text, motion, what
 * not to use, feasibility) from the locked script and the reference
 * descriptions the Creator picked; the Creator edits, approves, has the whole
 * set reviewed and locks a version the later visual steps read.
 */
export function VisualConceptStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const projectId = params.id ?? draft.projectId;
  const concept = useVisualConcept(projectId);
  const [tab, setTab] = useState<Tab>("rules");
  const [opened, setOpened] = useState<VisualConceptUnitKey | null>(null);

  if (!projectId) return <Navigate to="/" replace />;

  const state = concept.state;
  const readOnly = !flow.editable;

  function openUnit(key: VisualConceptUnitKey) {
    setOpened(key);
    setTab("rules");
    window.setTimeout(
      () =>
        document
          .getElementById(`concept-unit-${key}`)
          ?.scrollIntoView({ behavior: "smooth", block: "start" }),
      50,
    );
  }

  return (
    <div data-testid="visual-concept-page">
      <AppShell
        viewStep={VIEW_VISUAL_CONCEPT}
        wide
        title="Ngôn ngữ hình ảnh"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, concept.running !== null) : undefined}
      >
        {readOnly && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="concept-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {concept.notOpen && (
          <>
            <p className={researchStyles.hint} data-testid="concept-not-open">
              Chốt kịch bản ở bước 7 trước.
            </p>
            <div className={researchStyles.actions}>
              <Button
                variant="ghost"
                onClick={() => navigate(lockRoute(projectId))}
                data-testid="concept-go-lock"
              >
                ← Rà soát và chốt
              </Button>
            </div>
          </>
        )}
        {!state && !concept.notOpen && concept.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="concept-load-error">
              Không tải được bước Ngôn ngữ hình ảnh: {concept.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button
                variant="ghost"
                onClick={() => void concept.reload()}
                data-testid="concept-reload"
              >
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !concept.notOpen && !concept.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && !concept.notOpen && (
          <div className={researchStyles.body}>
            <VisualHeadCard
              projectId={projectId}
              state={state}
              concept={concept}
              readOnly={readOnly}
            />
            <div className={lockStyles.tabs} role="tablist">
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "rules"}
                onClick={() => setTab("rules")}
                data-testid="concept-tab-rules"
              >
                Quy tắc hình ảnh
                <span className={lockStyles.tabCount}>
                  {state.approved_count}/{state.units.length}
                </span>
              </button>
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "review"}
                onClick={() => setTab("review")}
                data-testid="concept-tab-review"
              >
                Rà soát và chốt
                <span className={lockStyles.tabCount}>{state.review.open_count} vấn đề</span>
              </button>
            </div>
            {tab === "rules" && (
              <>
                <ReferencePicker state={state} concept={concept} readOnly={readOnly} />
                {state.units.map((u) => (
                  <ConceptUnitCard
                    key={u.key}
                    projectId={projectId}
                    unit={u}
                    concept={concept}
                    readOnly={readOnly}
                    forceOpen={opened === u.key}
                    onFold={() => setOpened(null)}
                  />
                ))}
              </>
            )}
            {tab === "review" && (
              <ConceptReviewTab
                projectId={projectId}
                state={state}
                concept={concept}
                readOnly={readOnly}
                onOpenUnit={openUnit}
              />
            )}
            {concept.actionError && (
              <p className={researchStyles.error} role="alert" data-testid="concept-error">
                {concept.actionError}
              </p>
            )}
          </div>
        )}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          state && state.locked_version > 0
            ? `Đã chốt v${state.locked_version}. Sang Kế hoạch hình.`
            : "Chốt ngôn ngữ hình ảnh để sang Kế hoạch hình."
        }
        onBack={() => navigate(lockRoute(projectId))}
        onNext={() => navigate(visualPlanRoute(projectId))}
        nextDisabled={!state || state.locked_version === 0}
        nextLabel="Sang Kế hoạch hình"
        nextTestId="concept-next"
      />
    </div>
  );
}
