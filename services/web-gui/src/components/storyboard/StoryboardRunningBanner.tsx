import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import styles from "./storyboard.module.css";

const RUNNING_TEXT = {
  all: "AI đang viết shot các unit còn trống, mỗi unit một lượt, theo thứ tự chuỗi hình, rồi dựng khung.",
  assets: "AI đang lập danh sách tài sản. Tài sản bạn đã sửa tay giữ nguyên.",
  details:
    "AI đang viết chi tiết cho các tài sản chưa có, mỗi tài sản một lượt. Trường bạn đã sửa tay giữ nguyên.",
  detail: "AI đang viết chi tiết của một tài sản. Trường bạn đã sửa tay giữ nguyên.",
  review: "AI đang rà soát cả storyboard.",
  unit: "AI đang viết shot của một unit rồi dựng khung. Trường bạn đã sửa tay giữ nguyên.",
  asset: "Bộ vẽ đang vẽ tài sản.",
  shot: "Đang dựng lại khung của shot.",
};

interface StoryboardRunningBannerProps {
  storyboard: StoryboardControls;
}

/**
 * What the system's AI or drawer is doing on step 11, with the button that stops
 * it. It sticks to the top of the screen so the Creator can stop a run from
 * wherever they have scrolled; nothing shows while no run is in flight.
 */
export function StoryboardRunningBanner({ storyboard }: StoryboardRunningBannerProps) {
  const running = storyboard.running;
  if (running === null) return null;
  let text: string;
  if (typeof running !== "object") text = RUNNING_TEXT[running];
  else if ("unit" in running) text = `${RUNNING_TEXT.unit} (${running.unit})`;
  else if ("detail" in running) text = `${RUNNING_TEXT.detail} (${running.detail})`;
  else if ("asset" in running) text = `${RUNNING_TEXT.asset} (${running.asset})`;
  else text = `${RUNNING_TEXT.shot} (${running.shot})`;
  return (
    <div
      className={`${researchStyles.banner} ${styles.sticky}`}
      role="status"
      data-testid="sb-running"
    >
      <UnitStatusChip state="running" /> {text}{" "}
      <Button variant="ghost" onClick={storyboard.stop} data-testid="sb-stop">
        Dừng
      </Button>
    </div>
  );
}
