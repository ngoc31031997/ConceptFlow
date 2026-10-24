import { useState } from "react";
import type { IdeaAspect, IdeaRating, ProjectBriefField } from "../../api/client";
import type { ProjectBriefControls } from "../../hooks/useProjectBrief";
import { formatClock } from "../../utils/clockDuration";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { Button, Card } from "../ui";
import { UnitReviewBar } from "../units/UnitReviewBar";
import { UnitStatusChip } from "../units/UnitStatusChip";
import unitStyles from "../units/units.module.css";
import cardStyles from "../CreativeBriefCard.module.css";
import styles from "../idea/ProjectIdeaCard.module.css";
import { BriefEditModal } from "./BriefEditModal";
import { BriefInputModal } from "./BriefInputModal";
import { BriefRedoModal } from "./BriefRedoModal";

const FIELD_LABELS: Record<ProjectBriefField, string> = {
  audience: "Khán giả",
  duration_seconds: "Thời lượng",
  tone: "Giọng điệu",
  learning_objectives: "Mục tiêu học tập",
  avoid: "Cần tránh",
};

const ASPECTS: {
  key: "audienceFit" | "durationFit" | "objectives" | "avoidList";
  label: string;
}[] = [
  { key: "audienceFit", label: "Khán giả" },
  { key: "durationFit", label: "Thời lượng" },
  { key: "objectives", label: "Mục tiêu học tập" },
  { key: "avoidList", label: "Cần tránh" },
];

const RATING_LABELS: Record<IdeaRating, { label: string; className: string }> = {
  good: { label: "Tốt", className: styles.good },
  fair: { label: "Tạm ổn", className: styles.fair },
  weak: { label: "Yếu", className: styles.weak },
};

