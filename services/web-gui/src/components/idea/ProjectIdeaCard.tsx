import { useState } from "react";
import {
  describeAiRequest,
  generateProjectIdea,
  reviewProjectIdea,
  type IdeaAspect,
  type IdeaRating,
  type ProjectIdeaField,
} from "../../api/client";
import type { ProjectIdeaControls } from "../../hooks/useProjectIdea";
import { AiModelSelect } from "../AiModelSelect";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { Button, Card } from "../ui";
import { UnitReviewBar } from "../units/UnitReviewBar";
import { UnitStatusChip } from "../units/UnitStatusChip";
import unitStyles from "../units/units.module.css";
import cardStyles from "../CreativeBriefCard.module.css";
import styles from "./ProjectIdeaCard.module.css";
import { IdeaEditModal } from "./IdeaEditModal";
import { IdeaRunPreview } from "./IdeaRunPreview";

const FIELD_LABELS: Record<ProjectIdeaField, string> = {
  central_question: "Câu hỏi trung tâm",
  curiosity_points: "Điểm gây tò mò",
  scope: "Phạm vi",
};

const ASPECTS: { key: "clarity" | "researchability" | "scopeFit"; label: string }[] = [
  { key: "clarity", label: "Độ rõ" },
  { key: "researchability", label: "Khả năng nghiên cứu" },
  { key: "scopeFit", label: "Phạm vi" },
];

const RATING_LABELS: Record<IdeaRating, { label: string; className: string }> = {
  good: { label: "Tốt", className: styles.good },
  fair: { label: "Tạm ổn", className: styles.fair },
  weak: { label: "Yếu", className: styles.weak },
};

interface ProjectIdeaCardProps {
  projectId: string;
  /** The project exists on the server; before that there is no idea to load. */
  created: boolean;
  idea: ProjectIdeaControls;
  /** The system's own language model is configured; it only changes the hints, since an outside AI can run too. */
  aiAvailable: boolean;
  /** The dialog that has the AI write or redo the idea is open; the page opens it right after it creates the project. */
  writeOpen: boolean;
  onWriteOpenChange: (open: boolean) => void;
}

function Aspect({ label, aspect }: { label: string; aspect: IdeaAspect }) {
  const rating = RATING_LABELS[aspect.rating];
  return (
    <div className={styles.aspect}>
      <span className={styles.aspectName}>
        {label} <span className={`${styles.rating} ${rating.className}`}>{rating.label}</span>
      </span>
      <p>{aspect.comment}</p>
    </div>
  );
}

/**
 * Section 2 of the idea step: the idea the AI made of the Creator's text, the
 * AI's review of it, and the actions that decide it. The review comes before
 * the approve button (docs/ux-ui-design-rules.md, rule 1) and only advises;
 * approval needs the three fields filled and nothing else. "Sửa tay" and
 * "Làm lại kèm ghi chú" open their dialogs (rule 4); the AI actions run in the shared run dialog, which shows the result before it is stored (ADR-0052). Without a language model
 * the Creator writes the idea and approves it.
 */
