import { createContext, useEffect, useReducer, type Dispatch, type ReactNode } from "react";

export interface SubtitleStyle {
  fontSize: "small" | "medium" | "large";
  textColor: string;
  backgroundOpacity: number;
  position: "bottom" | "top";
}

/**
 * How subtitle_cues get delivered to YouTube (CR-015, ADR-0027):
 *   off     — no subtitles
 *   track   — a caption track the viewer can toggle with CC — searchable,
 *             auto-translatable, and it never paints over Manim's edge content
 *   burn_in — painted into the video frames (the only option before CR-015)
 *   both    — both at once, which means a viewer with CC on sees the text twice
 */
export type SubtitleMode = "off" | "track" | "burn_in" | "both";

/** Which of the three script situations the Creator picked in step 1. */
export type ScriptSource = "blank" | "draft" | "ready";

export interface ProjectDraft {
  projectId: string;
  scriptContent: string;
  scriptSource: ScriptSource;
  voiceLanguage: "vi" | "en";
  backgroundMusicPath: string | null;
  ttsEnabled: boolean;
  voiceId: string | null;
  subtitleMode: SubtitleMode;
  subtitleStyle: SubtitleStyle;
  renderQuality: RenderQuality;
  renderEngine: RenderEngine;
  videoFormatId: string;
  backgroundMusicVolume: number;
  videoOutputMode: VideoOutputMode;
  /**
   * The topic the Creator typed on the outline tab (1a) — kept in the draft
   * (not local component state) so switching to another tab and back, or a
   * reload, does not lose it. Never sent to the server on its own; it only
   * exists to keep filling the story_architect prompt on that tab.
   */
  authoringTopic: string;
  /**
   * CR-025 step 1 — the Story Architect story outline the Creator pasted
   * back and the server has saved (POST /v1/projects/:id/authoring/story).
   * Kept here so step 2 (Visual Director, currently a stub) can show it as
   * {{previous_output}} without a re-fetch.
   */
  authoringStory: string;
  /**
   * CR-025 step 2 — the Visual Director storyboard the Creator pasted back
   * and the server has saved (POST /v1/projects/:id/authoring/storyboard).
   * Kept here so step 3 (Manim Engineer, currently a stub) can show
   * story+storyboard as {{previous_output}} without a re-fetch.
   */
  authoringStoryboard: string;
  /**
   * True once this draft has been handed to the render saga. The draft then
   * belongs to a project that already exists, so reusing it would start a
   * second saga against the same project_id and overwrite the first video.
   * NewProjectPage resets on mount when it sees this.
   */
  hasSubmitted: boolean;
}

/**
 * Resolution/framerate for the render (CR-004 FR12.6). A 720p30 draft is for
 * checking the content quickly; anything published should be 1080p60 or better.
 */
export type RenderQuality = "480p15" | "720p30" | "1080p60" | "4k60";

/**
 * feature/remotion-engine — which engine renders scriptContent. "manim"
 * stays the default; Remotion is a minimal first cut (see
 * RenderEnginePicker.tsx). Backend defaults to "manim" too when this is
 * omitted, so every project created before this field existed keeps working.
 */
export type RenderEngine = "manim" | "remotion";

/**
 * Which output(s) this project produces (CR-007 follow-up). A short clip is
 * always cut from the rendered 16:9 video (CR-007 D1 — no standalone vertical
 * production), so "short" still renders the full long-form pipeline as
 * source; it only changes what generate_clips does and what step 5
 * (ResultPage) puts front and center — publishing a Shorts/TikTok clip stays
 * a manual upload outside this app either way (no auto-publish adapter).
 */
export type VideoOutputMode = "long" | "short" | "both";

export type ProjectDraftAction =
  | { type: "SET_SCRIPT"; payload: string }
  | { type: "SET_SCRIPT_SOURCE"; payload: ScriptSource }
  | { type: "SET_VOICE_LANGUAGE"; payload: "vi" | "en" }
  | { type: "SET_BACKGROUND_MUSIC"; payload: string | null }
  | { type: "SET_TTS_ENABLED"; payload: boolean }
  | { type: "SET_VOICE_ID"; payload: string | null }
  | { type: "SET_SUBTITLE_MODE"; payload: SubtitleMode }
  | { type: "SET_SUBTITLE_STYLE"; payload: Partial<SubtitleStyle> }
  | { type: "SET_RENDER_QUALITY"; payload: RenderQuality }
  | { type: "SET_RENDER_ENGINE"; payload: RenderEngine }
  | { type: "SET_VIDEO_OUTPUT_MODE"; payload: VideoOutputMode }
  | { type: "SET_VIDEO_FORMAT"; payload: string }
  | { type: "SET_BACKGROUND_MUSIC_VOLUME"; payload: number }
  | { type: "SET_AUTHORING_TOPIC"; payload: string }
  | { type: "SET_AUTHORING_STORY"; payload: string }
  | { type: "SET_AUTHORING_STORYBOARD"; payload: string }
  | { type: "MARK_SUBMITTED" }
  | { type: "RESUME_EDITING" }
  | { type: "RESET" };

