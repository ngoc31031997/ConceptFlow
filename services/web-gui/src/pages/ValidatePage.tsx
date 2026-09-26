import { useContext, useEffect, useState } from "react";
import { Link, useNavigate, useParams, useSearchParams } from "react-router-dom";
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
import { statusToStep, projectPhase, projectPath, VALIDATE_STEPS } from "../utils/pipelineLabels";
import { FLOW_REVIEW, FLOW_VALIDATE } from "../utils/flow";
import type { Project } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./ValidatePage.module.css";

// useOutlineReview cannot be called conditionally (Rules of Hooks) even
// though its buttons only render once `project` exists — this placeholder
// keeps the hook call unconditional without ever being read for real, since
// nothing renders the actions until isAwaitingReview && project are true.
const EMPTY_PROJECT: Project = { project_id: "", status: "draft", voice_language: "vi", scenes: [] };

/**
 * Bước 4 của 7 — "Validate": phần rẻ của saga, và điểm dừng trước phần đắt.
 *
 * CR-031 tách màn "Xử lý" cũ làm đôi ở đúng ranh giới mà saga vốn đã có: chạy
 * thử kịch bản (parse_script → validate_script, vài giây, không tốn gì) rồi
 * dừng ở `awaiting_review`; mọi thứ sau đó — TTS, render, ghép — mới là tiền
 * và thời gian thật. Gộp cả hai vào một màn khiến cổng duyệt trông như một
 * gián đoạn giữa chừng của quá trình render, nên Creator hoặc bấm duyệt cho
 * xong, hoặc ngồi đợi một saga đã dừng từ lâu. Tách ra thì bước 4 có đúng một
 * việc, và "Duyệt và sản xuất" là hành động chuyển bước chứ không phải một
 * nút lạc giữa thanh tiến trình.
 *
 * Trang này không tự nó quyết định gì trên server: cổng duyệt là của CR-024,
 * `ReviewEnabled` vẫn là thứ bật/tắt nó. Khi cổng tắt, saga không dừng và
 * effect bên dưới đẩy Creator thẳng sang bước 5.
 */
export function ValidatePage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  // ?view=1: mở chỉ để XEM lại bước 6/7 của một dự án đã đi xa hơn — không đẩy
  // sang màn đang sở hữu dự án, và không có nút hành động nào.
  const [search] = useSearchParams();
  const viewOnly = search.get("view") === "1";
  const viewStep = Number(search.get("step")) === FLOW_VALIDATE ? FLOW_VALIDATE : FLOW_REVIEW;
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

  // Duyệt xong là sang bước 5, nhưng KHÔNG điều hướng từ callback của nút
  // duyệt: cùng callback đó cũng chạy sau khi sửa một dòng lời thoại, và lúc
  // ấy Creator vẫn đang ở giữa việc duyệt. Bám vào trạng thái project là thứ
  // duy nhất phân biệt được hai lần gọi đó — và nó cũng xử lý luôn trường hợp
  // cổng duyệt tắt, hay Creator mở lại một bookmark cũ của bước 4.
  const phase = project ? projectPhase(project.status) : null;
  useEffect(() => {
    if (project && phase !== "validate" && !viewOnly) {
      navigate(projectPath(projectId, project.status), { replace: true });
    }
  }, [project, phase, viewOnly, projectId, navigate]);
  // Đang xem lại một bước đã qua (dự án đã sang phần sau): mọi thứ ở đây xong rồi.
  const reviewingPast = viewOnly && phase !== "validate";

  const isFailed =
    progressState.status === "failed" || Boolean(project?.status.startsWith("failed_at_"));
  const errorMessage = progressState.errorMessage ?? project?.error_message ?? "";
  const isCancelled = project?.run_state === "cancelled";

  // Mọi lỗi dừng ở bước này đều là lỗi đầu vào: parse_script và validate_script
  // chỉ đọc và chạy thử chính cái script Creator đưa vào. Thử lại y nguyên sẽ
  // hỏng y nguyên, nên "Quay lại sửa script" luôn có mặt ở đây — khác hẳn bước
  // 5, nơi một lần hỏng thường là hạ tầng và retry là việc đúng.
  const displayStep =
    (isFailed
      ? (progressState.currentStep ?? project?.status.replace("failed_at_", ""))
      : progressState.currentStep) ?? (project ? statusToStep(project.status) : null);
  const displayProgressState =
    displayStep && displayStep !== progressState.currentStep
      ? {
          ...progressState,
          currentStep: displayStep,
          status: isFailed ? progressState.status : ("in_progress" as const),
        }
      : progressState;

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
    <div data-testid="validate-page">
      <AppShell
        currentStep={viewOnly ? viewStep : isAwaitingReview ? FLOW_REVIEW : FLOW_VALIDATE}
        wide={isAwaitingReview || (reviewingPast && viewStep === FLOW_REVIEW)}
        headerAction={
          <Link to="/" className={glass.ghostBtn} style={{ textDecoration: "none" }}>
            Tạo video mới
          </Link>
        }
        title={
          isCancelled
            ? "Đã hủy kiểm tra"
            : isFailed
            ? "Kịch bản không chạy được"
            : isAwaitingReview
              ? "Duyệt dàn ý trước khi sản xuất"
              : "Đang kiểm tra kịch bản"
        }
        subtitle={
          isCancelled
            ? "Bạn đã dừng bước này. Bấm Chạy tiếp ở thanh trạng thái phía trên."
            : isFailed
            ? "Kịch bản chưa chạy được. Hãy sửa lại rồi chạy lại, chưa mất chi phí nào."
            : isAwaitingReview
              ? "Bạn có thể chỉnh sửa thoải mái ở bước này. Sau khi duyệt, hệ thống bắt đầu tạo video."
              : "Hệ thống đang kiểm tra kịch bản của bạn."
        }
      >
        {(isAwaitingReview || (reviewingPast && viewStep === FLOW_REVIEW)) && project ? (
          /*
            Two columns only while there is an outline to review: it can run
            to dozens of lines, and stacking it above the tracker used to push
            status far down a wall of text. The tracker moves to a sticky
            side column instead of disappearing.

            OutlineActions (Duyệt/Từ chối) sits at the TOP of that side
            column, above the tracker — bug report: the buttons used to live
            at the bottom of the outline list itself, which could run to
            dozens of lines and scroll them out of view exactly when needed.
          */
          <div className={styles.layout}>
            <OutlineReview project={project} outline={outline} />
            <div className={styles.tracker}>
              {isAwaitingReview && <OutlineActions outline={outline} />}
              <div className={glass.mtSm}>
                <ProgressTracker
                  progressState={displayProgressState}
                  steps={VALIDATE_STEPS}
                  isFailed={isFailed}
                  allDone={reviewingPast}
                />
              </div>
            </div>
          </div>
        ) : (
          <>
            {isFailed && !isCancelled && (
              <ErrorBanner
                errorMessage={retryError ?? errorMessage}
                onRetry={handleRetry}
                isRetrying={isRetrying}
                projectId={projectId}
                step={progressState.currentStep ?? project?.status.replace("failed_at_", "")}
              />
            )}
            <ProgressTracker
              progressState={displayProgressState}
              steps={VALIDATE_STEPS}
              isFailed={isFailed}
              allDone={reviewingPast}
            />
          </>
        )}
      </AppShell>
    </div>
  );
}
