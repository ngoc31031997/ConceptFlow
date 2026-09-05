import { createContext, useReducer, type Dispatch, type ReactNode } from "react";

export interface ProjectDraft {
  scriptContent: string;
  pluginId: string | null;
  voiceLanguage: "vi" | "en";
  backgroundMusicPath: string | null;
}

export type ProjectDraftAction =
  | { type: "SET_SCRIPT"; payload: string }
  | { type: "SET_PLUGIN"; payload: string }
  | { type: "SET_VOICE_LANGUAGE"; payload: "vi" | "en" }
  | { type: "SET_BACKGROUND_MUSIC"; payload: string | null }
  | { type: "RESET" };

const initialDraft: ProjectDraft = {
  scriptContent: "",
  pluginId: null,
  voiceLanguage: "vi",
  backgroundMusicPath: null,
};

function projectDraftReducer(state: ProjectDraft, action: ProjectDraftAction): ProjectDraft {
  switch (action.type) {
    case "SET_SCRIPT":
      return { ...state, scriptContent: action.payload };
    case "SET_PLUGIN":
      return { ...state, pluginId: action.payload };
    case "SET_VOICE_LANGUAGE":
      return { ...state, voiceLanguage: action.payload };
    case "SET_BACKGROUND_MUSIC":
      return { ...state, backgroundMusicPath: action.payload };
    case "RESET":
      return initialDraft;
  }
}

export const ProjectDraftContext = createContext<ProjectDraft>(initialDraft);
export const ProjectDraftDispatchContext = createContext<Dispatch<ProjectDraftAction>>(() => {});

export function ProjectDraftProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(projectDraftReducer, initialDraft);
  return (
    <ProjectDraftContext.Provider value={state}>
      <ProjectDraftDispatchContext.Provider value={dispatch}>
        {children}
      </ProjectDraftDispatchContext.Provider>
    </ProjectDraftContext.Provider>
  );
}