interface ProjectBriefCardProps {
  projectId: string;
  /** The project exists on the server; before that there is no brief to load. */
  created: boolean;
  brief: ProjectBriefControls;
  /** A language model is configured, so the AI can write, redo and review the brief. */
  aiAvailable: boolean;
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
 * Section 2 of the brief step: the brief the AI made of the approved idea, the
 * AI's review of it, and the actions that decide it. The review comes before
 * the approve button (docs/ux-ui-design-rules.md, rule 1) and only advises;
 * approval needs the five fields filled and the length inside the video form's
 * range. "Sửa tay" and "Làm lại kèm ghi chú" open their dialogs (rule 4).
 * Without a language model the Creator writes the brief and approves it.
 */
export function ProjectBriefCard({
  projectId,
  created,
  brief,
  aiAvailable,
}: ProjectBriefCardProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [redoOpen, setRedoOpen] = useState(false);
  const [inputOpen, setInputOpen] = useState(false);

  const screen = brief.screen;
  const stored = screen?.brief ?? null;
  const review = screen?.review ?? null;
  const ideaApproved = screen?.idea != null;
  const minSeconds = screen?.minDurationSeconds ?? 0;
  const maxSeconds = screen?.maxDurationSeconds ?? 0;

  const missing: ProjectBriefField[] = [];
  let outOfRange = false;
  if (stored) {
    if (!stored.audience.trim()) missing.push("audience");
    if (stored.durationSeconds <= 0) missing.push("duration_seconds");
    else outOfRange = stored.durationSeconds < minSeconds || stored.durationSeconds > maxSeconds;
    if (!stored.tone.trim()) missing.push("tone");
    if (stored.learningObjectives.length === 0) missing.push("learning_objectives");
    if (stored.avoid.length === 0) missing.push("avoid");
  }
  const working = brief.writing || brief.reviewing;
  const cannotApprove = missing.length > 0 || outOfRange;

  const chip = brief.writing ? (
    <UnitStatusChip state="running" testId="brief-status" />
  ) : stored ? (
    <UnitStatusChip state={stored.status} testId="brief-status" />
  ) : null;

  function mine(name: ProjectBriefField) {
    if (!stored?.creatorFields.includes(name)) return null;
    return (
      <span
        className={unitStyles.mine}
        title="AI giữ nguyên ô này khi làm lại"
        data-testid={`brief-mine-${name}`}
      >
        ✎ Bạn viết
      </span>
    );
  }

  function text(name: "audience" | "tone", value: string) {
    return (
      <div className={cardStyles.field} data-testid={`brief-field-${name}`}>
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

  function list(name: "learning_objectives" | "avoid", items: string[], ordered: boolean) {
    const Tag = ordered ? "ol" : "ul";
    return (
      <div className={cardStyles.field} data-testid={`brief-field-${name}`}>
        <span className={cardStyles.label}>
          {FIELD_LABELS[name]}
          {mine(name)}
        </span>
        {items.length > 0 ? (
          <Tag className={styles.points}>
            {items.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </Tag>
        ) : (
          <p className={cardStyles.value}>
            <span className={cardStyles.empty}>(trống)</span>
          </p>
        )}
      </div>
    );
  }

  function duration(seconds: number) {
    return (
      <div className={cardStyles.field} data-testid="brief-field-duration_seconds">
        <span className={cardStyles.label}>
          {FIELD_LABELS.duration_seconds}
          {mine("duration_seconds")}
        </span>
        <p className={cardStyles.value}>
          {seconds > 0 ? (
            <>
              {formatClock(seconds)} <span className={cardStyles.hint}>({seconds} giây)</span>
            </>
          ) : (
            <span className={cardStyles.empty}>(trống)</span>
          )}
        </p>
      </div>
    );
  }

  function reviewPanel() {
    if (!stored) return null;
    return (
      <div className={styles.review} data-testid="brief-review-panel">
        <div className={styles.reviewHead}>
          <h4>Đánh giá của AI</h4>
          {aiAvailable && (
            <Button
              variant="run"
              onClick={() => void brief.review()}
              disabled={working || brief.busy || cannotApprove}
              data-testid="brief-review-run"
            >
              {review ? "Đánh giá lại" : "Nhờ AI đánh giá"}
            </Button>
          )}
        </div>
        {!review && (
          <p className={cardStyles.hint} data-testid="brief-review-none">
            {aiAvailable
              ? "Chưa có đánh giá. AI đọc brief về khán giả, thời lượng, mục tiêu học tập và điều cần tránh; đánh giá chỉ để bạn tham khảo."
              : "Chưa cấu hình AI nên không có đánh giá. Bạn tự xem brief rồi duyệt."}
          </p>
        )}
        {review && (
          <>
            {screen?.reviewStale && (
              <p className={styles.banner} data-testid="brief-review-stale">
                Brief đã đổi sau đánh giá này. Bấm “Đánh giá lại” để AI đọc bản mới.
              </p>
            )}
            <div className={styles.aspects}>
              {ASPECTS.map((a) => (
                <Aspect key={a.key} label={a.label} aspect={review[a.key]} />
              ))}
            </div>
            {review.unclear.length > 0 && (
              <div data-testid="brief-review-unclear">
                <span className={styles.listTitle}>Chỗ chưa rõ</span>
                <ul className={styles.list}>
                  {review.unclear.map((u) => (
                    <li key={u}>{u}</li>
                  ))}
                </ul>
              </div>
            )}
            {review.suggestions.length > 0 && (
              <div data-testid="brief-review-suggestions">
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
        <AiOperationCard
          run={brief.reviewRun}
          active={brief.reviewing}
          testId="brief-review-progress"
          title="AI đang đọc brief"
        />
      </div>
    );
  }

  function body() {
    if (!created) {
      return <p className={cardStyles.hint}>Chưa có project. Hoàn thành bước Ý tưởng trước.</p>;
    }
    if (!screen) {
      if (brief.loadError) {
        return (
          <>
            <p className={cardStyles.error} role="alert">
              Không tải được brief: {brief.loadError}
            </p>
            <div className={cardStyles.actions}>
              <Button variant="ghost" onClick={brief.reload} data-testid="brief-reload">
                Tải lại
              </Button>
            </div>
          </>
        );
      }
      return (
        <p className={cardStyles.hint} role="status">
          Đang tải brief…
        </p>
      );
    }
    if (!stored) {
      if (brief.writing) return null;
      return (
        <>
          <p className={cardStyles.hint} data-testid="brief-none">
            {!ideaApproved
              ? "Chưa có brief. Duyệt ý tưởng ở bước 1 trước: brief được viết từ ý tưởng đã duyệt."
              : aiAvailable
                ? "Chưa có brief. AI viết từ ý tưởng đã duyệt, hoặc bạn tự viết."
                : "Chưa có brief. Chưa cấu hình AI nên bạn tự viết brief rồi duyệt."}
          </p>
          {ideaApproved && aiAvailable && (
            <AiModelSelect
              projectId={projectId}
              step="brief"
              label="Model AI viết brief"
              testId="brief-model"
            />
          )}
          <div className={cardStyles.actions}>
            <Button variant="ghost" onClick={() => setEditOpen(true)} data-testid="brief-write-own">
              Tự viết brief
            </Button>
            {aiAvailable && (
              <Button
                variant="run"
                onClick={() => void brief.generate()}
                disabled={!ideaApproved}
                data-testid="brief-generate"
              >
                AI viết brief
              </Button>
            )}
          </div>
        </>
      );
    }
    return (
      <>
        <p className={cardStyles.hint}>
          AI viết từ ý tưởng đã duyệt. Ô nào cũng sửa được bằng “Sửa tay”; ô bạn đã sửa thì AI giữ
          nguyên khi làm lại.
        </p>
        {screen.ideaChanged && (
          <p className={styles.banner} data-testid="brief-idea-changed">
            Ý tưởng ở bước 1 đã đổi sau khi AI viết brief này. Dùng “Làm lại kèm ghi chú” nếu muốn
            viết lại theo ý tưởng mới.
          </p>
        )}
        {text("audience", stored.audience)}
        {duration(stored.durationSeconds)}
        {text("tone", stored.tone)}
        {list("learning_objectives", stored.learningObjectives, true)}
        {list("avoid", stored.avoid, false)}
        {reviewPanel()}
        {stored.status === "pending" && missing.length > 0 && (
          <p className={cardStyles.hint} data-testid="brief-missing">
            Còn trống: {missing.map((f) => FIELD_LABELS[f]).join("; ")}. Điền đủ các ô rồi mới duyệt
            được.
          </p>
        )}
        {outOfRange && (
          <p className={cardStyles.hint} data-testid="brief-duration-range">
            Thời lượng của {screen.videoMode === "short" ? "video ngắn" : "video dài"} phải từ{" "}
            {formatClock(minSeconds)} đến {formatClock(maxSeconds)}. Sửa lại rồi mới duyệt được.
          </p>
        )}
        <UnitReviewBar
          status={stored.status}
          approveLabel="Duyệt brief"
          approveDisabled={cannotApprove}
          busy={brief.busy || working}
          canRedo={aiAvailable && ideaApproved}
          onEdit={() => setEditOpen(true)}
          onRedo={() => setRedoOpen(true)}
          onApprove={() => void brief.approve()}
          onUnapprove={() => void brief.unapprove()}
          testId="brief-actions"
        />
      </>
    );
  }

  return (
    <Card data-testid="brief-card">
      <div className={cardStyles.body}>
        <div className={cardStyles.head}>
          <div className={cardStyles.titleGroup}>
            <h3>② Định hướng sáng tạo</h3>
            <Button
              variant="view"
              className={cardStyles.sm}
              onClick={() => setInputOpen(true)}
              title="Xem ý tưởng đã duyệt ở bước 1 mà AI đọc để viết brief"
              data-testid="brief-input-open"
            >
              Xem đầu vào của AI
            </Button>
          </div>
          {chip}
        </div>
        {body()}
        <AiOperationCard
          run={brief.writeRun}
          active={brief.writing}
          testId="brief-progress"
          title="AI đang viết brief"
        />
        {brief.actionError && (
          <p className={cardStyles.error} role="alert" data-testid="brief-error">
            {brief.actionError}
          </p>
        )}
      </div>

      <BriefEditModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        brief={stored}
        minDurationSeconds={minSeconds}
        maxDurationSeconds={maxSeconds}
        onSave={brief.edit}
      />
      <BriefInputModal open={inputOpen} onClose={() => setInputOpen(false)} idea={brief.screen?.idea ?? null} />
      <BriefRedoModal
        open={redoOpen}
        onClose={() => setRedoOpen(false)}
        approved={stored?.status === "approved"}
        initialNote={brief.failedNote}
        modelSelect={
          aiAvailable ? (
            <AiModelSelect
              projectId={projectId}
              step="brief"
              label="Model AI viết brief"
              testId="brief-redo-model"
            />
          ) : undefined
        }
        onSend={(note) => void brief.generate(note)}
      />
    </Card>
  );
}
