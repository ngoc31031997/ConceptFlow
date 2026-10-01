import { useContext, useEffect, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ProgressTracker } from "../components/ProgressTracker";
import { OutlineReview } from "../components/OutlineReview";
import { ErrorBanner } from "../components/ErrorBanner";
import { ProductionSettingsPanel } from "../components/ProductionSettingsPanel";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { useSSE } from "../hooks/useSSE";
import { useProject } from "../hooks/useProject";
import { useOutlineReview } from "../hooks/useOutlineReview";
import { retryProject, ApiError } from "../api/client";
import { ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { statusToStep, projectPhase, projectPath, VALIDATE_STEPS, VALIDATE_SUBSTEP_NUMBERS } from "../utils/pipelineLabels";
import { FLOW_CODE, FLOW_REVIEW, FLOW_VALIDATE, flowTitle } from "../utils/flow";
import type { Project } from "../types";
import styles from "./ValidatePage.module.css";

// useOutlineReview cannot be called conditionally (Rules of Hooks) even
// though its buttons only render once `project` exists — this placeholder
// keeps the hook call unconditional without ever being read for real, since
// nothing renders the actions until isAwaitingReview && project are true.
const EMPTY_PROJECT: Project = { project_id: "", status: "draft", voice_language: "vi", scenes: [] };

/** Phần của màn Duyệt nội dung đang mở (?part=settings là phần 2). */
type ReviewPart = "review" | "settings";

/**
 * Bước 7 — "Kiểm tra tự động" (và 8 — "Duyệt nội dung", cùng màn): phần rẻ của
 * saga, và điểm dừng trước phần đắt.
 *
 * Bước 8 chia hai phần, mỗi phần một việc, cùng URL (`?part=settings`) để nút
 * Back của trình duyệt và tải lại trang giữ đúng phần đang xem: phần 1 duyệt
 * lời thoại, phần 2 chọn cài đặt xuất video (chất lượng, font, phụ đề, nhạc
 * nền) ngay trước nút "Duyệt và bắt đầu tạo video", vì bước Dựng hình và Ghép
 * video đọc chúng và là phần bắt đầu tốn tiền.
 *
 * Màn này dừng ở đúng ranh giới mà saga vốn đã có: chạy
 * thử kịch bản (parse_script → validate_script, vài giây, không tốn gì) rồi
 * dừng ở `awaiting_review`; mọi thứ sau đó — TTS, render, ghép — mới là tiền
 * và thời gian thật. Gộp cả hai vào một màn sẽ khiến cổng duyệt
 * trông như một gián đoạn giữa chừng của quá trình render. Bước 7 có đúng một
 * việc, và "Duyệt và sản xuất" là hành động chuyển bước chứ không phải một
 * nút lạc giữa thanh tiến trình.
 *
 * Trang này không tự nó quyết định gì trên server: cổng duyệt là của saga,
 * `ReviewEnabled` vẫn là thứ bật/tắt nó. Khi cổng tắt, saga không dừng và
 * effect bên dưới đẩy Creator thẳng sang bước 9 (TTS).
 */
export function ValidatePage() {
  const { id } = useParams<{ id: string }>();
  const projectId = id ?? "";
  const navigate = useNavigate();
  // ?view=1: mở chỉ để XEM lại bước 7/8 của một dự án đã đi xa hơn — không đẩy
  // sang màn đang sở hữu dự án, và không có nút hành động nào.
  const [search, setSearch] = useSearchParams();
  const part: ReviewPart = search.get("part") === "settings" ? "settings" : "review";
  const viewOnly = search.get("view") === "1";
  const viewStep = Number(search.get("step")) === FLOW_VALIDATE ? FLOW_VALIDATE : FLOW_REVIEW;
  const progressState = useSSE(projectId);
  const { project, refetch } = useProject(projectId);
  const dispatchDraft = useContext(ProjectDraftDispatchContext);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryError, setRetryError] = useState<string | null>(null);

  // Saga đang dừng chờ người, không phải đang chạy. Phân biệt hai thứ
  // này là cần thiết — nếu không Creator sẽ ngồi đợi một tiến trình
  // đã dừng từ lâu.
  const isAwaitingReview = project?.status === "awaiting_review";

  const outline = useOutlineReview(project ?? EMPTY_PROJECT, refetch, () => {
    // Server has already put this project back to draft (review_outline.go
    // Reject). The script is what needs fixing, so reopen the Code step; the
    // resume screen reloads the draft from the server first.
    dispatchDraft({ type: "RESUME_EDITING" });
    navigate(`/projects/${projectId}/resume?step=${FLOW_CODE}`);
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

  // Validate (7) and Review (8) share this screen; this is the one it shows now.
  const shownStep = viewOnly ? viewStep : isAwaitingReview ? FLOW_REVIEW : FLOW_VALIDATE;

  return (
    <div data-testid="validate-page">
      <AppShell
        currentStep={shownStep}
        wide={(isAwaitingReview && part === "review") || (reviewingPast && viewStep === FLOW_REVIEW)}
        // The title names the step as the rail does; what is happening
        // at it moves to the subtitle.
        title={flowTitle(shownStep)}
        subtitle={
          isCancelled
            ? "Đã hủy kiểm tra. Bạn đã dừng bước này. Bấm Chạy tiếp ở thanh trạng thái phía trên."
            : isFailed
            ? "Kịch bản không chạy được. Hãy sửa lại rồi chạy lại, chưa mất chi phí nào."
            : reviewingPast
            ? "Bước này đã chạy xong."
            : isAwaitingReview
              ? part === "settings"
                ? "Phần 2/2 · Chọn chất lượng, phụ đề và nhạc nền cho video, rồi duyệt."
                : "Phần 1/2 · Đọc và sửa lời thoại. Chưa tốn chi phí nào."
              : "Đang kiểm tra kịch bản của bạn."
        }
      >
        {isAwaitingReview && project && part === "settings" ? (
          <ProductionSettingsPanel
            key={project.project_id}
            project={project}
            stages={["render", "merge"]}
            hint="Dùng ở bước Dựng hình và Ghép video. Nếu một trong hai bước lỗi, bạn sửa được ngay tại đó rồi thử lại."
          />
        ) : (isAwaitingReview || (reviewingPast && viewStep === FLOW_REVIEW)) && project ? (
          /*
            Two columns while there is an outline to review: it can run to
            dozens of lines, and stacking it above the tracker would push
            status far down a wall of text. The tracker moves to a sticky side
            column instead of disappearing.
          */
          <div className={styles.layout}>
            <OutlineReview project={project} outline={outline} />
            <div className={styles.tracker}>
              <ProgressTracker
                progressState={displayProgressState}
                steps={VALIDATE_STEPS}
                stepNumbers={VALIDATE_SUBSTEP_NUMBERS}
                isFailed={isFailed}
                allDone={reviewingPast}
              />
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
              stepNumbers={VALIDATE_SUBSTEP_NUMBERS}
              isFailed={isFailed}
              allDone={reviewingPast}
            />
          </>
        )}
      </AppShell>

      {isAwaitingReview && project && !viewOnly && (
        part === "review" ? (
          <WizardNav
            allowWhenLocked
            hint="Đọc lại lời thoại, bấm vào một câu để sửa. Xong thì sang cài đặt xuất video."
            onBack={() => void outline.reject()}
            backLabel="Quay lại sửa script"
            backDisabled={outline.busy}
            backTestId="outline-reject"
            onNext={() => setSearch({ part: "settings" })}
            nextLabel="Tiếp: cài đặt xuất video"
            nextDisabled={outline.busy || outline.editing !== null}
            nextTestId="review-next"
          />
        ) : (
          <WizardNav
            allowWhenLocked
            hint={outline.error ?? "Bấm duyệt là hệ thống bắt đầu tạo giọng đọc, dựng hình và ghép video."}
            isBlocked={!!outline.error}
            onBack={() => setSearch({})}
            backLabel="Quay lại duyệt lời thoại"
            backTestId="review-back"
            onNext={() => void outline.approve()}
            nextLabel="Duyệt và bắt đầu tạo video"
            nextDisabled={outline.busy}
            nextTestId="outline-approve"
          />
        )
      )}
    </div>
  );
}
