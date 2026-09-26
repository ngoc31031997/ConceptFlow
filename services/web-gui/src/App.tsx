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
import { ScriptAuthoringSettingsStepPage } from "./pages/ScriptAuthoringSettingsStepPage";
import { ValidatePage } from "./pages/ValidatePage";
import { RenderPage } from "./pages/RenderPage";
import { ResultPage } from "./pages/ResultPage";
import { PublishPage } from "./pages/PublishPage";
import { OAuthCallbackPage } from "./pages/OAuthCallbackPage";
import { VideoListPage } from "./pages/VideoListPage";
import { ScriptOutlineStepPage } from "./pages/ScriptOutlineStepPage";
import { VisualDirectorStepPage } from "./pages/VisualDirectorStepPage";
import { ManimEngineerStepPage } from "./pages/ManimEngineerStepPage";
import { VideoArchetypeSettingsPage } from "./pages/VideoArchetypeSettingsPage";
import { PromptSettingsPage } from "./pages/PromptSettingsPage";
import { JournalPage } from "./pages/JournalPage";

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
            {/* "/" (Bước 1) chỉ còn tình huống "chỉ có ý tưởng" — chọn xong
                là tạo project ngay. "/create/script/settings" (Bước 2) chọn
                ngôn ngữ, engine, cách làm rồi mới vào chuỗi 3 tab bên dưới
                (xem ScriptPipelineTabs). Mỗi tab một URL riêng để Back/
                reload/bookmark đều chạy, và tab nào cũng bấm sang được bất
                kể tiến độ. */}
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/settings" element={<ScriptAuthoringSettingsStepPage />} />
            <Route path="/create/script/outline" element={<ScriptOutlineStepPage />} />
            <Route path="/create/script/storyboard" element={<VisualDirectorStepPage />} />
            <Route path="/create/script/code" element={<ManimEngineerStepPage />} />
            <Route path="/create/settings" element={<Navigate to="/create/script/settings" replace />} />
            <Route path="/create/review" element={<Navigate to="/create/script/settings" replace />} />

            {/* CR-025 — admin screen to edit pipeline prompt wording, outside
                the Creator wizard flow. */}
            <Route path="/settings/prompts" element={<PromptSettingsPage />} />
            <Route path="/settings/video-archetypes" element={<VideoArchetypeSettingsPage />} />

            {/* CR-031 — bước 4 (chạy thử + duyệt dàn ý) và bước 5 (sản xuất)
                là hai màn riêng. Mỗi trang tự đẩy sang trang kia khi trạng
                thái project không thuộc về nó (projectPath). */}
            <Route path="/projects/:id/resume" element={<ResumeProjectPage />} />
            <Route path="/projects/:id/validate" element={<ValidatePage />} />
            <Route path="/projects/:id/render" element={<RenderPage />} />
            <Route path="/projects/:id/result" element={<ResultPage />} />
            <Route path="/projects/:id/publish" element={<PublishPage />} />
            <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
            <Route path="/videos" element={<VideoListPage />} />
            <Route path="/journal" element={<JournalPage />} />
          </Routes>
          </ProjectFlowProvider>
        </BrowserRouter>
        </AuthoringRunProvider>
      </ProjectDraftProvider>
    </ThemeProvider>
  );
}
