import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { useRequireScript } from "../../src/hooks/useRequireScript";
import { ProjectDraftContext, defaultSubtitleStyle } from "../../src/context/ProjectDraftContext";
import type { ProjectDraft } from "../../src/context/ProjectDraftContext";

const BASE_DRAFT: ProjectDraft = {
  projectId: "p1",
  scriptContent: "",
  scriptSource: "code",
  voiceLanguage: "vi",
  backgroundMusicPath: null,
  ttsEnabled: true,
  voiceId: null,
  subtitleMode: "track",
  subtitleStyle: defaultSubtitleStyle,
  renderQuality: "1080p60",
  renderEngine: "manim",
  videoFont: "Be Vietnam Pro",
  videoFormatId: "visual_first_7min",
  backgroundMusicVolume: 0.2,
  videoOutputMode: "long",
  authoringMode: "manual",
  authoringModels: { story: "", storyboard: "", code: "" },
  authoringTopic: "",
  authoringStory: "",
  authoringStoryboard: "",
  hasSubmitted: false,
};

function wrapperFor(draft: ProjectDraft) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return (
      <MemoryRouter>
        <ProjectDraftContext.Provider value={draft}>{children}</ProjectDraftContext.Provider>
      </MemoryRouter>
    );
  };
}

describe("useRequireScript", () => {
  it("stays ready for a valid Remotion script, without running the Manim-only lint", () => {
    const draft: ProjectDraft = {
      ...BASE_DRAFT,
      renderEngine: "remotion",
      scriptContent: 'export const narrations: string[] = ["xin chào"];',
    };
    const { result } = renderHook(() => useRequireScript(), { wrapper: wrapperFor(draft) });
    // Bug: validateScript's Manim-only "class Scene" check used to run
    // unconditionally here, so a perfectly valid Remotion script silently
    // bounced the Creator back to "/" every time Settings/Review mounted.
    expect(result.current).toBe(true);
  });

  it("still requires valid Manim code when the engine is manim", () => {
    const draft: ProjectDraft = {
      ...BASE_DRAFT,
      renderEngine: "manim",
      scriptContent: "not a valid manim script",
    };
    const { result } = renderHook(() => useRequireScript(), { wrapper: wrapperFor(draft) });
    expect(result.current).toBe(false);
  });

  it("still requires non-empty content for Remotion", () => {
    const draft: ProjectDraft = { ...BASE_DRAFT, renderEngine: "remotion", scriptContent: "" };
    const { result } = renderHook(() => useRequireScript(), { wrapper: wrapperFor(draft) });
    expect(result.current).toBe(false);
  });
});
