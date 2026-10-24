import type { ResearchBrief, ResearchIdea } from "../../api/client";
import { formatClock } from "../../utils/clockDuration";
import cardStyles from "../CreativeBriefCard.module.css";
import ideaStyles from "../idea/ProjectIdeaCard.module.css";
import { Button, Dialog } from "../ui";
import styles from "./research.module.css";

interface ResearchInputModalProps {
  open: boolean;
  onClose: () => void;
  /** The input is being loaded again from the server. */
  loading: boolean;
  idea: ResearchIdea | null;
  brief: ResearchBrief | null;
}

function List({ items, ordered }: { items: string[]; ordered?: boolean }) {
  const Tag = ordered ? "ol" : "ul";
  return (
    <Tag className={ideaStyles.points}>
      {items.map((item) => (
        <li key={item}>{item}</li>
      ))}
    </Tag>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className={cardStyles.field}>
      <span className={cardStyles.label}>{label}</span>
      {children}
    </div>
  );
}

/**
 * What the research is planned from: the idea approved on step 1 and the brief
 * approved on step 2, read again from the server each time the dialog opens so
 * the Creator sees the data the AI receives.
 */
export function ResearchInputModal({ open, onClose, loading, idea, brief }: ResearchInputModalProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title="Đầu vào của bước Nghiên cứu"
      testId="research-input-dialog"
      footer={
        <Button variant="ghost" onClick={onClose} data-testid="research-input-cancel">
          Đóng
        </Button>
      }
    >
      <div className={styles.inputBody}>
        <p className={styles.hint}>Dữ liệu đã duyệt ở bước 1 và bước 2, đọc lại từ máy chủ. AI sinh câu hỏi từ đúng dữ liệu này.</p>
        {loading && (
          <p className={styles.hint} role="status" data-testid="research-input-loading">
            Đang tải lại…
          </p>
        )}
        {!loading && idea === null && (
          <p className={styles.error} role="alert" data-testid="research-input-no-idea">
            Ý tưởng ở bước 1 chưa được duyệt.
          </p>
        )}
        {!loading && brief === null && (
          <p className={styles.error} role="alert" data-testid="research-input-no-brief">
            Brief ở bước 2 chưa được duyệt.
          </p>
        )}
        {!loading && idea && (
          <section data-testid="research-input-idea">
            <h4>Ý tưởng (bước 1)</h4>
            <Field label="Câu hỏi trung tâm">
              <p className={cardStyles.value}>{idea.centralQuestion}</p>
            </Field>
            <Field label="Điểm gây tò mò">
              <List items={idea.curiosityPoints} />
            </Field>
            <Field label="Phạm vi">
              <p className={cardStyles.value}>{idea.scope}</p>
            </Field>
          </section>
        )}
        {!loading && brief && (
          <section data-testid="research-input-brief">
            <h4>Brief (bước 2)</h4>
            <Field label="Khán giả">
              <p className={cardStyles.value}>{brief.audience}</p>
            </Field>
            <Field label="Thời lượng">
              <p className={cardStyles.value}>{formatClock(brief.durationSeconds)}</p>
            </Field>
            <Field label="Giọng điệu">
              <p className={cardStyles.value}>{brief.tone}</p>
            </Field>
            <Field label="Mục tiêu học tập">
              <List items={brief.learningObjectives} ordered />
            </Field>
            <Field label="Cần tránh">
              <List items={brief.avoid} />
            </Field>
          </section>
        )}
      </div>
    </Dialog>
  );
}
