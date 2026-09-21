import { useMemo, useState, type ChangeEvent } from "react";
import { END_SCREEN_SNIPPETS, HOOK_SNIPPETS } from "./scriptTemplates";
import { useDebounce } from "../hooks/useDebounce";
import { Button, Card, TextArea } from "./ui";
import glass from "../styles/glass.module.css";
import styles from "./ScriptEditor.module.css";
import { formatDuration } from "../utils/durationEstimate";
import { stripMarkdownCodeFence, validateScript, validateRemotionScript } from "../utils/scriptValidation";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  /** Picks the language of the snippets this inserts (CR-008 FR21.4). */
  contentLanguage: "vi" | "en";
  /** WPM đo được của giọng đang chọn; bỏ trống thì dùng hằng số theo ngôn ngữ (CR-016 FR43). */
  wordsPerMinute?: number;
  /**
   * feature/remotion-engine — this editor used to be Manim-only (hardcoded
   * title, hook/end-screen snippets that insert self.hook()/self.narrate()
   * calls, and validateScript's self.narrate/ConceptFlowScene lint always
   * on). Now that "draft"/"ready" exist for Remotion too (ScriptAssistant),
   * pasting valid Remotion code here was failing the Manim-only lint with a
   * "Chưa tìm thấy class Scene" error that has nothing to do with Remotion.
   * Defaults to "manim" so every other call site (all existing tests) keeps
   * behaving exactly as before without passing this explicitly.
   */
  renderEngine?: "manim" | "remotion";
}

function UploadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 16V4M12 4L7 9M12 4L17 9" />
      <path d="M4 17V19a2 2 0 002 2h12a2 2 0 002-2v-2" />
    </svg>
  );
}

function TemplateIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M4 9h16M9 9v11" />
    </svg>
  );
}

function CheckCircleIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M8.5 12.5l2.3 2.3L16 10" />
    </svg>
  );
}

function WarningIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l10 18H2L12 3z" />
      <path d="M12 10v4M12 17.5v.01" />
    </svg>
  );
}

/**
 * The editor is now only an editor.
 *
 * It used to carry a "Công cụ AI" dropdown that mixed prompts meant to be
 * copied into another tool with snippets inserted right here — the getting-a-
 * script part of the job now belongs to ScriptAssistant, which explains the
 * round trip instead of hiding it in a menu.
 */
