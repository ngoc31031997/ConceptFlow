import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { VisualSequenceState } from "../api/client";
import { AppShell } from "../components/AppShell";
import lockStyles from "../components/lock/lock.module.css";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { Button } from "../components/ui";
import { SequenceBeatsCard } from "../components/visualsequence/SequenceBeatsCard";
import { SequenceFlowStrip } from "../components/visualsequence/SequenceFlowStrip";
import { SequenceHeadCard } from "../components/visualsequence/SequenceHeadCard";
import { SequenceMotifsCard } from "../components/visualsequence/SequenceMotifsCard";
import { SequenceReviewTab } from "../components/visualsequence/SequenceReviewTab";
import { SequenceRunCard } from "../components/visualsequence/SequenceRunCard";
import { SequenceRunningBanner } from "../components/visualsequence/SequenceRunningBanner";
import { motifsReady } from "../components/visualsequence/sequenceModel";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useVisualSequence, type SequenceNotOpen } from "../hooks/useVisualSequence";
import {
  VIEW_VISUAL_SEQUENCE,
  comingRoute,
  lockRoute,
  readOnlyReason,
  visualConceptRoute,
  visualPlanRoute,
} from "../utils/flow";

const SUBTITLE =
  "Với mỗi đơn vị hình của kế hoạch: vai trò trong mạch, nhịp, chuyển cảnh từ đơn vị trước và motif.";

const ROLE =
  "Các đơn vị hình nối với nhau thế nào để người xem theo kịp mạch? Không đổi thứ tự đơn vị của bước 9, không chọn shot, không dựng chuyển động chi tiết.";

type Tab = "sequence" | "review";

/** What each closed gate says and where it leads. */
const GATES: Record<
  Exclude<SequenceNotOpen, null>,
  { text: string; back: string; route: (id: string) => string }
> = {
  script: { text: "Chốt kịch bản ở bước 7 trước.", back: "← Rà soát và chốt", route: lockRoute },
  concept: {
    text: "Chốt Ngôn ngữ hình ảnh ở bước 8 trước.",
    back: "← Ngôn ngữ hình ảnh",
    route: visualConceptRoute,
  },
  plan: {
    text: "Chốt Kế hoạch hình ở bước 9 trước.",
    back: "← Kế hoạch hình",
    route: visualPlanRoute,
  },
};

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(s: VisualSequenceState, running: boolean): StepStripProps {
  const total = s.coverage.units_total;
  if (running) {
    return {
      tone: "run",
      label: "AI đang làm",
      message: "Unit đã duyệt và trường bạn sửa tay giữ nguyên.",
    };
  }
  if (s.locked_version > 0 && s.draft.length === 0) {
    return { tone: "ok", label: `Đã chốt v${s.locked_version}`, message: "Tiếp theo: storyboard." };
  }
  if (s.locked_version > 0) {
    return {
      tone: "wait",
      label: `v${s.locked_version} đã chốt · bản nháp v${s.next_version} có ${s.draft.length} thay đổi`,
      message: `Các bước sau vẫn dùng v${s.locked_version} tới khi bạn chốt bản mới.`,
    };
  }
  if (!motifsReady(s)) {
    return {
      tone: "wait",
      label: "Chờ motif",
      message: "Đặt motif và duyệt trước khi AI viết chuỗi.",
    };
  }
  if (s.coverage.units_sequenced < total) {
    return {
      tone: "wait",
      label: `Đã có chuỗi ${s.coverage.units_sequenced}/${total}`,
      message: "Cho AI viết chuỗi các beat còn trống.",
    };
  }
  if (s.items_approved < total) {
    return {
      tone: "wait",
      label: `Chờ bạn duyệt · ${s.items_approved}/${total}`,
      message: "Unit đã duyệt giữ nguyên khi beat khác làm lại.",
    };
  }
  return {
    tone: "wait",
    label: `Đã duyệt ${total}/${total}`,
    message: "Xử lý các vấn đề ở tab Rà soát và chốt rồi chốt.",
  };
}

/**
 * Step 10 of the 23-step flow: the visual sequence. Keeping the order of the
 * locked visual plan, it gives every unit its purpose in the flow, its pacing,
 * the transition into it and the motifs it carries. The Creator sets and
 * approves the motifs, has the AI write the beats, edits, approves, has the
 * whole sequence reviewed and locks a version the storyboard reads.
 */
