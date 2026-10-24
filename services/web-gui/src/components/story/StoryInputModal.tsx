import type { ResearchBrief, StoryFact } from "../../api/client";
import { formatClock } from "../../utils/clockDuration";
import cardStyles from "../CreativeBriefCard.module.css";
import ideaStyles from "../idea/ProjectIdeaCard.module.css";
import styles from "../research/research.module.css";
import { Button, Dialog } from "../ui";
import { FactTags } from "./FactTags";

interface StoryInputModalProps {
  open: boolean;
  onClose: () => void;
  /** The input is being loaded again from the server. */
  loading: boolean;
  brief: ResearchBrief | null;
  facts: StoryFact[];
  researchSkipped: boolean;
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
 * What the story is written from: the brief approved on step 2 and the facts
 * kept on step 3, read again from the server each time the dialog opens so the
 * Creator sees the data the AI receives.
 */
export function StoryInputModal({ open, onClose, loading, brief, facts, researchSkipped }: StoryInputModalProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title="Đầu vào của bước Câu chuyện"
      testId="story-input-dialog"
      footer={
        <Button variant="ghost" onClick={onClose} data-testid="story-input-cancel">
          Đóng
        </Button>
      }
    >
      <div className={styles.inputBody}>
        <p className={styles.hint}>Dữ liệu đã duyệt ở bước 2 và bước 3, đọc lại từ máy chủ. AI viết câu chuyện từ đúng dữ liệu này.</p>
        {loading && (
          <p className={styles.hint} role="status" data-testid="story-input-loading">
            Đang tải lại…
          </p>
        )}
        {!loading && brief === null && (
          <p className={styles.error} role="alert" data-testid="story-input-no-brief">
            Brief ở bước 2 chưa được duyệt.
          </p>
        )}
        {!loading && brief && (
          <section data-testid="story-input-brief">
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
        {!loading && (
          <section data-testid="story-input-facts">
            <h4>Dữ kiện (bước 3)</h4>
            {facts.length === 0 ? (
              <p className={styles.hint} data-testid="story-input-no-facts">
                {researchSkipped
                  ? "Bước Nghiên cứu đã bỏ qua: không có dữ kiện, AI chỉ nói điều định tính."
                  : "Chưa có dữ kiện nào được giữ."}
              </p>
            ) : (
              <ul className={ideaStyles.points}>
                {facts.map((f) => (
                  <li key={f.id}>
                    <FactTags ids={[f.id]} /> {f.claimVi}
                    {f.must ? " (bắt buộc nói)" : ""}
                  </li>
                ))}
              </ul>
            )}
          </section>
        )}
      </div>
    </Dialog>
  );
}
