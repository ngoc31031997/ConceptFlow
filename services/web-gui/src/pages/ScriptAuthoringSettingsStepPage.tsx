import { useContext, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { AuthoringModeBar } from "../components/AuthoringModeBar";
import { AuthoringModelPicker } from "../components/AuthoringModelPicker";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { NarrationPanel } from "../components/NarrationPanel";
import { VideoOutputModePicker } from "../components/VideoOutputModePicker";
import { VideoFormatPicker } from "../components/VideoFormatPicker";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { WizardNav } from "../components/WizardNav";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  saveLastUsedSettings,
} from "../context/ProjectDraftContext";
import { patchWizardSettings, type WizardSettingsPatch } from "../api/client";
import styles from "./WizardSteps.module.css";
import { FLOW_CONFIG, flowTitle } from "../utils/flow";

/**
 * Bước 2 — cấu hình mà các bước soạn (Kịch bản, Visual, Code) và TTS đọc:
 * ngôn ngữ, render engine, cách làm (manual/AI), giọng đọc, định dạng video
 * (quyết định số beat mà script phải theo) và kiểu đầu ra (script phải đánh
 * dấu clip). Chất lượng, font chữ trong video, phụ đề và nhạc nền chỉ được
 * bước Render/Merge đọc, nên được chọn ở màn Review và sửa được ngay khi bước
 * đó lỗi (ProductionSettingsPanel) — không nằm ở đây. Mọi mục đều có mặc định
 * nên đi thẳng qua được. Mỗi lần đổi một mục là một PATCH riêng lên project
 * (đã tạo ở Bước 1); "Tiếp tục" chỉ chốt bước.
 */
export function ScriptAuthoringSettingsStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  const { models: authoringModels, setModels: setAuthoringModels } = useAuthoringModels(draft.projectId);

  const formats = useVideoFormats();

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const isRemotion = draft.renderEngine === "remotion";

  // PATCH nối đuôi nhau để server nhận đúng thứ tự Creator đổi. Field nào lưu
  // lỗi được giữ ở `failedRef` và gửi lại cùng lần PATCH kế tiếp.
  const queueRef = useRef<Promise<void>>(Promise.resolve());
  const failedRef = useRef<WizardSettingsPatch>({});

  function send(fields: WizardSettingsPatch): Promise<boolean> {
    const projectId = draft.projectId;
    if (!projectId) return Promise.resolve(true);
    const run = queueRef.current.then(async () => {
      const body = { ...failedRef.current, ...fields };
      try {
        await patchWizardSettings(projectId, body);
        failedRef.current = {};
        setSaveError(null);
        return true;
      } catch {
        failedRef.current = body;
        setSaveError("Không lưu được cấu hình — kiểm tra kết nối rồi thử lại.");
        return false;
      }
    });
    queueRef.current = run.then(() => undefined);
    return run;
  }

  function handleEngineChange(engine: "manim" | "remotion") {
    dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
    void send({ renderEngine: engine });
  }

  async function handleContinue() {
    setSaving(true);
    try {
      // Các field đã lưu từng cái một; bấm tiếp chỉ chốt bước (và gửi lại
      // field nào trước đó lưu lỗi) rồi mới đi tiếp.
      if (await send({ confirm: true })) {
        // CR-028 FR86.1 — cấu hình này thành mặc định cho project kế tiếp.
        saveLastUsedSettings(draft);
        navigate("/create/script/outline");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div data-testid="script-authoring-settings-step-page">
      <AppShell
        currentStep={2}
        wide
        title={flowTitle(FLOW_CONFIG)}
        subtitle={`${isRemotion ? "Remotion" : "Manim"} · Chọn ngôn ngữ, giọng đọc và định dạng video. Chất lượng, phụ đề và nhạc nền chọn ở bước Review. Các mục đã có sẵn giá trị phù hợp, bạn có thể bấm Tiếp tục ngay.`}
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => {
              dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang });
              void send({ voiceLanguage: lang });
            }}
          />
        </div>

        <div className={styles.settingsRow}>
          <RenderEnginePicker value={draft.renderEngine} onChange={handleEngineChange} />
        </div>

        <div className={styles.settingsRow}>
          <AuthoringModeBar
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
          />
        </div>

        {authoringMode === "ai" && llm?.enabled && (
          <div className={styles.settingsRow}>
            <AuthoringModelPicker
              models={authoringModels}
              onChange={setAuthoringModels}
              options={llm.models ?? []}
              defaultModel={llm.default_model ?? ""}
              codeStats={llm.code_stats}
              codeStatsError={llm.code_stats_error}
            />
          </div>
        )}

        <div className={styles.settingsLayout}>
          <NarrationPanel
            voiceLanguage={draft.voiceLanguage}
            ttsEnabled={draft.ttsEnabled}
            onTtsEnabledChange={(enabled) => {
              dispatch({ type: "SET_TTS_ENABLED", payload: enabled });
              void send({ ttsEnabled: enabled });
            }}
            voiceId={draft.voiceId}
            onVoiceIdChange={(voiceId) => {
              dispatch({ type: "SET_VOICE_ID", payload: voiceId });
              void send({ voiceId: voiceId ?? "" });
            }}
          />

          <div className={styles.settingsRow}>
            <VideoFormatPicker
              formats={formats}
              value={draft.videoFormatId}
              onChange={(formatId) => {
                dispatch({ type: "SET_VIDEO_FORMAT", payload: formatId });
                void send({ videoFormatId: formatId });
              }}
            />

            <VideoOutputModePicker
              value={draft.videoOutputMode}
              onChange={(mode) => {
                dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode });
                void send({ videoOutputMode: mode });
              }}
            />
          </div>
        </div>
      </AppShell>

      <WizardNav
        isBlocked={!!saveError}
        nextDisabled={saving}
        hint={
          saveError ??
          (authoringMode === "ai" && llm?.enabled
            ? "AI sẽ giúp bạn ở các bước Kịch bản, Visual và Code."
            : "Bạn tự làm với ChatGPT, Claude… Có thể đổi sang AI làm giúp bất cứ lúc nào.")
        }
        onBack={() => navigate("/")}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Sang Kịch bản"}
        nextTestId="script-authoring-settings-next"
      />
    </div>
  );
}
