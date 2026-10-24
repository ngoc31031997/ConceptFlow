import { useState } from "react";
import type { BeatState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import storyStyles from "../story/story.module.css";
import { Button, Card } from "../ui";
import { BeatInputModal } from "./BeatInputModal";

interface BeatInputCardProps {
  state: BeatState;
  /** Reads the state again from the server; resolves when it has landed. */
  onReload: () => Promise<void>;
}

/**
 * The top of the beat sheet step: what the AI reads (the approved brief, the
 * kept facts and the approved story), with a button that shows it read again
 * from the server.
 */
export function BeatInputCard({ state, onReload }: BeatInputCardProps) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  // The input is read again each time the dialog opens, so it shows what the AI will get.
  function openInput() {
    setOpen(true);
    setLoading(true);
    void onReload().finally(() => setLoading(false));
  }

  return (
    <Card data-testid="beats-input-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <div className={researchStyles.titleGroup}>
            <h3>Đầu vào của AI</h3>
            <Button
              variant="view"
              className={researchStyles.sm}
              onClick={openInput}
              title="Xem brief, dữ kiện và câu chuyện đã duyệt mà AI đọc để viết các beat"
              data-testid="beats-input-open"
            >
              Xem đầu vào của AI
            </Button>
          </div>
        </div>
        <p className={storyStyles.inputSummary} data-testid="beats-input-summary">
          {state.input.briefApproved ? "Brief đã duyệt" : "Brief chưa duyệt"} · {state.facts.length} dữ kiện đang giữ · Câu chuyện đã duyệt
        </p>
      </div>
      <BeatInputModal open={open} onClose={() => setOpen(false)} loading={loading} input={state.input} facts={state.facts} />
    </Card>
  );
}
