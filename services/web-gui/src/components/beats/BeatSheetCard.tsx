import { useState } from "react";
import { ExternalRequestDescribed, redoBeat, type Beat, type BeatState, type ExternalRequest } from "../../api/client";
import type { BeatsControls } from "../../hooks/useBeats";
import { useDeepSeekQueue } from "../../hooks/useDeepSeekQueue";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { BeatInputModal } from "./BeatInputModal";
import styles from "./beats.module.css";

/** The request that has the AI write one beat, described and not sent, so the extension can make it later. */
async function describeWrite(projectId: string, beat: Beat): Promise<ExternalRequest> {
  try {
    await redoBeat(projectId, beat.key, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết beat này.");
}

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
  const unwrittenBeats = state.beats.filter((b) => !b.goal.trim() && !b.content.trim());
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/beats`)) void beats.reload();
  });
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);

  // One queue item per beat without content, in the order of the sheet, so each beat is
  // written after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, beat] of unwrittenBeats.entries()) {
        const outcome = await queue.add({
          label: `B${state.beats.indexOf(beat) + 1} · ${beat.title}`,
          project: projectId,
          request: await describeWrite(projectId, beat),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({ tone: "error", text: `Dừng ở beat ${i + 1}/${unwrittenBeats.length}: ${outcome.message}` });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} beat vào hàng đợi DeepSeek${already > 0 ? `, ${already} beat đã có trong hàng đợi` : ""}. Extension chạy lần lượt; bấm icon extension để xem.`,
      });
    } catch (err) {
      setQueueNote({ tone: "error", text: err instanceof Error ? err.message : "Không thêm được vào hàng đợi." });
    } finally {
      setQueueing(false);
    }
  }

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
              {queue.available && unwrittenBeats.length > 0 && (
                <Button variant="run" onClick={() => void queueAll()} disabled={busy || queueing} data-testid="beats-queue-all">
                  {queueing ? "Đang thêm…" : `Chạy tất cả bằng DeepSeek (${unwrittenBeats.length})`}
                </Button>
              )}
              <span className={researchStyles.grow} />
              <Button onClick={() => void beats.approveAll()} disabled={busy || pendingWithGoal === 0} data-testid="beats-approve-all">
                Duyệt tất cả beat còn lại
              </Button>
            </>
          )}
        </div>
        {queue.counts && (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
          <p className={researchStyles.hint} data-testid="beats-queue-state">
            Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ · {queue.counts.failed} lỗi{queue.counts.paused ? " · đang tạm dừng" : ""}.
          </p>
        )}
        {queueNote && (
          <p className={queueNote.tone === "error" ? researchStyles.error : researchStyles.hint} role={queueNote.tone === "error" ? "alert" : "status"} data-testid="beats-queue-note">
            {queueNote.text}
          </p>
        )}
      </div>
      <BeatInputModal open={inputOpen} onClose={() => setInputOpen(false)} loading={inputLoading} input={state.input} facts={state.facts} />
    </Card>
  );
}
