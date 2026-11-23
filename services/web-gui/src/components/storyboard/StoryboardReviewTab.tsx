import { useState } from "react";
import {
  describeAiRequest,
  reviewStoryboard,
  getStoryboardVersion,
  type StoryboardState,
} from "../../api/client";
import { visualErrorText } from "../../hooks/useVisualConcept";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import lockStyles from "../lock/lock.module.css";
import researchStyles from "../research/research.module.css";
import { Button, Card, CopyBox, Dialog, FormField, TextInput } from "../ui";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { StoryboardRunPreview } from "./StoryboardRunPreview";
import visualStyles from "../visual/visual.module.css";
import { ACCEPTABLE_CHECKS, allShots, blockersTooltip, keyTarget, storyboardCriteria } from "./storyboardModel";
import styles from "./storyboard.module.css";

const STATE_LABEL: Record<string, string> = {
  fixed: "✓ Đã sửa (AI không báo lại)",
  applied: "✓ Đã áp dụng gợi ý · về chờ duyệt",
};

interface StoryboardReviewTabProps {
  projectId: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  /** Opens the shot, unit or asset a finding points to on its own tab. */
  onOpen: (key: string) => void;
}

/**
 * "Rà soát và chốt" of step 11: the machine checks that fail, the AI review's
 * findings on shots, units and assets with what to do on each, the conditions to
 * lock, and the lock of a version with the versions already locked.
 */
