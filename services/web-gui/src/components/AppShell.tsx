import type { ReactNode } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ThemeToggle } from "./ThemeToggle";
import styles from "./AppShell.module.css";

/*
  The whole journey, one pill per screen the Creator actually passes through.
  The old three-step version compressed all of the work into "Soạn nội dung"
  and gave the two automatic phases equal billing, so it described the plumbing
  rather than the path.
*/
/*
  Chỉ 3 mục đầu là đích nhảy được — xem BACKTRACKABLE_STEPS. Các mục sau thuộc
  về một project đã tồn tại, nên đường của chúng cần :id mà thanh này không
  có; chúng ở đây để chỉ chỗ, không để bấm.
*/
const STEP_ROUTES = [
  "/",
  "/create/script/settings",
  "/create/script/outline",
  "/projects/:id/validate",
  "/projects/:id/render",
  "/projects/:id/result",
  "/projects/:id/publish",
];

/*
  CR-031 — bảy bước. "Cấu hình" (bước 2) gộp ngôn ngữ/engine/cách làm với giọng đọc/hình ảnh và đứng trước "Script" (chuỗi 1a-1b-1c); bước "Xem lại" đã bỏ, nút cuối của 1c chạy thẳng saga. "Script" cũ gộp cả tình huống, cấu hình ngôn ngữ/engine/
  cách làm và chuỗi 1a-1b-1c vào một pill duy nhất; tách "Ý tưởng" (chọn tình
  huống, nhập chủ đề) ra làm bước riêng để thanh tiến trình phản ánh đúng
  màn hình Creator đang đứng, thay vì gộp hai việc khác hẳn nhau (chọn ý
  tưởng vs. soạn script) vào một mục. "Xử lý" cũ cũng từng gộp hai việc rất
  khác nhau vào một màn: chạy thử kịch bản (vài giây, miễn phí, sửa được) và
  sản xuất thật (nhiều phút, tốn TTS/render, không dừng được) — tách thành
  "Validate" rồi "Xử lý" để cổng duyệt dàn ý nằm đúng ranh giới đó.
*/
const STEP_LABELS = [
  "Ý tưởng",
  "Cấu hình",
  "Script",
  "Validate",
  "Xử lý",
  "Kết quả",
  "Đăng",
] as const;

/** Số bước đầu tiên có URL không cần project id, nên nhảy ngược về được. */
const BACKTRACKABLE_STEPS = 3;

interface AppShellProps {
  currentStep?: 1 | 2 | 3 | 4 | 5 | 6 | 7;
  title: string;
  subtitle: string;
  wide?: boolean;
  /** Page-specific action rendered at the right of the top bar. */
  headerAction?: ReactNode;
  children: ReactNode;
}

function CheckIcon() {
  return (
    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

export function AppShell({ currentStep, title, subtitle, wide, headerAction, children }: AppShellProps) {
  const navigate = useNavigate();

  const handleStepClick = (stepNumber: number) => {
    // Only allow navigation to completed steps
    if (currentStep && stepNumber < currentStep && stepNumber <= BACKTRACKABLE_STEPS) {
      navigate(STEP_ROUTES[stepNumber - 1]);
    }
  };

  return (
    <div className={styles.stage}>
      <div className={`${styles.blob} ${styles.blob1}`} />
      <div className={`${styles.blob} ${styles.blob2}`} />
      <div className={`${styles.blob} ${styles.blob3}`} />

      <div className={styles.content}>
        <div className={styles.topbar}>
          <Link to="/" className={styles.logo} style={{ textDecoration: "none" }}>
            <img
              className={styles.logoMark}
              src="/icon-192.png"
              alt=""
              width={34}
              height={34}
              /* Trang trí thuần tuý: tên kênh đã nằm ngay cạnh ở logoName,
                 nên alt rỗng để trình đọc màn hình không đọc lặp hai lần. */
            />
            <div className={styles.logoName}>ConceptFlow</div>
          </Link>

          {currentStep && (
            <div className={styles.stepsPill}>
              {STEP_LABELS.map((label, index) => {
                const stepNumber = index + 1;
                const isActive = stepNumber === currentStep;
                const isDone = stepNumber < currentStep;
                const isClickable = isDone && stepNumber <= BACKTRACKABLE_STEPS;
                const className = [styles.stepItem, isActive ? styles.active : "", isDone ? styles.done : ""]
                  .filter(Boolean)
                  .join(" ");
                return (
                  <button
                    key={label}
                    type="button"
                    className={className}
                    onClick={() => handleStepClick(stepNumber)}
                    disabled={!isClickable}
                    aria-current={isActive ? "step" : undefined}
                    title={isClickable ? `Nhảy về ${label}` : undefined}
                    style={{ cursor: isClickable ? "pointer" : "default" }}
                  >
                    <span className={styles.stepNum}>{isDone ? <CheckIcon /> : stepNumber}</span>
                    {label}
                  </button>
                );
              })}
            </div>
          )}

          <div className={styles.headerActions}>
            <ThemeToggle />
            {headerAction}
            <Link to="/videos" className={styles.headerLink}>
              Danh sách video
            </Link>
            {/* CR-025 — admin entry to edit the authoring pipeline's prompt
                wording, deliberately outside the Creator's step pill above. */}
            <Link to="/settings/prompts" className={styles.headerLink}>
              Cài đặt prompt
            </Link>
          </div>
        </div>

        <div className={`${styles.mainCol} ${wide ? styles.mainColWide : ""}`}>
          <div className={styles.heading}>
            <h1>{title}</h1>
            <p>{subtitle}</p>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
