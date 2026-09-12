import { BrowserRouter, Routes, Route } from "react-router-dom";
import { ProjectDraftProvider } from "./context/ProjectDraftContext";
import { ScriptStepPage } from "./pages/ScriptStepPage";
import { SettingsStepPage } from "./pages/SettingsStepPage";
import { ReviewStepPage } from "./pages/ReviewStepPage";
import { RenderPage } from "./pages/RenderPage";
import { ResultPage } from "./pages/ResultPage";
import { PublishPage } from "./pages/PublishPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { VideoListPage } from "./pages/VideoListPage";

export function App() {
  return (
    <ProjectDraftProvider>
      <BrowserRouter>
        <Routes>
          {/* The three creation steps, each its own URL so Back works. */}
          <Route path="/" element={<ScriptStepPage />} />
          <Route path="/create/settings" element={<SettingsStepPage />} />
          <Route path="/create/review" element={<ReviewStepPage />} />

          <Route path="/projects/:id/render" element={<RenderPage />} />
          <Route path="/projects/:id/result" element={<ResultPage />} />
          <Route path="/projects/:id/publish" element={<PublishPage />} />
          <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
          <Route path="/videos" element={<VideoListPage />} />
        </Routes>
      </BrowserRouter>
    </ProjectDraftProvider>
  );
}
