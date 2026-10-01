import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, createProjectDraft, getAuthoringState } from "../api/client";
import { FLOW_CONFIG } from "../utils/flow";
import { Button } from "./ui";
import glass from "../styles/glass.module.css";

interface MakeShortButtonProps {
  /** The long video the short is made from. */
  projectId: string;
  contentLanguage: "vi" | "en";
}

/**
 * "Làm bản short dọc": starts a new project on the same topic as this long
 * video, built vertically at 1080x1920 with its own script and storyboard,
 * and linked to this video both ways. Opens step 2 (Cấu hình) of the new
 * project, where the short format and Remotion are already chosen.
 */
export function MakeShortButton({ projectId, contentLanguage }: MakeShortButtonProps) {
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    setBusy(true);
    setError(null);
    try {
      const { topic } = await getAuthoringState(projectId);
      const shortId = crypto.randomUUID();
      await createProjectDraft(shortId, topic, contentLanguage, "remotion", projectId);
      navigate(`/projects/${shortId}/resume?step=${FLOW_CONFIG}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
      setBusy(false);
    }
  }

  return (
    <div>
      <p className={glass.cardHint}>
        Một video dọc 30–60 giây cùng chủ đề, kịch bản và hình dựng riêng cho khung 9:16 — không cắt
        từ video này.
      </p>
      {error && (
        <p role="alert" className={`${glass.helperText} ${glass.mtXs}`}>
          Không tạo được bản short: {error}
        </p>
      )}
      <div className={`${glass.ctaRow} ${glass.mtSm}`}>
        <Button disabled={busy} onClick={handleClick} data-testid="make-short">
          {busy ? "Đang tạo..." : "Làm bản short dọc"}
        </Button>
      </div>
    </div>
  );
}
