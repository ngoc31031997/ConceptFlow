import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Card, TextArea } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { createProjectDraft } from "../api/client";

/**
 * Bước 1 — chỉ còn tình huống "chưa có gì, chỉ có ý tưởng" (các tình huống
 * đã có dàn ý/storyboard/code sẽ quay lại sau; xem CR-031 cho lịch sử). Ngôn
 * ngữ, render engine và cách làm (manual/AI) đã dời sang Bước 2
 * (ScriptAuthoringSettingsStepPage) — chọn xong ở đây là tạo project ngay,
 * để projectId sẵn sàng trước khi vào Bước 2.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  const topic = draft.authoringTopic;
  const canContinue = topic.trim().length > 0 && !creating;

  async function handleContinue() {
    if (!topic.trim()) return;
    setError(null);
    setCreating(true);
    try {
      await createProjectDraft(draft.projectId, topic.trim(), draft.voiceLanguage, draft.renderEngine);
      navigate("/create/script/settings");
    } catch {
      setError("Không tạo được project — kiểm tra kết nối rồi thử lại.");
    } finally {
      setCreating(false);
    }
  }

  return (
    <div data-testid="script-step-page">
      <AppShell
        currentStep={1}
        wide
        title="Bước 1 — Ý tưởng"
        subtitle="Bạn đang có sẵn tới đâu? Hiện tại chỉ hỗ trợ bắt đầu từ ý tưởng."
      >
        <Card
          title="Chưa có gì, chỉ có ý tưởng"
          hint="Nhập chủ đề rồi dựng dàn ý → storyboard → code cùng AI, từng bước một."
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
            ? "Đang tạo project..."
            : topic.trim()
              ? "Sẵn sàng khi bạn bấm Tiếp tục."
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
