import { useState } from "react";
import { Link } from "react-router-dom";
import type { FindingState, ReviewFinding } from "../../api/client";
import type { ScriptLockControls } from "../../hooks/useScriptLock";
import { scriptRoute } from "../../utils/flow";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, FormField, TextArea, TextInput } from "../ui";
import styles from "./lock.module.css";

interface LockReviewTabProps {
  projectId: string;
  lock: ScriptLockControls;
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

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
 * replacement line, add its note to a beat's handoff, or accept it with a
 * reason. "Rà soát lại" has the AI compare the script as it stands with the
 * findings it made before.
 */
export function LockReviewTab({ projectId, lock, aiAvailable, locked }: LockReviewTabProps) {
  const state = lock.state;
  const [accepting, setAccepting] = useState<ReviewFinding | null>(null);
  const [why, setWhy] = useState("");
  const [note, setNote] = useState("");
  if (!state) return null;
  const busy = locked || lock.busy || lock.running !== null;
  const reviewed = state.reviewed;

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
          {f.state === "accepted" && <div className={styles.detail}>Lý do: {f.why}</div>}
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
            <Button
              variant="ghost"
              className={researchStyles.sm}
              onClick={() => {
                setWhy("");
                setAccepting(f);
              }}
              disabled={busy}
              data-testid={`finding-accept-${f.id}`}
            >
              Chấp nhận, ghi lý do
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
            {aiAvailable ? (
              <Button variant="run" onClick={() => void lock.review()} disabled={busy || !state.scriptReady} data-testid="lock-review-run">
                {reviewed ? "Rà soát lại" : "AI rà soát"}
              </Button>
            ) : (
              <span className={researchStyles.hint}>Chưa cấu hình AI: chỉ có phần máy kiểm. Mục “AI chưa rà soát” chấp nhận kèm lý do được.</span>
            )}
            {aiAvailable && (
              <span className={researchStyles.hint}>
                {reviewed ? "AI so bản lời thoại hiện tại với các mục đã phát hiện." : "AI đọc cả kịch bản một lượt."}
              </span>
            )}
          </div>
          <AiOperationCard run={lock.run} active={lock.running === "review"} testId="lock-review-progress" title="AI đang rà soát kịch bản" />
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

      <Dialog
        open={accepting !== null}
        onClose={() => setAccepting(null)}
        title="Chấp nhận mục rà soát"
        hasInput
        testId="lock-accept"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAccepting(null)} data-testid="lock-accept-cancel">
              Huỷ
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                const id = accepting?.id;
                setAccepting(null);
                if (id) void lock.accept(id, why);
              }}
              data-testid="lock-accept-save"
            >
              Chấp nhận
            </Button>
          </>
        }
      >
        {accepting && (
          <>
            <p>
              <b>{accepting.title}</b>
            </p>
            <p className={researchStyles.hint}>{accepting.detail}</p>
          </>
        )}
        <FormField label="Lý do chấp nhận">
          <TextInput value={why} onChange={(e) => setWhy(e.target.value)} placeholder="Ví dụ: đã kiểm ở nguồn khác, sẽ bổ sung sau" data-testid="lock-accept-why" />
        </FormField>
        <p className={researchStyles.hint}>Lý do lưu cùng phiên bản kịch bản.</p>
      </Dialog>
    </>
  );
}
