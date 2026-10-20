import { useState } from "react";
import type { StoryApproach, StoryState } from "../../api/client";
import type { StoryControls } from "../../hooks/useStory";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { Button, Card } from "../ui";
import { ImpactConfirm } from "../research/ImpactConfirm";
import researchStyles from "../research/research.module.css";
import { FactTags } from "./FactTags";
import styles from "./story.module.css";

/** Where the Kiểu video table is managed; opened in a new tab so the story on screen stays. */
const ARCHETYPES_URL = "/settings/video-archetypes";

interface ApproachCardProps {
  projectId: string;
  story: StoryControls;
  state: StoryState;
  /** A language model is configured, so the AI can propose approaches. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

/** One proposed approach as a choice card. */
function ApproachOption({ approach, disabled, onPick }: { approach: StoryApproach; disabled: boolean; onPick: () => void }) {
  return (
    <button
      type="button"
      className={`${styles.approach} ${approach.selected ? styles.approachOn : ""}`}
      onClick={onPick}
      disabled={disabled}
      aria-pressed={approach.selected}
      data-testid={`approach-${approach.key}`}
    >
      <span className={styles.approachName}>
        <b>{approach.name}</b>
        <span className={styles.code}>{approach.code}</span>
        {approach.isNew && <span className={styles.newTag}>AI đề xuất kiểu mới</span>}
      </span>
      <span className={styles.small}>Góc kể: {approach.angle}</span>
      <span className={styles.small}>Vì sao hợp: {approach.why}</span>
      <FactTags ids={approach.refs} />
      <span className={`${styles.small} ${styles.open}`}>Câu mở mẫu: “{approach.open}”</span>
    </button>
  );
}

/**
 * Section 1 of the story step: the approaches the AI proposes from the brief
 * and the fact table, "Để AI tự chọn", "Xem tất cả kiểu" and "Gợi ý lại". A new
 * kind can be saved into the Kiểu video table. Changing the approach of a story
 * that is already approved asks first, since the fields the AI wrote are written again.
 */
export function ApproachCard({ projectId, story, state, aiAvailable, locked }: ApproachCardProps) {
  const [pending, setPending] = useState<string | null>(null);
  const running = story.running === "approaches";
  const hasFrame = state.fields.some((f) => f.text.trim() !== "");
  const chosen = state.approaches.find((a) => a.selected) ?? null;
  const disabled = locked || story.busy || story.running !== null;

  function pick(key: string) {
    if (key === state.approachKey) return;
    if (state.status === "approved" && hasFrame) setPending(key);
    else void story.selectApproach(key, false);
  }

  const name = (key: string) => state.approaches.find((a) => a.key === key)?.name ?? "AI tự chọn";

  return (
    <Card data-testid="story-approach-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>① Góc tiếp cận · kiểu video</h3>
          <span className={researchStyles.hint}>AI gợi ý từ bảng dữ kiện</span>
        </div>
        {state.approaches.length === 0 ? (
          <p className={researchStyles.hint} data-testid="story-approach-empty">
            {aiAvailable
              ? "Chưa có gợi ý. Bấm “AI gợi ý” để AI đọc brief và bảng dữ kiện, hoặc để AI tự chọn khi viết khung."
              : "Chưa cấu hình AI nên chưa có gợi ý góc. Bạn vẫn viết khung câu chuyện bằng tay."}
          </p>
        ) : (
          <div className={styles.approaches}>
            {state.approaches.map((a) => (
              <ApproachOption key={a.key} approach={a} disabled={disabled} onPick={() => pick(a.key)} />
            ))}
          </div>
        )}
        {aiAvailable && <AiModelSelect projectId={projectId} step="story_frame" label="Model AI cho bước Câu chuyện" testId="story-model" />}
        <AiOperationCard run={story.run} active={running} testId="story-approach-progress" title="AI đang gợi ý góc tiếp cận" />
        <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
          {chosen?.isNew &&
            (chosen.archetypeId ? (
              <span className={researchStyles.ok} data-testid="story-archetype-saved">
                ✓ Đã lưu vào bảng Kiểu video
              </span>
            ) : (
              <Button
                variant="primary"
                className={researchStyles.sm}
                onClick={() => void story.saveArchetype(chosen.key)}
                disabled={disabled}
                data-testid="story-archetype-save"
              >
                Lưu kiểu mới vào bảng Kiểu video
              </Button>
            ))}
          <Button
            variant="ghost"
            className={researchStyles.sm}
            onClick={() => pick("")}
            disabled={disabled}
            aria-pressed={state.approachKey === ""}
            data-testid="story-approach-auto"
          >
            {state.approachKey === "" ? "✓ AI tự chọn" : "Để AI tự chọn"}
          </Button>
          <Button
            variant="view"
            className={researchStyles.sm}
            onClick={() => window.open(ARCHETYPES_URL, "_blank", "noopener")}
            data-testid="story-approach-all"
          >
            Xem tất cả kiểu
          </Button>
          {aiAvailable && (
            <Button
              variant="run"
              className={researchStyles.sm}
              onClick={() => void story.generateApproaches("")}
              disabled={disabled || !state.briefApproved || !state.researchResolved}
              data-testid="story-approach-generate"
            >
              {state.approaches.length === 0 ? "AI gợi ý" : "Gợi ý lại"}
            </Button>
          )}
        </div>
      </div>
      <ImpactConfirm
        open={pending !== null}
        onClose={() => setPending(null)}
        title="Đổi góc tiếp cận?"
        lead={`Từ “${name(state.approachKey)}” sang “${pending === null ? "" : name(pending)}”.`}
        list={[
          "Các ô khung câu chuyện do AI viết được viết lại theo góc mới; ô bạn tự viết giữ nguyên",
          "Câu chuyện về chờ duyệt",
          "Bảng dữ kiện giữ nguyên",
        ]}
        undo="Chọn lại góc cũ thì AI viết lại các ô một lần nữa."
        choices={[
          {
            label: "Đổi góc",
            variant: "impact",
            onChoose: () => {
              if (pending !== null) void story.selectApproach(pending, true);
            },
            testId: "story-approach-confirm",
          },
        ]}
        testId="story-approach-dialog"
      />
    </Card>
  );
}
