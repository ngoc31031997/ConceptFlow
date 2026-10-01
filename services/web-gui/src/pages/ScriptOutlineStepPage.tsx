import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS, authoringChainSteps } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { TopicSummary } from "../components/TopicSummary";
import { WizardNav } from "../components/WizardNav";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getAuthoringState, saveAuthoringStory, createProjectDraft } from "../api/client";
import { stripMarkdownCodeFence } from "../utils/scriptValidation";
import { authoringHint } from "../utils/authoringHint";
import { useRenderedPrompt } from "../hooks/useRenderedPrompt";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { Card, Button, TextArea } from "../components/ui";
import styles from "./WizardSteps.module.css";
import { FLOW_STORY, flowTitle } from "../utils/flow";

/**
 * Bước 3 — Kịch bản (Story Architect), bước soạn đầu tiên của 3–6.
 *
 * Cùng bố cục với các bước 4–6, trên xuống: chủ đề (chỉ đọc, sửa ở bước 1),
 * thanh "Đã chọn … Đổi" kèm nút chạy AI, rồi thẻ prompt + thẻ kết quả (tự làm)
 * hoặc chỉ thẻ kết quả (AI). Nút chạy AI ở đây chạy cả chuỗi 3 → 6, và các
 * bước 4–6 phụ thuộc engine (RoleFor ở server), nên thanh này vẫn cho đổi
 * engine trước khi bấm chạy.
 */
export function ScriptOutlineStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const formats = useVideoFormats();
  const format = formats.find((f) => f.id === draft.videoFormatId);

  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Reload lands here directly: the server copy is the one source of truth
  // later steps read from, so rehydrate it the same way steps 4–6 do.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringTopic)) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (cancelled) return;
        if (state.story && !draft.authoringStory) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
        // Chủ đề nằm ở server, nên nó rehydrate được
        // như 4 artefact kia. "" nghĩa là project chưa lưu chủ đề.
        if (state.topic && !draft.authoringTopic) {
          dispatch({ type: "SET_AUTHORING_TOPIC", payload: state.topic });
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

  // The server fills the template — channel identity, beat sheet
  // at this voice's speaking rate, narration-language rule. The browser only
  // says which topic, language, format and voice the Creator has picked.
  const rendered = useRenderedPrompt({
    role: "story_architect",
    language: draft.voiceLanguage,
    topic: draft.authoringTopic,
    format_id: format?.id,
    format_version: format?.version,
    voice_id: draft.voiceId ?? undefined,
  });
  const prompt = rendered.prompt ?? (rendered.failed ? "Không tải được prompt." : "Đang tải prompt...");

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
  const topicIsEmpty = draft.authoringTopic.trim().length === 0;
  const llm = useLlmStatus();
  // Chế độ lấy từ project ở server (qua draft), nên mở lại dự án
  // ở bất cứ tab nào, trình duyệt nào, sau restart nào cũng đúng chế độ đã chọn.
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  // Chế độ AI chỉ "thật" khi máy chủ có provider: một draft chọn AI trên máy
  // chưa cấu hình key phải quay về đường copy tay, chứ không mất cả hai.
  const aiMode = authoringMode === "ai" && llm?.enabled === true;

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      // Chủ đề đi kèm dàn ý trong cùng một lượt lưu, để server có thứ điền
      // vào {{topic}} lúc tự render prompt.
      // Đổi dàn ý thì storyboard và code dựng từ bản cũ bị xóa ở server: hỏi
      // trước, rồi đọc lại để bản nháp không giữ thứ đã mất.
      const saved = await getAuthoringState(draft.projectId).catch(() => null);
      const changed = saved !== null && saved.story !== "" && saved.story !== draft.authoringStory;
      if (changed && (saved.storyboard || saved.code)) {
        const ok = window.confirm(
          "Dàn ý đã thay đổi. Hình ảnh và code đã dựng từ dàn ý cũ sẽ bị xóa để làm lại. Tiếp tục?",
        );
        if (!ok) return;
      }
      await saveAuthoringStory(draft.projectId, draft.authoringStory, draft.authoringTopic.trim());
      if (changed) {
        dispatch({
          type: "SYNC_AUTHORING",
          payload: { story: draft.authoringStory, storyboard: "", code: "" },
        });
      }
      navigate("/create/script/storyboard");
    } catch {
      setSaveError("Không lưu được dàn ý. Vui lòng thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError ?? authoringHint({ aiMode, isEmpty: storyIsEmpty, what: "dàn ý", ready: "Dàn ý đã sẵn sàng. Bước tiếp theo: dựng hình ảnh." });

  return (
    <div data-testid="script-outline-step-page">
      <AppShell
        currentStep={3}
        wide
        title={flowTitle(FLOW_STORY)}
        subtitle="Dựng dàn ý cho video từ chủ đề."
      >
        <TopicSummary topic={draft.authoringTopic} onEdit={() => navigate("/")} />

        <div className={styles.settingsRow}>
          <PipelineSettingsBar
            renderEngine={draft.renderEngine}
            onEngineChange={(engine) => {
              dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
              if (draft.projectId) {
                void createProjectDraft(draft.projectId, "", draft.voiceLanguage, engine).catch(() => {});
              }
            }}
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
            steps={authoringChainSteps(draft.renderEngine)}
            what="dàn ý"
            runDisabled={topicIsEmpty}
            runDisabledReason="Chưa có chủ đề — quay lại bước 1."
            beforeRun={async () => {
              // Chủ đề và engine lên server trước khi chuỗi chạy: prompt của
              // bước 3 cần chủ đề, các bước 4–6 cần đúng engine.
              await createProjectDraft(
                draft.projectId,
                draft.authoringTopic.trim(),
                draft.voiceLanguage,
                draft.renderEngine,
              );
            }}
            onFollow={(step) => navigate(step === "done" ? AUTHORING_STEP_PATHS.code : AUTHORING_STEP_PATHS[step])}
            onGenerated={(step, content) => {
              if (step === "story") dispatch({ type: "SET_AUTHORING_STORY", payload: content });
              else if (step === "storyboard") dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: content });
              else dispatch({ type: "SET_SCRIPT", payload: stripMarkdownCodeFence(content) });
            }}
          />
        </div>

        <div className={aiMode ? styles.scriptLayoutSingle : styles.scriptLayout}>
          {/* Ở chế độ AI, thẻ prompt không còn việc gì: server tự render đúng
              văn bản này rồi tự gọi. Đổi lại chế độ là nó quay lại nguyên vẹn. */}
          {!aiMode && (
            <Card
              title="1. Sao chép prompt"
              hint="Sao chép prompt rồi dán vào ChatGPT, Claude hoặc Gemini."
            >
              <TextArea
                readOnly
                value={prompt}
                rows={16}
                className={styles.promptTextarea}
                data-testid="script-outline-prompt"
              />
              <Button onClick={handleCopy} disabled={rendered.prompt === null || rendered.stale} className={styles.copyButton} data-testid="script-outline-copy">
                {copied ? "Đã sao chép" : "Sao chép prompt"}
              </Button>
            </Card>
          )}

          <Card
            title={aiMode ? "Dàn ý" : "2. Dán kết quả"}
            hint={
              aiMode
                ? "Kết quả của AI hiện ở đây để bạn chỉnh sửa, rồi bấm Tiếp tục."
                : "Dán dàn ý từ AI vào đây, rồi bấm Tiếp tục."
            }
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
        onBack={() => navigate("/create/script/settings")}
        backLabel="Quay lại cấu hình"
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={storyIsEmpty || saving}
        nextTestId="script-outline-step-next"
      />
    </div>
  );
}
