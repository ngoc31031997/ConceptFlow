import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { BeatState } from "../api/client";
import { AiModelSelect } from "../components/AiModelSelect";
import { AiOperationCard } from "../components/AiOperationCard";
import { AppShell } from "../components/AppShell";
import { BeatCard } from "../components/beats/BeatCard";
import { StructureCard } from "../components/beats/StructureCard";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card } from "../components/ui";
import { UnitRedoModal } from "../components/units/UnitRedoModal";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useBeats, type BeatsControls } from "../hooks/useBeats";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { VIEW_BEATS, comingRoute, readOnlyReason, storyRoute } from "../utils/flow";

const SUBTITLE =
  "Mỗi beat có một chức năng, một mục tiêu và các dữ kiện nó dùng. Thêm, xoá, đổi thứ tự beat; duyệt từng beat.";

const ROLE = "Chia câu chuyện thành các beat, mỗi beat một chức năng và một bước tiến. Chưa viết lời thoại.";

/** What a rewrite of the whole sheet does, for the dialog that asks first. */
const REWRITE_IMPACT = [
  "Beat bạn đã viết (mục tiêu, tên, nội dung…) giữ nguyên; các beat khác được AI viết lại hoặc bỏ.",
  "Mọi beat về chờ duyệt.",
];

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(state: BeatState, running: BeatsControls["running"]): StepStripProps {
  if (!state.storyApproved) {
    return { tone: "todo", label: "Chưa mở bước", message: "Duyệt câu chuyện ở bước Câu chuyện trước." };
  }
  if (running !== null) return { tone: "run", label: "AI đang viết", message: "Phần bạn tự viết giữ nguyên." };
  if (state.beats.length === 0) {
    return { tone: "todo", label: "Chưa có beat sheet", message: "AI chia câu chuyện thành các beat, hoặc bạn tự thêm từng beat." };
  }
  const done = state.beats.filter((b) => b.status === "approved").length;
  if (state.status === "approved") {
    return { tone: "ok", label: "Beat sheet đã duyệt", message: "Tiếp theo: viết kịch bản theo từng beat." };
  }
  return {
    tone: "wait",
    label: `Chờ bạn duyệt · ${done}/${state.beats.length} beat`,
    message: "Beat đã duyệt giữ nguyên khi beat khác làm lại.",
  };
}

/**
 * Step 5 of the 21-step flow: the beat sheet. The AI splits the approved story
 * into beats, each with a function, a goal, the facts it uses and a target
 * length; the Creator edits, reorders, adds and deletes beats and approves
 * each one. The structure card checks the whole sheet and only warns.
 */
export function BeatsStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const projectId = params.id ?? draft.projectId;
  const beats = useBeats(projectId, !!projectId);
  const [rewriteOpen, setRewriteOpen] = useState(false);

  if (!projectId) return <Navigate to="/" replace />;

  const state = beats.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;

  function content(s: BeatState) {
    const busy = locked || beats.busy || beats.running !== null;
    const pendingWithGoal = s.beats.filter((b) => b.status === "pending" && b.goal.trim() !== "").length;
    const writingSheet = beats.running === "sheet";
    const writingAdd = beats.running === "add";
    return (
      <>
        {!s.storyApproved && (
          <p className={researchStyles.hint} data-testid="beats-not-open">
            Duyệt câu chuyện ở bước Câu chuyện trước.
          </p>
        )}
        {s.storyApproved && s.beats.length === 0 && (
          <Card data-testid="beats-empty-card">
            <div className={researchStyles.body}>
              <p className={researchStyles.hint}>
                {aiAvailable
                  ? "Chưa có beat sheet. AI chia câu chuyện đã duyệt thành các beat, hoặc bạn tự thêm từng beat."
                  : "Chưa có beat sheet. Chưa cấu hình AI nên bạn tự thêm từng beat rồi duyệt."}
              </p>
              {aiAvailable && <AiModelSelect projectId={projectId} step="beat_sheet" label="Model AI cho bước Beat sheet" testId="beats-model" />}
              <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                {aiAvailable && (
                  <Button variant="run" onClick={() => void beats.generate("")} disabled={busy} data-testid="beats-generate">
                    AI viết beat sheet
                  </Button>
                )}
                <Button variant="ghost" onClick={() => void beats.add("", true, "")} disabled={busy} data-testid="beats-add-first">
                  + Thêm beat trống
                </Button>
              </div>
            </div>
          </Card>
        )}
        <AiOperationCard
          run={beats.run}
          active={writingSheet || writingAdd}
          testId="beats-progress"
          title={writingAdd ? "AI đang viết beat mới" : "AI đang viết beat sheet"}
        />
        {s.beats.length > 0 && (
          <>
            <StructureCard state={s} />
            <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
              {aiAvailable && (
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setRewriteOpen(true)} disabled={busy} data-testid="beats-rewrite">
                  Viết lại cả bảng
                </Button>
              )}
              <Button
                variant="ghost"
                className={researchStyles.sm}
                onClick={() => void beats.approveAll()}
                disabled={busy || pendingWithGoal === 0}
                data-testid="beats-approve-all"
              >
                Duyệt tất cả beat còn lại
              </Button>
            </div>
            {s.beats.map((beat, index) => (
              <BeatCard key={beat.key} projectId={projectId} beat={beat} index={index} state={s} beats={beats} aiAvailable={aiAvailable} locked={locked} />
            ))}
          </>
        )}
        {beats.actionError && (
          <p className={researchStyles.error} role="alert" data-testid="beats-error">
            {beats.actionError}
          </p>
        )}
      </>
    );
  }

  return (
    <div data-testid="beats-step-page">
      <AppShell
        viewStep={VIEW_BEATS}
        wide
        title="Dàn ý · beat sheet"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, beats.running) : undefined}
      >
        {locked && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="beats-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {!state && beats.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="beats-load-error">
              Không tải được bước Beat sheet: {beats.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={beats.reload} data-testid="beats-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !beats.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && <div className={researchStyles.body}>{content(state)}</div>}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          state?.status === "approved"
            ? "Beat sheet đã duyệt. Sang bước Viết kịch bản."
            : "Duyệt mọi beat để sang bước Viết kịch bản."
        }
        onBack={() => navigate(storyRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_BEATS + 1, projectId))}
        nextDisabled={state?.status !== "approved"}
        nextLabel="Lưu và sang viết kịch bản"
        nextTestId="beats-next"
      />

      <UnitRedoModal
        open={rewriteOpen}
        onClose={() => setRewriteOpen(false)}
        title="Viết lại cả bảng · Beat sheet"
        impact={{ items: REWRITE_IMPACT, failed: false }}
        initialNote={beats.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="beat_sheet" label="Model AI cho bước Beat sheet" testId="beats-rewrite-model" /> : undefined}
        onSend={(note) => void beats.generate(note)}
        testId="beats-rewrite-dialog"
      />
    </div>
  );
}
