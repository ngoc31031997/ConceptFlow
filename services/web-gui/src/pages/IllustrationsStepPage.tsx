import { useContext, useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { ProjectIllustrationsPanel, type IllustrationsSummary } from "../components/ProjectIllustrationsPanel";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getAuthoringState } from "../api/client";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import styles from "./WizardSteps.module.css";

/**
 * CR-045 — bước Hình minh hoạ, giữa Visual và Code, chỉ cho video Remotion.
 *
 * Trước đây việc lập danh sách và vẽ hình chạy ngầm ở đầu bước Code, nên thẻ
 * "Code" đứng vài phút mà chưa viết dòng code nào. Giờ nó là một tab riêng:
 * chạy bằng AI thì server lập danh sách từ storyboard và vẽ hình còn thiếu
 * (nhiều hình cùng lúc, mỗi hình một thanh tiến độ), Creator duyệt / sửa / bỏ
 * qua / xoá ngay tại đây, rồi mới sang Code.
 *
 * CR-046 (2026-09-27, đảo ngược FR9 của CR-045): giờ có số bước riêng (6) trên
 * thanh bước, hiện "Không dùng" khi renderEngine không phải Remotion, thay vì
 * ẩn hoàn toàn dưới bước Code như trước.
 */
export function IllustrationsStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const { mode, setMode } = useAuthoringMode(draft.projectId);
  const [summary, setSummary] = useState<IllustrationsSummary | null>(null);

  // Nạp lại storyboard sau khi tải lại trang: nút chạy cần biết đã có storyboard chưa.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStoryboard && draft.authoringTopic)) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (cancelled) return;
        if (state.story && !draft.authoringStory) dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        if (state.storyboard && !draft.authoringStoryboard) dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: state.storyboard });
        if (state.topic && !draft.authoringTopic) dispatch({ type: "SET_AUTHORING_TOPIC", payload: state.topic });
      })
      .catch(() => {
        /* best-effort — the Creator can still go back to fix earlier steps */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId]);

  // Manim vẽ bằng code, không có thư viện hình: bước này không tồn tại với nó.
  if (draft.renderEngine !== "remotion") return <Navigate to={AUTHORING_STEP_PATHS.code} replace />;

  const hasStoryboard = draft.authoringStoryboard.trim().length > 0;
  const hint = !hasStoryboard
    ? "Cần hoàn thành bước Visual trước."
    : !summary || summary.total === 0
      ? "Chưa có danh sách hình — chạy bằng AI hoặc lập danh sách từ storyboard."
      : summary.ready === summary.total
        ? `Đủ ${summary.total} hình — sang bước Code.`
        : `${summary.ready}/${summary.total} hình sẵn sàng — duyệt hoặc bỏ qua các hình còn lại trước khi chạy Code bằng AI.`;

  return (
    <div data-testid="illustrations-step-page">
      <AppShell
        currentStep={6}
        title="Bước 3 — Script"
        subtitle="Hình minh hoạ của video: dùng lại hình trong thư viện, AI vẽ hình còn thiếu, bạn duyệt trước khi viết code."
        wide
      >
        <div className={styles.settingsRow}>
          <PipelineSettingsBar
            renderEngine={draft.renderEngine}
            llm={llm}
            mode={mode}
            onModeChange={setMode}
            projectId={draft.projectId}
            steps={["illustrations"]}
            what="hình minh hoạ"
            runDisabled={!hasStoryboard}
            runDisabledReason="Cần hoàn thành bước Visual trước."
            onFollow={(step) => navigate(step === "done" ? AUTHORING_STEP_PATHS.illustrations : AUTHORING_STEP_PATHS[step])}
          />
        </div>

        {draft.projectId && (
          <div className={styles.settingsRow}>
            <ProjectIllustrationsPanel projectId={draft.projectId} onSummary={setSummary} />
          </div>
        )}
      </AppShell>

      <WizardNav
        hint={hint}
        onBack={() => navigate(AUTHORING_STEP_PATHS.storyboard)}
        backLabel="Quay lại Visual"
        onNext={() => navigate(AUTHORING_STEP_PATHS.code)}
        nextLabel="Sang bước Code"
        nextDisabled={!hasStoryboard}
        nextTestId="illustrations-step-next"
      />
    </div>
  );
}
