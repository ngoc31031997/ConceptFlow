import { useState } from "react";
import { generateHandoff, handoffBeatUnit, redoUnit, type LockBeat } from "../../api/client";
import type { ScriptLockControls } from "../../hooks/useScriptLock";
import { useUnitImpact } from "../../hooks/useUnitImpact";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { ExternalAiButton } from "../ExternalAiButton";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, FormField, TextInput } from "../ui";
import { UnitEditModal } from "../units/UnitEditModal";
import { UnitRedoModal } from "../units/UnitRedoModal";
import { UnitReviewBar } from "../units/UnitReviewBar";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import styles from "./lock.module.css";
import { ROLE_LABELS } from "./roles";

interface LockHandoffTabProps {
  projectId: string;
  lock: ScriptLockControls;
  aiAvailable: boolean;
  locked: boolean;
}

function HandoffBeatCard({ projectId, beat, lock, aiAvailable, locked }: { projectId: string; beat: LockBeat } & Omit<LockHandoffTabProps, "projectId">) {
  const [editing, setEditing] = useState(false);
  const [redoing, setRedoing] = useState(false);
  const unit = handoffBeatUnit(beat.key);
  const editImpact = useUnitImpact(projectId, unit, editing);
  const redoImpact = useUnitImpact(projectId, unit, redoing);
  const sections = lock.state?.sections ?? [];
  const writing = typeof lock.running === "object" && lock.running !== null && lock.running.beat === beat.key;
  const writingAll = lock.running === "handoff";
  const busy = locked || lock.busy || lock.running !== null;

  return (
    <Card data-testid={`handoff-beat-${beat.key}`}>
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>
            B{beat.number} · {beat.fn} · {beat.title}
          </h3>
          <span className={styles.row}>
            {writing ? <UnitStatusChip state="running" /> : beat.written && <UnitStatusChip state={beat.status} testId={`handoff-status-${beat.key}`} />}
            {beat.mine.length > 0 && (
              <span className={unitStyles.mine} title="AI giữ nguyên phần này khi viết lại" data-testid={`handoff-mine-${beat.key}`}>
                ✎ Bạn viết
              </span>
            )}
            {beat.stale && beat.written && (
              <span className={unitStyles.mine} data-testid={`handoff-stale-${beat.key}`}>
                lời thoại đã đổi, xem lại ghi chú
              </span>
            )}
          </span>
        </div>
        <p className={researchStyles.hint}>
          <b>Mục tiêu (từ beat sheet):</b> {beat.goal}
        </p>
        <AiOperationCard run={lock.run} active={writing || (writingAll && !beat.written)} testId={`handoff-progress-${beat.key}`} title="AI đang viết ghi chú bàn giao beat này" />

        {!beat.written ? (
          <div className={styles.row}>
            <ExternalAiButton
              title={`Ghi chú bàn giao B${beat.number} ${beat.title} · AI ngoài`}
              label="Viết ghi chú bằng AI ngoài"
              run={(external) => generateHandoff(projectId, beat.key, { external })}
              onDone={lock.reload}
              disabled={busy}
              testId={`handoff-write-external-${beat.key}`}
            />
            {aiAvailable && (
              <Button variant="run" className={researchStyles.sm} onClick={() => void lock.generate(beat.key)} disabled={busy} data-testid={`handoff-write-${beat.key}`}>
                AI viết ghi chú beat này
              </Button>
            )}
            <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(true)} disabled={busy} data-testid={`handoff-by-hand-${beat.key}`}>
              Tự viết
            </Button>
          </div>
        ) : (
          <>
            <div className={styles.sections}>
              {sections.map((s) => {
                const items = beat.note[s.key as keyof typeof beat.note] ?? [];
                return (
                  <div className={styles.section} key={s.key}>
                    <span className={unitStyles.mine} style={{ fontWeight: 700 }}>
                      {s.label}
                    </span>
                    <span className={styles.roles}>
                      đọc bởi{" "}
                      {s.roles.map((r) => (
                        <span className={styles.role} key={r}>
                          {ROLE_LABELS[r] ?? r}
                        </span>
                      ))}
                    </span>
                    {items.length > 0 ? (
                      <ul data-testid={`handoff-${beat.key}-${s.key}`}>
                        {items.map((it, i) => (
                          <li key={i}>{it}</li>
                        ))}
                      </ul>
                    ) : (
                      <span className={researchStyles.hint}>không có</span>
                    )}
                  </div>
                );
              })}
            </div>
            {beat.stale && (
              <div className={styles.row}>
                <Button variant="primary" className={researchStyles.sm} onClick={() => void lock.confirm(beat.key)} disabled={busy} data-testid={`handoff-confirm-${beat.key}`}>
                  Giữ ghi chú, duyệt lại
                </Button>
              </div>
            )}
            {!beat.stale && (
              <UnitReviewBar
                status={beat.status}
                approveLabel="Duyệt ghi chú"
                busy={busy}
                canRedo
                onEdit={() => setEditing(true)}
                onRedo={() => setRedoing(true)}
                onApprove={() => void lock.approve(beat.key)}
                onUnapprove={() => void lock.unapprove(beat.key)}
                testId={`handoff-review-${beat.key}`}
              />
            )}
            {beat.stale && (
              <div className={styles.row}>
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(true)} disabled={busy} data-testid={`handoff-stale-edit-${beat.key}`}>
                  Sửa tay
                </Button>
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setRedoing(true)} disabled={busy} data-testid={`handoff-stale-redo-${beat.key}`}>
                  Làm lại kèm ghi chú
                </Button>
              </div>
            )}
          </>
        )}
      </div>

      <UnitEditModal
        open={editing}
        onClose={() => setEditing(false)}
        title={`Sửa tay · Ghi chú bàn giao B${beat.number} ${beat.title}`}
        fields={sections.map((s) => ({
          name: s.key,
          label: s.label,
          value: (beat.note[s.key as keyof typeof beat.note] ?? []).join("\n"),
          rows: 3,
        }))}
        impact={editImpact}
        onSave={(changed) => lock.edit(beat.key, changed)}
        note="Mỗi dòng một mục. Sửa xong ghi chú về chờ duyệt."
        size="lg"
        testId={`handoff-edit-${beat.key}`}
      />
      <UnitRedoModal
        open={redoing}
        onClose={() => setRedoing(false)}
        title={`Làm lại kèm ghi chú · Ghi chú bàn giao B${beat.number} ${beat.title}`}
        impact={redoImpact}
        initialNote={lock.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="story" label="Model AI cho bước Rà soát và chốt" testId={`handoff-redo-model-${beat.key}`} /> : undefined}
        onSend={aiAvailable ? (note) => void lock.redo(beat.key, note) : undefined}
        external={{
          run: (note, external) => redoUnit(projectId, handoffBeatUnit(beat.key), note, { external }),
          onDone: lock.reload,
        }}
        testId={`handoff-redo-${beat.key}`}
      />
    </Card>
  );
}

