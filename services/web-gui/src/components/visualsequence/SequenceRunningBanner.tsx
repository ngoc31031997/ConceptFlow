import type { VisualSequenceControls } from "../../hooks/useVisualSequence";
import researchStyles from "../research/research.module.css";
import { Button } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import styles from "./sequence.module.css";

const RUNNING_TEXT = {
  all: "AI đang viết chuỗi các unit còn trống, mỗi unit một lượt, theo thứ tự kế hoạch hình.",
  review: "AI đang rà soát cả chuỗi hình.",
  motifs: "AI đang đặt motif. Motif bạn đã sửa tay giữ nguyên.",
  beat: "AI đang viết lại từng unit của một beat. Trường bạn đã sửa tay giữ nguyên.",
  unit: "AI đang viết chuỗi một unit. Trường bạn đã sửa tay giữ nguyên.",
};

interface SequenceRunningBannerProps {
  sequence: VisualSequenceControls;
}

/**
 * What the system's AI is doing on step 10, with the button that stops it. It
 * sticks to the top of the screen so the Creator can stop a run from wherever
 * they have scrolled; nothing shows while no run is in flight.
 */
export function SequenceRunningBanner({ sequence }: SequenceRunningBannerProps) {
  const running = sequence.running;
  if (running === null) return null;
  const text =
    typeof running !== "object"
      ? RUNNING_TEXT[running]
      : "unit" in running
        ? `${RUNNING_TEXT.unit} (${running.unit})`
        : `${RUNNING_TEXT.beat} (${running.beat})`;
  return (
    <div
      className={`${researchStyles.banner} ${styles.sticky}`}
      role="status"
      data-testid="seq-running"
    >
      <UnitStatusChip state="running" /> {text}{" "}
      <Button variant="ghost" onClick={sequence.stop} data-testid="seq-stop">
        Dừng
      </Button>
    </div>
  );
}
