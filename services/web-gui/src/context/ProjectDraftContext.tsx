import { createContext, useReducer, type Dispatch, type ReactNode } from "react";

export interface SubtitleStyle {
  fontSize: "small" | "medium" | "large";
  textColor: string;
  backgroundOpacity: number;
  position: "bottom" | "top";
}

export interface ProjectDraft {
  projectId: string;
  scriptContent: string;
  voiceLanguage: "vi" | "en";
  backgroundMusicPath: string | null;
  ttsEnabled: boolean;
  voiceId: string | null;
  subtitlesEnabled: boolean;
  subtitleStyle: SubtitleStyle;
  renderQuality: RenderQuality;
}

/**
 * Resolution/framerate for the render (CR-004 FR12.6). A 720p30 draft is for
 * checking the content quickly; anything published should be 1080p60 or better.
 */
export type RenderQuality = "720p30" | "1080p60" | "4k60";

export type ProjectDraftAction =
  | { type: "SET_SCRIPT"; payload: string }
  | { type: "SET_VOICE_LANGUAGE"; payload: "vi" | "en" }
  | { type: "SET_BACKGROUND_MUSIC"; payload: string | null }
  | { type: "SET_TTS_ENABLED"; payload: boolean }
  | { type: "SET_VOICE_ID"; payload: string | null }
  | { type: "SET_SUBTITLES_ENABLED"; payload: boolean }
  | { type: "SET_SUBTITLE_STYLE"; payload: Partial<SubtitleStyle> }
  | { type: "SET_RENDER_QUALITY"; payload: RenderQuality }
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
  voiceLanguage: "vi",
  backgroundMusicPath: null,
  ttsEnabled: true,
  voiceId: null,
  subtitlesEnabled: false,
  subtitleStyle: defaultSubtitleStyle,
  renderQuality: "1080p60",
};

function projectDraftReducer(state: ProjectDraft, action: ProjectDraftAction): ProjectDraft {
  switch (action.type) {
    case "SET_SCRIPT":
      return { ...state, scriptContent: action.payload };
    case "SET_VOICE_LANGUAGE":
      return { ...state, voiceLanguage: action.payload };
    case "SET_BACKGROUND_MUSIC":
      return { ...state, backgroundMusicPath: action.payload };
    case "SET_TTS_ENABLED":
      return { ...state, ttsEnabled: action.payload };
    case "SET_VOICE_ID":
      return { ...state, voiceId: action.payload };
    case "SET_SUBTITLES_ENABLED":
      return { ...state, subtitlesEnabled: action.payload };
    case "SET_RENDER_QUALITY":
      return { ...state, renderQuality: action.payload };
    case "SET_SUBTITLE_STYLE":
      return { ...state, subtitleStyle: { ...state.subtitleStyle, ...action.payload } };
    case "RESET":
      return { ...initialDraft, projectId: crypto.randomUUID() };
  }
}

export const ProjectDraftContext = createContext<ProjectDraft>(initialDraft);
export const ProjectDraftDispatchContext = createContext<Dispatch<ProjectDraftAction>>(() => {});

export function ProjectDraftProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(
    projectDraftReducer,
    initialDraft,
    (draft) => ({ ...draft, projectId: crypto.randomUUID() }),
  );
  return (
    <ProjectDraftContext.Provider value={state}>
      <ProjectDraftDispatchContext.Provider value={dispatch}>
        {children}
      </ProjectDraftDispatchContext.Provider>
    </ProjectDraftContext.Provider>
  );
}