export function StoryboardReviewTab({
  projectId,
  state,
  storyboard,
  readOnly,
  onOpen,
}: StoryboardReviewTabProps) {
  const [reviewOpen, setReviewOpen] = useState(false);
  const [lockOpen, setLockOpen] = useState(false);
  const [lockNote, setLockNote] = useState("");
  const [viewing, setViewing] = useState<{ version: number; text: string } | null>(null);
  const [viewError, setViewError] = useState<string | null>(null);
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const failing = state.checks.filter((c) => !c.ok);
  const locked = state.locked_version > 0;
  const upToDate = locked && state.draft.length === 0;
  const criteria = storyboardCriteria(state);
  const written = allShots(state).length > 0;
  const lockBlockers = state.can_lock ? [] : state.blockers;
  const versions = Array.from({ length: state.locked_version }, (_, i) => i + 1);

  async function openVersion(n: number) {
    setViewError(null);
    try {
      const v = await getStoryboardVersion(projectId, n);
      setViewing({ version: n, text: JSON.stringify(v.document, null, 2) });
    } catch (err) {
      setViewError(visualErrorText(err));
    }
  }

  return (
    <>
      <Card data-testid="sb-review">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Rà soát</h3>
            <span className={researchStyles.hint}>
              Máy luôn kiểm: shot thuộc unit có thật, tài sản đã duyệt, vị trí và góc máy hợp lệ, số
              dòng chữ, ảnh khung còn mới. AI rà soát cả storyboard.
            </span>
            <span className={researchStyles.grow} />
            <Button
              variant="run"
              onClick={() => setReviewOpen(true)}
              disabled={disabled || !written}
              title={written ? undefined : "Chưa có shot nào để rà soát"}
              data-testid="sb-review-run"
            >
              {state.review.reviewed ? "Rà soát lại" : "Rà soát bằng AI"}
            </Button>
            <AiRunPill runPath={`/v1/projects/${projectId}/storyboard/review`} onOpen={() => setReviewOpen(true)} testId="sb-review-ai-run" />
          </div>
          {!state.review.reviewed && failing.length === 0 && (
            <p className={researchStyles.hint} data-testid="sb-review-none">
              AI chưa rà soát. Rà soát sau khi các unit đã có shot.
            </p>
          )}
          <div className={lockStyles.findings}>
            {failing.map((c) => (
              <div key={c.id} className={lockStyles.finding} data-testid={`sb-check-row-${c.id}`}>
                <span className={lockStyles.eyebrow}>Máy kiểm</span>
                <b className={lockStyles.findingTitle}>{c.label}</b>
                <span className={lockStyles.detail}>{c.detail}</span>
                {c.accepted ? (
                  <span className={lockStyles.chipOk}>✓ Đã chấp nhận: {c.why}</span>
                ) : ACCEPTABLE_CHECKS.has(c.id) ? (
                  <div className={lockStyles.findingActions}>
                    <Button
                      variant="ghost"
                      onClick={() => void storyboard.accept(c.id, "")}
                      disabled={disabled}
                      data-testid={`sb-accept-${c.id}`}
                    >
                      Chấp nhận
                    </Button>
                  </div>
                ) : (
                  <span className={researchStyles.hint}>
                    Phải sửa ở tab Tài sản hoặc Shot; không chấp nhận bỏ qua được.
                  </span>
                )}
              </div>
            ))}
            {state.review.findings.map((f) => {
              const target = keyTarget(f.unit_key);
              return (
                <div key={f.id} className={lockStyles.finding} data-testid={`sb-finding-${f.id}`}>
                  <span className={lockStyles.eyebrow}>
                    {f.group_label} · {f.unit_key}
                  </span>
                  <b className={lockStyles.findingTitle}>{f.text}</b>
                  {f.detail && <span className={lockStyles.detail}>{f.detail}</span>}
                  {f.state === "open" ? (
                    <>
                      {f.fix && <span className={researchStyles.hint}>Gợi ý: {f.fix}</span>}
                      <div className={lockStyles.findingActions}>
                        {target !== "other" && (
                          <Button
                            variant="view"
                            onClick={() => onOpen(f.unit_key)}
                            data-testid={`sb-finding-open-${f.id}`}
                          >
                            Mở {f.unit_key}
                          </Button>
                        )}
                        {f.fix && f.field && (target === "shot" || target === "asset") && (
                          <Button
                            variant="primary"
                            onClick={() => void storyboard.applyFix(f.id)}
                            disabled={disabled}
                            data-testid={`sb-finding-apply-${f.id}`}
                          >
                            Áp dụng gợi ý
                          </Button>
                        )}
                        <Button
                          variant="ghost"
                          onClick={() => void storyboard.accept(f.id, "")}
                          disabled={disabled}
                          data-testid={`sb-finding-accept-${f.id}`}
                        >
                          Chấp nhận
                        </Button>
                      </div>
                    </>
                  ) : (
                    <span className={lockStyles.chipOk} data-testid={`sb-finding-state-${f.id}`}>
                      {f.state === "accepted" ? `✓ Đã chấp nhận: ${f.why}` : STATE_LABEL[f.state]}
                    </span>
                  )}
                </div>
              );
            })}
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

      <Card data-testid="sb-criteria">
        <div className={researchStyles.body}>
          <h3>Chốt storyboard</h3>
          <p className={researchStyles.hint}>
            Điều kiện: bước 10 đã chốt, mọi tài sản và shot đã duyệt, khung còn mới, đã rà soát,
            không còn vấn đề mở.
          </p>
          <ul className={visualStyles.criteria}>
            {criteria.map((c) => (
              <li key={c.label}>
                <span className={c.ok ? visualStyles.critOk : visualStyles.critNo}>
                  {c.ok ? "✓" : "✘"}
                </span>
                <span>
                  {c.label} {c.detail && <span className={researchStyles.hint}>· {c.detail}</span>}
                </span>
              </li>
            ))}
          </ul>
          {state.pending_units.length > 0 && (
            <div className={researchStyles.actions}>
              <span data-testid="sb-pending-units">
                Chưa duyệt đủ: <b>{state.pending_units.join(", ")}</b>
              </span>
              <Button
                variant="view"
                onClick={() => onOpen(state.pending_units[0])}
                data-testid="sb-go-pending"
              >
                Tới {state.pending_units[0]}
              </Button>
            </div>
          )}
          {lockBlockers.length > 0 && (
            <ul className={visualStyles.problems} data-testid="sb-blockers">
              {lockBlockers.map((b, i) => (
                <li key={`${i}-${b}`}>{b}</li>
              ))}
            </ul>
          )}
          <div className={researchStyles.actionsEnd}>
            {upToDate ? (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipOk}`}
                data-testid="sb-locked"
              >
                ✓ Đã chốt storyboard v{state.locked_version}
              </span>
            ) : (
              <Button
                variant={locked ? "impact" : "primary"}
                onClick={() => setLockOpen(true)}
                disabled={disabled || !state.can_lock}
                title={
                  state.can_lock
                    ? undefined
                    : state.blockers.length > 0
                      ? blockersTooltip(state.blockers)
                      : "Chưa đủ điều kiện chốt: xem các điều kiện ở trên"
                }
                data-testid="sb-lock"
              >
                Chốt storyboard v{state.next_version}
              </Button>
            )}
          </div>
          <p className={researchStyles.hint}>
            Khi chốt: ghi <code>storyboard.json</code> và ảnh khung của phiên bản vào{" "}
            <code>/shared/{"{dự án}"}/storyboard/v{state.next_version}/</code>.
          </p>
          <div className={styles.versions} data-testid="sb-versions">
            <b>Phiên bản đã chốt</b>
            {versions.length === 0 && <span className={researchStyles.hint}>Chưa có.</span>}
            {versions.map((n) => (
              <div key={n} className={researchStyles.actions}>
                <span>
                  v{n}
                  {n === state.locked_version && state.locked_at
                    ? ` · ${new Date(state.locked_at).toLocaleString("vi-VN")}${state.locked_note ? ` · ${state.locked_note}` : ""}`
                    : ""}
                </span>
                <Button
                  variant="view"
                  onClick={() => void openVersion(n)}
                  data-testid={`sb-version-${n}`}
                >
                  Xem
                </Button>
              </div>
            ))}
            {viewError && (
              <p className={researchStyles.error} role="alert" data-testid="sb-version-error">
                {viewError}
              </p>
            )}
          </div>
        </div>
      </Card>

      <AiRunDialog
        open={reviewOpen}
        onClose={() => setReviewOpen(false)}
        title={state.review.reviewed ? "Rà soát lại storyboard" : "Rà soát storyboard"}
        runPath={`/v1/projects/${projectId}/storyboard/review`}
        describe={(note) => describeAiRequest((external) => reviewStoryboard(projectId, note, { external }))}
        renderPreview={(p) => <StoryboardRunPreview kind="review" preview={p} />}
        impact={
          <p className={researchStyles.hint}>
            AI so storyboard hiện tại với các vấn đề cũ: vấn đề đã hết thì đánh dấu đã sửa, vấn đề
            mới thì thêm vào. Vấn đề trỏ tới một shot, một unit hoặc một tài sản.
          </p>
        }
        onApplied={storyboard.reload}
        testId="sb-review-dialog"
      />

      <Dialog
        open={lockOpen}
        onClose={() => setLockOpen(false)}
        title={`Chốt storyboard v${state.next_version}`}
        hasInput
        testId="sb-lock-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setLockOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant={locked ? "impact" : "primary"}
              onClick={() => {
                setLockOpen(false);
                void storyboard.lock(lockNote.trim()).then((ok) => ok && setLockNote(""));
              }}
              data-testid="sb-lock-confirm"
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
            data-testid="sb-lock-note"
          />
        </FormField>
        {locked ? (
          <div className={researchStyles.banner} data-testid="sb-lock-draft">
            <b>Đổi so với v{state.locked_version}:</b>
            <ul className={visualStyles.list}>
              {state.draft.map((d) => (
                <li key={d}>{d}</li>
              ))}
            </ul>
            Các bước sau đọc bản mới nhất; bản v{state.locked_version} vẫn giữ nguyên trong lịch
            sử.
          </div>
        ) : (
          <p className={researchStyles.hint} data-testid="sb-lock-first">
            Bản đầu tiên. Sau khi chốt, sửa tài sản hay shot tạo bản nháp mới; các bước sau vẫn
            dùng bản đã chốt tới lần chốt kế tiếp.
          </p>
        )}
      </Dialog>

      <Dialog
        open={viewing !== null}
        onClose={() => setViewing(null)}
        title={`Storyboard v${viewing?.version ?? ""}`}
        size="lg"
        testId="sb-version-dialog"
        footer={
          <Button variant="ghost" onClick={() => setViewing(null)}>
            Đóng
          </Button>
        }
      >
        <CopyBox text={viewing?.text ?? ""} what="tài liệu storyboard">
          <pre className={visualStyles.pre}>{viewing?.text}</pre>
        </CopyBox>
      </Dialog>
    </>
  );
}
