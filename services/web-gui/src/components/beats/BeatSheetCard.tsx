import { useState } from "react";
import type { BeatState } from "../../api/client";
import type { BeatsControls } from "../../hooks/useBeats";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { BeatInputModal } from "./BeatInputModal";
import styles from "./beats.module.css";

interface BeatSheetCardProps {
  projectId: string;
  state: BeatState;
  beats: BeatsControls;
  /** A language model is configured, so the AI can write the beats. */
  aiAvailable: boolean;
  /** The step cannot be changed now, or an action is running. */
  busy: boolean;
  /** Beats that have a goal and are not approved yet. */
  pendingWithGoal: number;
  onRewrite: () => void;
  onRebuild: () => void;
}

/**
 * The head of the beat sheet: what the AI reads, the shape of the sheet (a
 * strip with one segment per beat, as wide as its target length) and the checks
 * over the whole sheet, which only warn, and the actions that apply to the
 * whole sheet. While the skeleton is reviewed no length is decided yet, so the
 * strip is left out and the actions are "Dựng lại bộ khung" and "Duyệt bộ khung".
 */
export function BeatSheetCard({ projectId, state, beats, aiAvailable, busy, pendingWithGoal, onRewrite, onRebuild }: BeatSheetCardProps) {
  const [inputOpen, setInputOpen] = useState(false);
  const [inputLoading, setInputLoading] = useState(false);
  const skeleton = state.phase === "skeleton";
  const writingSheet = beats.running === "sheet";
  const writingAdd = beats.running === "add";
  const approved = state.beats.filter((b) => b.status === "approved").length;

  // The input is read again each time the dialog opens, so it shows what the AI will get.
  function openInput() {
    setInputOpen(true);
    setInputLoading(true);
    void beats.reload().finally(() => setInputLoading(false));
  }

  return (
    <Card data-testid="beats-sheet-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Beat sheet</h3>
          {skeleton ? (
            <span className={researchStyles.chip}>Chờ duyệt bộ khung</span>
          ) : (
            <UnitStatusChip state={state.status} testId="beats-status" />
          )}
          {!skeleton && (
            <span className={researchStyles.hint}>
              {approved}/{state.beats.length} beat đã duyệt
            </span>
          )}
        </div>
        <div className={styles.inputLine}>
          <span data-testid="beats-input-summary">
            <strong>Đầu vào:</strong> {state.input.briefApproved ? "Brief đã duyệt" : "Brief chưa duyệt"} · {state.facts.length} dữ kiện đang giữ · Câu chuyện đã duyệt
          </span>
          <span className={researchStyles.grow} />
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
        {!skeleton && (
          <div className={styles.scroll}>
            <div className={styles.timeline} data-testid="beats-timeline">
              {state.beats.map((beat, i) => (
                <div key={beat.key} className={styles.segment} style={{ flex: beat.dur }} title={beat.title}>
                  B{i + 1} · {beat.fn}
                  <br />
                  {beat.dur} s
                </div>
              ))}
            </div>
          </div>
        )}
        {!skeleton && state.rebalanced > 0 && (
          <p className={researchStyles.hint} data-testid="beats-rebalanced">
            Đã cân lại thời lượng của {state.rebalanced} beat cho khớp mục tiêu {state.targetLabel}. Chỉnh bằng nút ± 5 s ở từng beat nếu thấy chưa hợp lý.
          </p>
        )}
        <div className={styles.checks} data-testid="beats-checks">
          {state.checks.map((check) => (
            <span key={check.text} className={`${styles.check} ${check.ok ? "" : styles.checkWarn}`} data-ok={check.ok}>
              {check.ok ? "✓" : "!"} {check.text}
            </span>
          ))}
        </div>
        <AiOperationCard
          run={beats.run}
          active={writingSheet || writingAdd}
          testId="beats-progress"
          title={writingAdd ? "AI đang viết beat mới" : "AI đang viết nội dung các beat"}
        />
        <div className={styles.sheetBar}>
          {skeleton ? (
            <>
              <span className={researchStyles.hint}>Kiểm chức năng và dữ kiện của từng beat, rồi duyệt để AI viết nội dung.</span>
              <span className={researchStyles.grow} />
              <Button variant="ghost" onClick={onRebuild} disabled={busy} data-testid="beats-rebuild-skeleton">
                Dựng lại bộ khung
              </Button>
              <Button onClick={() => void beats.approveSkeleton()} disabled={busy} data-testid="beats-approve-skeleton">
                Duyệt bộ khung
              </Button>
            </>
          ) : (
            <>
              {aiAvailable && <AiModelSelect projectId={projectId} step="beat_sheet" label="Model AI cho bước Beat sheet" testId="beats-model" />}
              {aiAvailable && (
                <Button variant="ghost" onClick={onRewrite} disabled={busy} data-testid="beats-rewrite">
                  Viết lại cả bảng
                </Button>
              )}
              {aiAvailable && state.unwritten > 0 && (
                <Button variant="run" onClick={() => void beats.generate("", true)} disabled={busy} data-testid="beats-generate">
                  AI viết nội dung các beat chưa có ({state.unwritten})
                </Button>
              )}
              <span className={researchStyles.grow} />
              <Button onClick={() => void beats.approveAll()} disabled={busy || pendingWithGoal === 0} data-testid="beats-approve-all">
                Duyệt tất cả beat còn lại
              </Button>
            </>
          )}
        </div>
      </div>
      <BeatInputModal open={inputOpen} onClose={() => setInputOpen(false)} loading={inputLoading} input={state.input} facts={state.facts} />
    </Card>
  );
}
