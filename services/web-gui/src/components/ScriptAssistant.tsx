import { useEffect, useMemo, useState } from "react";
import { SelectableOption } from "./SelectableOption";
import type { VideoFormat } from "../types";
import { buildAdjustPromptFor, buildBeatSheetSection, NARRATION_LANGUAGE_RULE } from "./scriptPrompts";
import { getPromptTemplate } from "../api/client";
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
  /**
   * CR-025 — the dàn ý câu chuyện (Story Architect output) the Creator pasted
   * back, and its setter. Only used when source === "blank": that path now
   * asks for a story outline first (plain text), not Manim code.
   */
  storyOutline: string;
  onStoryOutlineChange: (value: string) => void;
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
    hint: "Nhưng chưa có lời thoại self.narrate(...) — AI sẽ thêm giúp bạn",
  },
  {
    value: "ready",
    label: "Script đã đúng chuẩn",
    hint: "Đã dùng self.narrate(\"...\"), kế thừa ConceptFlowScene — dán thẳng vào là chạy",
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
  storyOutline,
  onStoryOutlineChange,
}: ScriptAssistantProps) {
  const [topic, setTopic] = useState("");
  const [existingScript, setExistingScript] = useState("");
  const [copied, setCopied] = useState(false);

  // CR-025: the "blank" path's prompt now comes from the DB-backed
  // story_architect template instead of a hardcoded builder, so an editor
  // can change the wording without rebuilding web-gui. format_beats and the
  // narration-language rule stay computed client-side (they are data, not
  // editable prose) and get substituted into the fetched template text.
  const [storyArchitectTemplate, setStoryArchitectTemplate] = useState<string | null>(null);
  useEffect(() => {
    if (source !== "blank") return;
    let cancelled = false;
    getPromptTemplate("story_architect", contentLanguage)
      .then((template) => {
        if (!cancelled) setStoryArchitectTemplate(template.template_text);
      })
      .catch(() => {
        if (!cancelled) setStoryArchitectTemplate(null);
      });
    return () => {
      cancelled = true;
    };
  }, [source, contentLanguage]);

  const prompt = useMemo(() => {
    if (source !== "blank") return buildAdjustPromptFor(contentLanguage, existingScript);
    const formatBeats = format ? buildBeatSheetSection(format, contentLanguage, wordsPerMinute) : "";
    const base = storyArchitectTemplate ?? "Đang tải prompt...";
    return base
      .split("{{topic}}")
      .join(topic.trim() || "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]")
      .split("{{format_beats}}")
      .join(formatBeats)
      .split("{{narration_language_rule}}")
      .join(NARRATION_LANGUAGE_RULE[contentLanguage]);
  }, [source, contentLanguage, topic, existingScript, format, wordsPerMinute, storyArchitectTemplate]);

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
            Dán script của bạn vào ô soạn thảo bên dưới. Hệ thống sẽ kiểm tra ngay script có kế thừa{" "}
            <code>ConceptFlowScene</code> và dùng <code>{'self.narrate("...")'}</code> đúng chuẩn không.
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

          {/* Step 3 — CR-025: for "blank", this is now the story outline
              (plain structured text), not Manim code — the pipeline's next
              3 steps (Visual Director/Manim Engineer/Script Reviewer) turn
              it into code later. */}
          <div className={styles.step}>
            <span className={styles.stepNum}>3</span>
            <div className={styles.stepBody}>
              {source === "blank" ? (
                <>
                  <label className={styles.stepLabel} htmlFor="story-outline-input">
                    Dán kết quả AI trả về vào đây
                  </label>
                  <p className={styles.stepHint}>
                    Đây là dàn ý câu chuyện (câu hỏi cốt lõi, insight, lời thoại nháp từng beat) — KHÔNG
                    phải code. Dán nguyên văn phần AI trả lời, không cần chỉnh sửa.
                  </p>
                  <textarea
                    id="story-outline-input"
                    className={`${glass.textArea} ${styles.sourceTextarea}`}
                    data-testid="script-assistant-story-outline"
                    value={storyOutline}
                    onChange={(event) => onStoryOutlineChange(event.target.value)}
                    placeholder={"CÂU HỎI CỐT LÕI: ...\nINSIGHT CỐT LÕI: ...\n\nBEAT 1 — ...\n..."}
                    rows={8}
                  />
                </>
              ) : (
                <>
                  <div className={styles.stepLabel}>Copy đoạn code AI trả về, dán vào ô soạn thảo bên dưới</div>
                  <p className={styles.stepHint}>
                    Chỉ lấy phần code Python, không lấy phần AI giải thích. Dán xong hệ thống sẽ tự kiểm tra
                    định dạng.
                  </p>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
