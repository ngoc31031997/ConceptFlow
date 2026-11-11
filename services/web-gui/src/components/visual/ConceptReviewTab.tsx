import { useState } from "react";
import {
  generateVisualConceptUnit,
  getVisualConceptVersion,
  reviewVisualConcept,
  type VisualConceptFinding,
  type VisualConceptState,
  type VisualConceptUnitKey,
} from "../../api/client";
import { visualErrorText, type VisualConceptControls } from "../../hooks/useVisualConcept";
import lockStyles from "../lock/lock.module.css";
import researchStyles from "../research/research.module.css";
import { Button, Card, CopyBox, Dialog, FormField, TextInput } from "../ui";
import { ConceptAiDialog } from "./ConceptAiDialog";
import styles from "./visual.module.css";

const STATE_LABEL: Record<string, string> = {
  fixed: "✓ Đã sửa (AI không báo lại)",
  applied: "✓ Đã áp dụng gợi ý · phần về chờ duyệt",
};

/** The note an AI rewrite of the part starts with: the problem and the reviewer's advice. */
function fixNote(f: VisualConceptFinding): string {
  const advice = f.fix ? ` Gợi ý: ${f.fix}` : "";
  return `Sửa vấn đề này: ${f.text} ${f.detail}${advice}`.replace(/\s+/g, " ").trim();
}

interface ConceptReviewTabProps {
  projectId: string;
  state: VisualConceptState;
  concept: VisualConceptControls;
  readOnly: boolean;
  /** Opens a part on the rules tab. */
  onOpenUnit: (key: VisualConceptUnitKey) => void;
}

/**
 * "Rà soát và chốt": the machine checks that fail, the AI review's findings
 * with what to do on each, the five completion criteria, and the lock of a
 * version with the versions already locked.
 */
