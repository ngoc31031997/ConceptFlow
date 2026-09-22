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
import { ScriptOutlineStepPage } from "./pages/ScriptOutlineStepPage";
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
            {/* "/" only picks the situation (dựng từ đầu / đã có code / code
                chuẩn) — "dựng từ đầu" immediately hands off to the 3-tab
                sub-wizard below (see ScriptPipelineTabs), each tab its own
                URL so Back/reload/bookmarks all work and every tab stays
                reachable regardless of progress. */}
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/outline" element={<ScriptOutlineStepPage />} />
            <Route path="/create/script/storyboard" element={<VisualDirectorStepPage />} />
            <Route path="/create/script/code" element={<ManimEngineerStepPage />} />
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
