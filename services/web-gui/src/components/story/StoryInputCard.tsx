import { useState } from "react";
import type { StoryState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { StoryInputModal } from "./StoryInputModal";
import styles from "./story.module.css";

interface StoryInputCardProps {
  state: StoryState;
  /** Reads the state again from the server; resolves when it has landed. */
  onReload: () => Promise<void>;
}

/**
 * Section 1 of the story step: what the AI reads (the approved brief and the
 * facts kept on the research step), with a button that shows it read again from
 * the server.
 */
export function StoryInputCard({ state, onReload }: StoryInputCardProps) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  // The input is read again each time the dialog opens, so it shows what the AI will get.
  function openInput() {
    setOpen(true);
    setLoading(true);
    void onReload().finally(() => setLoading(false));
  }

  const factsText = state.researchSkipped
    ? "bước Nghiên cứu đã bỏ qua, không có dữ kiện"
    : `${state.facts.length} dữ kiện đang giữ`;

  return (
    <Card data-testid="story-input-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <div className={researchStyles.titleGroup}>
            <h3>① Đầu vào của AI</h3>
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
        </div>
        <p className={styles.inputSummary} data-testid="story-input-summary">
          {state.briefApproved ? "Brief đã duyệt" : "Brief chưa duyệt"} · {state.researchResolved ? factsText : "bảng dữ kiện chưa duyệt"}
        </p>
      </div>
      <StoryInputModal
        open={open}
        onClose={() => setOpen(false)}
        loading={loading}
        brief={state.brief}
        facts={state.facts}
        researchSkipped={state.researchSkipped}
      />
    </Card>
  );
}
