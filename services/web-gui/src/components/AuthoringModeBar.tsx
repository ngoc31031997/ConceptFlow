import { useState } from "react";
import { SelectableOption } from "./SelectableOption";
import { Button } from "./ui";
import { generateAuthoringStep, type AuthoringStep, type LlmStatus } from "../api/client";
import type { AuthoringMode } from "../context/ProjectDraftContext";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./AuthoringModeBar.module.css";

interface AuthoringModeBarProps {
  /**
   * Trạng thái provider, do trang sở hữu (useLlmStatus) chứ không phải thẻ
   * này: trang cũng phải biết nó để quyết định bày ô prompt copy tay hay
   * không. Hai nơi tự hỏi riêng thì có lúc chúng trả lời khác nhau, và Creator
   * rơi vào trạng thái không có đường nào — chế độ AI đang chọn nhưng không có
   * nút chạy, mà ô prompt cũng đã bị ẩn. `null` là chưa biết.
   */
  llm: LlmStatus | null;
  mode: AuthoringMode;
  onModeChange: (mode: AuthoringMode) => void;
  projectId: string;
  /** Tab hiện tại — server tự suy ra vai trò prompt từ nó (CR-027 FR78.5). */
  step: AuthoringStep;
  /** Bước này sinh ra cái gì, để câu chữ trên nút nói đúng việc nó làm. */
  what: string;
  /** Kết quả trả về, để trang nhét thẳng vào ô soạn thảo (FR78.2). */
  onGenerated: (content: string) => void;
  /**
   * Việc phải xong trước khi gọi — lưu chủ đề/kết quả bước trước lên server,
   * vì server render prompt từ dữ liệu của nó, không từ state trình duyệt
   * (FR80.1).
   */
  beforeRun?: () => Promise<void>;
  /** Bước 1d truyền kết quả lint vào {{lint_results}} (FR80.3). */
  lintResults?: string;
  /** Chặn nút chạy dù đã chọn chế độ AI — ví dụ chưa nhập chủ đề. */
  runDisabled?: boolean;
  runDisabledReason?: string;
}

const MODES: { value: AuthoringMode; label: string }[] = [
  { value: "manual", label: "Copy prompt ra ngoài" },
  { value: "ai", label: "Gọi API trực tiếp" },
];

/**
 * CR-027 FR79 — cách làm **cả bước 1**, đặt ở đầu mỗi tab 1a/1b/1c/1d.
 *
 * Một lựa chọn cho toàn bộ pipeline, không phải một nút riêng mỗi tab:
 * Creator đã quyết định chạy script này bằng API thì không muốn quyết định
 * lại ở 1b, 1c, 1d. Lựa chọn nằm trong draft nên nó sống qua việc đổi tab và
 * tải lại trang, và mặc định là `manual` — đúng cái mọi project vẫn làm trước
 * CR-027.
 *
 * Chế độ `manual` không bao giờ mất đi: nó là đường đi khi chưa có key, hết số
 * dư, provider sập, hoặc khi Creator muốn dùng ChatGPT/Claude/Gemini của mình
 * (FR77.4/FR83.2). Vì vậy khi máy chủ chưa cấu hình key, lựa chọn "Gọi API"
 * hiện ra ở trạng thái không chọn được kèm lý do, chứ không lẳng lặng biến mất
 * — Creator cần biết tính năng có tồn tại và thiếu gì để bật.
 */
export function AuthoringModeBar({
  llm,
  mode,
  onModeChange,
  projectId,
  step,
  what,
  onGenerated,
  beforeRun,
  lintResults,
  runDisabled,
  runDisabledReason,
}: AuthoringModeBarProps) {
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);

  // Chưa biết trạng thái: chưa vẽ thẻ, để nó không nhấp nháy giữa hai hình
  // dạng ngay khi trang mở.
  if (!llm) return null;

  const aiMode = mode === "ai" && llm.enabled;

  async function handleRun() {
    setRunning(true);
    setError(null);
    setNote(null);
    try {
      // Server render prompt từ dữ liệu của chính nó, nên những gì Creator vừa
      // gõ phải lên server trước, không thì prompt thiếu dữ liệu bước này cần.
      if (beforeRun) await beforeRun();
      const result = await generateAuthoringStep(projectId, step, lintResults);
      onGenerated(result.content);
      if (result.save_error) setNote(result.save_error);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Chạy bằng AI thất bại, hoặc chuyển về Copy prompt như cũ.");
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="authoring-mode-bar">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Cách làm bước 1</div>
        <p className={styles.hint}>
          {!llm.enabled
            ? llm.reason || "Chưa cấu hình API key nên chỉ có đường copy tay."
            : aiMode
              ? `Áp dụng cho cả 4 tab 1a–1d: hệ thống tự gọi ${llm.provider}, điền kết quả vào ô soạn thảo để bạn sửa. Không tự chuyển bước, không tự nộp render.`
              : "Áp dụng cho cả 4 tab 1a–1d: bạn copy prompt, dán vào ChatGPT/Claude/Gemini rồi dán kết quả về. Đổi sang 'Gọi API' bất cứ lúc nào."}
        </p>
      </div>

      <div className={styles.right}>
        <div className={selectable.row} role="radiogroup" aria-label="Cách làm bước 1">
          {MODES.map((option) => {
            const blocked = option.value === "ai" && !llm.enabled;
            return (
              <SelectableOption
                key={option.value}
                selected={mode === option.value && !blocked}
                onSelect={() => {
                  if (!blocked) onModeChange(option.value);
                }}
                label={option.label}
                inline
                testId={`authoring-mode-${option.value}`}
                ariaLabel={blocked ? `${option.label} (chưa cấu hình API key)` : option.label}
              />
            );
          })}
        </div>

        {aiMode && (
          <div className={styles.run}>
            <Button
              onClick={handleRun}
              disabled={running || runDisabled}
              data-testid={`run-with-ai-${step}`}
              title={runDisabled ? runDisabledReason : `Gọi trực tiếp ${llm.provider}`}
            >
              {running ? "AI đang chạy..." : `Chạy ${what} bằng AI`}
            </Button>
            {running && (
              <p className={styles.status} data-testid="run-with-ai-running">
                Có thể mất vài chục giây, đừng đóng trang.
              </p>
            )}
            {!running && runDisabled && runDisabledReason && (
              <p className={styles.status}>{runDisabledReason}</p>
            )}
            {error && (
              <p className={styles.error} data-testid="run-with-ai-error">
                {error}
              </p>
            )}
            {note && <p className={styles.status}>{note}</p>}
          </div>
        )}
      </div>
    </div>
  );
}
