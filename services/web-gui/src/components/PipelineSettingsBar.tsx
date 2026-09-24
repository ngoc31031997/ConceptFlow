import { useState } from "react";
import { RenderEnginePicker } from "./RenderEnginePicker";
import { AuthoringModeBar, AuthoringModeSwitch } from "./AuthoringModeBar";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import type { RenderEngine } from "../context/ProjectDraftContext";
import type { AuthoringMode, AuthoringStep, LlmStatus } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PipelineSettingsBar.module.css";

interface PipelineSettingsBarProps {
  renderEngine: RenderEngine;
  /**
   * Bỏ trống thì engine chỉ hiện trong dòng tóm tắt, không đổi được ở đây —
   * dùng cho 1b, tab duy nhất không cần biết engine để dựng storyboard
   * (storyboard đọc dàn ý, không đọc engine).
   */
  onEngineChange?: (engine: RenderEngine) => void;
  llm: LlmStatus | null;
  mode: AuthoringMode;
  onModeChange: (mode: AuthoringMode) => void;
  projectId: string;
  steps?: AuthoringStep[];
  what?: string;
  runDisabled?: boolean;
  runDisabledReason?: string;
  beforeRun?: () => Promise<void>;
  onGenerated?: (step: AuthoringStep, content: string) => void;
}

/**
 * CR-031 bug report — Creator đã chọn engine và cách làm ở bước 1, rồi bước
 * vào 1a/1b/1c lại thấy y nguyên hai bộ chọn đầy đủ, y như chưa chọn gì. Đúng
 * là hai lựa chọn này ĐỔI được ở bất cứ tab nào (đó là chủ ý), nhưng "đổi
 * được" và "phải nhìn lại từ đầu mỗi lần chuyển tab" là hai việc khác nhau.
 *
 * Ở 1a/1b/1c, control này mặc định thu gọn thành một dòng tóm tắt ("Manim ·
 * Copy prompt ra ngoài") — chỉ mở lại thành hai bộ chọn đầy đủ khi Creator
 * bấm "Đổi". Màn chọn tình huống (ScriptStepPage) không dùng component này:
 * ở đó là lần chọn thật đầu tiên, nên vẫn mở sẵn.
 */
export function PipelineSettingsBar({
  renderEngine,
  onEngineChange,
  llm,
  mode,
  onModeChange,
  projectId,
  steps,
  what,
  runDisabled,
  runDisabledReason,
  beforeRun,
  onGenerated,
}: PipelineSettingsBarProps) {
  const [expanded, setExpanded] = useState(false);
  const { running } = useAuthoringRun();
  const engineLabel = renderEngine === "remotion" ? "Remotion" : "Manim";
  const modeLabel = mode === "ai" && llm?.enabled ? "Gọi API trực tiếp" : "Copy prompt ra ngoài";

  return (
    <div className={styles.stack}>
    <div className={`${glass.card} ${styles.card}`} data-testid="pipeline-settings-bar">
      <div className={styles.summaryRow}>
        <div className={styles.summaryText}>
          <span>Đã chọn:</span>
          <span className={styles.summaryLabel}>{engineLabel}</span>
          <span className={styles.dot}>·</span>
          <span className={styles.summaryLabel}>{modeLabel}</span>
        </div>
        <button
          type="button"
          className={styles.toggle}
          onClick={() => setExpanded((v) => !v)}
          aria-expanded={expanded}
          disabled={running}
          title={running ? "AI đang chạy — chờ xong mới đổi được" : undefined}
          data-testid="pipeline-settings-toggle"
        >
          {expanded ? "Xong" : "Đổi"}
        </button>
      </div>

      {expanded && (
        <div className={styles.expanded}>
          {onEngineChange && <RenderEnginePicker value={renderEngine} onChange={onEngineChange} disabled={running} />}
          {llm && <AuthoringModeSwitch llm={llm} mode={mode} onModeChange={onModeChange} disabled={running} />}
        </div>
      )}

    </div>

      {/* Thẻ chạy nằm ngoài thẻ chọn: chỉ hiện khi chọn "Gọi API trực tiếp". */}
      <AuthoringModeBar
        llm={llm}
        mode={mode}
        onModeChange={onModeChange}
        projectId={projectId}
        steps={steps}
        what={what}
        runDisabled={runDisabled}
        runDisabledReason={runDisabledReason}
        beforeRun={beforeRun}
        onGenerated={onGenerated}
        showSwitch={false}
      />
    </div>
  );
}
