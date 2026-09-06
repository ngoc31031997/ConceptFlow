import { BrowserRouter, Routes, Route } from "react-router-dom";
import { ProjectDraftProvider } from "./context/ProjectDraftContext";
import { NewProjectPage } from "./pages/NewProjectPage";
import { RenderPage } from "./pages/RenderPage";
import { ResultPage } from "./pages/ResultPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { VideoListPage } from "./pages/VideoListPage";

export function App() {
  return (
    <ProjectDraftProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<NewProjectPage />} />
          <Route path="/projects/:id/render" element={<RenderPage />} />
          <Route path="/projects/:id/result" element={<ResultPage />} />
          <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
          <Route path="/videos" element={<VideoListPage />} />
        </Routes>
      </BrowserRouter>
    </ProjectDraftProvider>
  );
}
