import { BrowserRouter, Routes, Route } from "react-router-dom";
import { ProjectDraftProvider } from "./context/ProjectDraftContext";
import { AuthoringRunProvider } from "./context/AuthoringRunContext";
import { ThemeProvider } from "./context/ThemeContext";
import { NavigationLoader } from "./components/NavigationLoader";
import { KeyboardShortcutsHelp } from "./components/KeyboardShortcutsHelp";
import { ScriptStepPage } from "./pages/ScriptStepPage";
import { SettingsStepPage } from "./pages/SettingsStepPage";
import { ReviewStepPage } from "./pages/ReviewStepPage";
import { ValidatePage } from "./pages/ValidatePage";
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
        <AuthoringRunProvider>
        <BrowserRouter>
          <NavigationLoader />
          <KeyboardShortcutsHelp />
          <Routes>
            {/* "/" chỉ chọn ngôn ngữ, engine, cách làm và tình huống (ý tưởng
                / đã có dàn ý / đã có storyboard / đã có code); mỗi tình huống
                mở đúng một tab của chuỗi 3 tab bên dưới (xem
                ScriptPipelineTabs). Mỗi tab một URL riêng để Back/reload/
                bookmark đều chạy, và tab nào cũng bấm sang được bất kể tiến
                độ. */}
            <Route path="/" element={<ScriptStepPage />} />
            <Route path="/create/script/outline" element={<ScriptOutlineStepPage />} />
            <Route path="/create/script/storyboard" element={<VisualDirectorStepPage />} />
            <Route path="/create/script/code" element={<ManimEngineerStepPage />} />
            <Route path="/create/settings" element={<SettingsStepPage />} />
            <Route path="/create/review" element={<ReviewStepPage />} />

            {/* CR-025 — admin screen to edit pipeline prompt wording, outside
                the Creator wizard flow. */}
            <Route path="/settings/prompts" element={<PromptSettingsPage />} />

            {/* CR-031 — bước 4 (chạy thử + duyệt dàn ý) và bước 5 (sản xuất)
                là hai màn riêng. Mỗi trang tự đẩy sang trang kia khi trạng
                thái project không thuộc về nó (projectPath). */}
            <Route path="/projects/:id/validate" element={<ValidatePage />} />
            <Route path="/projects/:id/render" element={<RenderPage />} />
            <Route path="/projects/:id/result" element={<ResultPage />} />
            <Route path="/projects/:id/publish" element={<PublishPage />} />
            <Route path="/oauth/youtube/callback" element={<OAuthCallbackPage />} />
            <Route path="/videos" element={<VideoListPage />} />
          </Routes>
        </BrowserRouter>
        </AuthoringRunProvider>
      </ProjectDraftProvider>
    </ThemeProvider>
  );
}
