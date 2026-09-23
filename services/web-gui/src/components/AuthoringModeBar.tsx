import { useState } from "react";
import { Button } from "./ui";
import { generateAuthoringStep, type AuthoringStep, type LlmStatus } from "../api/client";
import type { AuthoringMode } from "../context/ProjectDraftContext";
import { useAuthoringRun, useAuthoringRunDispatch } from "../context/AuthoringRunContext";
import glass from "../styles/glass.module.css";
import styles from "./AuthoringModeBar.module.css";

/** Nhãn tiếng Việt của từng bước, để câu trạng thái nói đúng nó đang ở đâu. */
const STEP_LABELS: Record<AuthoringStep, string> = {
  story: "1a. Dàn ý",
  storyboard: "1b. Storyboard",
  code: "1c. Code",
};

const ALL_STEPS: AuthoringStep[] = ["story", "storyboard", "code"];

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
  /**
   * CR-030 — chuỗi bước mà một lần bấm sẽ chạy, theo đúng thứ tự. Tab 1a
   * truyền cả ba (`story`, `storyboard`, `code`): Creator chỉ nhập chủ đề rồi
   * bấm một lần, server chạy tuần tự, mỗi bước đọc kết quả bước trước đã lưu.
   * Tab 1b/1c truyền đúng một bước, để chạy lại riêng bước đó sau khi sửa tay.
   *
   * CR-031 — để trống ở màn chọn tình huống: ở đó chưa có chủ đề, chưa có
   * artefact nào để sinh, nên chỉ có công tắc chế độ chứ không có nút chạy.
   * Chọn chế độ ngay từ đó là có ích vì nó lưu lên project và đi theo sang cả
   * ba tab.
   */
  steps?: AuthoringStep[];
  /** Bước này sinh ra cái gì, để câu chữ trên nút nói đúng việc nó làm. */
  what?: string;
  /** Kết quả từng bước, để trang nhét thẳng vào ô soạn thảo (FR78.2). */
  onGenerated?: (step: AuthoringStep, content: string) => void;
  /**
   * Việc phải xong trước khi gọi — lưu chủ đề/kết quả bước trước lên server,
   * vì server render prompt từ dữ liệu của nó, không từ state trình duyệt
   * (FR80.1).
   */
  beforeRun?: () => Promise<void>;
  /** Chặn nút chạy dù đã chọn chế độ AI — ví dụ chưa nhập chủ đề. */
  runDisabled?: boolean;
  runDisabledReason?: string;
  /**
   * Mặc định hiện cả chữ giải thích lẫn công tắc chế độ. `false` chỉ để lại
   * nút chạy + trạng thái/lỗi/tiến độ — PipelineSettingsBar dùng khi đang thu
   * gọn, để Creator bấm chạy luôn mà không phải mở "Đổi".
   */
  showSwitch?: boolean;
}

const MODE_LABELS: Record<AuthoringMode, string> = {
  manual: "Copy prompt ra ngoài",
  ai: "Gọi API trực tiếp",
};

/**
 * CR-027 FR79 — cách làm **cả bước 3**, đặt ở đầu mỗi tab 1a/1b/1c.
 *
 * Một lựa chọn cho toàn bộ pipeline, không phải một nút riêng mỗi tab:
 * Creator đã quyết định chạy script này bằng API thì không muốn quyết định
 * lại ở 1b, 1c. Lựa chọn nằm trong draft nên nó sống qua việc đổi tab và tải
 * lại trang, và mặc định là `manual` — đúng cái mọi project vẫn làm trước
 * CR-027.
 *
 * Chế độ `manual` không bao giờ mất đi: nó là đường đi khi chưa có key, hết số
 * dư, provider sập, hoặc khi Creator muốn dùng ChatGPT/Claude/Gemini của mình
 * (FR77.4/FR83.2). Vì vậy khi máy chủ chưa cấu hình key, lựa chọn "Gọi API"
 * hiện ra ở trạng thái không chọn được kèm lý do, chứ không lẳng lặng biến mất
 * — Creator cần biết tính năng có tồn tại và thiếu gì để bật.
 *
 * CR-030 — ở chế độ AI, tab 1a chạy cả ba bước trong một lần bấm (`steps`).
 * Chuỗi chạy ở client chứ không phải một endpoint mới, vì mỗi lượt gọi đã tự
 * lưu kết quả lên server rồi: bước sau render prompt từ đúng dữ liệu bước
 * trước vừa lưu. Đổi lại, khi một bước giữa chừng hỏng thì những bước đã xong
 * vẫn còn nguyên, và Creator chạy tiếp từ tab đang dở thay vì mất cả chuỗi.
 */