export function ProjectIdeaCard({
  projectId,
  created,
  idea,
  aiAvailable,
  writeOpen,
  onWriteOpenChange,
}: ProjectIdeaCardProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const ideaPath = `/v1/projects/${projectId}/idea/generate`;
  const reviewPath = `/v1/projects/${projectId}/idea/review`;

  const screen = idea.screen;
  const stored = screen?.idea ?? null;
  const review = screen?.review ?? null;
  const missing: ProjectIdeaField[] = [];
  if (stored) {
    if (!stored.centralQuestion.trim()) missing.push("central_question");
    if (stored.curiosityPoints.length === 0) missing.push("curiosity_points");
    if (!stored.scope.trim()) missing.push("scope");
  }
  const chip = stored ? <UnitStatusChip state={stored.status} testId="idea-status" /> : null;

  function mine(name: ProjectIdeaField) {
    if (!stored?.creatorFields.includes(name)) return null;
    return (
      <span
        className={unitStyles.mine}
        title="AI giữ nguyên ô này khi làm lại"
        data-testid={`idea-mine-${name}`}
      >
        ✎ Bạn viết
      </span>
    );
  }

  function text(name: "central_question" | "scope", value: string) {
    return (
      <div className={cardStyles.field} data-testid={`idea-field-${name}`}>
        <span className={cardStyles.label}>
          {FIELD_LABELS[name]}
          {mine(name)}
        </span>
        <p className={cardStyles.value}>
          {value.trim() || <span className={cardStyles.empty}>(trống)</span>}
        </p>
      </div>
    );
  }

  function reviewPanel() {
    if (!stored) return null;
    return (
      <div className={styles.review} data-testid="idea-review-panel">
        <div className={styles.reviewHead}>
          <h4>Đánh giá của AI</h4>
          <div className={cardStyles.actions}>
            <Button
              variant="run"
              onClick={() => setReviewOpen(true)}
              disabled={idea.busy || missing.length > 0}
              data-testid="idea-review-run"
            >
              {review ? "Đánh giá lại" : "Nhờ AI đánh giá"}
            </Button>
            <AiRunPill
              runPath={reviewPath}
              onOpen={() => setReviewOpen(true)}
              testId="idea-review-ai-run"
            />
          </div>
        </div>
        {!review && (
          <p className={cardStyles.hint} data-testid="idea-review-none">
            {aiAvailable
              ? "Chưa có đánh giá. AI đọc ý tưởng về độ rõ, khả năng nghiên cứu và phạm vi; đánh giá chỉ để bạn tham khảo."
              : "Chưa cấu hình AI nên không có đánh giá. Bạn tự xem ý tưởng rồi duyệt."}
          </p>
        )}
        {review && (
          <>
            {screen?.reviewStale && (
              <p className={styles.banner} data-testid="idea-review-stale">
                Ý tưởng đã đổi sau đánh giá này. Bấm “Đánh giá lại” để AI đọc bản mới.
              </p>
            )}
            <div className={styles.aspects}>
              {ASPECTS.map((a) => (
                <Aspect key={a.key} label={a.label} aspect={review[a.key]} />
              ))}
            </div>
            {review.unclear.length > 0 && (
              <div data-testid="idea-review-unclear">
                <span className={styles.listTitle}>Chỗ chưa rõ</span>
                <ul className={styles.list}>
                  {review.unclear.map((u) => (
                    <li key={u}>{u}</li>
                  ))}
                </ul>
              </div>
            )}
            {review.suggestions.length > 0 && (
              <div data-testid="idea-review-suggestions">
                <span className={styles.listTitle}>Gợi ý cải thiện</span>
                <ul className={styles.list}>
                  {review.suggestions.map((s) => (
                    <li key={s}>{s}</li>
                  ))}
                </ul>
              </div>
            )}
          </>
        )}
      </div>
    );
  }

  function body() {
    if (!created) {
      return (
        <p className={cardStyles.hint}>
          {aiAvailable
            ? "Chưa có ý tưởng. Gõ ý tưởng ở trên rồi bấm “Tạo project và viết ý tưởng”: AI viết, bạn sửa rồi duyệt."
            : "Chưa có ý tưởng. Gõ ý tưởng ở trên rồi bấm “Tạo project”, sau đó tự viết ý tưởng và duyệt."}
        </p>
      );
    }
    if (!screen) {
      if (idea.loadError) {
        return (
          <>
            <p className={cardStyles.error} role="alert">
              Không tải được ý tưởng: {idea.loadError}
            </p>
            <div className={cardStyles.actions}>
              <Button variant="ghost" onClick={idea.reload} data-testid="idea-reload">
                Tải lại
              </Button>
            </div>
          </>
        );
      }
      return (
        <p className={cardStyles.hint} role="status">
          Đang tải ý tưởng…
        </p>
      );
    }
    if (!stored) {
      return (
        <>
          <p className={cardStyles.hint}>
            {aiAvailable
              ? "Chưa có ý tưởng. AI viết từ nội dung bạn đã gõ, hoặc bạn tự viết."
              : "Chưa có ý tưởng. AI viết từ nội dung bạn đã gõ (qua extension hoặc dán tay khi chưa cấu hình model hệ thống), hoặc bạn tự viết."}
          </p>
          {aiAvailable && (
            <AiModelSelect
              projectId={projectId}
              step="brief"
              label="Model AI viết ý tưởng"
              testId="idea-model"
            />
          )}
          <div className={cardStyles.actions}>
            <Button variant="ghost" onClick={() => setEditOpen(true)} data-testid="idea-write-own">
              Tự viết ý tưởng
            </Button>
            <Button
              variant="run"
              onClick={() => onWriteOpenChange(true)}
              data-testid="idea-generate"
            >
              AI viết ý tưởng
            </Button>
            <AiRunPill
              runPath={ideaPath}
              onOpen={() => onWriteOpenChange(true)}
              testId="idea-ai-run"
            />
          </div>
        </>
      );
    }
    return (
      <>
        <p className={cardStyles.hint}>
          AI viết từ nội dung bạn đã gõ. Ô nào cũng sửa được bằng “Sửa tay”; ô bạn đã sửa thì AI giữ
          nguyên khi làm lại.
        </p>
        {screen.inputChanged && (
          <p className={styles.banner} data-testid="idea-input-changed">
            Nội dung hoặc ngôn ngữ đã đổi sau khi AI viết ý tưởng này. Dùng “Làm lại kèm ghi chú”
            nếu muốn viết lại theo bản mới.
          </p>
        )}
        {text("central_question", stored.centralQuestion)}
        <div className={cardStyles.field} data-testid="idea-field-curiosity_points">
          <span className={cardStyles.label}>
            {FIELD_LABELS.curiosity_points}
            {mine("curiosity_points")}
          </span>
          {stored.curiosityPoints.length > 0 ? (
            <ul className={styles.points}>
              {stored.curiosityPoints.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          ) : (
            <p className={cardStyles.value}>
              <span className={cardStyles.empty}>(trống)</span>
            </p>
          )}
        </div>
        {text("scope", stored.scope)}
        {reviewPanel()}
        {stored.status === "pending" && missing.length > 0 && (
          <p className={cardStyles.hint} data-testid="idea-missing">
            Còn trống: {missing.map((f) => FIELD_LABELS[f]).join("; ")}. Điền đủ các ô rồi mới duyệt
            được.
          </p>
        )}
        <UnitReviewBar
          status={stored.status}
          approveLabel="Duyệt ý tưởng"
          approveDisabled={missing.length > 0}
          busy={idea.busy}
          canRedo
          onEdit={() => setEditOpen(true)}
          onRedo={() => onWriteOpenChange(true)}
          onApprove={() => void idea.approve()}
          onUnapprove={() => void idea.unapprove()}
          testId="idea-actions"
        />
      </>
    );
  }

  return (
    <Card data-testid="idea-card">
      <div className={cardStyles.body}>
        <div className={cardStyles.head}>
          <h3>② Ý tưởng</h3>
          {chip}
        </div>
        {body()}
        {idea.actionError && (
          <p className={cardStyles.error} role="alert" data-testid="idea-error">
            {idea.actionError}
          </p>
        )}
      </div>

      <IdeaEditModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        idea={stored}
        onSave={idea.edit}
      />
      <AiRunDialog
        open={writeOpen}
        onClose={() => onWriteOpenChange(false)}
        title={stored ? "Làm lại kèm ghi chú · Ý tưởng" : "Viết ý tưởng"}
        runPath={ideaPath}
        describe={(note) =>
          describeAiRequest((external) => generateProjectIdea(projectId, note, { external }))
        }
        renderPreview={(p) => <IdeaRunPreview kind="idea" preview={p} />}
        impact={
          <>
            <p className={unitStyles.note}>
              AI viết lại các ô bạn chưa sửa; ô “Bạn viết” giữ nguyên.
              {stored?.status === "approved" ? " Ý tưởng đang được duyệt sẽ về chờ duyệt." : ""}
            </p>
            {aiAvailable && (
              <AiModelSelect
                projectId={projectId}
                step="brief"
                label="Model AI viết ý tưởng"
                testId="idea-write-model"
              />
            )}
          </>
        }
        onApplied={idea.reload}
        testId="idea-write-dialog"
      />
      <AiRunDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title="Đánh giá ý tưởng"
        runPath={reviewPath}
        describe={() => describeAiRequest((external) => reviewProjectIdea(projectId, { external }))}
        noNote
        renderPreview={(p) => <IdeaRunPreview kind="review" preview={p} />}
        impact={
          <p className={unitStyles.note}>
            AI đọc ý tưởng về độ rõ, khả năng nghiên cứu và phạm vi. Đánh giá chỉ để bạn tham khảo:
            không đổi ý tưởng, không đổi trạng thái duyệt.
          </p>
        }
        onApplied={idea.reload}
        testId="idea-review-dialog"
      />
    </Card>
  );
}
