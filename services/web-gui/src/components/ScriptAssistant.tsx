import { useMemo, useState } from "react";
import { buildAdjustPromptFor, buildRemotionAdjustPromptFor } from "./scriptPrompts";
import { Card, TextArea } from "./ui";
import styles from "./ScriptAssistant.module.css";

interface ScriptAssistantProps {
  contentLanguage: "vi" | "en";
  /** Which engine's conventions the adjust-prompt/hints should talk about. */
  renderEngine: "manim" | "remotion";
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1" />
    </svg>
  );
}

/**
 * Trợ lý chuẩn hoá code sẵn có: nhận code Creator đã viết ở đâu đó, trả về
 * một prompt yêu cầu AI sửa nó cho khớp quy ước hệ thống. Nó CHỈ sinh prompt —
 * kết quả được dán vào ô soạn thảo của tab 1c, không phải vào đây.
 *
 * CR-031 — trước đây component này còn một nhánh "ready" chỉ gồm một đoạn
 * hướng dẫn và nút script mẫu, dùng ở màn "/" cũ. Màn đó giờ chỉ còn chọn
 * tình huống, và cả hai thứ kia đã về tab 1c (nơi ô soạn thảo thật sự nằm),
 * nên nhánh đó không còn ai gọi.
 */
export function ScriptAssistant({ contentLanguage, renderEngine }: ScriptAssistantProps) {
  const [existingScript, setExistingScript] = useState("");
  const [copied, setCopied] = useState(false);

  const prompt = useMemo(
    () =>
      renderEngine === "remotion"
        ? buildRemotionAdjustPromptFor(contentLanguage, existingScript)
        : buildAdjustPromptFor(contentLanguage, existingScript),
    [contentLanguage, existingScript, renderEngine],
  );

  const isFilled = existingScript.trim().length > 0;

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  return (
    <Card data-testid="script-assistant">
      <div data-testid="script-assistant-guide">
        <div className={styles.step}>
          <span className={styles.stepNum}>1</span>
          <div className={styles.stepBody}>
            <label className={styles.stepLabel} htmlFor="assistant-input">
              Dán code {renderEngine === "remotion" ? "Remotion" : "Manim"} hiện có của bạn
            </label>
            <TextArea
              id="assistant-input"
              className={styles.sourceTextarea}
              data-testid="script-assistant-existing"
              value={existingScript}
              onChange={(event) => setExistingScript(event.target.value)}
              placeholder={
                renderEngine === "remotion"
                  ? "export const narrations = [...];\n\nfunction CreatorComposition(...) {\n  ...\n}"
                  : "class MyScene(Scene):\n    def construct(self):\n        ..."
              }
              rows={5}
            />
          </div>
        </div>

        <div className={styles.step}>
          <span className={styles.stepNum}>2</span>
          <div className={styles.stepBody}>
            <div className={styles.stepLabel}>Copy prompt rồi dán vào ChatGPT, Claude hoặc Gemini</div>
            <div className={styles.copyRow}>
              <button
                type="button"
                className={styles.copyButton}
                data-testid="script-assistant-copy"
                onClick={handleCopy}
              >
                <CopyIcon />
                {copied ? "Đã copy!" : "Copy prompt"}
              </button>
              <span className={`${styles.copyStatus} ${isFilled ? styles.copyStatusOn : ""}`}>
                <span className={styles.copyStatusDot} aria-hidden="true" />
                {isFilled ? "Đã gắn script của bạn" : "Chưa dán script"}
              </span>
            </div>
            <details className={styles.preview}>
              <summary className={styles.previewSummary}>Xem trước nội dung prompt</summary>
              <TextArea
                className={styles.previewTextarea}
                data-testid="script-assistant-prompt"
                value={prompt}
                readOnly
                rows={10}
              />
            </details>
          </div>
        </div>

        <div className={styles.step}>
          <span className={styles.stepNum}>3</span>
          <div className={styles.stepBody}>
            <div className={styles.stepLabel}>Copy đoạn code AI trả về, dán vào ô soạn thảo bên cạnh</div>
            <p className={styles.stepHint}>
              {renderEngine === "remotion"
                ? "Chỉ lấy phần code TypeScript, không lấy phần AI giải thích."
                : "Chỉ lấy phần code Python, không lấy phần AI giải thích. Dán xong hệ thống sẽ tự kiểm tra định dạng."}
            </p>
          </div>
        </div>
      </div>
    </Card>
  );
}
