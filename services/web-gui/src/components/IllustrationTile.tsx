import { useState, type ReactNode } from "react";
import { illustrationPreviewUrl, type Illustration } from "../api/client";
import { illustrationStatusLabel } from "../utils/illustrationLabels";
import { StyleWarnings } from "./StyleWarnings";
import glass from "../styles/glass.module.css";
import styles from "./IllustrationTile.module.css";

interface IllustrationTileProps {
  illustration: Illustration;
  folderName?: string;
  busy?: boolean;
  /** Bumped by the parent after a re-render, so the <img> fetches the new file. */
  bust?: number;
  selected?: boolean;
  onOpen?: () => void;
  /** Extra buttons under the picture (Duyệt, Dựng lại, Vẽ lại bằng AI...). */
  actions?: ReactNode;
  /** "Nhờ AI sửa các cảnh báo này": opens the redraw with this note filled in. */
  onFixWarnings?: (note: string) => void;
}

function badgeClass(ill: Illustration): string {
  if (ill.builtin || ill.exemplar) return glass.badgeNeutral;
  return ill.status === "approved" ? glass.badgeSuccess : glass.badgeProgress;
}

/**
 * One drawing of the library as a tile: the PNG still, which turns
 * into its GIF (the figure's own motion) while hovered or focused.
 */
export function IllustrationTile({ illustration: ill, folderName, busy, bust = 0, selected, onOpen, actions, onFixWarnings }: IllustrationTileProps) {
  const [moving, setMoving] = useState(false);
  const [broken, setBroken] = useState(false);
  const [warningsOpen, setWarningsOpen] = useState(false);
  const src = `${illustrationPreviewUrl(ill, moving ? "gif" : "png")}&b=${bust}`;
  return (
    <li
      className={`${styles.tile} ${selected ? styles.selected : ""} ${warningsOpen ? styles.wide : ""}`}
      data-testid={`illustration-tile-${ill.name}`}
      onMouseEnter={() => setMoving(true)}
      onMouseLeave={() => setMoving(false)}
    >
      <button
        type="button"
        className={styles.picture}
        onClick={onOpen}
        onFocus={() => setMoving(true)}
        onBlur={() => setMoving(false)}
        aria-label={`Mở ${ill.title}`}
      >
        {busy ? (
          <span className={styles.placeholder}>Đang dựng…</span>
        ) : broken ? (
          <span className={styles.placeholder}>Chưa có ảnh xem trước</span>
        ) : (
          <img src={src} alt={ill.title} loading="lazy" onError={() => setBroken(true)} onLoad={() => setBroken(false)} />
        )}
      </button>
      <div className={styles.meta}>
        <span className={styles.title}>{ill.title}</span>
        <span className={styles.name}>
          &lt;{ill.name} /&gt;{folderName ? ` · ${folderName}` : ""}
        </span>
        <span className={`${glass.badge} ${badgeClass(ill)}`} data-testid={`illustration-status-${ill.name}`}>
          <span className={glass.badgeDot} aria-hidden="true" />
          {illustrationStatusLabel(ill)}
        </span>
        {ill.kind === "backdrop" && (
          <span className={`${glass.badge} ${glass.badgeNeutral}`} data-testid={`illustration-kind-${ill.name}`}>
            Nền
          </span>
        )}
        {ill.warnings?.length > 0 && (
          <StyleWarnings
            warnings={ill.warnings}
            name={ill.name}
            onFix={!ill.builtin && !ill.exemplar ? onFixWarnings : undefined}
            onToggle={setWarningsOpen}
          />
        )}
      </div>
      {actions && <div className={styles.actions}>{actions}</div>}
    </li>
  );
}
