import { useState } from "react";
import {
  describeAiRequest,
  getVisualSequenceVersion,
  reviewVisualSequence,
  type VisualSequenceState,
} from "../../api/client";
import { visualErrorText } from "../../hooks/useVisualConcept";
import type { VisualSequenceControls } from "../../hooks/useVisualSequence";
import lockStyles from "../lock/lock.module.css";
import researchStyles from "../research/research.module.css";
import { Button, Card, CopyBox, Dialog, FormField, TextInput } from "../ui";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { SequenceRunPreview } from "./SequenceRunPreview";
import visualStyles from "../visual/visual.module.css";
import { sequenceCriteria } from "./sequenceModel";

const STATE_LABEL: Record<string, string> = {
  fixed: "✓ Đã sửa (AI không báo lại)",
  applied: "✓ Đã áp dụng gợi ý · về chờ duyệt",
};

/** Which machine checks can be accepted with a reason; the others must be fixed. */
const ACCEPTABLE_CHECKS = new Set(["m-seq-via", "m-motif-count"]);

interface SequenceReviewTabProps {
  projectId: string;
  state: VisualSequenceState;
  sequence: VisualSequenceControls;
  readOnly: boolean;
  /** Opens a unit (`VP-…`) or a motif (`MOTIF-…`) on the sequence tab. */
  onOpen: (key: string) => void;
}

/**
 * "Rà soát và chốt" of step 10: the machine checks that fail, the AI review's
 * findings on units and motifs with what to do on each, the completion
 * criteria, and the lock of a version with the versions already locked.
 */