/**
 * The second tab: the shared glossary and, for every beat, the five sections
 * of its handoff note with the roles that read each. The AI writes a beat's
 * note at a time; the Creator edits sections by hand, redoes a note with a
 * remark and approves it.
 */
export function LockHandoffTab({ projectId, lock, aiAvailable, locked }: LockHandoffTabProps) {
  const state = lock.state;
  const [adding, setAdding] = useState(false);
  const [term, setTerm] = useState({ term: "", say: "", draw: "", avoid: "" });
  const [error, setError] = useState<string | null>(null);
  if (!state) return null;
  const busy = locked || lock.busy || lock.running !== null;
  const unwritten = state.beats.filter((b) => !b.written).length;
  const toApprove = state.beats.filter((b) => b.written && (b.status === "pending" || b.stale)).length;

  async function save() {
    setError(null);
    try {
      await lock.addTerm(term);
      setAdding(false);
      setTerm({ term: "", say: "", draw: "", avoid: "" });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không thêm được thuật ngữ.");
    }
  }

  return (
    <>
      {unwritten > 0 && (
        <Card data-testid="handoff-write-card">
          <div className={researchStyles.body}>
            <p className={researchStyles.hint}>
              {aiAvailable
                ? `${unwritten} beat chưa có ghi chú bàn giao. AI viết lần lượt từng beat, hoặc bạn tự viết.`
                : `${unwritten} beat chưa có ghi chú bàn giao. Chưa cấu hình AI nên bạn tự viết từng beat.`}
            </p>
            {aiAvailable && (
              <>
                <AiModelSelect projectId={projectId} step="story" label="Model AI cho bước Rà soát và chốt" testId="lock-model" />
                <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                  <Button variant="run" onClick={() => void lock.generate("")} disabled={busy} data-testid="handoff-generate">
                    AI viết ghi chú bàn giao
                  </Button>
                  <span className={researchStyles.hint}>Ghi chú đã có giữ nguyên; thuật ngữ AI đề xuất gộp vào bảng chung.</span>
                </div>
              </>
            )}
          </div>
        </Card>
      )}

      <Card data-testid="glossary-card">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Thuật ngữ dùng chung</h3>
            <span className={styles.roles}>
              đọc bởi{" "}
              {["art_world", "storyboard", "voice", "illustration"].map((r) => (
                <span className={styles.role} key={r}>
                  {ROLE_LABELS[r]}
                </span>
              ))}
            </span>
          </div>
          {state.glossary.length === 0 ? (
            <p className={researchStyles.hint}>Chưa có thuật ngữ nào.</p>
          ) : (
            <div className={styles.scroll}>
              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Thuật ngữ</th>
                    <th>Nói là</th>
                    <th>Vẽ là</th>
                    <th>Tránh</th>
                  </tr>
                </thead>
                <tbody>
                  {state.glossary.map((g) => (
                    <tr key={g.term} data-testid={`glossary-${g.term}`}>
                      <td>
                        {g.term} {g.mine && <span className={unitStyles.mine}>✎</span>}
                      </td>
                      <td>{g.say}</td>
                      <td>{g.draw}</td>
                      <td>{g.avoid}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          <div>
            <Button variant="ghost" className={researchStyles.sm} onClick={() => setAdding(true)} disabled={busy} data-testid="glossary-add">
              + Thêm thuật ngữ
            </Button>
          </div>
        </div>
      </Card>

      {state.beats.map((b) => (
        <HandoffBeatCard key={b.key} projectId={projectId} beat={b} lock={lock} aiAvailable={aiAvailable} locked={locked} />
      ))}

      {state.beats.some((b) => b.written) && (
        <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
          <Button onClick={() => void lock.approveAll()} disabled={busy || toApprove === 0} data-testid="handoff-approve-all">
            Duyệt tất cả ghi chú còn lại
          </Button>
        </div>
      )}

      <Dialog
        open={adding}
        onClose={() => setAdding(false)}
        title="Thêm thuật ngữ"
        hasInput
        testId="glossary-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAdding(false)} data-testid="glossary-cancel">
              Huỷ
            </Button>
            <Button variant="primary" onClick={() => void save()} disabled={term.term.trim() === ""} data-testid="glossary-save">
              Thêm thuật ngữ
            </Button>
          </>
        }
      >
        {(["term", "say", "draw", "avoid"] as const).map((k) => (
          <FormField key={k} label={{ term: "Thuật ngữ", say: "Nói là", draw: "Vẽ là", avoid: "Tránh" }[k]}>
            <TextInput value={term[k]} onChange={(e) => setTerm((t) => ({ ...t, [k]: e.target.value }))} data-testid={`glossary-${k}`} />
          </FormField>
        ))}
        <p className={researchStyles.hint}>Mỹ thuật, Storyboard, Giọng và Hình minh hoạ đọc bảng này.</p>
        {error && (
          <p className={researchStyles.error} role="alert">
            {error}
          </p>
        )}
      </Dialog>
    </>
  );
}
