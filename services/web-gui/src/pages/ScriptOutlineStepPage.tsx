import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ScriptPipelineTabs } from "../components/ScriptPipelineTabs";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringStory } from "../api/client";
import { buildBeatSheetSection, NARRATION_LANGUAGE_RULE } from "../components/scriptPrompts";
import { useVoiceCalibration, wordsPerMinuteFor } from "../hooks/useVoiceCalibration";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { Card, Button, TextInput, TextArea } from "../components/ui";
import styles from "./WizardSteps.module.css";

const TOPIC_PLACEHOLDER = "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]";

/**
 * Bước 1a (Story Architect) — first tab of the "Bước 1 — Script" sub-wizard.
 * Used to be baked into ScriptStepPage + ScriptAssistant as the "blank"
 * situation; pulled out into its own tab/route so all 4 pipeline steps
 * (dàn ý/storyboard/code/duyệt) are visible and reachable at once (see
 * ScriptPipelineTabs), instead of a single hidden path through "/".
 *
 * The engine choice (Manim vs Remotion) is NOT asked here — this step's
 * output (a plain-text story outline) is identical either way; only step 1c
 * (Code) needs to know which engine, to fetch the right system prompt.
 */
export function ScriptOutlineStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const calibration = useVoiceCalibration();
  const formats = useVideoFormats();
  const format = formats.find((f) => f.id === draft.videoFormatId);

  const [template, setTemplate] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Reload lands here directly (or the Creator jumps back to "1a" from a
  // later tab) — the in-memory draft survives via localStorage already, but
  // the server copy is the one source of truth later steps read from, so
  // rehydrate it the same way the other 3 tabs do.
  useEffect(() => {
    if (!draft.projectId || draft.authoringStory) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (!cancelled && state.story) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
      })
      .catch(() => {
        /* best-effort — the Creator can still type/paste normally */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId]);

  useEffect(() => {
    let cancelled = false;
    getPromptTemplate("story_architect", draft.voiceLanguage)
      .then((t) => {
        if (!cancelled) setTemplate(t.template_text);
      })
      .catch(() => {
        if (!cancelled) setTemplate(null);
      });
    return () => {
      cancelled = true;
    };
  }, [draft.voiceLanguage]);

  const formatBeats = format ? buildBeatSheetSection(format, draft.voiceLanguage, wordsPerMinuteFor(calibration, draft.voiceId)) : "";
  const prompt = (template ?? "Đang tải prompt...")
    .split("{{topic}}")
    .join(draft.authoringTopic.trim() || TOPIC_PLACEHOLDER)
    .split("{{format_beats}}")
    .join(formatBeats)
    .split("{{narration_language_rule}}")
    .join(NARRATION_LANGUAGE_RULE[draft.voiceLanguage]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  const storyIsEmpty = draft.authoringStory.trim().length === 0;

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringStory(draft.projectId, draft.authoringStory);
      navigate("/create/script/storyboard");
    } catch {
      setSaveError("Không lưu được dàn ý, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : storyIsEmpty
      ? "Dán dàn ý câu chuyện AI trả về để tiếp tục"
      : "Dàn ý đã sẵn sàng — bước tiếp theo sẽ dựng storyboard hình ảnh";

  return (
    <div data-testid="script-outline-step-page">
      <AppShell currentStep={1} wide title="Bước 1 — Script" subtitle="1a. Dựng dàn ý câu chuyện với Story Architect.">
        <ScriptPipelineTabs
          active="outline"
          outlineDone={!storyIsEmpty}
          storyboardDone={draft.authoringStoryboard.trim().length > 0}
          codeDone={draft.scriptContent.trim().length > 0}
        />

        <div className={styles.scriptLayout}>
          <Card title="1. Copy prompt" hint="Nhập chủ đề, copy prompt rồi dán vào ChatGPT, Claude hoặc Gemini.">
            <TextInput
              type="text"
              data-testid="script-outline-topic"
              value={draft.authoringTopic}
              onChange={(event) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: event.target.value })}
              placeholder="Ví dụ: Vòng lặp for trong Java, khi nào dùng while thay thế"
              style={{ marginBottom: 12 }}
            />
            <TextArea
              readOnly
              value={prompt}
              rows={16}
              className={styles.promptTextarea}
              data-testid="script-outline-prompt"
            />
            <Button onClick={handleCopy} className={styles.copyButton} data-testid="script-outline-copy">
              {copied ? "Đã copy!" : "Copy prompt"}
            </Button>
          </Card>

          <Card
            title="2. Dán kết quả"
            hint="Dán dàn ý AI trả về, rồi bấm Tiếp tục để chuyển sang bước 1b (Storyboard)."
          >
            <TextArea
              id="story-outline-input"
              value={draft.authoringStory}
              onChange={(event) => dispatch({ type: "SET_AUTHORING_STORY", payload: event.target.value })}
              rows={16}
              className={styles.promptTextarea}
              placeholder={"CÂU HỎI CỐT LÕI: ...\nINSIGHT CỐT LÕI: ...\n\nBEAT 1 — ...\n..."}
              data-testid="script-outline-story-input"
            />
          </Card>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError}
        onBack={() => navigate("/")}
        backLabel="Quay lại chọn tình huống"
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={storyIsEmpty || saving}
        nextTestId="script-outline-step-next"
      />
    </div>
  );
}
