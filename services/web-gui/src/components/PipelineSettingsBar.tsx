import { useState } from "react";
import { usePresence } from "../hooks/usePresence";
import { AuthoringModeBar, AuthoringModeSwitch } from "./AuthoringModeBar";
import { AuthoringModelPicker } from "./AuthoringModelPicker";
import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import type { AuthoringMode, AuthoringStep, LlmStatus } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PipelineSettingsBar.module.css";

interface PipelineSettingsBarProps {
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
  onFollow?: (step: AuthoringStep | "done") => void;
}

/**
 * Creator đã chọn cách làm ở bước 1; ở 1a/1b/1c hiện lại y nguyên bộ chọn đầy
 * đủ sẽ trông như chưa chọn gì. Lựa chọn này ĐỔI được ở bất cứ tab nào (đó là chủ ý), nhưng "đổi
 * được" và "phải nhìn lại từ đầu mỗi lần chuyển tab" là hai việc khác nhau.
 *
 * Ở 1a/1b/1c, control này mặc định thu gọn thành một dòng tóm tắt ("Tự làm
 * với ChatGPT, Claude…") — chỉ mở lại thành bộ chọn đầy đủ khi Creator
 * bấm "Đổi". Màn chọn tình huống (ScriptStepPage) không dùng component này:
 * ở đó là lần chọn thật đầu tiên, nên vẫn mở sẵn.
 */
export function PipelineSettingsBar({
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
  onFollow,
}: PipelineSettingsBarProps) {
  const [expanded, setExpanded] = useState(false);
  const { running } = useAuthoringRun();
  const { mounted: showPickers, closing } = usePresence(expanded);
  const { models, setModels } = useAuthoringModels(projectId);
  const aiOn = mode === "ai" && !!llm?.enabled;
  const modeLabel = mode === "ai" && llm?.enabled ? "Để AI làm giúp" : "Tự làm với ChatGPT, Claude…";

  return (
    <div className={styles.stack}>
    <div className={`${glass.card} ${styles.card}`} data-testid="pipeline-settings-bar">
      <div className={styles.summaryRow}>
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
        <div className={styles.summaryText}>
          <span>Đã chọn:</span>
          <span className={styles.summaryLabel}>{modeLabel}</span>
        </div>
      </div>

      {showPickers && (
        <div className={`${styles.expanded} ${closing ? glass.revealOut : glass.reveal}`}>
          {llm && <AuthoringModeSwitch llm={llm} mode={mode} onModeChange={onModeChange} disabled={running} />}
          {aiOn && (
            <AuthoringModelPicker
              models={models}
              onChange={setModels}
              options={llm?.models ?? []}
              defaultModel={llm?.default_model ?? ""}
              disabled={running}
              codeStats={llm?.code_stats}
              codeStatsError={llm?.code_stats_error}
            />
          )}
        </div>
      )}

      {/* Nút chạy AI nằm chung thẻ với dòng "Đã chọn": Đổi cách làm và chạy
          là cùng một quyết định, tách hai thẻ khiến màn hình rời rạc. */}
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
        onFollow={onFollow}
        showSwitch={false}
        embedded
      />
    </div>
    </div>
  );
}
