import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import {
  getAuthoringState,
  saveAuthoringStory,
  createProjectDraft,
  type SimilarProject,
} from "../api/client";
import { stripMarkdownCodeFence } from "../utils/scriptValidation";
import { useRenderedPrompt } from "../hooks/useRenderedPrompt";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { useDebounce } from "../hooks/useDebounce";
import { Card, Button, TextInput, TextArea } from "../components/ui";
import styles from "./WizardSteps.module.css";

/**
 * Bước 1a (Story Architect) — first tab of the "Bước 3 — Script" sub-wizard.
 * Used to be baked into ScriptStepPage + ScriptAssistant as the "blank"
 * situation; pulled out into its own tab/route so all 3 pipeline steps
 * (dàn ý/storyboard/code) are visible and reachable at once (see
 * ScriptPipelineTabs), instead of a single hidden path through "/".
 *
 * The engine choice (Manim vs Remotion) does not change THIS step's own
 * prompt — a plain-text story outline reads the same either way — but
 * CR-030's "chạy cả bước 3 bằng AI" button runs 1b (storyboard) and 1c
 * (code) too, and those two DO branch by engine (RoleFor on the server). So
 * the picker lives here as well, not only on 1c: choosing it up front, before
 * the chain runs, is the only way the chain's own storyboard/code calls see
 * the right engine instead of always defaulting to Manim.
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
  // CR-028 FR83.1/FR83.2/FR85 — the project row (and its topic) is created/
  // updated on the server as soon as the Creator stops typing, instead of
  // waiting for POST /v1/sagas/render (docs/review/data-flow-review.md's
  // "orphan draft" risk). similarProjects backs the FR85 collision banner.
  const [similarProjects, setSimilarProjects] = useState<SimilarProject[]>([]);
  const debouncedTopic = useDebounce(draft.authoringTopic.trim(), 600);

  useEffect(() => {
    if (!draft.projectId || !debouncedTopic) {
      setSimilarProjects([]);
      return;
    }
    let cancelled = false;
    createProjectDraft(draft.projectId, debouncedTopic, draft.voiceLanguage)
      .then(({ similarProjects }) => {
        if (!cancelled) setSimilarProjects(similarProjects);
      })
      .catch(() => {
        // Best-effort — a Creator offline or mid-render (FR84.2 lock) can
        // still type/paste normally; the collision warning just won't show.
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId, debouncedTopic, draft.voiceLanguage]);

  // Reload lands here directly (or the Creator jumps back to "1a" from a
  // later tab) — the in-memory draft survives via localStorage already, but
  // the server copy is the one source of truth later steps read from, so
  // rehydrate it the same way the other 3 tabs do.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringTopic)) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (cancelled) return;
        if (state.story && !draft.authoringStory) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
        // CR-027 D0 — chủ đề giờ cũng nằm ở server, nên nó rehydrate được
        // như 4 artefact kia. "" nghĩa là project tạo trước CR-027.
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

  // CR-040 FR113: the server fills the template — channel identity, beat sheet
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
  // CR-031 — "Đã có dàn ý" vào đúng tab này, nhưng để dán chứ không để sinh.
  // Cùng một màn hình, hai nửa khác nhau được dùng, nên chữ phải nói rõ nửa
  // nào là việc của Creator lúc này.
  const hasOwnOutline = draft.scriptSource === "outline";
  const llm = useLlmStatus();
  // CR-027 FR79 — chế độ lấy từ project ở server (qua draft), nên mở lại dự án
  // ở bất cứ tab nào, trình duyệt nào, sau restart nào cũng đúng chế độ đã chọn.
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  // Chế độ AI chỉ "thật" khi máy chủ có provider: một draft chọn AI trên máy
  // chưa cấu hình key phải quay về đường copy tay, chứ không mất cả hai.
  const aiMode = authoringMode === "ai" && llm?.enabled === true;

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      // CR-027 D0 — chủ đề đi kèm dàn ý trong cùng một lượt lưu. Trước đây
      // nó chỉ sống trong localStorage của trình duyệt, nên server không có
      // gì để điền vào {{topic}} lúc tự render prompt (FR77).
      // Đổi dàn ý thì storyboard và code dựng từ bản cũ bị xoá ở server: hỏi
      // trước, rồi đọc lại để bản nháp không giữ thứ đã mất.
      const saved = await getAuthoringState(draft.projectId).catch(() => null);
      const changed = saved !== null && saved.story !== "" && saved.story !== draft.authoringStory;
      if (changed && (saved.storyboard || saved.code)) {
        const ok = window.confirm(
          "Dàn ý đã đổi. Storyboard và code dựng từ dàn ý cũ sẽ bị xoá để làm lại. Tiếp tục?",
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
      setSaveError("Không lưu được dàn ý, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : storyIsEmpty
      ? hasOwnOutline
        ? "Dán dàn ý sẵn có của bạn để tiếp tục"
        : "Dán dàn ý câu chuyện AI trả về để tiếp tục"
      : "Dàn ý đã sẵn sàng — bước tiếp theo sẽ dựng storyboard hình ảnh";

  return (
    <div data-testid="script-outline-step-page">
      <AppShell
        currentStep={3}
        wide
        title="Bước 3 — Script"
        subtitle={
          hasOwnOutline
            ? "Dán dàn ý sẵn có của bạn vào ô bên phải — không cần chạy Story Architect."
            : "Dựng dàn ý câu chuyện với Story Architect."
        }
      >
        {/* CR-031 bug report — engine và cách làm đã chốt ở màn chọn tình
            huống; hiện lại y nguyên hai bộ chọn đầy đủ ở mỗi tab đọc như thể
            chưa chọn gì. PipelineSettingsBar thu gọn thành một dòng tóm tắt,
            mở rộng khi Creator bấm "Đổi" — vẫn đổi được ở đây (CR-030: nút
            "chạy cả bước 3" bên dưới gọi luôn cả 1b/1c, nên đổi engine phải
            xong TRƯỚC khi bấm chạy, không phải ở 1c lúc đã muộn). */}
        <div className={styles.settingsRow} style={{ marginBottom: 16 }}>
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
            steps={["story", "storyboard", "code"]}
            what="dàn ý"
            runDisabled={topicIsEmpty}
            runDisabledReason="Nhập chủ đề trước đã — server điền {{topic}} từ chủ đề đã lưu."
            beforeRun={async () => {
              // Chủ đề bình thường được lưu bởi effect debounce; nếu Creator
              // bấm ngay sau khi gõ thì nó chưa kịp lên server, và prompt sẽ
              // thiếu đúng cái thứ duy nhất bước này cần. Engine đi kèm ở đây
              // nữa, làm lưới an toàn cho lượt lưu ở onChange phía trên —
              // chuỗi 1b/1c phải thấy đúng engine trước khi chạy, không phải
              // sau.
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

        <div className={styles.scriptLayout}>
          <Card
            title={hasOwnOutline ? "Chủ đề" : aiMode ? "1. Chủ đề" : "1. Copy prompt"}
            hint={
              hasOwnOutline
                ? "Đã có dàn ý rồi thì chủ đề chỉ để đặt tên và đối chiếu trùng lặp — prompt bên dưới bỏ qua được."
                : aiMode
                  ? "Chủ đề là tất cả những gì bước này cần — server tự điền nó vào prompt khi gọi AI."
                  : "Nhập chủ đề, copy prompt rồi dán vào ChatGPT, Claude hoặc Gemini."
            }
          >
            <TextInput
              type="text"
              data-testid="script-outline-topic"
              value={draft.authoringTopic}
              onChange={(event) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: event.target.value })}
              placeholder="Ví dụ: Vòng lặp for trong Java, khi nào dùng while thay thế"
              style={{ marginBottom: 12 }}
            />
            {similarProjects.length > 0 && (
              <div className={styles.topicCollisionBanner} data-testid="topic-collision-banner">
                Chủ đề này trùng với {similarProjects.length} project khác:{" "}
                {similarProjects.map((p, i) => (
                  <span key={p.projectId}>
                    {i > 0 && ", "}
                    <a href="/videos" target="_blank" rel="noreferrer">
                      {p.topic || p.projectId} ({p.status})
                    </a>
                  </span>
                ))}
                . Bạn vẫn có thể tiếp tục — đây chỉ là cảnh báo.
              </div>
            )}
            {/* Ở chế độ AI, ô prompt để copy không còn việc gì: server tự
                render đúng văn bản này rồi tự gọi. Đổi lại chế độ là nó quay
                lại nguyên vẹn — không có gì bị xoá. */}
            {!aiMode && (
              <>
                <TextArea
                  readOnly
                  value={prompt}
                  rows={16}
                  className={styles.promptTextarea}
                  data-testid="script-outline-prompt"
                />
                <Button onClick={handleCopy} disabled={rendered.prompt === null || rendered.stale} className={styles.copyButton} data-testid="script-outline-copy">
                  {copied ? "Đã copy!" : "Copy prompt"}
                </Button>
              </>
            )}
          </Card>

          <Card
            title={hasOwnOutline ? "Dàn ý của bạn" : aiMode ? "2. Dàn ý" : "2. Dán kết quả"}
            hint={
              hasOwnOutline
                ? "Dán dàn ý sẵn có vào đây, rồi bấm Tiếp tục để chuyển sang bước Visual."
                : aiMode
                  ? "Kết quả AI sinh ra hiện ở đây để bạn sửa, rồi bấm Tiếp tục để chuyển sang bước Visual."
                  : "Dán dàn ý AI trả về, rồi bấm Tiếp tục để chuyển sang bước Visual."
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
