import { useMemo, useState } from "react";
import { buildAdjustPromptFor, buildRemotionAdjustPromptFor } from "./scriptPrompts";
import { Card, Button, TextArea } from "./ui";
import styles from "./ScriptAssistant.module.css";

/**
 * "Dựng từ đầu" (blank) moved out to its own 4-tab sub-wizard
 * (ScriptOutlineStepPage → .../storyboard → .../code → .../review, see
 * ScriptPipelineTabs) — this component now only covers the two situations
 * that skip that pipeline entirely: the Creator already has SOME code and
 * either needs it adjusted to fit this system's conventions ("draft") or it
 * already fits and just needs pasting into the editor ("ready").
 */
export type ScriptSource = "draft" | "ready";

interface ScriptAssistantProps {
  contentLanguage: "vi" | "en";
  /** Which engine's conventions the adjust-prompt/hints should talk about. */
  renderEngine: "manim" | "remotion";
  /** Replaces the editor's content — used by "dùng script mẫu" (Manim only). */
  onUseTemplate: () => void;
  source: ScriptSource;
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
 * "draft": the Creator has an existing script that doesn't yet fit this
 * system's conventions — collects it, hands back an adjust-prompt ready to
 * paste into an external AI, whose result gets pasted into the sibling
 * ScriptEditor (not here — this only produces the prompt).
 *
 * "ready": the script already fits; no AI round trip needed at all.
 */
export function ScriptAssistant({ contentLanguage, renderEngine, onUseTemplate, source }: ScriptAssistantProps) {
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

  if (source === "ready") {
    return (
      <Card data-testid="script-assistant">
        <div data-testid="script-assistant-ready">
          <p className={styles.panelLead}>
            {renderEngine === "remotion" ? (
              <>
                Dán code của bạn vào ô soạn thảo bên cạnh. Đảm bảo đã có{" "}
                <code>export const narrations</code> và <code>{'<Composition id="creator" ...>'}</code>{" "}
                đúng chuẩn hệ thống.
              </>
            ) : (
              <>
                Dán script của bạn vào ô soạn thảo bên cạnh. Hệ thống sẽ kiểm tra ngay script có kế thừa{" "}
                <code>ConceptFlowScene</code> và dùng <code>{'self.narrate("...")'}</code> đúng chuẩn không.
              </>
            )}
          </p>
          {renderEngine === "manim" && (
            <Button variant="ghost" onClick={onUseTemplate} data-testid="script-assistant-template">
              Hoặc xem một script mẫu chạy được ngay
            </Button>
          )}
        </div>
      </Card>
    );
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
