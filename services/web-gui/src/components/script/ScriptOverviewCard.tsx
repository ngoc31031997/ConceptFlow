import { useState } from "react";
import type { ScriptState } from "../../api/client";
import type { LinePlayer } from "../../hooks/useLinePlayer";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { ScriptReplaceDialog } from "./ScriptReplaceDialog";
import styles from "./script.module.css";

/** "4:28" for a length in seconds. */
export function formatSeconds(seconds: number): string {
  const total = Math.round(seconds);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

interface ScriptOverviewCardProps {
  projectId: string;
  state: ScriptState;
  player: LinePlayer;
  /** The step cannot be changed or listened to now. */
  disabled: boolean;
  /** The lines were changed by a find-and-replace; the screen loads them again. */
  onReload: () => void | Promise<void>;
}

/**
 * The whole script at a glance: how many lines and beats, how long it runs
 * against the target (from the voice files that exist, else an estimate), the
 * checks over the opening, the closing, the flags and the joins, and the
 * button that plays the script from the top. A check that fails is a warning.
 */
export function ScriptOverviewCard({ projectId, state, player, disabled, onReload }: ScriptOverviewCardProps) {
  const [replacing, setReplacing] = useState(false);
  const { stats } = state;
  const ids = state.beats.flatMap((b) => b.lines.map((l) => l.id));
  const playingAny = player.playingId !== null || player.waitingId !== null;
  return (
    <Card data-testid="script-overview-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Toàn bộ kịch bản</h3>
          <Button variant="ghost" className={researchStyles.sm} onClick={() => setReplacing(true)} disabled={disabled || ids.length === 0} data-testid="script-replace-open">
            Tìm và thay
          </Button>
          <Button
            variant="ghost"
            className={researchStyles.sm}
            onClick={() => (playingAny ? player.stop() : void player.play(ids))}
            disabled={disabled || ids.length === 0}
            data-testid="script-play-all"
          >
            {playingAny ? "❚❚ Dừng" : "▶ Nghe cả kịch bản"}
          </Button>
        </div>
        <dl className={styles.kv}>
          <dt>Lời thoại</dt>
          <dd data-testid="script-counts">
            {stats.lines} câu · {stats.written}/{stats.beats} beat có lời
          </dd>
          <dt>Ước tính</dt>
          <dd data-testid="script-total">
            {formatSeconds(stats.totalSeconds)} · mục tiêu {state.targetLabel}{" "}
            <span className={`${styles.check} ${state.inRange ? "" : styles.checkWarn}`}>{state.inRange ? "✓" : "! ngoài khung"}</span>
            {stats.estimated > 0 && <span className={researchStyles.hint}> · {stats.estimated} câu chưa nghe thử, tính theo tốc độ đọc</span>}
          </dd>
        </dl>
        <div className={styles.checks} data-testid="script-checks">
          {state.checks.map((check) => (
            <span key={check.text} className={`${styles.check} ${check.ok ? "" : styles.checkWarn}`} data-ok={check.ok}>
              {check.ok ? "✓" : "!"} {check.text}
            </span>
          ))}
        </div>
        {player.error && (
          <p className={researchStyles.error} role="alert" data-testid="script-player-error">
            {player.error}
          </p>
        )}
      </div>
      <ScriptReplaceDialog open={replacing} onClose={() => setReplacing(false)} projectId={projectId} onApplied={onReload} />
    </Card>
  );
}
