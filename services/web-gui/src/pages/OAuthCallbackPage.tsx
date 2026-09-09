import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { completeYoutubeAuthCallback } from "../api/client";
import { AppShell } from "../components/AppShell";
import glass from "../styles/glass.module.css";

export function OAuthCallbackPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);

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
        navigate(result.state ? `/projects/${result.state}/result` : "/", { replace: true });
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : String(err));
      });
  }, [searchParams, navigate]);

  return (
    // Connecting YouTube happens from the result screen, so this is step 5
    // ("Đăng"). It read 3 — the last step of the wizard back when creation
    // was three screens — which lit up "Xem lại" mid-publish.
    <AppShell currentStep={5} title="Đang kết nối YouTube..." subtitle="Vui lòng chờ trong giây lát.">
      {error && (
        <p role="alert" className={glass.helperText}>
          {error}
        </p>
      )}
    </AppShell>
  );
}
