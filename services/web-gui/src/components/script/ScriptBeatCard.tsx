import { useState } from "react";
import { scriptBeatUnit, type ScriptBeat, type ScriptLine } from "../../api/client";
import type { ScriptControls } from "../../hooks/useScript";
import { useUnitImpact } from "../../hooks/useUnitImpact";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { FactTags } from "../story/FactTags";
import { Button, Card } from "../ui";
import { UnitEditModal } from "../units/UnitEditModal";
import { UnitRedoModal } from "../units/UnitRedoModal";
import { UnitReviewBar } from "../units/UnitReviewBar";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { formatSeconds } from "./ScriptOverviewCard";
import styles from "./script.module.css";

interface ScriptBeatCardProps {
  projectId: string;
  beat: ScriptBeat;
  script: ScriptControls;
  /** A language model is configured, so the AI can write, redo and suggest. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
  /** A line to mark (opened from another screen). */
  highlight?: string | null;
}

/** A suggestion the AI gave for one line, waiting for the Creator to apply or drop it. */
interface Suggestion {
  kind: "line" | "join";
  lineId: string;
  text: string;
}

function readingLabel(line: ScriptLine, playing: boolean, waiting: boolean): string {
  if (playing) return "● Đang phát";
  if (waiting || line.audio.state === "pending" || line.audio.state === "running") return "… Đang đọc";
  return "▶ Nghe";
}

/** How far a beat's dialogue may run from its target length before the card warns, as a fraction of the target. */
const LENGTH_TOLERANCE = 0.15;

/**
 * How far a beat's dialogue runs from the length the beat sheet gave it:
 * "" within the tolerance, otherwise "ngắn N%" or "dài N%" of the target.
 */
function lengthDeviation(seconds: number, targetSeconds: number): string {
  if (targetSeconds <= 0) return "";
  const off = (seconds - targetSeconds) / targetSeconds;
  if (Math.abs(off) <= LENGTH_TOLERANCE) return "";
  return `${off < 0 ? "ngắn" : "dài"} ${Math.round(Math.abs(off) * 100)}%`;
}

/**
 * One beat's dialogue: its lines with their length, facts and flags, the
 * join with the beat before, listening to a line or the whole beat, and the
 * review bar. A flagged line or a bad join offers the AI's suggestion, shown
 * for the Creator to apply (an edit of that line) or drop. Hand edits go
 * through the "Sửa tay" dialog (docs/ux-ui-design-rules.md, rule 4).
 */