export const defaultSubtitleStyle: SubtitleStyle = {
  fontSize: "medium",
  textColor: "#FFFFFF",
  backgroundOpacity: 0.6,
  position: "bottom",
};

const initialDraft: ProjectDraft = {
  projectId: "",
  scriptContent: "",
  scriptSource: "blank",
  voiceLanguage: "vi",
  backgroundMusicPath: null,
  ttsEnabled: true,
  voiceId: null,
  // CR-015 FR41.2: caption track is the default for long-form YouTube —
  // searchable, auto-translatable, and never painted over the frame.
  // Burn-in remains available as an explicit Creator choice.
  subtitleMode: "track",
  subtitleStyle: defaultSubtitleStyle,
  renderQuality: "1080p60",
  renderEngine: "manim",
  videoFormatId: "visual_first_7min",
  backgroundMusicVolume: 0.2,
  videoOutputMode: "long",
  authoringTopic: "",
  authoringStory: "",
  authoringStoryboard: "",
  hasSubmitted: false,
};

const STORAGE_KEY = "conceptflow.draft.v1";

/**
 * The Creator's last-picked voice, kept separately from the per-project draft
 * blob above. That blob is meant to reset with every new/submitted project
 * (RESET, or loadDraft() refusing to restore a submitted one) — but a voice
 * choice is a channel-level preference, not something tied to one script. Bug
 * report: without this, every new video defaulted back to whichever voice
 * happens to sort first (NarrationPanel's own fallback), forcing a reselect
 * every single time.
 */
const LAST_VOICE_KEY = "conceptflow.lastVoiceId";

function loadLastVoiceId(): string | null {
  try {
    return window.localStorage.getItem(LAST_VOICE_KEY);
  } catch {
    return null;
  }
}

function saveLastVoiceId(voiceId: string | null): void {
  try {
    if (voiceId) {
      window.localStorage.setItem(LAST_VOICE_KEY, voiceId);
    }
  } catch {
    /* storage unavailable or full — the preference simply will not persist */
  }
}

/**
 * CR-028 FR86 — "một bộ cấu hình lần cuối dùng" toàn cục: engine, quality,
 * TTS, giọng, sub, nhạc nền, output mode, hình dạng video. Saved once when
 * the Creator finishes wizard step 6 (SettingsStepPage's onNext calls
 * saveLastUsedSettings), read back to prefill every new draft from then on
 * — same client-only posture as LAST_VOICE_KEY above (a browser-level
 * convenience, not business data that needs to sync across devices).
 */
const LAST_SETTINGS_KEY = "conceptflow.lastUsedSettings.v1";

type LastUsedSettings = Pick<
  ProjectDraft,
  | "ttsEnabled"
  | "voiceId"
  | "subtitleMode"
  | "subtitleStyle"
  | "renderQuality"
  | "renderEngine"
  | "videoFormatId"
  | "backgroundMusicPath"
  | "backgroundMusicVolume"
  | "videoOutputMode"
>;

function loadLastUsedSettings(): Partial<LastUsedSettings> {
  try {
    const raw = window.localStorage.getItem(LAST_SETTINGS_KEY);
    return raw ? (JSON.parse(raw) as Partial<LastUsedSettings>) : {};
  } catch {
    return {};
  }
}

/**
 * Call once a project's step-6 settings are final (SettingsStepPage's
 * "Tiếp tục"). Deliberately not saved on every keystroke while still
 * editing — a half-finished change to one project's settings must not leak
 * into the next project's defaults before the Creator confirms it (FR86.2).
 */
export function saveLastUsedSettings(draft: ProjectDraft): void {
  try {
    const settings: LastUsedSettings = {
      ttsEnabled: draft.ttsEnabled,
      voiceId: draft.voiceId,
      subtitleMode: draft.subtitleMode,
      subtitleStyle: draft.subtitleStyle,
      renderQuality: draft.renderQuality,
      renderEngine: draft.renderEngine,
      videoFormatId: draft.videoFormatId,
      backgroundMusicPath: draft.backgroundMusicPath,
      backgroundMusicVolume: draft.backgroundMusicVolume,
      videoOutputMode: draft.videoOutputMode,
    };
    window.localStorage.setItem(LAST_SETTINGS_KEY, JSON.stringify(settings));
  } catch {
    /* storage unavailable or full — the preference simply will not persist */
  }
}

