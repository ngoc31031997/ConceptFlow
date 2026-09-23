import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { completeYoutubeAuthCallback } from "../api/client";
import { AppShell } from "../components/AppShell";
import glass from "../styles/glass.module.css";

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [connectedChannel, setConnectedChannel] = useState<string | null>(null);

  useEffect(() => {
    const code = searchParams.get("code");
    const state = searchParams.get("state");

    if (!code) {
      setError("Thiếu mã xác thực từ Google");
      return;
    }

    completeYoutubeAuthCallback(code, state)
      .then((result) => {
        if (!result.connected) {
          setError(result.error ?? "Kết nối YouTube thất bại");
          return;
        }
        // Which channel was actually consented is decided on Google's
        // chooser, not here, so it is worth confirming on the way back.
        if (result.channel_title) {
          setConnectedChannel(result.channel_title);
        }
        navigate(result.state ? `/projects/${result.state}/result` : "/", { replace: true });
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : String(err));
      });
  }, [searchParams, navigate]);

  return (
    // Nối YouTube bắt đầu từ màn kết quả, nên đây là bước 6 ("Kết quả") —
    // Creator sẽ quay lại đúng chỗ đó sau khi nối xong.
    <AppShell
      currentStep={6}
      title={connectedChannel ? `Đã nối kênh ${connectedChannel}` : "Đang kết nối YouTube..."}
      subtitle="Vui lòng chờ trong giây lát."
    >
      {error && (
        <p role="alert" className={glass.helperText}>
          {error}
        </p>
      )}
    </AppShell>
  );
}