export function ScriptBeatCard({ projectId, beat, script, aiAvailable, locked, highlight }: ScriptBeatCardProps) {
  const [editing, setEditing] = useState(false);
  const [redoing, setRedoing] = useState(false);
  const [suggestion, setSuggestion] = useState<Suggestion | null>(null);
  const unit = scriptBeatUnit(beat.key);
  const deviation = lengthDeviation(beat.seconds, beat.targetSeconds);
  const editImpact = useUnitImpact(projectId, unit, editing);
  const redoImpact = useUnitImpact(projectId, unit, redoing);

  const { player } = script;
  const busy = locked || script.busy || script.running !== null;
  const writing = typeof script.running === "object" && script.running !== null && "beat" in script.running && script.running.beat === beat.key;
  const writingAll = script.running === "write";
  const asking = typeof script.running === "object" && script.running !== null && "suggest" in script.running ? script.running.suggest : null;
  const lastId = beat.lines.length > 0 ? beat.lines[beat.lines.length - 1].id : "start";
  const ids = beat.lines.map((l) => l.id);
  const playingBeat = ids.some((id) => id === player.playingId || id === player.waitingId);
  const listenDisabled = locked || ids.length === 0;

  async function ask(kind: "line" | "join", lineId: string) {
    if (suggestion?.lineId === lineId && suggestion.kind === kind) {
      setSuggestion(null);
      return;
    }
    const text = await script.suggest(kind, lineId);
    if (text) setSuggestion({ kind, lineId, text });
  }

  async function apply() {
    if (!suggestion) return;
    const ok = await script.setLine(beat.key, suggestion.lineId, suggestion.text);
    if (ok) setSuggestion(null);
  }

  const suggestionBox = (lineId: string, kind: "line" | "join") =>
    suggestion?.lineId === lineId && suggestion.kind === kind ? (
      <div className={styles.suggestion} data-testid={`script-suggestion-${lineId}`}>
        <b>AI gợi ý</b>
        <span>{suggestion.text}</span>
        <div className={styles.actions}>
          <Button variant="ghost" className={researchStyles.sm} onClick={() => setSuggestion(null)} data-testid={`script-suggestion-drop-${lineId}`}>
            Bỏ
          </Button>
          <Button variant="primary" className={researchStyles.sm} onClick={() => void apply()} disabled={busy} data-testid={`script-suggestion-apply-${lineId}`}>
            Áp dụng
          </Button>
        </div>
      </div>
    ) : null;

  const askButton = (kind: "line" | "join", lineId: string) =>
    aiAvailable ? (
      <Button variant="view" className={researchStyles.sm} onClick={() => void ask(kind, lineId)} disabled={busy} data-testid={`script-ask-${kind}-${lineId}`}>
        {asking === lineId ? "AI đang gợi ý…" : suggestion?.lineId === lineId && suggestion.kind === kind ? "Ẩn gợi ý" : "Xem gợi ý"}
      </Button>
    ) : null;

  return (
    <Card data-testid={`script-beat-${beat.key}`}>
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>
            B{beat.number} · {beat.fn} · {beat.title}
          </h3>
          <span className={styles.actions}>
            {writing ? <UnitStatusChip state="running" testId={`script-status-${beat.key}`} /> : beat.written && <UnitStatusChip state={beat.status} testId={`script-status-${beat.key}`} />}
            <span className={unitStyles.mine} data-testid={`script-length-${beat.key}`}>
              {beat.written && `${formatSeconds(beat.seconds)} / `}mục tiêu {formatSeconds(beat.targetSeconds)}
              {beat.written && deviation && <span className={`${styles.check} ${styles.checkWarn}`}> · {deviation}</span>}
            </span>
            {beat.mine.length > 0 && (
              <span className={unitStyles.mine} title="AI giữ nguyên phần này khi viết lại" data-testid={`script-mine-${beat.key}`}>
                ✎ Bạn viết
              </span>
            )}
            {beat.stale && beat.written && (
              <span className={unitStyles.mine} data-testid={`script-stale-${beat.key}`}>
                beat đã đổi, xem lại lời thoại
              </span>
            )}
          </span>
        </div>
        <p className={researchStyles.hint}>Mục tiêu: {beat.goal || "(chưa có)"}</p>

        <AiOperationCard run={script.run} active={writing || (writingAll && !beat.written)} showFailure={script.runBeat === null || script.runBeat === beat.key} testId={`script-progress-${beat.key}`} title="AI đang viết lời thoại beat này" />

        {beat.join && (
          <>
            <div className={`${styles.note} ${beat.join.ok ? "" : styles.noteBad}`} data-testid={`script-join-${beat.key}`} data-ok={beat.join.ok}>
              <span className={`${unitStyles.mine}`}>{beat.join.ok ? "✓ nối" : "! nối"}</span>
              <span className={styles.noteText}>
                B{beat.number - 1} → B{beat.number}: {beat.join.msg}
              </span>
              {!beat.join.ok && beat.lines.length > 0 && askButton("join", beat.lines[0].id)}
            </div>
            {beat.lines.length > 0 && suggestionBox(beat.lines[0].id, "join")}
          </>
        )}

        {!beat.written ? (
          <>
            <p className={researchStyles.hint} data-testid={`script-empty-${beat.key}`}>
              Chưa có lời thoại.{aiAvailable ? " AI viết từ mục tiêu và nội dung của beat, hoặc bạn tự viết." : " Chưa cấu hình AI nên bạn tự viết."}
            </p>
            <div className={styles.actions}>
              {aiAvailable && (
                <Button variant="run" className={researchStyles.sm} onClick={() => void script.generate(beat.key, "")} disabled={busy} data-testid={`script-write-${beat.key}`}>
                  AI viết lời thoại beat này
                </Button>
              )}
              <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(true)} disabled={busy} data-testid={`script-write-by-hand-${beat.key}`}>
                Tự viết
              </Button>
            </div>
          </>
        ) : (
          <>
            <div className={styles.lines}>
              {beat.lines.map((line) => {
                const playing = player.playingId === line.id;
                const waiting = player.waitingId === line.id;
                return (
                  <div key={line.id} style={{ display: "contents" }}>
                    <div className={styles.line} id={`ln-${line.id}`} data-current={highlight === line.id || playing} data-testid={`script-line-${line.id}`}>
                      <span className={styles.lineId}>
                        {line.id.slice(1)}
                        <span className={styles.lineSecs}>
                          {line.estimated ? "~" : ""}
                          {line.seconds.toFixed(1)} s
                        </span>
                      </span>
                      <span className={styles.lineBody}>
                        <span data-testid={`script-line-text-${line.id}`}>{line.text}</span>
                        {line.spoken && (
                          <span className={styles.spoken} data-testid={`script-line-spoken-${line.id}`}>
                            <b>Đọc:</b> {line.spoken}
                          </span>
                        )}
                        <span className={styles.tags}>
                          <FactTags ids={line.facts} />
                          {line.flag && (
                            <span className={styles.flag} title={line.flag.msg} data-testid={`script-flag-${line.id}`}>
                              {line.flag.kind}
                            </span>
                          )}
                        </span>
                        {line.audio.state === "failed" && (
                          <span className={styles.audioFail} role="alert" data-testid={`script-audio-error-${line.id}`}>
                            Không đọc được: {line.audio.error}
                          </span>
                        )}
                      </span>
                      <Button
                        variant="ghost"
                        className={researchStyles.sm}
                        onClick={() => (playing || waiting ? player.stop() : void player.play([line.id]))}
                        disabled={locked}
                        data-testid={`script-play-${line.id}`}
                      >
                        {readingLabel(line, playing, waiting)}
                      </Button>
                    </div>
                    {line.flag && (
                      <div className={`${styles.note} ${styles.noteBad}`} data-testid={`script-flagnote-${line.id}`}>
                        <span className={styles.noteText}>
                          <b>{line.flag.kind}:</b> {line.flag.msg}
                        </span>
                        {askButton("line", line.id)}
                      </div>
                    )}
                    {suggestionBox(line.id, "line")}
                  </div>
                );
              })}
            </div>
            <div className={styles.actions}>
              <Button
                variant="ghost"
                className={researchStyles.sm}
                onClick={() => (playingBeat ? player.stop() : void player.play(ids))}
                disabled={listenDisabled}
                data-testid={`script-play-beat-${beat.key}`}
              >
                {playingBeat ? "❚❚ Dừng" : "▶ Nghe cả beat"}
              </Button>
            </div>
            <UnitReviewBar
              status={beat.status}
              approveLabel="Duyệt beat"
              busy={busy}
              canRedo={aiAvailable}
              onEdit={() => setEditing(true)}
              onRedo={() => setRedoing(true)}
              onApprove={() => void script.approve(beat.key)}
              onUnapprove={() => void script.unapprove(beat.key)}
              testId={`script-review-${beat.key}`}
            />
          </>
        )}
      </div>

      <UnitEditModal
        open={editing}
        onClose={() => setEditing(false)}
        title={`Sửa tay · B${beat.number} ${beat.title}`}
        fields={[
          ...beat.lines.flatMap((l) => [
            { name: `line:${l.id}`, label: `Câu ${l.id.slice(1)}`, value: l.text, rows: 2 },
            { name: `spoken:${l.id}`, label: `Chữ để đọc câu ${l.id.slice(1)} (để trống thì đọc đúng như chữ trên)`, value: l.spoken ?? "", rows: 2 },
          ]),
          { name: `add:${lastId}`, label: beat.lines.length === 0 ? "Câu đầu tiên" : "Thêm một câu mới ở cuối", value: "", rows: 2 },
        ]}
        impact={editImpact}
        onSave={(changed) => script.edit(beat.key, changed)}
        note="Xoá trắng một câu để bỏ câu đó. Công thức hay ký hiệu giọng đọc sai thì ghi cách đọc vào ô “Chữ để đọc” (ví dụ: Đê bằng một trừ mờ, rồi chia cho bê); phụ đề vẫn hiện chữ ở ô trên. Sửa xong beat về chờ duyệt; câu đã sửa phải nghe thử lại."
        size="lg"
        testId={`script-edit-${beat.key}`}
      />
      <UnitRedoModal
        open={redoing}
        onClose={() => setRedoing(false)}
        title={`Làm lại kèm ghi chú · B${beat.number} ${beat.title}`}
        impact={redoImpact}
        initialNote={script.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="story" label="Model AI cho bước Viết kịch bản" testId={`script-redo-model-${beat.key}`} /> : undefined}
        onSend={(note) => void script.redo(beat.key, note)}
        testId={`script-redo-${beat.key}`}
      />
    </Card>
  );
}
