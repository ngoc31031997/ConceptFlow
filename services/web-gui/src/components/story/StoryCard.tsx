import { useState } from "react";
import { generateStory, type StoryFieldKey, type StoryPatch, type StoryState } from "../../api/client";
import { useDeepSeekRun } from "../../hooks/useDeepSeekRun";
import type { StoryControls } from "../../hooks/useStory";
import { AiBridgeRunStatus } from "../AiBridgeRunStatus";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { ExternalAiDialog } from "../ExternalAiDialog";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { FactTags } from "./FactTags";
import { StoryArcEditModal } from "./StoryArcEditModal";
import { StoryInputModal } from "./StoryInputModal";
import { StoryTextEditModal } from "./StoryTextEditModal";
import styles from "./story.module.css";

interface StoryCardProps {
  projectId: string;
  story: StoryControls;
  state: StoryState;
  /** A language model is configured, so the AI can write the story. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
  /** Opens "Làm lại kèm ghi chú" for the whole story. */
  onRedo: () => void;
}

type Engine = "system" | "deepseek" | "outside";

const ENGINES: { id: Engine; label: string }[] = [
  { id: "system", label: "AI hệ thống" },
  { id: "deepseek", label: "DeepSeek" },
  { id: "outside", label: "AI ngoài" },
];

const LABELS: Record<StoryFieldKey, string> = {
  central_question: "Câu hỏi trung tâm",
  core_answer: "Câu trả lời cốt lõi",
  narrative_arc: "Mạch kể",
};

function Mine({ field, owned }: { field: StoryFieldKey; owned: boolean }) {
  if (!owned) return null;
  return (
    <span className={unitStyles.mine} title="AI giữ nguyên ô này khi viết lại" data-testid={`story-mine-${field}`}>
      ✎ Bạn viết
    </span>
  );
}

/**
 * The story step in one card: what the AI reads, the central question, the
 * core answer and the narrative arc, the bar that writes the story, and the
 * approval at the foot. Each field reads as text with its own "Sửa tay"
 * (docs/ux-ui-design-rules.md, rule 4); one the Creator wrote is tagged and
 * kept by the AI. With no story yet the bar writes it on the engine picked
 * beside it; with one, "Làm lại kèm ghi chú" opens the redo dialog.
 */
