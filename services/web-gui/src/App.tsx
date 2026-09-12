import { BrowserRouter, Routes, Route } from "react-router-dom";
import { ProjectDraftProvider } from "./context/ProjectDraftContext";
import { ThemeProvider } from "./context/ThemeContext";
import { NavigationLoader } from "./components/NavigationLoader";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { ScriptStepPage } from "./pages/ScriptStepPage";
import { SettingsStepPage } from "./pages/SettingsStepPage";
import { ReviewStepPage } from "./pages/ReviewStepPage";
import { RenderPage } from "./pages/RenderPage";
import { ResultPage } from "./pages/ResultPage";
import { PublishPage } from "./pages/PublishPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { VideoListPage } from "./pages/VideoListPage";
import { VisualDirectorStepPage } from "./pages/VisualDirectorStepPage";
import { ManimEngineerStepPage } from "./pages/ManimEngineerStepPage";
import { PromptSettingsPage } from "./pages/PromptSettingsPage";

export function App() {
  return (
    <ThemeProvider>
      <ProjectDraftProvider>
        <BrowserRouter>
          <NavigationLoader />
          <KeyboardShortcutsHelp />
          <Routes>
            {/* The three creation steps, each its own URL so Back works. */}
            <Route path="/" element={<ScriptStepPage />} />
            {/* CR-025 step 2 — reached from step 1's "blank" (Story Architect)
                path; step 3 (Manim Engineer) is still a stub. */}
            <Route path="/create/visual-director" element={<VisualDirectorStepPage />} />
            <Route path="/create/manim-engineer" element={<ManimEngineerStepPage />} />
            <Route path="/create/settings" element={<SettingsStepPage />} />
            <Route path="/create/review" element={<ReviewStepPage />} />

            {/* CR-025 — admin screen to edit pipeline prompt wording, outside
                the Creator wizard flow. */}
            <Route path="/settings/prompts" element={<PromptSettingsPage />} />

            <Route path="/projects/:id/render" element={<RenderPage />} />
            <Route path="/projects/:id/result" element={<ResultPage />} />
            <Route path="/projects/:id/publish" element={<PublishPage />} />
            <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
            <Route path="/videos" element={<VideoListPage />} />
          </Routes>
        </BrowserRouter>
      </ProjectDraftProvider>
    </ThemeProvider>
  );
}
