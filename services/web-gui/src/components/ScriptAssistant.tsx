import { useMemo, useState } from "react";
import { SelectableOption } from "./SelectableOption";
import type { VideoFormat } from "../types";
import { buildAdjustPromptFor, buildGenerationPromptFor } from "./scriptPrompts";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./ScriptAssistant.module.css";

/** Which of the three situations the Creator is actually in. */
export type ScriptSource = "blank" | "draft" | "ready";

interface ScriptAssistantProps {
  contentLanguage: "vi" | "en";
  /** Format đã chọn — prompt sẽ mang beat sheet của nó (CR-019 FR54). */
  format?: VideoFormat;
  wordsPerMinute?: number;
  /** Replaces the editor's content — used by "dùng script mẫu". */
  onUseTemplate: () => void;
  source: ScriptSource;
  onSourceChange: (source: ScriptSource) => void;
}

const SOURCES: { value: ScriptSource; label: string; hint: string }[] = [
  {
    value: "blank",
    label: "Chưa có gì, chỉ có ý tưởng",
    hint: "Nhập chủ đề, AI sẽ viết script Manim hoàn chỉnh cho bạn",
  },
  {
    value: "draft",
    label: "Đã có script Manim",
    hint: "Nhưng chưa có marker lời thoại — AI sẽ thêm giúp bạn",
  },
  {
    value: "ready",
    label: "Script đã đúng chuẩn",
    hint: "Đã có # NARRATION và self.wait(AUTO) — dán thẳng vào là chạy",
  },
];

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="12" height="12" rx="2" />
      <path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1" />
    </svg>
  );
}

/**
 * Getting a usable script means a round trip through an external AI, and the
 * old "Công cụ AI" dropdown never said so. It listed five items that mixed two
 * unrelated actions — prompts you copy somewhere else, and text inserted right
 * here — under a name that described neither.
 *
 * This asks which of three situations the Creator is in, collects the one
 * thing the prompt is missing (a topic, or their existing script), and hands
 * back a prompt that is ready to paste with nothing left to edit by hand.
 */
export function ScriptAssistant({
  contentLanguage,
  format,
  wordsPerMinute,
  onUseTemplate,
  source,
  onSourceChange,
}: ScriptAssistantProps) {
  const [topic, setTopic] = useState("");
  const [existingScript, setExistingScript] = useState("");
  const [copied, setCopied] = useState(false);

  const prompt = useMemo(
    () =>
      source === "blank"
        ? buildGenerationPromptFor(contentLanguage, topic, format, wordsPerMinute)
        : buildAdjustPromptFor(contentLanguage, existingScript),
    [source, contentLanguage, topic, existingScript, format, wordsPerMinute],
  );

  // The prompt is copyable either way, but saying it is incomplete is more
  // useful than silently handing over one with a placeholder still in it.
  const isFilled = source === "blank" ? topic.trim().length > 0 : existingScript.trim().length > 0;

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
    <div className={glass.card} data-testid="script-assistant">
      <div className={glass.cardTitle}>Bạn đang ở tình huống nào?</div>
      <p className={styles.lead}>
        Chọn đúng tình huống của bạn — các bước bên dưới sẽ đổi theo.
      </p>

      <div className={selectable.stack} role="radiogroup" aria-label="Tình huống script">
        {SOURCES.map((option) => (
          <SelectableOption
            key={option.value}
            selected={source === option.value}
            onSelect={() => onSourceChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`script-source-${option.value}`}
          />
        ))}
      </div>

      {source === "ready" && (
        <div className={styles.panel} data-testid="script-assistant-ready">
          <p className={styles.panelLead}>
            Dán script của bạn vào ô soạn thảo bên dưới. Hệ thống sẽ kiểm tra ngay số marker{" "}
            <code>{"# NARRATION"}</code> có khớp số <code>self.wait(AUTO)</code> không.
          </p>
          <button type="button" className={glass.ghostBtn} onClick={onUseTemplate} data-testid="script-assistant-template">
            Hoặc xem một script mẫu chạy được ngay
          </button>
        </div>
      )}

      {source !== "ready" && (
        <div className={styles.panel} data-testid="script-assistant-guide">
          {/* Step 1 — the one input the prompt is missing. */}
          <div className={styles.step}>
            <span className={styles.stepNum}>1</span>
            <div className={styles.stepBody}>
              <label className={styles.stepLabel} htmlFor="assistant-input">
                {source === "blank" ? "Chủ đề video của bạn là gì?" : "Dán script Manim hiện có của bạn"}
              </label>
              {source === "blank" ? (
                <input
                  id="assistant-input"
                  type="text"
                  className={glass.textInput}
                  data-testid="script-assistant-topic"
                  value={topic}
                  onChange={(event) => setTopic(event.target.value)}
                  placeholder="Ví dụ: Vòng lặp for trong Java, khi nào dùng while thay thế"
                />
              ) : (
                <textarea
                  id="assistant-input"
                  className={`${glass.textArea} ${styles.sourceTextarea}`}
                  data-testid="script-assistant-existing"
                  value={existingScript}
                  onChange={(event) => setExistingScript(event.target.value)}
                  placeholder={"class MyScene(Scene):\n    def construct(self):\n        ..."}
                  rows={5}
                />
              )}
            </div>
          </div>

          {/* Step 2 — copy a prompt that needs no further editing. */}
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
                  {isFilled
                    ? source === "blank"
                      ? "Đã gắn chủ đề của bạn"
                      : "Đã gắn script của bạn"
                    : source === "blank"
                      ? "Chưa nhập chủ đề"
                      : "Chưa dán script"}
                </span>
              </div>
              <details className={styles.preview}>
                <summary className={styles.previewSummary}>Xem trước nội dung prompt</summary>
                <textarea
                  className={`${glass.textArea} ${styles.previewTextarea}`}
                  data-testid="script-assistant-prompt"
                  value={prompt}
                  readOnly
                  rows={10}
                />
              </details>
            </div>
          </div>

          {/* Step 3 — the step nobody was ever told about. */}
          <div className={styles.step}>
            <span className={styles.stepNum}>3</span>
            <div className={styles.stepBody}>
              <div className={styles.stepLabel}>Copy đoạn code AI trả về, dán vào ô soạn thảo bên dưới</div>
              <p className={styles.stepHint}>
                Chỉ lấy phần code Python, không lấy phần AI giải thích. Dán xong hệ thống sẽ tự kiểm tra
                định dạng.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