export function VisualSequenceStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const projectId = params.id ?? draft.projectId;
  const sequence = useVisualSequence(projectId);
  const [tab, setTab] = useState<Tab>("sequence");

  if (!projectId) return <Navigate to="/" replace />;

  const state = sequence.state;
  const readOnly = !flow.editable;
  const gate = sequence.notOpen ? GATES[sequence.notOpen] : null;

  /** Shows the sequence tab and scrolls to `id` once it has rendered. */
  function scrollTo(id: string) {
    setTab("sequence");
    window.setTimeout(
      () => document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" }),
      50,
    );
  }

  /** Opens a unit, or the motif card for a motif. */
  function open(key: string) {
    scrollTo(key.startsWith("MOTIF-") ? `seq-motif-${key}` : `seq-unit-${key}`);
  }

  /** The unit's card, or its beat when the beat has no sequence yet and shows no unit card. */
  function openUnit(unitId: string) {
    const beat = state?.beats.find((b) => b.units.some((u) => u.unit_id === unitId));
    scrollTo(beat && !beat.written ? `seq-beat-${beat.key}` : `seq-unit-${unitId}`);
  }

  return (
    <div data-testid="visual-sequence-page">
      <AppShell
        viewStep={VIEW_VISUAL_SEQUENCE}
        wide
        title="Chuỗi hình"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, sequence.running !== null) : undefined}
      >
        {readOnly && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="seq-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {gate && (
          <>
            <p className={researchStyles.hint} data-testid="seq-not-open">
              {gate.text}
            </p>
            <div className={researchStyles.actions}>
              <Button
                variant="ghost"
                onClick={() => navigate(gate.route(projectId))}
                data-testid="seq-go-back"
              >
                {gate.back}
              </Button>
            </div>
          </>
        )}
        {!state && !gate && sequence.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="seq-load-error">
              Không tải được bước Chuỗi hình: {sequence.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button
                variant="ghost"
                onClick={() => void sequence.reload()}
                data-testid="seq-reload"
              >
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !gate && !sequence.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && !gate && (
          <div className={researchStyles.body}>
            <SequenceRunningBanner sequence={sequence} />
            <div className={lockStyles.tabs} role="tablist">
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "sequence"}
                onClick={() => setTab("sequence")}
                data-testid="seq-tab-sequence"
              >
                Chuỗi hình
                <span className={lockStyles.tabCount}>
                  {state.items_approved}/{state.coverage.units_total}
                </span>
              </button>
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "review"}
                onClick={() => setTab("review")}
                data-testid="seq-tab-review"
              >
                Rà soát và chốt
                <span className={lockStyles.tabCount}>{state.review.open_count} vấn đề</span>
              </button>
            </div>
            {tab === "sequence" && (
              <>
                <SequenceHeadCard state={state} />
                <SequenceMotifsCard
                  projectId={projectId}
                  state={state}
                  sequence={sequence}
                  readOnly={readOnly}
                />
                <SequenceRunCard
                  projectId={projectId}
                  state={state}
                  sequence={sequence}
                  readOnly={readOnly}
                />
                <SequenceFlowStrip state={state} onOpenUnit={openUnit} />
                <SequenceBeatsCard
                  projectId={projectId}
                  state={state}
                  sequence={sequence}
                  readOnly={readOnly}
                />
              </>
            )}
            {tab === "review" && (
              <SequenceReviewTab
                projectId={projectId}
                state={state}
                sequence={sequence}
                readOnly={readOnly}
                onOpen={open}
              />
            )}
            {sequence.actionError && (
              <p className={researchStyles.error} role="alert" data-testid="seq-error">
                {sequence.actionError}
              </p>
            )}
          </div>
        )}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          state && state.locked_version > 0
            ? `Đã chốt v${state.locked_version}. Sang Storyboard.`
            : state && state.blockers.length > 0
              ? `Chưa chốt được: ${state.blockers.join(" ")}`
              : "Chốt chuỗi hình để sang Storyboard."
        }
        onBack={() => navigate(visualPlanRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_VISUAL_SEQUENCE + 1, projectId))}
        nextDisabled={!state || state.locked_version === 0}
        nextLabel="Sang Storyboard"
        nextTestId="seq-next"
      />
    </div>
  );
}
