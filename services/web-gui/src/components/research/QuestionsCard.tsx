import { useEffect, useRef, useState } from "react";
import type { QuestionCoverage, ResearchBrief, ResearchIdea, ResearchQuestion } from "../../api/client";
import type { OperationRun } from "../../hooks/useOperationRun";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { Button, Card, TextInput } from "../ui";
import unitStyles from "../units/units.module.css";
import { CoverageChip } from "./CoverageChip";
import { ImpactConfirm } from "./ImpactConfirm";
import { ResearchInputModal } from "./ResearchInputModal";
import styles from "./research.module.css";

interface QuestionRowProps {
  question: ResearchQuestion;
  coverage: QuestionCoverage | undefined;
  disabled: boolean;
  onSave: (id: string, text: string) => void;
  onDelete: (question: ResearchQuestion) => void;
}

/** One question as a field: it is saved when the Creator leaves it with a changed, non-empty text. */
function QuestionRow({ question, coverage, disabled, onSave, onDelete }: QuestionRowProps) {
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
        <CoverageChip coverage={coverage} testId={`question-coverage-${question.id}`} />
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

interface QuestionsCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  coverage: QuestionCoverage[];
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
 * Section 1 of the research step: the questions to look up. The AI writes
 * them from the brief and nobody approves them; every question is a field the
 * Creator edits, deletes (after a confirmation) or adds to. Regenerating keeps
 * the questions the Creator wrote, unless they agree to replace those too.
 */
export function QuestionsCard({
  projectId,
  questions,
  coverage,
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
  const [regenerating, setRegenerating] = useState(false);
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

  function startGenerate() {
    if (mine.length > 0) setRegenerating(true);
    else void onGenerate(false);
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
          <span className={styles.hint}>AI tự sinh từ brief · không cần duyệt · sửa, xoá, thêm tuỳ ý</span>
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
        {aiAvailable && !locked && (
          <AiModelSelect projectId={projectId} step="research" label="Model AI sinh câu hỏi" testId="research-model" />
        )}
        <div className={styles.actions}>
          <Button variant="ghost" className={styles.sm} onClick={() => setAdding(true)} disabled={locked || adding} data-testid="question-add">
            + Thêm câu hỏi
          </Button>
          <Button
            variant={mine.length > 0 && questions.length > 0 ? "impact" : "run"}
            className={styles.sm}
            onClick={startGenerate}
            disabled={locked || running || !aiAvailable}
            title={aiAvailable ? undefined : "Chưa cấu hình AI"}
            data-testid="question-generate"
          >
            {generateLabel}
          </Button>
        </div>
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
        open={regenerating}
        onClose={() => setRegenerating(false)}
        title="Sinh lại câu hỏi từ brief?"
        lead={`Câu hỏi bạn đã viết hoặc sửa: ${mine.map((q) => q.id).join(", ")}.`}
        list={[
          "AI thay các câu hỏi còn lại bằng câu mới (mã mới, không dùng lại mã cũ)",
          "Có thể giữ câu của bạn, hoặc thay cả những câu đó",
          "Bảng dữ kiện giữ nguyên",
        ]}
        undo="Chữ bạn đã sửa không lấy lại được nếu chọn thay."
        choices={[
          { label: "Giữ câu của tôi", variant: "run", onChoose: () => void onGenerate(false), testId: "question-generate-keep" },
          { label: "Thay cả câu tôi đã sửa", variant: "impact", onChoose: () => void onGenerate(true), testId: "question-generate-replace" },
        ]}
        testId="question-generate-dialog"
      />
    </Card>
  );
}
