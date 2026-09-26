import { useContext, useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { getAuthoringState, getProject } from "../api/client";
import { ProjectDraftDispatchContext, defaultSubtitleStyle } from "../context/ProjectDraftContext";
import type { ProjectDraft } from "../context/ProjectDraftContext";
import { projectPath } from "../utils/pipelineLabels";
import { authoringRoute } from "../utils/flow";
import type { Project } from "../types";
import type { AuthoringState } from "../api/client";
import glass from "../styles/glass.module.css";

const DEFAULT_MUSIC_VOLUME = 0.2;

/** Dựng lại bản nháp phía client từ những gì server đã lưu (bước 1-3). */
function draftFromServer(project: Project, state: AuthoringState): Partial<ProjectDraft> {
  const style = project.subtitle_style;
  return {
    projectId: project.project_id,
    scriptContent: project.script_content || state.code,
    voiceLanguage: project.voice_language,
    renderEngine: project.render_engine ?? "manim",
    videoFont: project.video_font || "Be Vietnam Pro",
    ttsEnabled: project.tts_enabled ?? true,
    voiceId: project.voice_id || null,
    subtitleMode: (project.subtitle_mode as ProjectDraft["subtitleMode"]) || "track",
    subtitleStyle: style
      ? {
          // Projects saved before the font choice existed rendered in DejaVu
          // Sans; showing the new default here would misreport them.
          fontFamily: style.font_family || "DejaVu Sans",
          fontSize: style.font_size,
          textColor: style.text_color,
          backgroundOpacity: style.background_opacity,
          position: style.position,
        }
      : defaultSubtitleStyle,
    renderQuality: project.render_quality ?? "1080p60",
    videoFormatId: project.video_format_id ?? "case_study_essay_8min",
    videoOutputMode: project.video_output_mode ?? "long",
    backgroundMusicPath: project.background_music_path ?? null,
    backgroundMusicVolume: project.background_music_volume || DEFAULT_MUSIC_VOLUME,
    authoringMode: state.mode ?? "manual",
    authoringModels: {
      story: state.story_model ?? "",
      storyboard: state.storyboard_model ?? "",
      code: state.code_model ?? "",
    },
    authoringTopic: state.topic,
    authoringStory: state.story,
    authoringStoryboard: state.storyboard,
  };
}

/**
 * Mở lại một project đang dở ở máy/trình duyệt bất kỳ: nạp bước 1-3 đã lưu trên
 * server vào bản nháp, rồi đưa Creator tới đúng bước `wizard_step`. Project đã
 * chạy saga (bước 4+) thì đi thẳng tới màn của trạng thái hiện tại.
 */
export function ResumeProjectPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  // ?edit=1: "Quay lại sửa script" từ màn lỗi — project chưa render gì nên vẫn
  // mở lại được để sửa, không đẩy về màn theo dõi của trạng thái lỗi.
  const [search] = useSearchParams();
  const editAfterFailure = search.get("edit") === "1";
  // ?view=1&step=N: mở project (kể cả đã khoá) ở bước authoring N để XEM lại.
  // Các màn soạn tự chuyển sang chỉ đọc khi server không cho sửa nữa.
  const viewOnly = search.get("view") === "1";
  const stepParam = Number(search.get("step")) || 0;
  const dispatch = useContext(ProjectDraftDispatchContext);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const project = await getProject(id);
        if (cancelled) return;
        if (project.status !== "draft" && !editAfterFailure && !viewOnly) {
          // projectPath("draft") sẽ trỏ lại đây, nên chỉ gọi khi đã qua draft.
          navigate(projectPath(id, project.status), { replace: true });
          return;
        }
        const state = await getAuthoringState(id);
        if (cancelled) return;
        dispatch({ type: "LOAD_PROJECT", payload: draftFromServer(project, state) });
        // Đích lấy từ vị trí server suy ra (flow_step: bước xa nhất đang ở), không
        // phải màn nào mở lần cuối. Với dự án đã qua bước 5 mà chưa chỉ định bước
        // (sửa sau lỗi), mở tab code — nơi có nội dung xa nhất.
        const flowStep = project.flow_step ?? 2;
        const step = stepParam || (flowStep > 5 ? 5 : flowStep);
        navigate(authoringRoute(step), { replace: true });
      } catch {
        if (!cancelled) setError("Không mở lại được dự án. Dự án có thể đã bị xóa hoặc kết nối bị gián đoạn.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, dispatch, navigate, editAfterFailure, viewOnly, stepParam]);

  return (
    <AppShell title="Đang mở dự án" subtitle="Đang tải lại nội dung bạn đã lưu.">
      {error ? (
        <p role="alert" className={glass.helperText}>
          {error} <Link to="/videos">Về danh sách video</Link>
        </p>
      ) : (
        <p className={glass.helperText}>Đang tải...</p>
      )}
    </AppShell>
  );
}
