import { useEffect } from "react";
import { BrowserRouter, Routes, Route, Navigate, useLocation, useNavigate } from "react-router-dom";
import { ProjectDraftProvider } from "./context/ProjectDraftContext";
import { AuthoringRunProvider } from "./context/AuthoringRunContext";
import { ProjectFlowProvider } from "./context/ProjectFlowContext";
import { ThemeProvider } from "./context/ThemeContext";
import { NavigationLoader } from "./components/NavigationLoader";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { ScriptStepPage } from "./pages/ScriptStepPage";
import { ResumeProjectPage } from "./pages/ResumeProjectPage";
import { CreativeBriefStepPage } from "./pages/CreativeBriefStepPage";
import { ValidatePage } from "./pages/ValidatePage";
import { RenderPage } from "./pages/RenderPage";
import { ResultPage } from "./pages/ResultPage";
import { PublishPage } from "./pages/PublishPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { VideoListPage } from "./pages/VideoListPage";
import { ScriptOutlineStepPage } from "./pages/ScriptOutlineStepPage";
import { VisualDirectorStepPage } from "./pages/VisualDirectorStepPage";
import { CodeStepPage } from "./pages/CodeStepPage";
import { IllustrationsStepPage } from "./pages/IllustrationsStepPage";
import { VideoArchetypeSettingsPage } from "./pages/VideoArchetypeSettingsPage";
import { VisualLanguagePage } from "./pages/VisualLanguagePage";
import { IllustrationLibraryPage } from "./pages/IllustrationLibraryPage";
import { PromptSettingsPage } from "./pages/PromptSettingsPage";
import { JournalPage } from "./pages/JournalPage";
import { FlowMapPage } from "./pages/FlowMapPage";
import { AiModelSettingsPage } from "./pages/AiModelSettingsPage";
import { StepPreviewPage } from "./pages/StepPreviewPage";
import { ComingStepPage } from "./pages/ComingStepPage";
import { LockStepPage } from "./pages/LockStepPage";
import { ScriptWriteStepPage } from "./pages/ScriptWriteStepPage";
import { ResearchStepPage } from "./pages/ResearchStepPage";
import { BeatsStepPage } from "./pages/BeatsStepPage";
import { StoryStepPage } from "./pages/StoryStepPage";

/**
 * The wizard draft lives only in memory (see ProjectDraftContext), so a page
 * load that lands on a /create/* step has no draft behind it — send it back
 * to step 1 instead of showing an empty step. Runs once per page load; in-app
 * navigation never remounts App.
 */
function RedirectStaleWizardLoad() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  useEffect(() => {
    if (pathname.startsWith("/create/")) navigate("/", { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return null;
}

export function App() {
  return (
    <ThemeProvider>
      <ProjectDraftProvider>
        <AuthoringRunProvider>
        <BrowserRouter>
          <ProjectFlowProvider>
          <RedirectStaleWizardLoad />
          <NavigationLoader />
          <KeyboardShortcutsHelp />
          <Routes>
            {/* "/" (Bước 1) viết ý tưởng từ đề tài và tạo project.
                "/create/script/brief" (Bước 2) chọn dạng video, ngôn ngữ,
                thời lượng rồi viết brief từ ý tưởng đã duyệt. Mỗi bước một
                URL riêng để Back/reload/bookmark đều chạy, và bước nào cũng
                bấm sang được bất kể tiến độ. */}
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/brief" element={<CreativeBriefStepPage />} />
            <Route path="/create/script/settings" element={<Navigate to="/create/script/brief" replace />} />
            <Route path="/create/script/outline" element={<ScriptOutlineStepPage />} />
            <Route path="/create/script/storyboard" element={<VisualDirectorStepPage />} />
            {/* The video's drawings, between Visual and Code. */}
            <Route path="/create/script/illustrations" element={<IllustrationsStepPage />} />
            <Route path="/create/script/code" element={<CodeStepPage />} />
            {/* Read-only preview of a step not reached yet. */}
            <Route path="/create/preview/:step" element={<StepPreviewPage />} />
            <Route path="/create/settings" element={<Navigate to="/create/script/brief" replace />} />
            <Route path="/create/review" element={<Navigate to="/create/script/brief" replace />} />

            {/* Admin screen to edit pipeline prompt wording, outside
                the Creator wizard flow. */}
            <Route path="/settings/prompts" element={<PromptSettingsPage />} />
            <Route path="/settings/ai-model" element={<AiModelSettingsPage />} />
            <Route path="/settings/video-archetypes" element={<VideoArchetypeSettingsPage />} />
            <Route path="/settings/visual-languages" element={<VisualLanguagePage />} />
            {/* The illustration library, folders of reviewed drawings. */}
            <Route path="/settings/illustrations" element={<IllustrationLibraryPage />} />

            {/* Bước 4 (chạy thử + duyệt dàn ý) và bước 5 (sản xuất)
                là hai màn riêng. Mỗi trang tự đẩy sang trang kia khi trạng
                thái project không thuộc về nó (projectPath). */}
            <Route path="/projects/:id/preview/:step" element={<StepPreviewPage />} />
            <Route path="/projects/:id/research" element={<ResearchStepPage />} />
            <Route path="/projects/:id/story" element={<StoryStepPage />} />
            <Route path="/projects/:id/beats" element={<BeatsStepPage />} />
            <Route path="/projects/:id/script" element={<ScriptWriteStepPage />} />
            <Route path="/projects/:id/lock" element={<LockStepPage />} />
            <Route path="/projects/:id/coming/:n" element={<ComingStepPage />} />
            <Route path="/create/coming/:n" element={<ComingStepPage />} />
            <Route path="/projects/:id/resume" element={<ResumeProjectPage />} />
            <Route path="/projects/:id/validate" element={<ValidatePage />} />
            <Route path="/projects/:id/render" element={<RenderPage />} />
            <Route path="/projects/:id/result" element={<ResultPage />} />
            <Route path="/projects/:id/publish" element={<PublishPage />} />
            <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
            <Route path="/videos" element={<VideoListPage />} />
            <Route path="/journal" element={<JournalPage />} />
            <Route path="/flow" element={<FlowMapPage />} />
          </Routes>
          </ProjectFlowProvider>
        </BrowserRouter>
        </AuthoringRunProvider>
      </ProjectDraftProvider>
    </ThemeProvider>
  );
}
