import { useNavigate } from "react-router-dom";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import styles from "./ScriptPipelineTabs.module.css";

export type ScriptPipelineTab = "outline" | "storyboard" | "code";

const TABS: { key: ScriptPipelineTab; path: string; label: string }[] = [
  { key: "outline", path: "/create/script/outline", label: "1a. Dàn ý" },
  { key: "storyboard", path: "/create/script/storyboard", label: "1b. Storyboard" },
  { key: "code", path: "/create/script/code", label: "1c. Code" },
];

function CheckIcon() {
  return (
    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

interface ScriptPipelineTabsProps {
  active: ScriptPipelineTab;
  /** Whether each earlier tab already has content — shown as a checkmark for
      orientation only; every tab stays clickable regardless (CR: Creator
      explicitly wants to jump to any step at any time, not a locked stepper). */
  outlineDone: boolean;
  storyboardDone: boolean;
  codeDone: boolean;
}

/**
 * "Bước 3 — Script" used to be one page that silently hopped between
 * separate URLs (/, /create/visual-director, /create/manim-engineer) with no
 * way to see the other steps or jump back to one already done except the
 * couple of hardcoded "Quay lại X" links. This makes all three steps visible
 * at once and always reachable — no step is ever locked behind finishing an
 * earlier one, so a Creator who wants to tweak the outline after already
 * generating code can just click "1a".
 *
 * CR-030 — tab "1d. Duyệt" (Script Reviewer) đã bị bỏ hẳn: bước 1 giờ kết
 * thúc ở 1c và đi thẳng sang /create/settings.
 */
export function ScriptPipelineTabs({ active, outlineDone, storyboardDone, codeDone }: ScriptPipelineTabsProps) {
  const navigate = useNavigate();
  // CR bug report — chạy chuỗi AI rồi lỡ bấm sang tab khác giữa chừng khiến
  // Creator tưởng chuỗi đã dừng (form của tab mới không biết gì về nó nữa,
  // dù state chạy giờ đã dùng chung). Khoá cả 3 tab lại trong lúc chạy: cách
  // duy nhất để theo dõi tiến độ là đứng yên nhìn panel, đúng ý người dùng
  // muốn — 3 prompt hiện cùng lúc, disable, không cho thao tác chồng lên.
  const { running } = useAuthoringRun();
  const doneByKey: Record<ScriptPipelineTab, boolean> = {
    outline: outlineDone,
    storyboard: storyboardDone,
    code: codeDone,
  };

  return (
    <div className={styles.tabs} role="tablist" aria-label="Các bước dựng script">
      {TABS.map((tab, index) => {
        const isActive = tab.key === active;
        const isDone = doneByKey[tab.key];
        const className = [styles.tab, isActive ? styles.active : "", isDone ? styles.done : ""]
          .filter(Boolean)
          .join(" ");
        return (
          <button
            key={tab.key}
            type="button"
            role="tab"
            aria-selected={isActive}
            className={className}
            disabled={running && !isActive}
            title={running && !isActive ? "Đang chạy AI — chờ xong rồi hẵng chuyển tab" : undefined}
            onClick={() => navigate(tab.path)}
            data-testid={`script-tab-${tab.key}`}
          >
            <span className={styles.tabNum}>{isDone ? <CheckIcon /> : index + 1}</span>
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
