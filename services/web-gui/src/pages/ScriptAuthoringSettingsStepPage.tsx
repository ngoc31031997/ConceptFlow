import { useContext, useRef, useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { TopicSummary } from "../components/TopicSummary";
import { VideoArchetypePicker } from "../components/VideoArchetypePicker";
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
import { createProjectDraft, patchWizardSettings, type SimilarProject, type WizardSettingsPatch } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./WizardSteps.module.css";
import { FLOW_CONFIG, flowTitle } from "../utils/flow";

interface SettingsSectionProps {
  /** "1. Nội dung video": số thứ tự là thứ tự quyết định. */
  title: string;
  hint: string;
  testId: string;
  children: ReactNode;
}

/** Một nhóm cài đặt có đánh số trên màn Cấu hình. */
function SettingsSection({ title, hint, testId, children }: SettingsSectionProps) {
  return (
    <section className={styles.settingsSection} data-testid={testId}>
      <header className={styles.settingsSectionHead}>
        <h2>{title}</h2>
        <p>{hint}</p>
      </header>
      {children}
    </section>
  );
}

/**
 * Bước 2 — cấu hình mà các bước soạn (Kịch bản, Hình ảnh, Code) và Giọng đọc
 * đọc, chia ba nhóm theo thứ tự quyết định: nội dung video (ngôn ngữ, kiểu
 * video, format, kiểu đầu ra), giọng đọc, cách soạn (engine, AI hay tự làm,
 * model). Kiểu video nằm ngay trên format vì nó gợi ý format; lựa chọn được
 * ghi thành "kiểu: X" ở cuối chủ đề nên đổi kiểu là lưu lại chủ đề.
 *
 * Chất lượng, font chữ trong video, phụ đề và nhạc nền chỉ bước Dựng hình/Ghép
 * video đọc, nên được chọn ở màn Duyệt nội dung và sửa được ngay khi bước đó
 * lỗi (ProductionSettingsPanel). Mọi mục đều có mặc định nên đi thẳng qua
 * được. Mỗi lần đổi một mục là một lần lưu riêng lên project (đã tạo ở Bước
 * 1); "Tiếp tục" chỉ chốt bước.
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
  // Dự án trùng chủ đề, do Bước 1 chuyển sang lúc tạo project.
  const location = useLocation();
  const similarProjects = (location.state as { similarProjects?: SimilarProject[] } | null)?.similarProjects ?? [];

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

  // Kiểu video sống trong chủ đề ("kiểu: X"), nên đổi kiểu là lưu lại chủ đề,
  // nối đuôi cùng hàng đợi với các field khác.
  function handleTopicChange(topic: string) {
    dispatch({ type: "SET_AUTHORING_TOPIC", payload: topic });
    const projectId = draft.projectId;
    if (!projectId) return;
    const run = queueRef.current.then(async () => {
      try {
        await createProjectDraft(projectId, topic.trim(), draft.voiceLanguage, draft.renderEngine);
        setSaveError(null);
      } catch {
        setSaveError("Không lưu được kiểu video — kiểm tra kết nối rồi thử lại.");
      }
    });
    queueRef.current = run;
  }

  function handleFormatChange(formatId: string) {
    dispatch({ type: "SET_VIDEO_FORMAT", payload: formatId });
    void send({ videoFormatId: formatId });
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
        // Cấu hình này thành mặc định cho project kế tiếp.
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
        subtitle={`${isRemotion ? "Remotion" : "Manim"} · Ba nhóm cài đặt, làm từ trên xuống. Chất lượng, phụ đề và nhạc nền chọn ở bước Duyệt nội dung. Các mục đã có sẵn giá trị phù hợp, bạn có thể bấm Tiếp tục ngay.`}
      >
        <TopicSummary topic={draft.authoringTopic} onEdit={() => navigate("/")} />
        {similarProjects.length > 0 && (
          <div className={styles.topicCollisionBanner} data-testid="topic-collision-banner">
            Chủ đề này giống {similarProjects.length} dự án khác:{" "}
            {similarProjects.map((p, i) => (
              <span key={p.projectId}>
                {i > 0 && ", "}
                <a href="/videos" target="_blank" rel="noreferrer">
                  {p.topic || p.projectId} ({p.status})
                </a>
              </span>
            ))}
            . Bạn vẫn có thể tiếp tục.
          </div>
        )}

        <SettingsSection
          title="1. Nội dung video"
          hint="Video nói bằng ngôn ngữ nào, theo kiểu nào, dài bao lâu."
          testId="settings-section-content"
        >
          <div className={styles.settingsRow}>
            <ContentLanguagePicker
              value={draft.voiceLanguage}
              onChange={(lang) => {
                dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang });
                void send({ voiceLanguage: lang });
              }}
            />
            <div className={glass.card} style={{ padding: 22 }}>
              <VideoArchetypePicker
                topic={draft.authoringTopic}
                onTopicChange={handleTopicChange}
                formats={formats}
                formatId={draft.videoFormatId}
                onFormatChange={handleFormatChange}
              />
            </div>
          </div>
          <div className={styles.settingsRow}>
            <VideoFormatPicker formats={formats} value={draft.videoFormatId} onChange={handleFormatChange} />
            <VideoOutputModePicker
              value={draft.videoOutputMode}
              onChange={(mode) => {
                dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode });
                void send({ videoOutputMode: mode });
              }}
            />
          </div>
        </SettingsSection>

        <SettingsSection
          title="2. Giọng đọc"
          hint="Có lời đọc hay không, và giọng nào đọc."
          testId="settings-section-voice"
        >
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
        </SettingsSection>

        <SettingsSection
          title="3. Cách soạn"
          hint="Công cụ dựng hình, và AI làm giúp hay bạn tự làm với ChatGPT, Claude…"
          testId="settings-section-authoring"
        >
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
        </SettingsSection>
      </AppShell>

      <WizardNav
        isBlocked={!!saveError}
        nextDisabled={saving}
        hint={
          saveError ??
          (authoringMode === "ai" && llm?.enabled
            ? "AI sẽ giúp bạn ở các bước Kịch bản, Hình ảnh và Code."
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