/**
 * A draft only lived in memory, so a reload mid-edit threw away a script the
 * Creator may have spent a while getting right. Persisting is best-effort:
 * private browsing and a full quota both throw, and neither is worth failing
 * the render over.
 */
function loadDraft(): ProjectDraft {
  const fresh = {
    ...initialDraft,
    ...loadLastUsedSettings(),
    projectId: crypto.randomUUID(),
    // LAST_VOICE_KEY predates FR86 and stays authoritative for voiceId
    // specifically — same value in practice, but no behaviour change for
    // anyone already relying on it.
    voiceId: loadLastVoiceId(),
  };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fresh;
    const stored = JSON.parse(raw) as Partial<ProjectDraft>;
    // A submitted draft is spent — never restore it onto a new session, but
    // still carry over the last voice (see LAST_VOICE_KEY's docstring).
    if (stored.hasSubmitted) return fresh;
    return { ...fresh, ...stored, projectId: stored.projectId ?? fresh.projectId };
  } catch {
    return fresh;
  }
}

function saveDraft(draft: ProjectDraft): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(draft));
  } catch {
    /* storage unavailable or full — the draft simply will not survive a reload */
  }
}

function projectDraftReducer(state: ProjectDraft, action: ProjectDraftAction): ProjectDraft {
  switch (action.type) {
    case "SET_SCRIPT":
      return { ...state, scriptContent: action.payload };
    case "SET_SCRIPT_SOURCE":
      return { ...state, scriptSource: action.payload };
    case "SET_VOICE_LANGUAGE":
      return { ...state, voiceLanguage: action.payload };
    case "SET_BACKGROUND_MUSIC":
      return { ...state, backgroundMusicPath: action.payload };
    case "SET_TTS_ENABLED":
      return { ...state, ttsEnabled: action.payload };
    case "SET_VOICE_ID":
      return { ...state, voiceId: action.payload };
    case "SET_SUBTITLE_MODE":
      return { ...state, subtitleMode: action.payload };
    case "SET_BACKGROUND_MUSIC_VOLUME":
      return { ...state, backgroundMusicVolume: action.payload };
    case "SET_RENDER_QUALITY":
      return { ...state, renderQuality: action.payload };
    case "SET_RENDER_ENGINE":
      return { ...state, renderEngine: action.payload };
    case "SET_VIDEO_OUTPUT_MODE":
      return { ...state, videoOutputMode: action.payload };
    case "SET_VIDEO_FORMAT":
      return { ...state, videoFormatId: action.payload };
    case "SET_AUTHORING_TOPIC":
      return { ...state, authoringTopic: action.payload };
    case "SET_AUTHORING_STORY":
      return { ...state, authoringStory: action.payload };
    case "SET_AUTHORING_STORYBOARD":
      return { ...state, authoringStoryboard: action.payload };
    case "MARK_SUBMITTED":
      return { ...state, hasSubmitted: true };
    // CR-024's "Quay lại sửa script" (outline rejected): the render saga has
    // already restarted the SAME project_id from scratch server-side
    // (StartRenderSagaUseCase upserts it back to StatusDraft), so the fix here
    // is the mirror image of MARK_SUBMITTED — clear the flag, keep everything
    // else (scriptContent, projectId) so ScriptStepPage's own
    // "hasSubmitted → RESET" effect does not wipe the very script the Creator
    // came back to fix.
    case "RESUME_EDITING":
      return { ...state, hasSubmitted: false };
    case "SET_SUBTITLE_STYLE":
      return { ...state, subtitleStyle: { ...state.subtitleStyle, ...action.payload } };
    case "RESET":
      return {
        ...initialDraft,
        ...loadLastUsedSettings(),
        projectId: crypto.randomUUID(),
        voiceId: loadLastVoiceId(),
      };
  }
}

export const ProjectDraftContext = createContext<ProjectDraft>(initialDraft);
export const ProjectDraftDispatchContext = createContext<Dispatch<ProjectDraftAction>>(() => {});

export function ProjectDraftProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(projectDraftReducer, initialDraft, loadDraft);

  useEffect(() => {
    saveDraft(state);
  }, [state]);

  useEffect(() => {
    saveLastVoiceId(state.voiceId);
  }, [state.voiceId]);

  return (
    <ProjectDraftContext.Provider value={state}>
      <ProjectDraftDispatchContext.Provider value={dispatch}>
        {children}
      </ProjectDraftDispatchContext.Provider>
    </ProjectDraftContext.Provider>
  );
}
