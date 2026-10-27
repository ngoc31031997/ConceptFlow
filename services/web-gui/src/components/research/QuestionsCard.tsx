import { useEffect, useRef, useState } from "react";
import { generateResearchQuestions, type QuestionCoverage, type ResearchBrief, type ResearchIdea, type ResearchQuestion } from "../../api/client";
import type { OperationRun } from "../../hooks/useOperationRun";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { useDeepSeekRun } from "../../hooks/useDeepSeekRun";
import { AiBridgeRunStatus } from "../AiBridgeRunStatus";
import { ExternalAiDialog } from "../ExternalAiDialog";
import { Button, Card, TextInput } from "../ui";
import unitStyles from "../units/units.module.css";
import { CoverageChip } from "./CoverageChip";
import { FollowUpPrompt } from "./FollowUpPrompt";
import { ImpactConfirm } from "./ImpactConfirm";
import { ResearchInputModal } from "./ResearchInputModal";
import styles from "./research.module.css";

interface QuestionRowProps {
  question: ResearchQuestion;
  coverage: QuestionCoverage | undefined;
  /** A link of one of the question's facts is still being checked. */
  checking: boolean;
  disabled: boolean;
  onSave: (id: string, text: string) => void;
  onDelete: (question: ResearchQuestion) => void;
}

/** One question as a field: it is saved when the Creator leaves it with a changed, non-empty text. */
function QuestionRow({ question, coverage, checking, disabled, onSave, onDelete }: QuestionRowProps) {
  const [value, setValue] = useState(question.text);
  useEffect(() => setValue(question.text), [question.text]);

  function commit() {
    const text = value.trim();
    if (text === question.text) return;
    if (text === "") {
      setValue(question.text);
      return;
    }
    onSave(question.id, text);
  }

  return (
    <div className={styles.qrow} data-testid={`question-${question.id}`}>
      <span className={styles.qid}>{question.id}</span>
      <TextInput
        className={styles.qinput}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") e.currentTarget.blur();
        }}
        placeholder="Nhập câu hỏi cần tra"
        aria-label={`Câu hỏi ${question.id}`}
        disabled={disabled}
        data-testid={`question-input-${question.id}`}
      />
      <span className={styles.qmeta}>
        {question.mine && (
          <span className={unitStyles.mine} title="AI giữ nguyên câu này khi sinh lại" data-testid={`question-mine-${question.id}`}>
            ✎ Bạn viết
          </span>
        )}
        {question.stale && (
          <span className={`${styles.chip} ${styles.chipInfo}`} data-testid={`question-stale-${question.id}`}>
            ↻ cần sinh lại
          </span>
        )}
        <CoverageChip coverage={coverage} checking={checking} testId={`coverage-chip-${question.id}`} />
        {coverage && !checking && coverage.badFacts.length > 0 && (
          <span className={styles.warn} data-testid={`coverage-bad-${question.id}`}>
            {coverage.badFacts.join(", ")} không khớp
          </span>
        )}
      </span>
      <Button
        variant="danger"
        className={styles.sm}
        onClick={() => onDelete(question)}
        disabled={disabled}
        aria-label={`Xoá câu hỏi ${question.id}`}
        data-testid={`question-delete-${question.id}`}
      >
        ×
      </Button>
    </div>
  );
}

type Engine = "system" | "deepseek" | "outside";

const ENGINES: { id: Engine; label: string }[] = [
  { id: "system", label: "AI hệ thống" },
  { id: "deepseek", label: "DeepSeek" },
  { id: "outside", label: "AI ngoài" },
];

interface QuestionsCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  coverage: QuestionCoverage[];
  /** Ids of the questions that have a fact whose link is still being checked. */
  checking: ReadonlySet<string>;
  /** The brief the questions come from changed since the AI wrote them. */
  stale: boolean;
  /** A language model is configured, so the AI can write the questions. */
  aiAvailable: boolean;
  running: boolean;
  run: OperationRun;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
  /** The approved idea and brief the questions are written from. */
  idea: ResearchIdea | null;
  brief: ResearchBrief | null;
  /** Reads the idea and brief again from the server. */
  onReloadInput: () => Promise<void>;
  onGenerate: (replaceEdited: boolean) => Promise<boolean>;
  onAdd: (text: string) => Promise<boolean>;
  onSave: (id: string, text: string) => Promise<boolean>;
  onDelete: (id: string) => Promise<boolean>;
}

/**
 * Section 1 of the research step: the questions to look up, each with how well
 * it is sourced. The AI writes them from the brief and nobody approves them;
 * every question is a field the Creator edits, deletes (after a confirmation)
 * or adds to. One button generates them on the engine picked beside it: the
 * system's AI, DeepSeek or an AI outside. Regenerating keeps the questions the
 * Creator wrote, unless they agree to replace those too.
 */
