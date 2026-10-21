import { useState } from "react";
import { BEAT_DURATION_STEP, BEAT_FNS, BEAT_MIN_SECONDS, beatUnit, type Beat, type BeatFn, type BeatState } from "../../api/client";
import type { BeatsControls } from "../../hooks/useBeats";
import { useUnitImpact } from "../../hooks/useUnitImpact";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import { FactTags } from "../story/FactTags";
import { ImpactConfirm } from "../research/ImpactConfirm";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dropdown } from "../ui";
import { UnitEditModal } from "../units/UnitEditModal";
import { UnitRedoModal } from "../units/UnitRedoModal";
import { UnitReviewBar } from "../units/UnitReviewBar";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import styles from "./beats.module.css";

interface BeatCardProps {
  projectId: string;
  beat: Beat;
  /** Position in the sheet, from 0. */
  index: number;
  state: BeatState;
  beats: BeatsControls;
  /** A language model is configured, so the AI can write and redo beats. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

/** The text of a beat the Creator edits in the dialog, in the order the card shows it. */
const TEXT_FIELDS = [
  { name: "title", label: "Tên beat", rows: 2 },
  { name: "goal", label: "Mục tiêu · bước tiến của beat", rows: 3 },
  { name: "content", label: "Nội dung chính", rows: 4 },
  { name: "next", label: "Chuyển sang beat sau", rows: 3 },
] as const;

function Block({ label, text, testId }: { label: string; text: string; testId: string }) {
  return (
    <div className={styles.control} data-testid={testId}>
      <span className={styles.controlLabel}>{label}</span>
      <span className={text.trim() ? undefined : researchStyles.hint}>{text.trim() || "(trống)"}</span>
    </div>
  );
}

/**
 * One beat: its function, target length, goal, content, facts and how it
 * joins the next beat, with the review bar and the moves that change the
 * sheet (up, down, add after, delete). Text edits go through the "Sửa tay"
 * dialog (docs/ux-ui-design-rules.md, rule 4); the function, length and facts
 * change in place. Deleting asks first and names the facts the script must
 * state that no other beat holds.
 */
export function BeatCard({ projectId, beat, index, state, beats, aiAvailable, locked }: BeatCardProps) {
  const [editing, setEditing] = useState(false);
  const [redoing, setRedoing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const unit = beatUnit(beat.key);
  const editImpact = useUnitImpact(projectId, unit, editing);
  const redoImpact = useUnitImpact(projectId, unit, redoing);

  const no = index + 1;
  const last = index === state.beats.length - 1;
  const busy = locked || beats.busy || beats.running !== null;
  const writing = typeof beats.running === "object" && beats.running !== null && beats.running.beat === beat.key;
  const unusedFacts = state.facts.filter((f) => !beat.facts.includes(f.id));
  const isMine = (field: string) => beat.mine.includes(field);
  const canShorten = beat.dur - BEAT_DURATION_STEP >= BEAT_MIN_SECONDS;

  const deleteList = [
    "Beat này bị xoá khỏi bảng; các beat còn lại giữ nguyên.",
    ...(beat.onlyMust.length > 0
      ? [`${beat.onlyMust.join(", ")} bắt buộc phải nói nhưng chỉ có trong beat này: sẽ không còn trong beat nào.`]
      : []),
    "Lời thoại và ghi chú bàn giao của beat này (nếu đã có) sẽ không còn chỗ để bám.",
  ];

  return (
    <Card data-testid={`beat-${beat.key}`}>
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>
            B{no} · {beat.title}
          </h3>
          <span className={styles.moves}>
            {writing ? <UnitStatusChip state="running" testId={`beat-status-${beat.key}`} /> : <UnitStatusChip state={beat.status} testId={`beat-status-${beat.key}`} />}
            {beat.mine.length > 0 && (
              <span className={unitStyles.mine} title="AI giữ nguyên phần này khi viết lại" data-testid={`beat-mine-${beat.key}`}>
                ✎ Bạn viết
              </span>
            )}
          </span>
        </div>

        <AiOperationCard run={beats.run} active={writing} testId={`beat-progress-${beat.key}`} title="AI đang làm lại beat này" />

        <div className={styles.pair}>
          <div className={styles.control}>
            <span className={styles.controlLabel}>Chức năng{isMine("fn") ? " ✎" : ""}</span>
            <Dropdown
              value={beat.fn}
              options={BEAT_FNS.map((fn) => ({ value: fn, label: fn }))}
              onChange={(fn) => void beats.setFn(beat, fn as BeatFn)}
              disabled={busy}
              aria-label={`Chức năng beat B${no}`}
              data-testid={`beat-fn-${beat.key}`}
            />
          </div>
          <div className={styles.control}>
            <span className={styles.controlLabel}>Thời lượng mục tiêu{isMine("dur") ? " ✎" : ""}</span>
            <div className={styles.stepper}>
              <Button
                variant="ghost"
                className={researchStyles.sm}
                onClick={() => void beats.shiftDuration(beat, -1)}
                disabled={busy || !canShorten}
                data-testid={`beat-dur-down-${beat.key}`}
              >
                − {BEAT_DURATION_STEP} s
              </Button>
              <span className={styles.seconds} data-testid={`beat-dur-${beat.key}`}>
                {beat.dur} s
              </span>
              <Button variant="ghost" className={researchStyles.sm} onClick={() => void beats.shiftDuration(beat, 1)} disabled={busy} data-testid={`beat-dur-up-${beat.key}`}>
                + {BEAT_DURATION_STEP} s
              </Button>
            </div>
          </div>
        </div>

        <Block label="Mục tiêu · bước tiến của beat" text={beat.goal} testId={`beat-goal-${beat.key}`} />
        <Block label="Nội dung chính" text={beat.content} testId={`beat-content-${beat.key}`} />

        <div className={styles.control}>
          <span className={styles.controlLabel}>Dữ kiện dùng{isMine("facts") ? " ✎" : ""}</span>
          <div className={styles.factRow} data-testid={`beat-facts-${beat.key}`}>
            {beat.facts.length === 0 && <span className={researchStyles.hint}>chưa gắn dữ kiện</span>}
            {beat.facts.map((id) => (
              <span key={id} className={styles.factTag}>
                <FactTags ids={[id]} />
                <button
                  type="button"
                  className={styles.factDrop}
                  disabled={busy}
                  onClick={() => void beats.setFacts(beat, beat.facts.filter((f) => f !== id))}
                  aria-label={`Bỏ ${id} khỏi beat B${no}`}
                  data-testid={`beat-fact-drop-${beat.key}-${id}`}
                >
                  ×
                </button>
              </span>
            ))}
          </div>
          {unusedFacts.length > 0 && (
            <Dropdown
              className={styles.addFact}
              value=""
              placeholder="+ Gắn dữ kiện…"
              options={unusedFacts.map((f) => ({
                value: f.id,
                label: `${f.id} · ${f.claimVi.length > 52 ? `${f.claimVi.slice(0, 52)}…` : f.claimVi}`,
              }))}
              onChange={(id) => id && void beats.setFacts(beat, [...beat.facts, id])}
              disabled={busy}
              aria-label={`Gắn dữ kiện vào beat B${no}`}
              data-testid={`beat-fact-add-${beat.key}`}
            />
          )}
        </div>

        {!last && <Block label="Chuyển sang beat sau" text={beat.next} testId={`beat-next-${beat.key}`} />}

        <UnitReviewBar
          status={beat.status}
          approveLabel="Duyệt beat"
          approveDisabled={beat.goal.trim() === ""}
          busy={busy}
          canRedo={aiAvailable}
          onEdit={() => setEditing(true)}
          onRedo={() => setRedoing(true)}
          onApprove={() => void beats.approve(beat.key)}
          onUnapprove={() => void beats.unapprove(beat.key)}
          testId={`beat-review-${beat.key}`}
        />
        {beat.goal.trim() === "" && beat.status === "pending" && (
          <p className={researchStyles.hint}>Viết mục tiêu của beat rồi mới duyệt được.</p>
        )}

        <div className={styles.moves}>
          <Button variant="ghost" className={researchStyles.sm} onClick={() => void beats.move(beat.key, -1)} disabled={busy || index === 0} data-testid={`beat-up-${beat.key}`}>
            ↑ Lên
          </Button>
          <Button variant="ghost" className={researchStyles.sm} onClick={() => void beats.move(beat.key, 1)} disabled={busy || last} data-testid={`beat-down-${beat.key}`}>
            ↓ Xuống
          </Button>
          <Button
            variant="run"
            className={researchStyles.sm}
            onClick={() => void beats.add(beat.key, !aiAvailable, "")}
            disabled={busy}
            data-testid={`beat-add-${beat.key}`}
          >
            {aiAvailable ? "+ Thêm beat sau" : "+ Thêm beat trống sau"}
          </Button>
          <Button variant="danger" className={researchStyles.sm} onClick={() => setDeleting(true)} disabled={busy} data-testid={`beat-delete-${beat.key}`}>
            Xoá beat
          </Button>
        </div>
      </div>

      <UnitEditModal
        open={editing}
        onClose={() => setEditing(false)}
        title={`Sửa tay · B${no} ${beat.title}`}
        fields={TEXT_FIELDS.map((f) => ({ name: f.name, label: f.label, value: beat[f.name], rows: f.rows }))}
        impact={editImpact}
        onSave={(changed) => beats.edit(beat.key, changed)}
        note="Phần bạn sửa thành “Bạn viết”: AI giữ nguyên khi viết lại. Xoá trắng mục tiêu, nội dung hoặc chuyển tiếp thì AI viết lại phần đó. Sửa xong beat về chờ duyệt."
        size="lg"
        testId={`beat-edit-${beat.key}`}
      />
      <UnitRedoModal
        open={redoing}
        onClose={() => setRedoing(false)}
        title={`Làm lại kèm ghi chú · B${no} ${beat.title}`}
        impact={redoImpact}
        initialNote={beats.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="beat_sheet" label="Model AI cho bước Beat sheet" testId={`beat-redo-model-${beat.key}`} /> : undefined}
        onSend={(note) => void beats.redo(beat.key, note)}
        testId={`beat-redo-${beat.key}`}
      />
      <ImpactConfirm
        open={deleting}
        onClose={() => setDeleting(false)}
        title={`Xoá beat B${no}?`}
        lead={`“${beat.title}” (${beat.fn}, ${beat.dur} s)`}
        list={deleteList}
        undo="Không hoàn tác được. Muốn có lại thì thêm beat mới hoặc cho AI viết lại cả bảng."
        choices={[{ label: "Xoá beat", variant: "danger", onChoose: () => void beats.remove(beat.key), testId: `beat-delete-confirm-${beat.key}` }]}
        testId={`beat-delete-dialog-${beat.key}`}
      />
    </Card>
  );
}
