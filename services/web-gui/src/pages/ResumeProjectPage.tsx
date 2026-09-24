import { useContext, useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { getAuthoringState, getProject } from "../api/client";
import { ProjectDraftDispatchContext, defaultSubtitleStyle } from "../context/ProjectDraftContext";
import type { ProjectDraft } from "../context/ProjectDraftContext";
import { projectPath } from "../utils/pipelineLabels";
import type { Project } from "../types";
import type { AuthoringState } from "../api/client";
import glass from "../styles/glass.module.css";

const DEFAULT_MUSIC_VOLUME = 0.2;

/** Bước 3 có ba tab; mở lại tab xa nhất đã có nội dung, để không bắt Creator bấm lại từ 1a. */
function scriptTabPath(state: AuthoringState): string {
  if (state.code.trim()) return "/create/script/code";
  if (state.storyboard.trim()) return "/create/script/storyboard";
  return "/create/script/outline";
}

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
    videoFormatId: project.video_format_id ?? "visual_first_7min",
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
  const editAfterFailure = useSearchParams()[0].get("edit") === "1";
  const dispatch = useContext(ProjectDraftDispatchContext);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const project = await getProject(id);
        if (cancelled) return;
        if (project.status !== "draft" && !editAfterFailure) {
          // projectPath("draft") sẽ trỏ lại đây, nên chỉ gọi khi đã qua draft.
          navigate(projectPath(id, project.status), { replace: true });
          return;
        }
        const state = await getAuthoringState(id);
        if (cancelled) return;
        dispatch({ type: "LOAD_PROJECT", payload: draftFromServer(project, state) });
        const step = project.wizard_step ?? 1;
        const target = editAfterFailure ? scriptTabPath(state) : step <= 1 ? "/" : step === 2 ? "/create/script/settings" : scriptTabPath(state);
        navigate(target, { replace: true });
      } catch {
        if (!cancelled) setError("Không mở lại được project — có thể nó đã bị xoá hoặc mất kết nối.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id, dispatch, navigate, editAfterFailure]);

  return (
    <AppShell title="Đang mở lại project" subtitle="Nạp lại những gì bạn đã lưu ở các bước trước.">
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
