import { useState } from "react";
import { Link } from "react-router-dom";
import { describeAiRequest, reviewScript, type FindingState, type ReviewFinding } from "../../api/client";
import type { ScriptLockControls } from "../../hooks/useScriptLock";
import { scriptRoute } from "../../utils/flow";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import researchStyles from "../research/research.module.css";
import { Button, Card, FormField, TextArea } from "../ui";
import styles from "./lock.module.css";
import { LockRunPreview } from "./LockRunPreview";

interface LockReviewTabProps {
  projectId: string;
  lock: ScriptLockControls;
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

/** What the server keeps for a finding accepted without a reason. */
const NO_REASON = "không ghi lý do";

const CHIP: Record<FindingState, { label: string; className: string }> = {
  open: { label: "! cần xử lý", className: styles.chipOpen },
  ok: { label: "✓ đạt", className: styles.chipOk },
  fixed: { label: "✓ đã sửa", className: styles.chipOk },
  accepted: { label: "– đã chấp nhận", className: styles.chipInfo },
  info: { label: "i thông tin", className: styles.chipInfo },
};

/** The first line of the script a finding is about, as the script screen takes it in the address. */
function lineLink(projectId: string, line: string): string {
  return `${scriptRoute(projectId)}?line=${encodeURIComponent(line)}`;
}

/**
 * The first tab: what the machine and the AI found over the whole script. A
 * finding offers what fits it: open the line, open research, apply the AI's
 * replacement line, add its note to a beat's handoff, or accept it as it
 * stands in one click. "Rà soát lại" has the AI compare the script as it stands with the
 * findings it made before.
 */
export function LockReviewTab({ projectId, lock, aiAvailable, locked }: LockReviewTabProps) {
  const state = lock.state;
  const [note, setNote] = useState("");
  const [reviewOpen, setReviewOpen] = useState(false);
  if (!state) return null;
  const busy = locked || lock.busy || lock.running !== null;
  const reviewed = state.reviewed;
  const reviewPath = `/v1/projects/${projectId}/script-lock/review`;

  const row = (f: ReviewFinding) => {
    const chip = CHIP[f.state];
    const open = f.state === "open";
    return (
      <div className={styles.finding} key={f.id} data-testid={`finding-${f.id}`} data-state={f.state}>
        <span className={`${chip.className}`} data-testid={`finding-chip-${f.id}`}>
          {chip.label}
        </span>
        <div>
          <span className={styles.eyebrow}>{f.groupLabel}</span>
          <br />
          <span className={styles.findingTitle}>{f.title}</span>
          {f.detail && (
            <>
              <br />
              <span className={styles.detail}>{f.detail}</span>
            </>
          )}
          {open && f.fix && <div className={styles.quote}>Gợi ý: {f.fix}</div>}
          {f.state === "accepted" && f.why && f.why !== NO_REASON && <div className={styles.detail}>Lý do: {f.why}</div>}
        </div>
        {open && (
          <div className={styles.findingActions}>
            {f.line && (
              <Link to={lineLink(projectId, f.line)} data-testid={`finding-open-line-${f.id}`}>
                <Button variant="view" className={researchStyles.sm} type="button">
                  Mở câu đó
                </Button>
              </Link>
            )}
            {f.research && (
              <Link to={`/projects/${projectId}/research`} data-testid={`finding-open-research-${f.id}`}>
                <Button variant="view" className={researchStyles.sm} type="button">
                  Mở ở Nghiên cứu
                </Button>
              </Link>
            )}
            {f.fix && (
              <Button variant="primary" className={researchStyles.sm} onClick={() => void lock.applyFix(f.id)} disabled={busy} data-testid={`finding-apply-${f.id}`}>
                Áp dụng gợi ý
              </Button>
            )}
            {f.handoff && (
              <Button variant="primary" className={researchStyles.sm} onClick={() => void lock.toHandoff(f.id)} disabled={busy} data-testid={`finding-handoff-${f.id}`}>
                Thêm vào ghi chú bàn giao
              </Button>
            )}
            <Button variant="ghost" className={researchStyles.sm} onClick={() => void lock.accept(f.id, "")} disabled={busy} data-testid={`finding-accept-${f.id}`}>
              Chấp nhận
            </Button>
          </div>
        )}
      </div>
    );
  };

  return (
    <>
      <Card data-testid="lock-review-card">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Kiểm tra</h3>
            <span className={researchStyles.hint}>máy kiểm và AI rà soát · {state.openCount} mục cần xử lý</span>
          </div>
          <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
            <Button variant="run" onClick={() => setReviewOpen(true)} disabled={busy || !state.scriptReady} data-testid="lock-review-run">
              {reviewed ? "Rà soát lại" : "Rà soát bằng AI"}
            </Button>
            <AiRunPill runPath={reviewPath} onOpen={() => setReviewOpen(true)} testId="lock-review-ai-run" />
            <span className={researchStyles.hint}>
              {reviewed ? "AI so bản lời thoại hiện tại với các mục đã phát hiện." : "AI đọc cả kịch bản một lượt."}
              {!aiAvailable && " Chưa cấu hình AI của hệ thống: chạy bằng extension hoặc thủ công."}
            </span>
          </div>
          <div className={styles.findings}>{state.findings.map(row)}</div>
        </div>
      </Card>

      <Card data-testid="lock-notes-card">
        <div className={researchStyles.body}>
          <h3>Ghi chú rà soát</h3>
          <span className={researchStyles.hint}>AI rà soát viết; bạn thêm ghi chú của mình. Lưu cùng phiên bản kịch bản.</span>
          <div className={styles.notes}>
            {state.aiNotes.map((n, i) => (
              <div className={styles.noteLine} key={`ai-${i}`}>
                AI: {n}
              </div>
            ))}
            {state.myNotes.map((n, i) => (
              <div className={styles.noteLine} key={`me-${i}`}>
                Bạn: {n}
              </div>
            ))}
          </div>
          <FormField label="Ghi chú của bạn">
            <TextArea value={note} onChange={(e) => setNote(e.target.value)} rows={2} placeholder="Ví dụ: beat 2 có thể rút ngắn" data-testid="lock-note-input" />
          </FormField>
          <div>
            <Button
              variant="primary"
              className={researchStyles.sm}
              onClick={() => void lock.addNote(note).then((ok) => ok && setNote(""))}
              disabled={busy || note.trim() === ""}
              data-testid="lock-note-add"
            >
              Thêm ghi chú
            </Button>
          </div>
        </div>
      </Card>

      <AiRunDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title={reviewed ? "Rà soát lại kịch bản" : "Rà soát kịch bản"}
        runPath={reviewPath}
        describe={() => describeAiRequest((external) => reviewScript(projectId, { external }))}
        noNote
        renderPreview={(p) => <LockRunPreview kind="review" preview={p} />}
        impact={
          <p className={researchStyles.hint}>
            AI so bản lời thoại hiện tại với các mục đã phát hiện: mục đã hết thì đánh dấu đã sửa, mục mới thì thêm vào.
          </p>
        }
        onApplied={lock.reload}
        testId="lock-review-dialog"
      />
    </>
  );
}