export function AuthoringModeBar({
  llm,
  mode,
  onModeChange,
  projectId,
  steps = [],
  what = "",
  onGenerated,
  beforeRun,
  runDisabled,
  runDisabledReason,
  showSwitch = true,
}: AuthoringModeBarProps) {
  // Trạng thái "đang chạy" sống ở AuthoringRunContext, ngoài component này —
  // dùng chung cho cả 3 tab 1a/1b/1c, để tab vừa mở thấy đúng một chuỗi đang
  // chạy dở ở tab khác thay vì tưởng mình rảnh và cho bấm chạy chồng lên.
  const run = useAuthoringRun();
  const dispatchRun = useAuthoringRunDispatch();
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);

  // Chưa biết trạng thái: chưa vẽ thẻ, để nó không nhấp nháy giữa hai hình
  // dạng ngay khi trang mở.
  if (!llm) return null;

  const aiMode = mode === "ai" && llm.enabled;
  const isChain = steps.length > 1;
  const canRun = steps.length > 0;
  const running = run.running;
  // Có chuỗi khác (tab khác) đang chạy, không phải chuỗi của chính nút này —
  // câu trạng thái phải nói rõ đang chờ cái gì, không chỉ "đang chạy" chung
  // chung khiến Creator tưởng máy đứng hình.
  const runningElsewhere = running && run.steps !== steps && run.steps.join() !== steps.join();

  async function handleRun() {
    dispatchRun({ type: "START", steps });
    setError(null);
    setNote(null);
    let at: AuthoringStep | null = null;
    try {
      // Server render prompt từ dữ liệu của chính nó, nên những gì Creator vừa
      // gõ phải lên server trước, không thì prompt thiếu dữ liệu bước này cần.
      if (beforeRun) await beforeRun();
      for (let i = 0; i < steps.length; i += 1) {
        const step = steps[i];
        // Biến cục bộ chứ không đọc lại state `progress` ở khối catch: state
        // vừa set chưa nhìn thấy được trong cùng một lượt chạy, nên câu lỗi sẽ
        // chỉ sai tên bước.
        at = step;
        dispatchRun({ type: "PROGRESS", index: i });
        const result = await generateAuthoringStep(projectId, step);
        onGenerated?.(step, result.content);
        if (result.save_error) {
          // Nội dung sinh ra được nhưng không lưu được: bước sau sẽ render
          // prompt từ dữ liệu cũ trên server, tức là làm sai đề. Dừng chuỗi
          // ngay, giữ lại thứ vừa sinh trong ô soạn thảo.
          setNote(result.save_error);
          break;
        }
      }
    } catch (err) {
      const where = at && isChain ? ` (dừng ở ${STEP_LABELS[at]})` : "";
      setError(
        (err instanceof Error
          ? err.message
          : "Chạy bằng AI thất bại, hoặc chuyển về Copy prompt như cũ.") + where,
      );
    } finally {
      dispatchRun({ type: "FINISH" });
    }
  }

  const runLabel = isChain ? `Chạy cả bước 3 bằng AI (1a → 1b → 1c)` : `Chạy ${what} bằng AI`;

  return (
    <div
      className={showSwitch ? `${glass.card} ${styles.card}` : styles.flat}
      data-testid="authoring-mode-bar"
    >
      {showSwitch && (
        <div className={styles.text}>
          <div className={glass.cardTitle}>Cách làm bước 3</div>
          <p className={styles.hint}>
            {!llm.enabled
              ? llm.reason || "Chưa cấu hình API key nên chỉ có đường copy tay."
              : aiMode
                ? `Áp dụng cho cả 3 tab 1a–1c: hệ thống tự gọi ${llm.provider}, điền kết quả vào ô soạn thảo để bạn sửa. Không tự chuyển bước, không tự nộp render.`
                : "Áp dụng cho cả 3 tab 1a–1c: bạn copy prompt, dán vào ChatGPT/Claude/Gemini rồi dán kết quả về. Đổi sang 'Gọi API' bất cứ lúc nào."}
          </p>
        </div>
      )}

      <div className={showSwitch ? styles.right : styles.rightFlat}>
        {showSwitch && (
          <div className={styles.switchRow}>
            <button
              type="button"
              data-testid="authoring-mode-manual"
              className={`${styles.switchLabelBtn} ${!aiMode ? styles.switchLabelActive : ""}`}
              disabled={running}
              onClick={() => onModeChange("manual")}
            >
              {MODE_LABELS.manual}
            </button>
            <button
              type="button"
              role="switch"
              aria-checked={aiMode}
              aria-label="Cách làm bước 3"
              data-testid="authoring-mode-switch"
              className={`${styles.switch} ${aiMode ? styles.switchOn : ""}`}
              disabled={!llm.enabled || running}
              title={!llm.enabled ? llm.reason || "Chưa cấu hình API key" : undefined}
              onClick={() => onModeChange(mode === "ai" ? "manual" : "ai")}
            >
              <span className={styles.switchKnob} aria-hidden="true" />
            </button>
            <button
              type="button"
              data-testid="authoring-mode-ai"
              className={`${styles.switchLabelBtn} ${aiMode ? styles.switchLabelActive : ""}`}
              disabled={!llm.enabled || running}
              title={!llm.enabled ? llm.reason || "Chưa cấu hình API key" : undefined}
              onClick={() => {
                if (llm.enabled) onModeChange("ai");
              }}
            >
              {MODE_LABELS.ai}
            </button>
          </div>
        )}

        {aiMode && canRun && (
          <div className={styles.run}>
            <Button
              onClick={handleRun}
              disabled={running || runDisabled}
              data-testid={`run-with-ai-${steps[0]}`}
              title={
                runningElsewhere
                  ? "Một chuỗi khác đang chạy — chờ xong đã"
                  : runDisabled
                    ? runDisabledReason
                    : `Gọi trực tiếp ${llm.provider}`
              }
            >
              {running && <span className={styles.spinner} aria-hidden="true" />}
              {running ? "AI đang chạy…" : runLabel}
            </Button>
            {running && (
              <p className={styles.status} data-testid="run-with-ai-running">
                {runningElsewhere
                  ? `Đang chạy ở tab khác: ${
                      run.currentIndex >= 0 ? STEP_LABELS[run.steps[run.currentIndex]] : "..."
                    }. Chờ xong rồi mới chạy tiếp được.`
                  : run.currentIndex >= 0 && isChain
                    ? `Bước ${run.currentIndex + 1}/${steps.length} — ${STEP_LABELS[steps[run.currentIndex]]}. Có thể mất vài phút, đừng đóng trang.`
                    : "Có thể mất vài chục giây, đừng đóng trang."}
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

      {/* Chạy cả chuỗi (1a → 1b → 1c) đụng đúng chỗ Creator từng bị lạc: bấm
          chạy ở 1a rồi lỡ chuyển sang 1b/1c xem tiến độ, màn đó trước đây
          không biết gì về chuỗi đang chạy. Giờ panel này hiện trên CẢ BA tab
          bất cứ khi nào một chuỗi nhiều bước đang chạy, nên đứng ở tab nào
          cũng thấy đủ ba prompt và biết đang chờ đúng bước nào. */}
      {running && (
        <div className={styles.runPanel} data-testid="authoring-run-panel">
          <div className={styles.progressTrack} role="progressbar" aria-label="Tiến độ AI" aria-busy="true">
            <div className={styles.progressBar} />
          </div>
          {run.steps.length > 1 && (
            <ol className={styles.stepper}>
              {ALL_STEPS.map((step, index) => {
                const status = index < run.currentIndex ? "done" : index === run.currentIndex ? "running" : "pending";
                return (
                  <li
                    key={step}
                    className={`${styles.stepItem} ${styles[`stepItem_${status}`]}`}
                    data-testid={`authoring-run-panel-${step}`}
                    aria-current={status === "running" ? "step" : undefined}
                  >
                    <span className={styles.stepIcon} aria-hidden="true">
                      {status === "done" ? "✓" : status === "running" ? <span className={styles.spinnerSm} /> : index + 1}
                    </span>
                    <span className={styles.stepText}>
                      <span className={styles.stepName}>{STEP_LABELS[step]}</span>
                      <span className={styles.stepNote}>
                        {status === "done" ? "Xong" : status === "running" ? "Đang chạy" : "Chờ"}
                      </span>
                    </span>
                  </li>
                );
              })}
            </ol>
          )}
        </div>
      )}
    </div>
  );
}