export function SequenceReviewTab({
  projectId,
  state,
  sequence,
  readOnly,
  onOpen,
}: SequenceReviewTabProps) {
  const [reviewOpen, setReviewOpen] = useState(false);
  const reviewRunPath = `/v1/projects/${projectId}/visual-sequence/review`;
  const [lockOpen, setLockOpen] = useState(false);
  const [lockNote, setLockNote] = useState("");
  const [viewing, setViewing] = useState<{ version: number; text: string } | null>(null);
  const [viewError, setViewError] = useState<string | null>(null);
  const disabled = readOnly || sequence.busy || sequence.running !== null;
  const failing = state.checks.filter((c) => !c.ok);
  const v2 = state.locked_version > 0;
  const criteria = sequenceCriteria(state);
  const written = state.coverage.units_sequenced > 0;

  async function openVersion(n: number) {
    setViewError(null);
    try {
      const v = await getVisualSequenceVersion(projectId, n);
      setViewing({ version: n, text: JSON.stringify(v.document, null, 2) });
    } catch (err) {
      setViewError(visualErrorText(err));
    }
  }

  return (
    <>
      <Card data-testid="seq-review">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Vấn đề cần xử lý trước khi chốt</h3>
            <span className={researchStyles.hint}>
              AI rà soát cả chuỗi cùng kế hoạch hình và ngôn ngữ hình ảnh
            </span>
            <span className={researchStyles.grow} />
            <Button
              variant="run"
              onClick={() => setReviewOpen(true)}
              disabled={disabled || !written}
              title={written ? undefined : "Chưa có chuỗi nào để rà soát"}
              data-testid="seq-review-run"
            >
              {state.review.reviewed ? "Rà soát lại" : "Rà soát bằng AI"}
            </Button>
            <AiRunPill
              runPath={reviewRunPath}
              onOpen={() => setReviewOpen(true)}
              testId="seq-review-ai-run"
            />
          </div>
          {!state.review.reviewed && failing.length === 0 && (
            <p className={researchStyles.hint} data-testid="seq-review-none">
              AI chưa rà soát. Rà soát sau khi các beat đã có chuỗi.
            </p>
          )}
          <div className={lockStyles.findings}>
            {failing.map((c) => (
              <div key={c.id} className={lockStyles.finding} data-testid={`seq-check-row-${c.id}`}>
                <span className={lockStyles.eyebrow}>Máy kiểm</span>
                <b className={lockStyles.findingTitle}>{c.label}</b>
                <span className={lockStyles.detail}>{c.detail}</span>
                {c.accepted ? (
                  <span className={lockStyles.chipOk}>✓ Đã chấp nhận: {c.why}</span>
                ) : ACCEPTABLE_CHECKS.has(c.id) ? (
                  <div className={lockStyles.findingActions}>
                    <Button
                      variant="ghost"
                      onClick={() => void sequence.accept(c.id, "")}
                      disabled={disabled}
                      data-testid={`seq-accept-${c.id}`}
                    >
                      Chấp nhận
                    </Button>
                  </div>
                ) : (
                  <span className={researchStyles.hint}>
                    Phải sửa ở tab Chuỗi hình; không chấp nhận bỏ qua được.
                  </span>
                )}
              </div>
            ))}
            {state.review.findings.map((f) => (
              <div key={f.id} className={lockStyles.finding} data-testid={`seq-finding-${f.id}`}>
                <span className={lockStyles.eyebrow}>
                  {f.group_label} · {f.unit_key}
                </span>
                <b className={lockStyles.findingTitle}>{f.text}</b>
                {f.detail && <span className={lockStyles.detail}>{f.detail}</span>}
                {f.state === "open" ? (
                  <>
                    {f.fix && <span className={researchStyles.hint}>Gợi ý: {f.fix}</span>}
                    <div className={lockStyles.findingActions}>
                      <Button
                        variant="view"
                        onClick={() => onOpen(f.unit_key)}
                        data-testid={`seq-finding-open-${f.id}`}
                      >
                        Mở {f.unit_key}
                      </Button>
                      {f.fix && f.field && (
                        <Button
                          variant="primary"
                          onClick={() => void sequence.applyFix(f.id)}
                          disabled={disabled}
                          data-testid={`seq-finding-apply-${f.id}`}
                        >
                          Áp dụng gợi ý
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        onClick={() => void sequence.accept(f.id, "")}
                        disabled={disabled}
                        data-testid={`seq-finding-accept-${f.id}`}
                      >
                        Chấp nhận
                      </Button>
                    </div>
                  </>
                ) : (
                  <span className={lockStyles.chipOk} data-testid={`seq-finding-state-${f.id}`}>
                    {f.state === "accepted" ? `✓ Đã chấp nhận: ${f.why}` : STATE_LABEL[f.state]}
                  </span>
                )}
              </div>
            ))}
          </div>
          {state.review.notes.length > 0 && (
            <div className={lockStyles.notes}>
              <b>Nhận xét của AI</b>
              <ul className={visualStyles.list}>
                {state.review.notes.map((n, i) => (
                  <li key={i}>{n}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </Card>

      <Card data-testid="seq-criteria">
        <div className={researchStyles.body}>
          <h3>Tiêu chí hoàn thành</h3>
          <ul className={visualStyles.criteria}>
            {criteria.map((c) => (
              <li key={c.label}>
                <span className={c.ok ? visualStyles.critOk : visualStyles.critNo}>
                  {c.ok ? "✓" : "○"}
                </span>
                <span>
                  {c.label} {c.detail && <span className={researchStyles.hint}>· {c.detail}</span>}
                </span>
              </li>
            ))}
          </ul>
          {state.pending_units.length > 0 && (
            <div className={researchStyles.actions}>
              <span data-testid="seq-pending-units">
                Chưa duyệt: <b>{state.pending_units.join(", ")}</b>
              </span>
              <Button
                variant="view"
                onClick={() => onOpen(state.pending_units[0])}
                data-testid="seq-go-pending"
              >
                Tới {state.pending_units[0]}
              </Button>
            </div>
          )}
          {state.blockers.length > 0 && (
            <ul className={visualStyles.problems} data-testid="seq-blockers">
              {state.blockers.map((b) => (
                <li key={b}>{b}</li>
              ))}
            </ul>
          )}
          <div className={researchStyles.actionsEnd}>
            {v2 && state.draft.length === 0 ? (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipOk}`}
                data-testid="seq-locked"
              >
                ✓ Đã chốt chuỗi hình v{state.locked_version}
              </span>
            ) : (
              <Button
                variant={v2 ? "impact" : "primary"}
                onClick={() => setLockOpen(true)}
                disabled={disabled || !state.can_lock}
                title={
                  state.can_lock ? undefined : "Chưa đủ điều kiện chốt: xem các tiêu chí ở trên"
                }
                data-testid="seq-lock"
              >
                Chốt chuỗi hình v{state.next_version}
              </Button>
            )}
          </div>
          <div className={researchStyles.body} data-testid="seq-versions">
            <b>Phiên bản đã chốt</b>
            {state.versions.length === 0 && <span className={researchStyles.hint}>Chưa có.</span>}
            {state.versions.map((v) => (
              <div key={v.version} className={researchStyles.actions}>
                <span>
                  v{v.version} · kế hoạch hình v{v.plan_version} · ngôn ngữ hình ảnh v
                  {v.concept_version} · kịch bản v{v.script_version} ·{" "}
                  {new Date(v.locked_at).toLocaleString("vi-VN")} · {v.note}
                </span>
                <Button
                  variant="view"
                  onClick={() => void openVersion(v.version)}
                  data-testid={`seq-version-${v.version}`}
                >
                  Xem
                </Button>
              </div>
            ))}
            {viewError && (
              <p className={researchStyles.error} role="alert">
                {viewError}
              </p>
            )}
          </div>
        </div>
      </Card>

      <AiRunDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title={state.review.reviewed ? "Rà soát lại chuỗi hình" : "Rà soát chuỗi hình"}
        runPath={reviewRunPath}
        describe={(note) =>
          describeAiRequest((external) => reviewVisualSequence(projectId, note, { external }))
        }
        renderPreview={(p) => <SequenceRunPreview kind="review" preview={p} />}
        impact={
          <p className={researchStyles.hint}>
            AI so chuỗi hiện tại với các vấn đề cũ: vấn đề đã hết thì đánh dấu đã sửa, vấn đề mới
            thì thêm vào. Nhóm vấn đề như bước 8 và 9: độ chính xác, nhất quán, rõ ràng, khả thi.
          </p>
        }
        onApplied={sequence.reload}
        testId="seq-review-dialog"
      />

      <Dialog
        open={lockOpen}
        onClose={() => setLockOpen(false)}
        title={`Chốt chuỗi hình v${state.next_version}`}
        hasInput
        testId="seq-lock-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setLockOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant={v2 ? "impact" : "primary"}
              onClick={() => {
                setLockOpen(false);
                void sequence.lock(lockNote.trim()).then((ok) => ok && setLockNote(""));
              }}
              data-testid="seq-lock-confirm"
            >
              Chốt v{state.next_version}
            </Button>
          </>
        }
      >
        <FormField label="Ghi chú phiên bản (không bắt buộc)">
          <TextInput
            value={lockNote}
            onChange={(e) => setLockNote(e.target.value)}
            maxLength={200}
            data-testid="seq-lock-note"
          />
        </FormField>
        {v2 && (
          <div className={researchStyles.banner} data-testid="seq-lock-draft">
            <b>Đổi so với v{state.locked_version}:</b>
            <ul className={visualStyles.list}>
              {state.draft.map((d) => (
                <li key={d}>{d}</li>
              ))}
            </ul>
            Các bước sau đọc bản mới nhất; bản v{state.locked_version} vẫn giữ nguyên trong lịch sử.
          </div>
        )}
      </Dialog>

      <Dialog
        open={viewing !== null}
        onClose={() => setViewing(null)}
        title={`Chuỗi hình v${viewing?.version ?? ""}`}
        size="lg"
        testId="seq-version-dialog"
        footer={
          <Button variant="ghost" onClick={() => setViewing(null)}>
            Đóng
          </Button>
        }
      >
        <CopyBox text={viewing?.text ?? ""} what="tài liệu chuỗi hình">
          <pre className={visualStyles.pre}>{viewing?.text}</pre>
        </CopyBox>
      </Dialog>
    </>
  );
}