export function ConceptReviewTab({
  projectId,
  state,
  concept,
  readOnly,
  onOpenUnit,
}: ConceptReviewTabProps) {
  const [reviewOpen, setReviewOpen] = useState(false);
  const [applying, setApplying] = useState<VisualConceptFinding | null>(null);
  const [lockOpen, setLockOpen] = useState(false);
  const [lockNote, setLockNote] = useState("");
  const [viewing, setViewing] = useState<{ version: number; text: string } | null>(null);
  const [viewError, setViewError] = useState<string | null>(null);
  const disabled = readOnly || concept.busy || concept.running !== null;
  const failing = state.checks.filter((c) => !c.ok);
  const v2 = state.locked_version > 0;

  async function openVersion(n: number) {
    setViewError(null);
    try {
      const v = await getVisualConceptVersion(projectId, n);
      setViewing({ version: n, text: JSON.stringify(v.document, null, 2) });
    } catch (err) {
      setViewError(visualErrorText(err));
    }
  }

  return (
    <>
      <Card data-testid="concept-review">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Vấn đề cần xử lý trước khi duyệt</h3>
            <span className={researchStyles.hint}>
              AI rà soát cả bộ quy tắc với hồ sơ nghiên cứu, brief và kịch bản
            </span>
            <span className={researchStyles.grow} />
            <Button
              variant="run"
              onClick={() => setReviewOpen(true)}
              disabled={disabled}
              data-testid="concept-review-run"
            >
              {state.review.reviewed ? "Rà soát lại" : "Rà soát bằng AI"}
            </Button>
          </div>
          {!state.review.reviewed && failing.length === 0 && (
            <p className={researchStyles.hint} data-testid="concept-review-none">
              AI chưa rà soát. Rà soát sau khi đã có các phần.
            </p>
          )}
          <div className={lockStyles.findings}>
            {failing.map((c) => (
              <div
                key={c.id}
                className={lockStyles.finding}
                data-testid={`concept-check-row-${c.id}`}
              >
                <span className={lockStyles.eyebrow}>Máy kiểm</span>
                <b className={lockStyles.findingTitle}>{c.label}</b>
                <span className={lockStyles.detail}>{c.detail}</span>
                {c.accepted ? (
                  <span className={`${lockStyles.chipOk}`}>✓ Đã chấp nhận: {c.why}</span>
                ) : (
                  <div className={lockStyles.findingActions}>
                    <Button
                      variant="ghost"
                      onClick={() => void concept.accept(c.id, "")}
                      disabled={disabled}
                      data-testid={`concept-accept-quick-${c.id}`}
                    >
                      Chấp nhận
                    </Button>
                  </div>
                )}
              </div>
            ))}
            {state.review.findings.map((f) => (
              <div
                key={f.id}
                className={lockStyles.finding}
                data-testid={`concept-finding-${f.id}`}
              >
                <span className={lockStyles.eyebrow}>
                  {f.group_label} · phần {f.unit_number}
                </span>
                <b className={lockStyles.findingTitle}>{f.text}</b>
                {f.detail && <span className={lockStyles.detail}>{f.detail}</span>}
                {f.state === "open" ? (
                  <>
                    {f.fix && <span className={researchStyles.hint}>Gợi ý: {f.fix}</span>}
                    <div className={lockStyles.findingActions}>
                      <Button
                        variant="view"
                        onClick={() => onOpenUnit(f.unit_key)}
                        data-testid={`concept-finding-open-${f.id}`}
                      >
                        Mở phần {f.unit_number}
                      </Button>
                      <Button
                        variant="primary"
                        onClick={() => (f.direct ? void concept.applyFix(f.id) : setApplying(f))}
                        disabled={disabled}
                        data-testid={`concept-finding-apply-${f.id}`}
                      >
                        {f.direct ? "Áp dụng gợi ý" : "Nhờ AI sửa"}
                      </Button>
                      <Button
                        variant="ghost"
                        onClick={() => void concept.accept(f.id, "")}
                        disabled={disabled}
                        data-testid={`concept-finding-accept-quick-${f.id}`}
                      >
                        Chấp nhận
                      </Button>
                    </div>
                  </>
                ) : (
                  <span className={lockStyles.chipOk} data-testid={`concept-finding-state-${f.id}`}>
                    {f.state === "accepted" ? `✓ Đã chấp nhận: ${f.why}` : STATE_LABEL[f.state]}
                  </span>
                )}
              </div>
            ))}
          </div>
          {state.review.notes.length > 0 && (
            <div className={lockStyles.notes}>
              <b>Nhận xét của AI</b>
              <ul className={styles.list}>
                {state.review.notes.map((n, i) => (
                  <li key={i}>{n}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      </Card>

      <Card data-testid="concept-criteria">
        <div className={researchStyles.body}>
          <h3>Tiêu chí hoàn thành</h3>
          <ul className={styles.criteria}>
            {state.criteria.map((c) => (
              <li key={c.label}>
                <span className={c.ok ? styles.critOk : styles.critNo}>{c.ok ? "✓" : "○"}</span>
                <span>
                  {c.label} <span className={researchStyles.hint}>· {c.detail}</span>
                </span>
              </li>
            ))}
          </ul>
          {state.blockers.length > 0 && (
            <ul className={styles.problems} data-testid="concept-blockers">
              {state.blockers.map((b) => (
                <li key={b}>{b}</li>
              ))}
            </ul>
          )}
          <div className={researchStyles.actionsEnd}>
            {state.locked_version > 0 && state.draft.length === 0 ? (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipOk}`}
                data-testid="concept-locked"
              >
                ✓ Đã chốt ngôn ngữ hình ảnh v{state.locked_version}
              </span>
            ) : (
              <Button
                variant={v2 ? "impact" : "primary"}
                onClick={() => setLockOpen(true)}
                disabled={disabled || !state.can_lock}
                data-testid="concept-lock"
              >
                Chốt ngôn ngữ hình ảnh v{state.next_version}
              </Button>
            )}
          </div>
          {state.versions.length > 0 && (
            <div className={researchStyles.body} data-testid="concept-versions">
              <b>Phiên bản đã chốt</b>
              {state.versions.map((v) => (
                <div key={v.version} className={researchStyles.actions}>
                  <span>
                    v{v.version} · kịch bản v{v.script_version} ·{" "}
                    {new Date(v.locked_at).toLocaleString("vi-VN")} · {v.note}
                  </span>
                  <Button
                    variant="view"
                    onClick={() => void openVersion(v.version)}
                    data-testid={`concept-version-${v.version}`}
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
          )}
        </div>
      </Card>

      <ConceptAiDialog
        open={applying !== null}
        onClose={() => setApplying(null)}
        title={`Nhờ AI sửa · phần ${applying?.unit_number ?? ""} · ${applying?.unit_title ?? ""}`}
        impact={
          <p className={researchStyles.hint}>
            AI viết lại phần này theo vấn đề đã nêu. Các phần khác giữ nguyên; trường bạn đã sửa tay
            giữ nguyên trừ khi ghi chú yêu cầu đổi. Phần này về “chờ duyệt”.
          </p>
        }
        initialNote={applying ? fixNote(applying) : ""}
        onSystem={(note) => {
          if (applying) void concept.generateUnit(applying.unit_key, note, applying.id);
        }}
        external={{
          run: (note, external) =>
            applying
              ? generateVisualConceptUnit(projectId, applying.unit_key, note, { external }, applying.id)
              : Promise.reject(new Error("no finding picked")),
          onDone: concept.reload,
        }}
        testId="concept-apply-dialog"
      />

      <ConceptAiDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title={state.review.reviewed ? "Rà soát lại bộ quy tắc" : "Rà soát bộ quy tắc"}
        impact={
          <p className={researchStyles.hint}>
            AI so bộ quy tắc hiện tại với các vấn đề cũ: vấn đề đã hết thì đánh dấu đã sửa, vấn đề
            mới thì thêm vào.
          </p>
        }
        onSystem={(note) => void concept.review(note)}
        external={{
          run: (note, external) => reviewVisualConcept(projectId, note, { external }),
          onDone: concept.reload,
        }}
        testId="concept-review-dialog"
      />


      <Dialog
        open={lockOpen}
        onClose={() => setLockOpen(false)}
        title={`Chốt ngôn ngữ hình ảnh v${state.next_version}`}
        hasInput
        testId="concept-lock-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setLockOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant={v2 ? "impact" : "primary"}
              onClick={() => {
                setLockOpen(false);
                void concept.lock(lockNote.trim()).then((ok) => ok && setLockNote(""));
              }}
              data-testid="concept-lock-confirm"
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
            data-testid="concept-lock-note"
          />
        </FormField>
        {v2 && (
          <div className={researchStyles.banner} data-testid="concept-lock-draft">
            <b>Đổi so với v{state.locked_version}:</b>
            <ul className={styles.list}>
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
        title={`Ngôn ngữ hình ảnh v${viewing?.version ?? ""}`}
        size="lg"
        testId="concept-version-dialog"
        footer={
          <Button variant="ghost" onClick={() => setViewing(null)}>
            Đóng
          </Button>
        }
      >
        <CopyBox text={viewing?.text ?? ""} what="tài liệu ngôn ngữ hình ảnh">
          <pre className={styles.pre}>{viewing?.text}</pre>
        </CopyBox>
      </Dialog>
    </>
  );
}
