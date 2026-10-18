import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, getProjectVisualLanguage, type PinnedVisualLanguage } from "../api/client";
import { Button } from "./ui";
import glass from "../styles/glass.module.css";

/**
 * The visual language this video is pinned to: its name and version, set when
 * the project was created. Enabling another language later does not change it,
 * so the card only reads; "Xem ở Cài đặt" opens the language's page.
 */
export function PinnedLanguageCard({ projectId }: { projectId: string | null }) {
  const navigate = useNavigate();
  const [pinned, setPinned] = useState<PinnedVisualLanguage | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    setPinned(null);
    setError(null);
    getProjectVisualLanguage(projectId)
      .then((p) => {
        if (!cancelled) setPinned(p);
      })
      .catch((e: unknown) => {
        if (cancelled) return;
        setError(
          e instanceof ApiError && e.status === 404
            ? "Video này chưa ghim ngôn ngữ hình, nên chưa vẽ hay render được."
            : "Không đọc được ngôn ngữ hình của video này. Vui lòng thử lại.",
        );
      });
    return () => {
      cancelled = true;
    };
  }, [projectId]);

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="pinned-language">
      {pinned ? (
        <p>
          <strong data-testid="pinned-language-name">
            {pinned.name} v{pinned.visual_language_version}
          </strong>{" "}
          · ghim lúc tạo project · bật ngôn ngữ khác sau này không đổi video này
        </p>
      ) : error ? (
        <p role="alert" data-testid="pinned-language-error">
          {error}
        </p>
      ) : (
        <p className={glass.cardHint}>Đang đọc ngôn ngữ hình…</p>
      )}
      <Button variant="view" onClick={() => navigate("/settings/visual-languages")} data-testid="pinned-language-open">
        Xem ở Cài đặt
      </Button>
    </div>
  );
}
