import { useState } from "react";
import { ApiError, startRenderSaga, suggestShortScript } from "../api/client";
import { buildShortScriptSystemPrompt } from "./scriptPrompts";
import glass from "../styles/glass.module.css";
import styles from "./ScriptAssistant.module.css";

interface ShortScriptAssistantProps {
  /** Project video dài sẽ được liên kết làm companion (CR-026 FR73.1). */
  sourceProjectId: string;
  sourceScriptContent?: string;
  contentLanguage: "vi" | "en";
  onCreated: (newProjectId: string) => void;
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
 * Tạo một bản Shorts/TikTok RIÊNG cho một video dài đã có — kịch bản của
 * riêng nó, không phải cắt từ video dài (CR-026). Hai lối lấy script, dùng
 * song song: copy prompt ra ChatGPT/Claude/Gemini (giống hệt luồng đã quen
 * với script dài), hoặc để AI nội bộ (Ollama) soạn thẳng một bản nháp.
 *
 * Nộp xong tạo một PROJECT MỚI (`video_output_mode: "short"`), liên kết
 * 2 chiều với project hiện tại qua `companion_project_id` — không sửa
 * project dài đang có (FR72.1).
 */
export function ShortScriptAssistant({
  sourceProjectId,
  sourceScriptContent,
  contentLanguage,
  onCreated,
}: ShortScriptAssistantProps) {
  const [topic, setTopic] = useState("");
  const [draftScript, setDraftScript] = useState("");
  const [copied, setCopied] = useState(false);
  const [isSuggesting, setIsSuggesting] = useState(false);
  const [suggestError, setSuggestError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const prompt = buildShortScriptSystemPrompt(contentLanguage, topic);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  async function handleSuggestWithLocalAI() {
    setIsSuggesting(true);
    setSuggestError(null);
    try {
      const result = await suggestShortScript({
        topic,
        language: contentLanguage,
        source_script_content: sourceScriptContent,
      });
      // Nháp AI, không tự nộp — Creator vẫn sửa được trước khi bấm nộp
      // (FR71.3), cùng nguyên tắc với "Copy prompt".
      setDraftScript(result.script_content);
    } catch (err) {
      setSuggestError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsSuggesting(false);
    }
  }

  async function handleSubmit() {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const newProjectId = crypto.randomUUID();
      await startRenderSaga({
        project_id: newProjectId,
        script_content: draftScript,
        voice_language: contentLanguage,
        tts_enabled: true,
        subtitle_mode: "track",
        video_output_mode: "short",
        companion_project_id: sourceProjectId,
      });
      onCreated(newProjectId);
    } catch (err) {
      setSubmitError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsSubmitting(false);
    }
  }

  const isFilled = topic.trim().length > 0;

  return (
    <div data-testid="short-script-assistant">
      <div className={styles.step}>
        <span className={styles.stepNum}>1</span>
        <div className={styles.stepBody}>
          <label className={styles.stepLabel} htmlFor="short-script-topic">
            Chủ đề bản ngắn (có thể khác trọng tâm video dài — chọn lát cắt hay nhất)
          </label>
          <input
            id="short-script-topic"
            type="text"
            className={glass.textInput}
            data-testid="short-script-topic"
            value={topic}
            onChange={(event) => setTopic(event.target.value)}
            placeholder="Ví dụ: Vì sao vòng lặp for lại chạy đúng 5 lần"
          />
        </div>
      </div>

      <div className={styles.step}>
        <span className={styles.stepNum}>2</span>
        <div className={styles.stepBody}>
          <div className={styles.stepLabel}>Chọn một cách để có script</div>
          <div className={styles.copyRow}>
            <button
              type="button"
              className={styles.copyButton}
              data-testid="short-script-copy"
              onClick={handleCopy}
            >
              <CopyIcon />
              {copied ? "Đã copy!" : "Copy prompt (dán vào ChatGPT/Claude/Gemini)"}
            </button>
            <button
              type="button"
              className={glass.ghostBtn}
              data-testid="short-script-suggest-ai"
              onClick={handleSuggestWithLocalAI}
              disabled={isSuggesting}
            >
              {isSuggesting ? "Đang soạn..." : "Hoặc soạn bằng AI nội bộ"}
            </button>
            <span className={`${styles.copyStatus} ${isFilled ? styles.copyStatusOn : ""}`}>
              <span className={styles.copyStatusDot} aria-hidden="true" />
              {isFilled ? "Đã gắn chủ đề của bạn" : "Chưa nhập chủ đề"}
            </span>
          </div>
          {suggestError && (
            <p role="alert" className={glass.helperText} style={{ marginTop: 8 }}>
              {suggestError}
            </p>
          )}
          <details className={styles.preview}>
            <summary className={styles.previewSummary}>Xem trước nội dung prompt</summary>
            <textarea
              className={`${glass.textArea} ${styles.previewTextarea}`}
              data-testid="short-script-prompt-preview"
              value={prompt}
              readOnly
              rows={8}
            />
          </details>
        </div>
      </div>

      <div className={styles.step}>
        <span className={styles.stepNum}>3</span>
        <div className={styles.stepBody}>
          <label className={styles.stepLabel} htmlFor="short-script-draft">
            Dán (hoặc sửa) script ngắn ở đây rồi nộp
          </label>
          <textarea
            id="short-script-draft"
            className={`${glass.textArea} ${styles.sourceTextarea}`}
            data-testid="short-script-draft"
            value={draftScript}
            onChange={(event) => setDraftScript(event.target.value)}
            rows={8}
            placeholder={'from conceptflow import *\n\nclass ...Scene(ConceptFlowScene):\n    def construct(self):\n        with self.clip("short"):\n            ...'}
          />
          {submitError && (
            <p role="alert" className={glass.helperText} style={{ marginTop: 8 }}>
              {submitError}
            </p>
          )}
          <div className={glass.ctaRow} style={{ marginTop: 12 }}>
            <button
              type="button"
              className={glass.btnPrimary}
              data-testid="short-script-submit"
              onClick={handleSubmit}
              disabled={isSubmitting || draftScript.trim().length === 0}
            >
              {isSubmitting ? "Đang bắt đầu..." : "Bắt đầu render bản ngắn"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
