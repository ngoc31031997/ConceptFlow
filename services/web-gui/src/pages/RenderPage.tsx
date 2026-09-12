import { useContext, useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { OutlineReview } from "../components/OutlineReview";
import { OutlineActions } from "../components/OutlineActions";
import { ErrorBanner } from "../components/ErrorBanner";
import { AppShell } from "../components/AppShell";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { useOutlineReview } from "../hooks/useOutlineReview";
import { retryProject, ApiError } from "../api/client";
import { ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import type { Project } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./RenderPage.module.css";

// Steps whose input comes straight from the "Soạn nội dung" screen
// (script/plugin/category) — a failure here is most likely a bad input,
// so the user should go back and fix it rather than blindly retry the
// same input against the same failing step.
// validate_script (CR-020) is where a Creator's own script mistake (bad API
// call, crashed dry run, missing beat) surfaces — the Creator needs to go
// fix the script, not retry the same broken one. classify_scenes no longer
// exists in the saga but is kept here for old projects that failed on it
// before that step was removed.
const INPUT_RELATED_STEPS = new Set(["parse_script", "validate_script", "classify_scenes"]);

// useOutlineReview cannot be called conditionally (Rules of Hooks) even
// though its buttons only render once `project` exists — this placeholder
// keeps the hook call unconditional without ever being read for real, since
// nothing renders the actions until isAwaitingReview && project are true.
const EMPTY_PROJECT: Project = { project_id: "", status: "draft", voice_language: "vi", scenes: [] };

export function RenderPage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  const progressState = useSSE(projectId);
  const { project, refetch } = useProject(projectId);
  const dispatchDraft = useContext(ProjectDraftDispatchContext);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);

  // CR-024: Saga đang dừng chờ người, không phải đang chạy. Phân biệt hai thứ
  // này là cả điểm của FR69.5 — nếu không Creator sẽ ngồi đợi một tiến trình
  // đã dừng từ lâu.
  const isAwaitingReview = project?.status === "awaiting_review";

  const outline = useOutlineReview(project ?? EMPTY_PROJECT, refetch, () => {
    // Server has already put this project_id back to draft (review_outline.go
    // Reject). RESUME_EDITING keeps the script and project_id intact — only
    // clears hasSubmitted — so ScriptStepPage does not wipe them via its own
    // reset-on-mount.
    dispatchDraft({ type: "RESUME_EDITING" });
    navigate("/");
  });

  const isFailed =
    progressState.status === "failed" || Boolean(project?.status.startsWith("failed_at_"));
  const errorMessage = progressState.errorMessage ?? project?.error_message ?? "";
  const failedStep = progressState.currentStep ?? project?.status.replace("failed_at_", "") ?? null;
  const isInputError = isFailed && failedStep !== null && INPUT_RELATED_STEPS.has(failedStep);

  useEffect(() => {
    if (project?.status === "ready_to_publish") {
      navigate(`/projects/${projectId}/result`);
    }
  }, [project?.status, projectId, navigate]);

  async function handleRetry() {
    setIsRetrying(true);
    setRetryError(null);
    try {
      await retryProject(projectId);
    } catch (err) {
      setRetryError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsRetrying(false);
    }
  }

  return (
    <div data-testid="render-page">
      <AppShell
        currentStep={4}
        wide={isAwaitingReview}
        headerAction={
          <Link to="/" className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
        title={
          isFailed
            ? "Đã xảy ra lỗi"
            : isAwaitingReview
              ? "Duyệt dàn ý trước khi sản xuất"
              : "Đang xử lý video"
        }
        subtitle={
          isFailed
            ? "Một bước trong quá trình xử lý không hoàn tất."
            : isAwaitingReview
              ? "Chưa tạo giọng đọc, chưa render — sửa gì cũng không tốn gì. Duyệt xong hệ thống mới bắt đầu tốn tiền/thời gian."
              : "Hệ thống đang tạo hoạt hình, giọng đọc và ghép video cho bạn."
        }
      >
        {isAwaitingReview && project ? (
          /*
            Two columns only while there is an outline to review: it can run
            to dozens of lines, and stacking it above the tracker used to push
            status far down a wall of text. The tracker moves to a sticky
            side column instead of disappearing — Creator still sees where
            the saga is while reading/editing the outline.

            OutlineActions (Duyệt/Từ chối) sits at the TOP of that side
            column, above the tracker — bug report: the buttons used to live
            at the bottom of the outline list itself, which could run to
            dozens of lines and scroll them out of view exactly when needed.
          */
          <div className={styles.layout}>
            <OutlineReview project={project} outline={outline} />
            <div className={styles.tracker}>
              <OutlineActions outline={outline} />
              <div className={glass.mtSm}>
                <ProgressTracker
                  progressState={failedStep ? { ...progressState, currentStep: failedStep } : progressState}
                  isFailed={isFailed}
                />
              </div>
            </div>
          </div>
        ) : (
          <>
            {/*
              The tracker stays visible after a failure. Replacing it outright
              hid how far the pipeline actually got, which is the first thing
              you want to know when deciding between retrying and going back
              to the script.
            */}
            {isFailed && (
              <ErrorBanner
                errorMessage={retryError ?? errorMessage}
                onRetry={handleRetry}
                isRetrying={isRetrying}
                onBack={isInputError ? () => navigate("/") : undefined}
              />
            )}
            <ProgressTracker
              progressState={failedStep ? { ...progressState, currentStep: failedStep } : progressState}
              isFailed={isFailed}
            />
          </>
        )}
      </AppShell>
    </div>
  );
}
