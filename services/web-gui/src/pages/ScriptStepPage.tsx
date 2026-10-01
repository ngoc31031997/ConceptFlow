import { useContext, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Card, TextArea } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { createProjectDraft } from "../api/client";
import { FLOW_INIT, flowTitle } from "../utils/flow";

/**
 * Bước 1 — Ý tưởng: chỉ có chủ đề, và đây là chỗ duy nhất sửa chủ đề. Ngôn
 * ngữ, kiểu video, giọng đọc và cách soạn chọn ở Bước 2
 * (ScriptAuthoringSettingsStepPage). "Tiếp tục" tạo project ngay, để
 * projectId sẵn sàng trước khi vào Bước 2, và chuyển danh sách dự án trùng
 * chủ đề sang đó qua router state.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const location = useLocation();
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  // "Tạo video mới" → projectId mới. Xoá cờ khỏi history để reload/Quay lại
  // không reset thêm lần nữa và làm mất chủ đề vừa gõ.
  useEffect(() => {
    const st = location.state as { newVideo?: boolean } | null;
    if (!st?.newVideo) return;
    dispatch({ type: "RESET" });
    navigate(location.pathname, { replace: true, state: null });
  }, [location.key]); // eslint-disable-line react-hooks/exhaustive-deps

  const topic = draft.authoringTopic;
  const canContinue = topic.trim().length > 0 && !creating;

  async function handleContinue() {
    if (!topic.trim()) return;
    setError(null);
    setCreating(true);
    try {
      const { similarProjects } = await createProjectDraft(
        draft.projectId,
        topic.trim(),
        draft.voiceLanguage,
        draft.renderEngine,
      );
      // Bước 2 hiện cảnh báo trùng chủ đề ngay đầu màn.
      navigate("/create/script/settings", { state: { similarProjects } });
    } catch {
      setError("Không tạo được dự án. Vui lòng kiểm tra kết nối và thử lại.");
    } finally {
      setCreating(false);
    }
  }

  return (
    <div data-testid="script-step-page">
      <AppShell
        currentStep={1}
        wide
        title={flowTitle(FLOW_INIT)}
        subtitle="Bắt đầu video mới từ một ý tưởng."
      >
        <Card
          title="Chưa có gì, chỉ có ý tưởng"
          hint="Nhập chủ đề, sau đó cùng AI dựng dàn ý, hình ảnh và code."
        >
          <TextArea
            value={topic}
            onChange={(e) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: e.target.value })}
            placeholder="Ý tưởng/chủ đề video của bạn là gì?"
            rows={4}
            data-testid="script-step-topic"
          />
        </Card>
      </AppShell>

      <WizardNav
        hint={
          error ??
          (creating
            ? "Đang tạo dự án..."
            : topic.trim()
              ? "Sẵn sàng."
              : "Nhập ý tưởng để tiếp tục.")
        }
        isBlocked={!!error}
        onNext={handleContinue}
        nextLabel={creating ? "Đang tạo..." : "Tiếp tục"}
        nextDisabled={!canContinue}
        nextTestId="script-step-next"
      />
    </div>
  );
}
