import { useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { StoryState } from "../api/client";
import { AiModelSelect } from "../components/AiModelSelect";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import type { StepStripProps } from "../components/StepStrip";
import { StoryCard } from "../components/story/StoryCard";
import { StoryInputCard } from "../components/story/StoryInputCard";
import researchStyles from "../components/research/research.module.css";
import { Button, Card } from "../components/ui";
import { UnitRedoModal } from "../components/units/UnitRedoModal";
import { UnitStatusChip } from "../components/units/UnitStatusChip";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useStory } from "../hooks/useStory";
import { VIEW_STORY, beatsRoute, readOnlyReason } from "../utils/flow";

const SUBTITLE =
  "Chọn góc kể và dựng mạch từ câu hỏi đến lời giải, từ brief và bảng dữ kiện đã duyệt. Sửa từng ô, hoặc cho AI làm lại kèm ghi chú.";

const ROLE =
  "Quyết định cách kể trước khi viết lời: câu hỏi trung tâm, câu trả lời cốt lõi và mạch kể gồm các nhịp, mỗi nhịp dựa vào những dữ kiện nào.";

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(state: StoryState, writing: boolean): StepStripProps {
  if (!state.briefApproved || !state.researchResolved) {
    return {
      tone: "todo",
      label: "Chưa mở bước",
      message: !state.briefApproved
        ? "Duyệt brief ở bước Creative brief trước."
        : "Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước.",
    };
  }
  if (writing) {
    return { tone: "run", label: "AI đang viết", message: "Ô bạn tự viết giữ nguyên." };
  }
  if (!state.story) {
    return { tone: "todo", label: "Chưa có câu chuyện", message: "AI viết câu chuyện, hoặc bạn tự viết từng ô." };
  }
  if (state.status === "approved") {
    return { tone: "ok", label: "Câu chuyện đã duyệt", message: "Tiếp theo: beat sheet." };
  }
  return {
    tone: "wait",
    label: "Chờ bạn duyệt câu chuyện",
    message: "Sửa từng ô hoặc cho AI làm lại kèm ghi chú; duyệt khi cả câu chuyện ổn.",
  };
}

/**
 * Step 4 of the 21-step flow: build the story. The AI reads the brief approved
 * on step 2 and the facts kept on step 3 and writes the central question, the
 * core answer and the narrative arc; the Creator edits each field or has the
 * AI redo it with a note, and approves the story as a whole. Changing any
 * field of an approved story sends it back to review.
 */
export function StoryStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const projectId = params.id ?? draft.projectId;
  const story = useStory(projectId, !!projectId);
  const [redoOpen, setRedoOpen] = useState(false);

  if (!projectId) return <Navigate to="/" replace />;

  const state = story.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;

  function content(s: StoryState) {
    const hasStory = s.story !== null;
    const busy = locked || story.busy || story.writing;
    return (
      <>
        <StoryInputCard state={s} onReload={story.reload} />
        <StoryCard story={story} state={s} aiAvailable={aiAvailable} locked={locked} />
        <Card data-testid="story-approve-card">
          <div className={researchStyles.body}>
            <div className={researchStyles.head}>
              <h3>③ Duyệt câu chuyện</h3>
              <UnitStatusChip state={s.status} testId="story-approve-status" />
            </div>
            <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
              {s.status === "approved" ? (
                <Button variant="ghost" onClick={() => void story.unapprove()} disabled={busy} data-testid="story-unapprove">
                  Bỏ duyệt
                </Button>
              ) : (
                <>
                  {aiAvailable && hasStory && (
                    <Button variant="ghost" onClick={() => setRedoOpen(true)} disabled={busy} data-testid="story-redo-all">
                      Làm lại kèm ghi chú
                    </Button>
                  )}
                  <Button variant="primary" onClick={() => void story.approve()} disabled={busy || !hasStory} data-testid="story-approve">
                    Duyệt câu chuyện
                  </Button>
                </>
              )}
            </div>
          </div>
        </Card>
      </>
    );
  }

  return (
    <div data-testid="story-step-page">
      <AppShell
        viewStep={VIEW_STORY}
        wide
        title="Xây dựng câu chuyện"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, story.writing) : undefined}
      >
        {locked && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="story-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {!state && story.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="story-load-error">
              Không tải được bước Câu chuyện: {story.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={story.reload} data-testid="story-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !story.loadError && (
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
            ? "Câu chuyện đã duyệt. Sang bước Beat sheet."
            : "Duyệt câu chuyện để sang bước Beat sheet."
        }
        onBack={() => navigate(`/projects/${projectId}/research`)}
        onNext={() => navigate(beatsRoute(projectId))}
        nextDisabled={state?.status !== "approved"}
        nextLabel="Lưu và sang beat sheet"
        nextTestId="story-next"
      />

      <UnitRedoModal
        open={redoOpen}
        onClose={() => setRedoOpen(false)}
        title="Làm lại kèm ghi chú · Câu chuyện"
        impact={{ items: [], failed: false }}
        initialNote={story.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="story_frame" label="Model AI cho bước Câu chuyện" testId="story-redo-all-model" /> : undefined}
        onSend={(note) => void story.generate(note)}
        testId="story-redo-all-dialog"
      />
    </div>
  );
}
