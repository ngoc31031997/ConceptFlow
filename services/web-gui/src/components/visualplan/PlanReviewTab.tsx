import { useState } from "react";
import { getVisualPlanVersion, reviewVisualPlan, type VisualPlanState } from "../../api/client";
import { visualErrorText } from "../../hooks/useVisualConcept";
import type { VisualPlanControls } from "../../hooks/useVisualPlan";
import lockStyles from "../lock/lock.module.css";
import researchStyles from "../research/research.module.css";
import { Button, Card, CopyBox, Dialog, FormField, TextInput } from "../ui";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
import visualStyles from "../visual/visual.module.css";
import { planCriteria, seconds } from "./planModel";

const STATE_LABEL: Record<string, string> = {
  fixed: "✓ Đã sửa (AI không báo lại)",
  applied: "✓ Đã áp dụng gợi ý · unit về chờ duyệt",
};

/** Minutes and seconds ("6:12") for a long total. */
function clock(s: number): string {
  const total = Math.round(s);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

interface PlanReviewTabProps {
  projectId: string;
  state: VisualPlanState;
  plan: VisualPlanControls;
  readOnly: boolean;
  /** Opens a unit on the plan tab. */
  onOpenUnit: (unitId: string) => void;
  /** Goes to the first line no unit covers. */
  onGoUncovered: (lineId: string) => void;
}

/**
 * "Rà soát và chốt": the machine checks that fail, the AI review's findings
 * with what to do on each, the completion criteria, and the lock of a version
 * with the versions already locked.
 */
export function PlanReviewTab({ projectId, state, plan, readOnly, onOpenUnit, onGoUncovered }: PlanReviewTabProps) {
  const [reviewOpen, setReviewOpen] = useState(false);
  const [lockOpen, setLockOpen] = useState(false);
  const [lockNote, setLockNote] = useState("");
  const [viewing, setViewing] = useState<{ version: number; text: string } | null>(null);
  const [viewError, setViewError] = useState<string | null>(null);
  const disabled = readOnly || plan.busy || plan.running !== null;
  const failing = state.checks.filter((c) => !c.ok);
  const v2 = state.locked_version > 0;
  const criteria = planCriteria(state);

  async function openVersion(n: number) {
    setViewError(null);
    try {
      const v = await getVisualPlanVersion(projectId, n);
      setViewing({ version: n, text: JSON.stringify(v.document, null, 2) });
    } catch (err) {
      setViewError(visualErrorText(err));
    }
  }

  return (
    <>
      <Card data-testid="plan-review">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Vấn đề cần xử lý trước khi chốt</h3>
            <span className={researchStyles.hint}>
              AI rà soát cả kế hoạch cùng kịch bản, ngôn ngữ hình ảnh và hồ sơ nghiên cứu
            </span>
            <span className={researchStyles.grow} />
            <Button
              variant="run"
              onClick={() => setReviewOpen(true)}
              disabled={disabled || state.units_total === 0}
              data-testid="plan-review-run"
            >
              {state.review.reviewed ? "Rà soát lại" : "Rà soát bằng AI"}
            </Button>
          </div>
          {!state.review.reviewed && failing.length === 0 && (
            <p className={researchStyles.hint} data-testid="plan-review-none">
              AI chưa rà soát. Rà soát sau khi đã có kế hoạch của các beat.
            </p>
          )}
          <div className={lockStyles.findings}>
            {state.coverage.uncovered.length > 0 && (
              <div className={lockStyles.finding} data-testid="plan-coverage-row">
                <span className={lockStyles.eyebrow}>Máy kiểm</span>
                <b className={lockStyles.findingTitle}>Độ phủ: {state.coverage.uncovered.length} câu chưa thuộc unit nào</b>
                <span className={lockStyles.detail}>
                  {state.coverage.uncovered.join(", ")}. Mỗi câu phải thuộc đúng một unit; không chấp nhận bỏ qua được.
                </span>
                <div className={lockStyles.findingActions}>
                  <Button variant="view" onClick={() => onGoUncovered(state.coverage.uncovered[0])}>
                    Tới câu chưa phủ
                  </Button>
                </div>
              </div>
            )}
            {failing.map((c) => (
              <div key={c.id} className={lockStyles.finding} data-testid={`plan-check-row-${c.id}`}>
                <span className={lockStyles.eyebrow}>Máy kiểm</span>
                <b className={lockStyles.findingTitle}>{c.label}</b>
                <span className={lockStyles.detail}>{c.detail}</span>
                {c.accepted ? (
                  <span className={lockStyles.chipOk}>✓ Đã chấp nhận: {c.why}</span>
                ) : (
                  <div className={lockStyles.findingActions}>
                    <Button
                      variant="ghost"
                      onClick={() => void plan.accept(c.id, "")}
                      disabled={disabled}
                      data-testid={`plan-accept-quick-${c.id}`}
                    >
                      Chấp nhận
                    </Button>
                  </div>
                )}
              </div>
            ))}
            {state.review.findings.map((f) => (
              <div key={f.id} className={lockStyles.finding} data-testid={`plan-finding-${f.id}`}>
                <span className={lockStyles.eyebrow}>
                  {f.group_label} · {f.unit_key}
                </span>
                <b className={lockStyles.findingTitle}>{f.text}</b>
                {f.detail && <span className={lockStyles.detail}>{f.detail}</span>}
                {f.state === "open" ? (
                  <>
                    {f.fix && <span className={researchStyles.hint}>Gợi ý: {f.fix}</span>}
                    <div className={lockStyles.findingActions}>
                      <Button variant="view" onClick={() => onOpenUnit(f.unit_key)} data-testid={`plan-finding-open-${f.id}`}>
                        Mở {f.unit_key}
                      </Button>
                      {f.fix && f.field && (
                        <Button
                          variant="primary"
                          onClick={() => void plan.applyFix(f.id)}
                          disabled={disabled}
                          data-testid={`plan-finding-apply-${f.id}`}
                        >
                          Áp dụng gợi ý
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        onClick={() => void plan.accept(f.id, "")}
                        disabled={disabled}
                        data-testid={`plan-finding-accept-quick-${f.id}`}
                      >
                        Chấp nhận
                      </Button>
                    </div>
                  </>
                ) : (
                  <span className={lockStyles.chipOk} data-testid={`plan-finding-state-${f.id}`}>
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

      <Card data-testid="plan-criteria">
        <div className={researchStyles.body}>
          <h3>Tiêu chí hoàn thành</h3>
          <ul className={visualStyles.criteria}>
            {criteria.map((c) => (
              <li key={c.label}>
                <span className={c.ok ? visualStyles.critOk : visualStyles.critNo}>{c.ok ? "✓" : "○"}</span>
                <span>
                  {c.label} {c.detail && <span className={researchStyles.hint}>· {c.detail}</span>}
                </span>
              </li>
            ))}
          </ul>
          {state.timing.over_brief && (
            <p className={researchStyles.banner} role="status" data-testid="plan-over-brief">
              ⚠ Tổng thời lượng dự kiến {clock(state.timing.planned_s)} vượt thời lượng creative brief (
              {clock(state.timing.brief_s)}) {seconds(state.timing.planned_s - state.timing.brief_s)}. Chỉ cảnh báo,
              không chặn chốt.
            </p>
          )}
          {state.blockers.length > 0 && (
            <ul className={visualStyles.problems} data-testid="plan-blockers">
              {state.blockers.map((b) => (
                <li key={b}>{b}</li>
              ))}
            </ul>
          )}
          <div className={researchStyles.actionsEnd}>
            {v2 && state.draft.length === 0 ? (
              <span className={`${researchStyles.chip} ${researchStyles.chipOk}`} data-testid="plan-locked">
                ✓ Đã chốt kế hoạch hình v{state.locked_version}
              </span>
            ) : (
              <Button
                variant={v2 ? "impact" : "primary"}
                onClick={() => setLockOpen(true)}
                disabled={disabled || !state.can_lock}
                title={state.can_lock ? undefined : "Chưa đủ điều kiện chốt: xem các tiêu chí ở trên"}
                data-testid="plan-lock"
              >
                Chốt kế hoạch hình v{state.next_version}
              </Button>
            )}
          </div>
          {state.versions.length > 0 && (
            <div className={researchStyles.body} data-testid="plan-versions">
              <b>Phiên bản đã chốt</b>
              {state.versions.map((v) => (
                <div key={v.version} className={researchStyles.actions}>
                  <span>
                    v{v.version} · kịch bản v{v.script_version} · ngôn ngữ hình ảnh v{v.concept_version} ·{" "}
                    {new Date(v.locked_at).toLocaleString("vi-VN")} · {v.note}
                  </span>
                  <Button variant="view" onClick={() => void openVersion(v.version)} data-testid={`plan-version-${v.version}`}>
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
          )}
        </div>
      </Card>

      <ConceptAiDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title={state.review.reviewed ? "Rà soát lại kế hoạch hình" : "Rà soát kế hoạch hình"}
        impact={
          <p className={researchStyles.hint}>
            AI so kế hoạch hiện tại với các vấn đề cũ: vấn đề đã hết thì đánh dấu đã sửa, vấn đề mới thì thêm vào.
          </p>
        }
        onSystem={(note) => void plan.review(note)}
        external={{
          run: (note, external) => reviewVisualPlan(projectId, note, { external }),
          onDone: plan.reload,
        }}
        testId="plan-review-dialog"
      />


      <Dialog
        open={lockOpen}
        onClose={() => setLockOpen(false)}
        title={`Chốt kế hoạch hình v${state.next_version}`}
        hasInput
        testId="plan-lock-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setLockOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant={v2 ? "impact" : "primary"}
              onClick={() => {
                setLockOpen(false);
                void plan.lock(lockNote.trim()).then((ok) => ok && setLockNote(""));
              }}
              data-testid="plan-lock-confirm"
            >
              Chốt v{state.next_version}
            </Button>
          </>
        }
      >
        <FormField label="Ghi chú phiên bản (không bắt buộc)">
          <TextInput value={lockNote} onChange={(e) => setLockNote(e.target.value)} maxLength={200} data-testid="plan-lock-note" />
        </FormField>
        {v2 && (
          <div className={researchStyles.banner} data-testid="plan-lock-draft">
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
        title={`Kế hoạch hình v${viewing?.version ?? ""}`}
        size="lg"
        testId="plan-version-dialog"
        footer={
          <Button variant="ghost" onClick={() => setViewing(null)}>
            Đóng
          </Button>
        }
      >
        <CopyBox text={viewing?.text ?? ""} what="tài liệu kế hoạch hình">
          <pre className={visualStyles.pre}>{viewing?.text}</pre>
        </CopyBox>
      </Dialog>
    </>
  );
}
