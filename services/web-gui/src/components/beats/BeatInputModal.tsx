import type { BeatFact, BeatInput } from "../../api/client";
import cardStyles from "../CreativeBriefCard.module.css";
import ideaStyles from "../idea/ProjectIdeaCard.module.css";
import styles from "../research/research.module.css";
import { FactTags } from "../story/FactTags";
import { Button, Dialog } from "../ui";

interface BeatInputModalProps {
  open: boolean;
  onClose: () => void;
  /** The input is being loaded again from the server. */
  loading: boolean;
  input: BeatInput;
  facts: BeatFact[];
}

/** A block of text exactly as the prompt gets it, line breaks kept. */
const BLOCK = { whiteSpace: "pre-wrap" } as const;

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className={cardStyles.field}>
      <span className={cardStyles.label}>{label}</span>
      {children}
    </div>
  );
}

/**
 * What the beats are written from: the topic, the brief approved on step 2, the
 * facts kept on step 3 and the story approved on step 4, read again from the
 * server each time the dialog opens so the Creator sees the data the AI receives.
 */
export function BeatInputModal({ open, onClose, loading, input, facts }: BeatInputModalProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title="Đầu vào của bước Beat sheet"
      testId="beats-input-dialog"
      footer={
        <Button variant="ghost" onClick={onClose} data-testid="beats-input-close">
          Đóng
        </Button>
      }
    >
      <div className={styles.inputBody}>
        <p className={styles.hint}>
          Dữ liệu đã duyệt ở các bước 2, 3 và 4, đọc lại từ máy chủ. Bộ khung dựng từ câu chuyện bằng code, không gọi AI; khi AI viết nội dung các beat, nó đọc đúng dữ liệu này.
        </p>
        {loading && (
          <p className={styles.hint} role="status" data-testid="beats-input-loading">
            Đang tải lại…
          </p>
        )}
        {!loading && (
          <>
            <section data-testid="beats-input-topic">
              <h4>Đề tài</h4>
              <p className={cardStyles.value}>{input.topic}</p>
            </section>
            <section data-testid="beats-input-brief">
              <h4>Brief (bước 2)</h4>
              {input.briefApproved ? (
                <p className={cardStyles.value} style={BLOCK}>
                  {input.brief}
                </p>
              ) : (
                <p className={styles.error} role="alert" data-testid="beats-input-no-brief">
                  Brief ở bước 2 chưa được duyệt.
                </p>
              )}
            </section>
            <section data-testid="beats-input-facts">
              <h4>Dữ kiện (bước 3)</h4>
              {facts.length === 0 ? (
                <p className={styles.hint} data-testid="beats-input-no-facts">
                  Không có dữ kiện nào được giữ: AI chỉ nói điều định tính.
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
            <section data-testid="beats-input-story">
              <h4>Câu chuyện (bước 4)</h4>
              <Field label="Các trường AI đọc">
                <p className={cardStyles.value} style={BLOCK}>
                  {input.story}
                </p>
              </Field>
            </section>
          </>
        )}
      </div>
    </Dialog>
  );
}