export function StoryCard({ projectId, story, state, aiAvailable, locked, onRedo }: StoryCardProps) {
  const [editing, setEditing] = useState<"central_question" | "core_answer" | "narrative_arc" | null>(null);
  const [inputOpen, setInputOpen] = useState(false);
  const [inputLoading, setInputLoading] = useState(false);
  const [outside, setOutside] = useState(false);
  // DeepSeek's answer the step could not read, shown in the outside-AI dialog for the Creator to fix.
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  const deepSeek = useDeepSeekRun({
    run: (external) => generateStory(projectId, "", { external }),
    onDone: story.reload,
    onUnreadable: (reply, message) => {
      setUnreadable({ reply, message });
      setOutside(true);
    },
  });
  const [pickedEngine, setPickedEngine] = useState<Engine | null>(null);
  const engineAvailable: Record<Engine, boolean> = { system: aiAvailable, deepseek: deepSeek.available, outside: true };
  // The Creator's pick, or the first engine that can run when that one cannot.
  const engine: Engine = pickedEngine && engineAvailable[pickedEngine] ? pickedEngine : (ENGINES.find((e) => engineAvailable[e.id])?.id ?? "outside");
  const s = state.story;
  const open = state.briefApproved && state.researchResolved;
  const busy = locked || story.busy || story.writing || deepSeek.busy;
  const owns = (key: StoryFieldKey) => s?.creatorFields.includes(key) ?? false;
  const approved = s?.status === "approved";

  const chip = story.writing ? (
    <UnitStatusChip state="running" testId="story-status" />
  ) : (
    <UnitStatusChip state={state.status} testId="story-status" />
  );

  async function save(patch: StoryPatch) {
    await story.edit(patch);
  }

  // The input is read again each time the dialog opens, so it shows what the AI will get.
  function openInput() {
    setInputOpen(true);
    setInputLoading(true);
    void story.reload().finally(() => setInputLoading(false));
  }

  function generate() {
    if (engine === "system") void story.generate("");
    else if (engine === "deepseek") void deepSeek.start();
    else {
      setUnreadable(null);
      setOutside(true);
    }
  }

  const factsText = state.researchSkipped ? "bước Nghiên cứu đã bỏ qua, không có dữ kiện" : `${state.facts.length} dữ kiện đang giữ`;

  const textField = (key: "central_question" | "core_answer", value: string) => (
    <div className={styles.field} data-testid={`story-field-${key}`}>
      <span className={styles.fieldHead}>
        {LABELS[key]}
        <Mine field={key} owned={owns(key)} />
      </span>
      {value.trim() ? <p className={styles.text}>{value}</p> : <p className={`${styles.text} ${styles.empty}`}>(trống)</p>}
      <div className={styles.fieldActions}>
        <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(key)} disabled={busy} data-testid={`story-edit-${key}`}>
          Sửa tay
        </Button>
      </div>
    </div>
  );

  return (
    <Card data-testid="story-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Câu chuyện</h3>
          {chip}
          <span className={researchStyles.grow} />
          {open && !s && (
            <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing("central_question")} disabled={busy} data-testid="story-write-by-hand">
              Tự viết từng ô
            </Button>
          )}
        </div>
        <div className={styles.inputLine}>
          <span data-testid="story-input-summary">
            <strong>Đầu vào:</strong> {state.briefApproved ? "Brief đã duyệt" : "Brief chưa duyệt"} · {state.researchResolved ? factsText : "bảng dữ kiện chưa duyệt"}
          </span>
          <span className={researchStyles.grow} />
          <Button
            variant="view"
            className={researchStyles.sm}
            onClick={openInput}
            title="Xem brief và dữ kiện đã duyệt mà AI đọc để viết câu chuyện"
            data-testid="story-input-open"
          >
            Xem đầu vào của AI
          </Button>
        </div>
        {!open && (
          <p className={researchStyles.hint} data-testid="story-not-open">
            {!state.briefApproved ? "Duyệt brief ở bước Creative brief trước." : "Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước."}
          </p>
        )}
        {open && !s && (
          <>
            <p className={researchStyles.hint}>
              {aiAvailable
                ? "Chưa có câu chuyện. AI chọn góc kể và xây mạch từ brief và dữ kiện, hoặc bạn tự viết từng ô."
                : "Chưa có câu chuyện. Chưa cấu hình AI nên bạn tự viết từng ô rồi duyệt."}
            </p>
          </>
        )}
        {s && (state.briefStale || state.researchStale) && (
          <p className={researchStyles.hint} data-testid="story-stale">
            ↻ {state.briefStale && state.researchStale ? "Brief và bảng dữ kiện" : state.briefStale ? "Brief" : "Bảng dữ kiện"} đã đổi sau khi câu chuyện được viết. Làm lại kèm ghi chú nếu cần viết lại.
          </p>
        )}
        <AiOperationCard run={story.run} active={story.writing} testId="story-progress" title="AI đang viết câu chuyện" />
        {s && (
          <>
            {textField("central_question", s.centralQuestion)}
            {textField("core_answer", s.coreAnswer)}
            <div className={styles.field} data-testid="story-field-narrative_arc">
              <span className={styles.fieldHead}>
                {LABELS.narrative_arc}
                <Mine field="narrative_arc" owned={owns("narrative_arc")} />
              </span>
              {s.narrativeArc.length === 0 ? (
                <p className={`${styles.text} ${styles.empty}`}>(trống)</p>
              ) : (
                <ol className={styles.arc} data-testid="story-arc">
                  {s.narrativeArc.map((b, i) => (
                    <li key={i}>
                      <span className={styles.beatName}>{b.beat}</span>: {b.text}
                      <FactTags ids={b.facts} />
                    </li>
                  ))}
                </ol>
              )}
              <div className={styles.fieldActions}>
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing("narrative_arc")} disabled={busy} data-testid="story-edit-narrative_arc">
                  Sửa tay
                </Button>
              </div>
            </div>
          </>
        )}
        {open && !approved && (
          <div className={styles.engineBar}>
            {s ? (
              <Button variant="ghost" onClick={onRedo} disabled={busy} data-testid="story-redo-all">
                Làm lại kèm ghi chú
              </Button>
            ) : (
              <>
                <span className={researchStyles.hint} id="story-engine-label">
                  Sinh bằng
                </span>
                <div className={styles.seg} role="radiogroup" aria-labelledby="story-engine-label">
                  {ENGINES.filter((e) => e.id !== "deepseek" || deepSeek.available).map((e) => (
                    <button
                      key={e.id}
                      type="button"
                      role="radio"
                      aria-checked={engine === e.id}
                      className={engine === e.id ? `${styles.segItem} ${styles.segOn}` : styles.segItem}
                      onClick={() => setPickedEngine(e.id)}
                      disabled={busy || !engineAvailable[e.id]}
                      title={engineAvailable[e.id] ? undefined : "Chưa cấu hình AI"}
                      data-testid={`story-engine-${e.id}`}
                    >
                      {e.label}
                    </button>
                  ))}
                </div>
                {engine === "system" && !locked && (
                  <AiModelSelect projectId={projectId} step="story_frame" label="Model AI cho bước Câu chuyện" testId="story-model" />
                )}
                <span className={researchStyles.grow} />
                {deepSeek.stage !== null ? (
                  <AiBridgeRunStatus stage={deepSeek.stage} modes={deepSeek.modes} onCancel={deepSeek.cancel} testId="story-generate-deepseek" />
                ) : (
                  <Button variant="run" onClick={generate} disabled={busy} data-testid="story-generate">
                    {deepSeek.busy ? "Đang chạy…" : "Viết câu chuyện"}
                  </Button>
                )}
              </>
            )}
          </div>
        )}
        {deepSeek.error && (
          <p className={researchStyles.error} role="alert" data-testid="story-generate-deepseek-error">
            {deepSeek.error}
          </p>
        )}
        {story.actionError && (
          <p className={researchStyles.error} role="alert" data-testid="story-error">
            {story.actionError}
          </p>
        )}
        <div className={styles.foot}>
          <span className={researchStyles.hint}>
            {approved ? "Bỏ duyệt để sửa hoặc làm lại." : s ? "Duyệt khi cả câu chuyện ổn. Ô bạn tự viết được AI giữ nguyên." : "Cần có câu chuyện mới duyệt được."}
          </span>
          {approved ? (
            <Button variant="ghost" onClick={() => void story.unapprove()} disabled={busy} data-testid="story-unapprove">
              Bỏ duyệt
            </Button>
          ) : (
            <Button variant="primary" onClick={() => void story.approve()} disabled={busy || !s} data-testid="story-approve">
              Duyệt câu chuyện
            </Button>
          )}
        </div>
      </div>

      <StoryInputModal
        open={inputOpen}
        onClose={() => setInputOpen(false)}
        loading={inputLoading}
        brief={state.brief}
        facts={state.facts}
        researchSkipped={state.researchSkipped}
      />
      <ExternalAiDialog
        open={outside}
        onClose={() => setOutside(false)}
        title="Câu chuyện · AI ngoài"
        run={(external) => generateStory(projectId, "", { external })}
        onDone={story.reload}
        initialReply={unreadable?.reply}
        initialError={unreadable?.message}
        testId="story-generate-external-dialog"
      />

      <StoryTextEditModal
        open={editing === "central_question"}
        onClose={() => setEditing(null)}
        label={LABELS.central_question}
        value={s?.centralQuestion ?? ""}
        rows={2}
        approved={approved}
        onSave={(text) => save({ centralQuestion: text })}
        testId="story-edit-question-dialog"
      />
      <StoryTextEditModal
        open={editing === "core_answer"}
        onClose={() => setEditing(null)}
        label={LABELS.core_answer}
        value={s?.coreAnswer ?? ""}
        rows={4}
        approved={approved}
        onSave={(text) => save({ coreAnswer: text })}
        testId="story-edit-answer-dialog"
      />
      <StoryArcEditModal
        open={editing === "narrative_arc"}
        onClose={() => setEditing(null)}
        arc={s?.narrativeArc ?? []}
        approved={approved}
        onSave={(arc) => save({ narrativeArc: arc })}
      />
    </Card>
  );
}
