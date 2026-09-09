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
  backgroundMusicVolume: number;
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
export type RenderQuality = "720p30" | "1080p60" | "4k60";

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
  | { type: "SET_BACKGROUND_MUSIC_VOLUME"; payload: number }
  | { type: "MARK_SUBMITTED" }
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
  backgroundMusicVolume: 0.2,
  hasSubmitted: false,
};

const STORAGE_KEY = "conceptflow.draft.v1";

/**
 * A draft only lived in memory, so a reload mid-edit threw away a script the
 * Creator may have spent a while getting right. Persisting is best-effort:
 * private browsing and a full quota both throw, and neither is worth failing
 * the render over.
 */
function loadDraft(): ProjectDraft {
  const fresh = { ...initialDraft, projectId: crypto.randomUUID() };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return fresh;
    const stored = JSON.parse(raw) as Partial<ProjectDraft>;
    // A submitted draft is spent — never restore it onto a new session.
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
    case "MARK_SUBMITTED":
      return { ...state, hasSubmitted: true };
    case "SET_SUBTITLE_STYLE":
      return { ...state, subtitleStyle: { ...state.subtitleStyle, ...action.payload } };
    case "RESET":
      return { ...initialDraft, projectId: crypto.randomUUID() };
  }
}

export const ProjectDraftContext = createContext<ProjectDraft>(initialDraft);
export const ProjectDraftDispatchContext = createContext<Dispatch<ProjectDraftAction>>(() => {});

export function ProjectDraftProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(projectDraftReducer, initialDraft, loadDraft);

  useEffect(() => {
    saveDraft(state);
  }, [state]);

  return (
    <ProjectDraftContext.Provider value={state}>
      <ProjectDraftDispatchContext.Provider value={dispatch}>
        {children}
      </ProjectDraftDispatchContext.Provider>
    </ProjectDraftContext.Provider>
  );
}