export function ScriptEditor({
  value,
  onChange,
  contentLanguage,
  wordsPerMinute,
  renderEngine = "manim",
}: ScriptEditorProps) {
  const isRemotion = renderEngine === "remotion";
  // Debounce the script value to avoid expensive validation on every keystroke
  const debouncedValue = useDebounce(value, 500);

  // validateScript only understands Manim's self.narrate/ConceptFlowScene
  // conventions — running it against Remotion code would just report a
  // confident-looking but meaningless "Chưa tìm thấy class Scene".
  // validateRemotionScript checks Remotion's own structural requirements
  // instead (narrations export, Composition id="creator", etc.).
  const validation = useMemo(
    () => validateScript(debouncedValue, contentLanguage, wordsPerMinute),
    [debouncedValue, contentLanguage, wordsPerMinute],
  );
  const remotionValidation = useMemo(() => validateRemotionScript(debouncedValue), [debouncedValue]);
  const [importError, setImportError] = useState<string | null>(null);
  const hasScript = value.trim().length > 0;

  function handleFileImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      onChange(stripMarkdownCodeFence(String(reader.result ?? "")));
      setImportError(null);
    };
    reader.onerror = () => setImportError("Không đọc được file, thử lại hoặc dán trực tiếp");
    reader.readAsText(file);
    event.target.value = "";
  }

  return (
    <Card
      title={isRemotion ? "Script Remotion (.tsx)" : "Script Manim (.py)"}
      headerAction={
        <div className={styles.headerActions}>
          {/* Snippets insert self.hook()/self.call_to_action() calls — a
              Manim-only convention Remotion code has no equivalent for. */}
          {hasScript && !isRemotion && (
            <>
              <Button
                variant="ghost"
                data-testid="script-editor-insert-hook"
                onClick={() => onChange(`${value}\n${HOOK_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn hook mở đầu
              </Button>
              <Button
                variant="ghost"
                data-testid="script-editor-insert-end-screen"
                onClick={() => onChange(`${value}\n${END_SCREEN_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn end screen
              </Button>
            </>
          )}
          <label className={glass.ghostBtn}>
            <UploadIcon />
            Nhập từ file
            <input
              type="file"
              accept={isRemotion ? ".tsx,.ts" : ".py"}
              onChange={handleFileImport}
              className={styles.hiddenFileInput}
            />
          </label>
        </div>
      }
    >
      <TextArea
        className={styles.textarea}
        data-testid="new-project-script-textarea"
        value={value}
        onChange={(event) => onChange(stripMarkdownCodeFence(event.target.value))}
        placeholder={
          isRemotion
            ? 'export const narrations: string[] = [\n  "Loi thoai cho canh nay",\n];\n\nfunction CreatorComposition({segments = []}) {\n  return <Segments segments={segments}>{(index) => <TitleText>{narrations[index]}</TitleText>}</Segments>;\n}'
            : 'from conceptflow import *\n\nclass DemoScene(ConceptFlowScene):\n    def construct(self):\n        self.narrate("Loi thoai cho canh nay")'
        }
        rows={11}
      />

      {importError && (
        <div className={styles.validationError} role="alert">
          <WarningIcon />
          {importError}
        </div>
      )}

      {hasScript && isRemotion && (
        <div
          id="script-validation"
          className={remotionValidation.isValid ? styles.validationOk : styles.validationError}
          data-testid="script-editor-validation"
        >
          {remotionValidation.isValid ? (
            <>
              <CheckCircleIcon />
              Hợp lệ: {remotionValidation.narrationCount} đoạn lời thoại.
            </>
          ) : (
            <>
              <WarningIcon />
              {remotionValidation.message}
            </>
          )}
        </div>
      )}

      {hasScript && !isRemotion && (
        <div
          id="script-validation"
          className={validation.isValid ? styles.validationOk : styles.validationError}
          data-testid="script-editor-validation"
        >
          {validation.isValid ? (
            <>
              <CheckCircleIcon />
              Hợp lệ: {validation.narrationCount} đoạn lời thoại, {validation.totalWords} từ.
            </>
          ) : (
            <>
              <WarningIcon />
              {validation.message}
            </>
          )}
        </div>
      )}

      {hasScript && !isRemotion && validation.narrationCount > 0 && (
        <div className={styles.durationEstimate} data-testid="script-duration-estimate">
          <div className={styles.durationHeadline}>
            ≈ {formatDuration(validation.estimatedNarrationSeconds)} lời thoại
          </div>
          <div className={styles.durationCaveat}>
            Chưa tính thời gian animation, nên video thật sẽ dài hơn con số này.{" "}
            {wordsPerMinute
              ? "Tốc độ đọc lấy từ số đo thật của giọng bạn đang chọn."
              : "Tốc độ đọc là mức trung bình; sau vài video hệ thống sẽ hiệu chỉnh theo giọng bạn chọn."}
          </div>
          <ol className={styles.durationBreakdown}>
            {validation.narrations.map((narration, index) => (
              <li key={index}>
                <span className={styles.durationBreakdownTime}>
                  {formatDuration(narration.seconds)}
                </span>
                <span className={styles.durationBreakdownText}>{narration.text}</span>
              </li>
            ))}
          </ol>
        </div>
      )}

      <div className={glass.cardHint}>
        {isRemotion ? (
          <>
            Mỗi câu lời thoại là một phần tử trong <code>export const narrations</code> — hệ thống tạo
            giọng đọc cho từng câu theo đúng thứ tự, và <code>{'<Segments>'}</code> hiển thị hình ảnh
            khớp với đoạn đang đọc.
          </>
        ) : (
          <>
            Mỗi câu lời thoại là một lời gọi <code>{'self.narrate("...")'}</code> — hệ thống tạo giọng
            đọc cho từng câu và giữ animation đúng bằng thời lượng audio thật. Lời gọi này dùng được
            cả trong vòng lặp và trong hàm.
          </>
        )}
      </div>
    </Card>
  );
}
