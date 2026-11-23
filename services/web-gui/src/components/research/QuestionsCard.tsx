import { useEffect, useRef, useState } from "react";
import {
  describeAiRequest,
  generateResearchQuestions,
  type QuestionCoverage,
  type ResearchBrief,
  type ResearchIdea,
  type ResearchQuestion,
} from "../../api/client";
import { AiModelSelect } from "../AiModelSelect";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { isOpen, useAiRuns } from "../airun/AiRunsContext";
import { Button, Card, TextInput } from "../ui";
import unitStyles from "../units/units.module.css";
import { CoverageChip } from "./CoverageChip";
import { FollowUpPrompt } from "./FollowUpPrompt";
import { ImpactConfirm } from "./ImpactConfirm";
import { QuestionsRunPreview } from "./QuestionsRunPreview";
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

interface QuestionsCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  coverage: QuestionCoverage[];
  /** Ids of the questions that have a fact whose link is still being checked. */
  checking: ReadonlySet<string>;
  /** The brief the questions come from changed since the AI wrote them. */
  stale: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
  /** The approved idea and brief the questions are written from. */
  idea: ResearchIdea | null;
  brief: ResearchBrief | null;
  /** Reads the idea and brief again from the server. */
  onReloadInput: () => Promise<void>;
  onAdd: (text: string) => Promise<boolean>;
  onSave: (id: string, text: string) => Promise<boolean>;
  onDelete: (id: string) => Promise<boolean>;
}

/**
 * Section 1 of the research step: the questions to look up, each with how well
 * it is sourced. The AI writes them from the brief and nobody approves them;
 * every question is a field the Creator edits, deletes (after a confirmation)
 * or adds to. One button runs the AI on the step's channel and shows the new
 * questions before they are stored. Regenerating keeps the questions the
 * Creator wrote, unless they agree to replace those too.
 */
export function QuestionsCard({
  projectId,
  questions,
  coverage,
  checking,
  stale,
  locked,
  idea,
  brief,
  onReloadInput,
  onAdd,
  onSave,
  onDelete,
}: QuestionsCardProps) {
  const [adding, setAdding] = useState(false);
  const [draft, setDraft] = useState("");
  const [deleting, setDeleting] = useState<ResearchQuestion | null>(null);
  // Regenerating while the Creator has questions of their own asks first whether to replace those too.
  const [regenerating, setRegenerating] = useState(false);
  // The AI run dialog; `replaceEdited` is what the run in it does with the Creator's own questions.
  const [runOpen, setRunOpen] = useState(false);
  const [replaceEdited, setReplaceEdited] = useState(false);
  const { runs } = useAiRuns();
  const runPath = `/v1/projects/${projectId}/research/questions/generate`;
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
    if (isOpen(runs[runPath])) setRunOpen(true);
    else if (mine.length > 0) setRegenerating(true);
    else {
      setReplaceEdited(false);
      setRunOpen(true);
    }
  }

  function chooseGenerate(replace: boolean) {
    setReplaceEdited(replace);
    setRunOpen(true);
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
              Chưa có câu hỏi nào. AI sinh từ brief, hoặc bạn tự thêm.
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
          {!locked && <AiModelSelect projectId={projectId} step="research" label="Model AI sinh câu hỏi" testId="research-model" />}
          <span className={styles.grow} />
          <AiRunPill runPath={runPath} onOpen={() => setRunOpen(true)} testId="question-generate-run" />
          <Button
            variant={mine.length > 0 && questions.length > 0 ? "impact" : "run"}
            className={styles.sm}
            onClick={startGenerate}
            disabled={locked}
            data-testid="question-generate"
          >
            {generateLabel}
          </Button>
        </div>
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
          { label: "Giữ câu của tôi", variant: "run", onChoose: () => chooseGenerate(false), testId: "question-generate-keep" },
          { label: "Thay cả câu tôi đã sửa", variant: "impact", onChoose: () => chooseGenerate(true), testId: "question-generate-replace" },
        ]}
        testId="question-generate-dialog"
      />
      <AiRunDialog
        open={runOpen}
        onClose={() => setRunOpen(false)}
        title={questions.length === 0 ? "Sinh câu hỏi nghiên cứu" : "Sinh lại câu hỏi nghiên cứu"}
        runPath={runPath}
        describe={(note) => describeAiRequest((external) => generateResearchQuestions(projectId, { replaceEdited, note }, { external }))}
        renderPreview={(p) => <QuestionsRunPreview preview={p} />}
        impact={
          <p className={styles.hint}>
            {replaceEdited
              ? "AI thay mọi câu hỏi, kể cả câu bạn đã viết hoặc sửa. Câu mới có mã mới; bảng dữ kiện giữ nguyên."
              : "AI thay các câu hỏi do AI viết bằng câu mới (mã mới, không dùng lại mã cũ); câu bạn đã viết hoặc sửa được giữ. Bảng dữ kiện giữ nguyên."}
          </p>
        }
        onApplied={onReloadInput}
        testId="question-generate-run-dialog"
      />
    </Card>
  );
}
