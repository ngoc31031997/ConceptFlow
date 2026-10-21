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
import { ImpactConfirm } from "../components/research/ImpactConfirm";
import { UnitRedoModal } from "../components/units/UnitRedoModal";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useBeats, type BeatsControls } from "../hooks/useBeats";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { VIEW_BEATS, comingRoute, readOnlyReason, storyRoute } from "../utils/flow";

const SUBTITLE =
  "Bước 1: duyệt bộ khung (chức năng và dữ kiện của từng beat, dựng từ câu chuyện). Bước 2: AI viết nội dung từng beat, rồi bạn duyệt.";

const ROLE = "Chia câu chuyện thành các beat, mỗi beat một chức năng và một bước tiến. Chưa viết lời thoại.";

/** What building the skeleton again does, for the dialog that asks first. */
const REBUILD_IMPACT = [
  "Mọi beat của bộ khung hiện tại bị xoá, kể cả chức năng và dữ kiện bạn đã sửa.",
  "Bộ khung mới được dựng lại từ câu chuyện đã duyệt, không gọi AI.",
];

/** What a rewrite of the whole sheet does, for the dialog that asks first. */
const REWRITE_IMPACT = [
  "AI viết lại nội dung của mọi beat theo bộ khung đã duyệt; chức năng, dữ kiện và phần bạn đã viết giữ nguyên.",
  "Mọi beat về chờ duyệt.",
];

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(state: BeatState, running: BeatsControls["running"]): StepStripProps {
  if (!state.storyApproved) {
    return { tone: "todo", label: "Chưa mở bước", message: "Duyệt câu chuyện ở bước Câu chuyện trước." };
  }
  if (running !== null) return { tone: "run", label: "AI đang viết", message: "Phần bạn tự viết giữ nguyên." };
  if (state.phase === "empty") {
    return { tone: "todo", label: "Chưa có bộ khung", message: "Dựng bộ khung từ câu chuyện, hoặc bạn tự thêm từng beat." };
  }
  if (state.phase === "skeleton") {
    return { tone: "wait", label: "Chờ bạn duyệt bộ khung", message: "Kiểm chức năng và dữ kiện của từng beat, rồi duyệt để AI viết nội dung." };
  }
  if (state.unwritten > 0) {
    return { tone: "todo", label: `Chưa viết nội dung · ${state.unwritten}/${state.beats.length} beat`, message: "AI viết lần lượt từng beat theo bộ khung đã duyệt." };
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
 * Step 5 of the 21-step flow: the beat sheet, in two phases. First a skeleton
 * built from the approved story with no AI (one beat per step of the chain,
 * each with a function and its facts) which the Creator reviews and approves;
 * then the AI writes the content and length of each beat, one at a time, and
 * the Creator edits, reorders, adds and deletes beats and approves each one.
 * The structure card checks the whole sheet and only warns.
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
  const [rebuildOpen, setRebuildOpen] = useState(false);

  if (!projectId) return <Navigate to="/" replace />;

  const state = beats.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;

  function content(s: BeatState) {
    const busy = locked || beats.busy || beats.running !== null;
    const pendingWithGoal = s.beats.filter((b) => b.status === "pending" && b.goal.trim() !== "").length;
    const writingSheet = beats.running === "sheet";
    const writingAdd = beats.running === "add";
    const skeleton = s.phase === "skeleton";
    return (
      <>
        {!s.storyApproved && (
          <p className={researchStyles.hint} data-testid="beats-not-open">
            Duyệt câu chuyện ở bước Câu chuyện trước.
          </p>
        )}
        {s.storyApproved && s.phase === "empty" && (
          <Card data-testid="beats-empty-card">
            <div className={researchStyles.body}>
              <p className={researchStyles.hint}>
                Chưa có beat sheet. Bộ khung được dựng từ câu chuyện đã duyệt: một beat mở đầu, mỗi bước của chuỗi nhân quả một beat, một beat kết. Không cần AI.
              </p>
              <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                <Button onClick={() => void beats.buildSkeleton()} disabled={busy} data-testid="beats-build-skeleton">
                  Dựng bộ khung từ câu chuyện
                </Button>
                <Button variant="ghost" onClick={() => void beats.add("", true, "")} disabled={busy} data-testid="beats-add-first">
                  + Thêm beat trống
                </Button>
              </div>
            </div>
          </Card>
        )}
        {s.beats.length > 0 && (
          <>
            <StructureCard state={s} />
            {s.beats.map((beat, index) => (
              <BeatCard key={beat.key} projectId={projectId} beat={beat} index={index} state={s} beats={beats} aiAvailable={aiAvailable} locked={locked} />
            ))}
            <AiOperationCard
              run={beats.run}
              active={writingSheet || writingAdd}
              testId="beats-progress"
              title={writingAdd ? "AI đang viết beat mới" : "AI đang viết nội dung các beat"}
            />
            {skeleton && (
              <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
                <Button variant="ghost" onClick={() => setRebuildOpen(true)} disabled={busy} data-testid="beats-rebuild-skeleton">
                  Dựng lại bộ khung
                </Button>
                <Button onClick={() => void beats.approveSkeleton()} disabled={busy} data-testid="beats-approve-skeleton">
                  Duyệt bộ khung
                </Button>
              </div>
            )}
            {!skeleton && (
              <>
                {aiAvailable && <AiModelSelect projectId={projectId} step="beat_sheet" label="Model AI cho bước Beat sheet" testId="beats-model" />}
                <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
                  {aiAvailable && (
                    <Button variant="ghost" onClick={() => setRewriteOpen(true)} disabled={busy} data-testid="beats-rewrite">
                      Viết lại cả bảng
                    </Button>
                  )}
                  {aiAvailable && s.unwritten > 0 && (
                    <Button variant="run" onClick={() => void beats.generate("", true)} disabled={busy} data-testid="beats-generate">
                      AI viết nội dung các beat chưa có ({s.unwritten})
                    </Button>
                  )}
                  <Button onClick={() => void beats.approveAll()} disabled={busy || pendingWithGoal === 0} data-testid="beats-approve-all">
                    Duyệt tất cả beat còn lại
                  </Button>
                </div>
              </>
            )}
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
            : state?.phase === "skeleton"
              ? "Duyệt bộ khung, để AI viết nội dung, rồi duyệt mọi beat."
              : "Duyệt mọi beat để sang bước Viết kịch bản."
        }
        onBack={() => navigate(storyRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_BEATS + 1, projectId))}
        nextDisabled={state?.status !== "approved"}
        nextLabel="Lưu và sang viết kịch bản"
        nextTestId="beats-next"
      />

      <ImpactConfirm
        open={rebuildOpen}
        onClose={() => setRebuildOpen(false)}
        title="Dựng lại bộ khung?"
        lead="Bộ khung hiện tại của beat sheet"
        list={REBUILD_IMPACT}
        undo="Không hoàn tác được."
        choices={[{ label: "Dựng lại bộ khung", variant: "danger", onChoose: () => void beats.buildSkeleton(), testId: "beats-rebuild-confirm" }]}
        testId="beats-rebuild-dialog"
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
