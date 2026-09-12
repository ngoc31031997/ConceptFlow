import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { getPromptTemplate } from "../api/client";
import styles from "./WizardSteps.module.css";

/**
 * CR-025 step 3 (Manim Engineer) — STUB, mirroring how step 2 (Visual
 * Director) was stubbed before it was wired up in this same change: shows
 * the manim_engineer template filled with story+storyboard as
 * {{previous_output}}, so the pipeline stays visibly continuous, but it does
 * not yet collect generated code back or persist anything. Full wiring
 * (steps 3-4: Manim code generation, Script Reviewer verdict) is a
 * documented TODO.
 */
export function ManimEngineerStepPage() {
  const draft = useContext(ProjectDraftContext);
  const navigate = useNavigate();
  const [prompt, setPrompt] = useState("Đang tải...");

  useEffect(() => {
    let cancelled = false;
    const previousOutput = [draft.authoringStory, draft.authoringStoryboard]
      .filter((part) => part.trim().length > 0)
      .join("\n\n---\n\n");
    getPromptTemplate("manim_engineer", draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const filled = template.template_text.split("{{previous_output}}").join(
          previousOutput || "(chưa có dàn ý/storyboard đã lưu ở các bước trước)",
        );
        setPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setPrompt("Không tải được template manim_engineer.");
      });
    return () => {
      cancelled = true;
    };
  }, [draft.voiceLanguage, draft.authoringStory, draft.authoringStoryboard]);

  return (
    <div data-testid="manim-engineer-step-page">
      <AppShell
        title="Bước 3 — Manim Engineer (sắp ra mắt)"
        subtitle="Bước này đang được hoàn thiện. Bên dưới là prompt sẽ dùng để sinh code Manim từ storyboard bước 2."
        wide
      >
        <div className={styles.scriptLayout}>
          <div>
            <p>
              <strong>Sắp ra mắt:</strong> dán code AI trả về, hệ thống sẽ kiểm tra và chuyển sang bước 4
              (Script Reviewer). Hiện tại bạn có thể copy prompt bên dưới để tự thử quy trình thủ công.
            </p>
            <textarea readOnly value={prompt} rows={24} style={{ width: "100%", fontFamily: "monospace" }} />
          </div>
        </div>
      </AppShell>

      <WizardNav
        hint="Bước 3-4 của quy trình soạn kịch bản mới đang được hoàn thiện."
        onNext={() => navigate("/create/settings")}
        nextLabel="Tạm dùng script hiện có để tiếp tục"
        nextTestId="manim-engineer-step-next"
      />
    </div>
  );
}
