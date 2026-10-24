import type { BriefIdea } from "../../api/client";
import cardStyles from "../CreativeBriefCard.module.css";
import ideaStyles from "../idea/ProjectIdeaCard.module.css";
import { Button, Dialog } from "../ui";

interface BriefInputModalProps {
  open: boolean;
  onClose: () => void;
  /** The idea approved on step 1; null while it is not approved. */
  idea: BriefIdea | null;
}

/**
 * What the brief is written from: the idea approved on step 1, as the brief
 * screen loaded it from the server, so the Creator sees the data the AI receives.
 */
export function BriefInputModal({ open, onClose, idea }: BriefInputModalProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      size="lg"
      title="Đầu vào của bước Creative brief"
      testId="brief-input-dialog"
      footer={
        <Button variant="ghost" onClick={onClose} data-testid="brief-input-close">
          Đóng
        </Button>
      }
    >
      {idea ? (
        <section data-testid="brief-input-idea">
          <div className={cardStyles.field}>
            <span className={cardStyles.label}>Câu hỏi trung tâm</span>
            <p className={cardStyles.value} data-testid="brief-idea-question">
              {idea.centralQuestion}
            </p>
          </div>
          <div className={cardStyles.field}>
            <span className={cardStyles.label}>Điểm gây tò mò</span>
            <ul className={ideaStyles.points}>
              {idea.curiosityPoints.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          </div>
          <div className={cardStyles.field}>
            <span className={cardStyles.label}>Phạm vi</span>
            <p className={cardStyles.value}>{idea.scope}</p>
          </div>
        </section>
      ) : (
        <p className={cardStyles.error} role="alert" data-testid="brief-idea-missing">
          Ý tưởng ở bước 1 chưa được duyệt. Duyệt ý tưởng trước: brief được viết từ ý tưởng đã duyệt.
        </p>
      )}
    </Dialog>
  );
}