export function QuestionsCard({
  projectId,
  questions,
  coverage,
  checking,
  stale,
  aiAvailable,
  running,
  run,
  locked,
  idea,
  brief,
  onReloadInput,
  onGenerate,
  onAdd,
  onSave,
  onDelete,
}: QuestionsCardProps) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [deleting, setDeleting] = useState<ResearchQuestion | null>(null);
  // Regenerating while the Creator has questions of their own asks first, for the system's AI or an AI outside.
  const [regenerating, setRegenerating] = useState<"system" | "outside" | "deepseek" | null>(null);
  // The dialog of the AI outside the system; the value is whether it replaces the Creator's own questions.
  const [outside, setOutside] = useState<boolean | null>(null);
  // DeepSeek's answer the step could not read, shown in that dialog for the Creator to fix.
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  // Whether the DeepSeek run in progress replaces the Creator's own questions.
  const deepSeekReplace = useRef(false);
  const deepSeek = useDeepSeekRun({
    run: (external) => generateResearchQuestions(projectId, { replaceEdited: deepSeekReplace.current }, { external }),
    onDone: onReloadInput,
    onUnreadable: (reply, message) => {
      setUnreadable({ reply, message });
      setOutside(deepSeekReplace.current);
    },
  });
  const [pickedEngine, setPickedEngine] = useState<Engine | null>(null);
  const engineAvailable: Record<Engine, boolean> = { system: aiAvailable, deepseek: deepSeek.available, outside: true };
  // The Creator's pick, or the first engine that can run when that one cannot.
  const engine: Engine = pickedEngine && engineAvailable[pickedEngine] ? pickedEngine : (ENGINES.find((e) => engineAvailable[e.id])?.id ?? "outside");
  // The dialog that shows what the AI reads.
  const [inputOpen, setInputOpen] = useState(false);
  const [inputLoading, setInputLoading] = useState(false);
  // The row for a new question takes focus as it appears.
  const newRow = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (adding) newRow.current?.querySelector("input")?.focus();
  }, [adding]);

  const mine = questions.filter((q) => q.mine);
  const coverageOf = (id: string) => coverage.find((c) => c.question === id);

  async function finishAdding() {
    const text = draft.trim();
    setAdding(false);
    setDraft("");
    if (text) await onAdd(text);
  }

  // The input is read again each time the dialog opens, so it shows what the AI will get.
  function openInput() {
    setInputOpen(true);
    setInputLoading(true);
    void onReloadInput().finally(() => setInputLoading(false));
  }

  function startOutside(replaceEdited: boolean) {
    setUnreadable(null);
    setOutside(replaceEdited);
  }

  function startDeepSeek(replaceEdited: boolean) {
    deepSeekReplace.current = replaceEdited;
    void deepSeek.start();
  }

  function startGenerate(via: "system" | "outside" | "deepseek") {
    if (mine.length > 0) setRegenerating(via);
    else if (via === "system") void onGenerate(false);
    else if (via === "outside") startOutside(false);
    else startDeepSeek(false);
  }

  function chooseGenerate(replaceEdited: boolean) {
    if (regenerating === "outside") startOutside(replaceEdited);
    else if (regenerating === "deepseek") startDeepSeek(replaceEdited);
    else void onGenerate(replaceEdited);
  }

  const generateLabel = questions.length === 0 ? "Sinh câu hỏi từ brief" : "Sinh lại từ brief";

  return (
    <Card data-testid="research-questions-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <div className={styles.titleGroup}>
            <h3>① Câu hỏi cần tra</h3>
            <Button
              variant="view"
              className={styles.sm}
              onClick={openInput}
              title="Xem ý tưởng và brief đã duyệt mà AI đọc để sinh câu hỏi"
              data-testid="research-input-open"
            >
              Xem đầu vào của AI
            </Button>
          </div>
        </div>
        {stale && (
          <p className={styles.banner} role="status" data-testid="research-questions-stale">
            Brief vừa đổi: câu hỏi nghiên cứu cần sinh lại theo brief mới. Dữ kiện đã có giữ nguyên.
          </p>
        )}
        <div className={styles.questions}>
          {questions.length === 0 && !adding && (
            <span className={styles.hint} data-testid="research-questions-empty">
              Chưa có câu hỏi nào. {aiAvailable ? "AI sinh từ brief, hoặc bạn tự thêm." : "Chưa cấu hình AI nên bạn tự thêm câu hỏi."}
            </span>
          )}
          {questions.map((q) => (
            <QuestionRow
              key={q.id}
              question={q}
              coverage={coverageOf(q.id)}
              checking={checking.has(q.id)}
              disabled={locked}
              onSave={(id, text) => void onSave(id, text)}
              onDelete={setDeleting}
            />
          ))}
          {adding && (
            <div className={styles.qrow} ref={newRow} data-testid="question-new">
              <span className={styles.qid}>+</span>
              <TextInput
                className={styles.qinput}
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                onBlur={() => void finishAdding()}
                onKeyDown={(e) => {
                  if (e.key === "Enter") e.currentTarget.blur();
                }}
                placeholder="Nhập câu hỏi cần tra"
                aria-label="Câu hỏi mới"
                data-testid="question-new-input"
              />
            </div>
          )}
        </div>
        <div className={styles.actions}>
          <Button variant="ghost" className={styles.sm} onClick={() => setAdding(true)} disabled={locked || adding} data-testid="question-add">
            + Thêm câu hỏi
          </Button>
          <span className={styles.grow} />
          <FollowUpPrompt projectId={projectId} questions={questions} coverage={coverage} checking={checking} />
        </div>
        <div className={styles.engineBar}>
          <span className={styles.hint} id="question-engine-label">
            Sinh bằng
          </span>
          <div className={styles.seg} role="radiogroup" aria-labelledby="question-engine-label">
            {ENGINES.filter((e) => e.id !== "deepseek" || deepSeek.available).map((e) => (
              <button
                key={e.id}
                type="button"
                role="radio"
                aria-checked={engine === e.id}
                className={engine === e.id ? `${styles.segItem} ${styles.segOn}` : styles.segItem}
                onClick={() => setPickedEngine(e.id)}
                disabled={locked || running || deepSeek.busy || !engineAvailable[e.id]}
                title={engineAvailable[e.id] ? undefined : "Chưa cấu hình AI"}
                data-testid={`question-engine-${e.id}`}
              >
                {e.label}
              </button>
            ))}
          </div>
          {engine === "system" && !locked && (
            <AiModelSelect projectId={projectId} step="research" label="Model AI sinh câu hỏi" testId="research-model" />
          )}
          <span className={styles.grow} />
          {deepSeek.stage !== null ? (
            <AiBridgeRunStatus stage={deepSeek.stage} modes={deepSeek.modes} onCancel={deepSeek.cancel} testId="question-generate-deepseek" />
          ) : (
            <Button
              variant={mine.length > 0 && questions.length > 0 ? "impact" : "run"}
              className={styles.sm}
              onClick={() => startGenerate(engine)}
              disabled={locked || running || deepSeek.busy}
              data-testid="question-generate"
            >
              {deepSeek.busy ? "Đang chạy…" : generateLabel}
            </Button>
          )}
        </div>
        {deepSeek.error && (
          <p className={styles.error} role="alert" data-testid="question-generate-deepseek-error">
            {deepSeek.error}
          </p>
        )}
        <AiOperationCard run={run} active={running} testId="research-progress" title="AI đang sinh câu hỏi" />
      </div>

      <ResearchInputModal
        open={inputOpen}
        onClose={() => setInputOpen(false)}
        loading={inputLoading}
        idea={idea}
        brief={brief}
      />
      <ImpactConfirm
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        title={deleting ? `Xoá câu hỏi ${deleting.id}?` : ""}
        lead={deleting ? `“${deleting.text}”` : undefined}
        list={
          deleting
            ? [
                `Dữ kiện đã có của ${deleting.id} giữ nguyên`,
                "Câu hỏi này không còn được tính độ phủ và không có trong prompt bổ sung",
              ]
            : []
        }
        undo="Không khôi phục được; muốn hỏi lại thì thêm câu hỏi mới."
        choices={[
          {
            label: "Xoá câu hỏi",
            variant: "danger",
            onChoose: () => deleting && void onDelete(deleting.id),
            testId: "question-delete-confirm",
          },
        ]}
        testId="question-delete-dialog"
      />
      <ImpactConfirm
        open={regenerating !== null}
        onClose={() => setRegenerating(null)}
        title="Sinh lại câu hỏi từ brief?"
        lead={`Câu hỏi bạn đã viết hoặc sửa: ${mine.map((q) => q.id).join(", ")}.`}
        list={[
          "AI thay các câu hỏi còn lại bằng câu mới (mã mới, không dùng lại mã cũ)",
          "Có thể giữ câu của bạn, hoặc thay cả những câu đó",
          "Bảng dữ kiện giữ nguyên",
        ]}
        undo="Chữ bạn đã sửa không lấy lại được nếu chọn thay."
        choices={[
          { label: "Giữ câu của tôi", variant: "run", onChoose: () => chooseGenerate(false), testId: "question-generate-keep" },
          { label: "Thay cả câu tôi đã sửa", variant: "impact", onChoose: () => chooseGenerate(true), testId: "question-generate-replace" },
        ]}
        testId="question-generate-dialog"
      />
      <ExternalAiDialog
        open={outside !== null}
        onClose={() => setOutside(null)}
        title="Câu hỏi nghiên cứu · AI ngoài"
        run={(external) => generateResearchQuestions(projectId, { replaceEdited: outside === true }, { external })}
        onDone={onReloadInput}
        initialReply={unreadable?.reply}
        initialError={unreadable?.message}
        testId="question-generate-external-dialog"
      />
    </Card>
  );
}
