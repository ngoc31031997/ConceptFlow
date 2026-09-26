import type { ReactNode } from "react";
import { useContext, useState } from "react";
import { Link, NavLink, useParams } from "react-router-dom";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useStepNav } from "../hooks/useStepNav";
import { StepRail, readRailCollapsed, writeRailCollapsed } from "./StepRail";
import { StatusStrip } from "./StatusStrip";
import { ReadOnlyContext } from "../context/ReadOnlyContext";
import { readOnlyReason } from "../utils/flow";
import { ThemeToggle } from "./ThemeToggle";
import { AccentPicker } from "./AccentPicker";
import { ProjectErrorBadge } from "./ProjectErrorBadge";
import styles from "./AppShell.module.css";

/*
  Hành trình 13 bước (xem utils/flow.ts): mỗi pill là một bước Creator đi qua.
  Bấm được mọi bước đã tới (tới bước xa nhất server ghi nhận) — để XEM lại. Có
  sửa được hay không là chuyện của server: chỉ draft hoặc dự án lỗi mới sửa
  được; còn lại mở ở chế độ chỉ đọc (xem `readOnly` bên dưới).
*/
interface AppShellProps {
  /** Bước trong flow 13 bước mà màn này đang hiển thị (1-13). */
  currentStep?: number;
  title: string;
  subtitle: string;
  wide?: boolean;
  /** Page-specific action rendered at the right of the top bar. */
  headerAction?: ReactNode;
  children: ReactNode;
}

const navCls = ({ isActive }: { isActive: boolean }) =>
  [styles.sideLink, isActive ? styles.sideLinkActive : ""].filter(Boolean).join(" ");

export function AppShell({ currentStep, title, subtitle, wide, headerAction, children }: AppShellProps) {
  const draft = useContext(ProjectDraftContext);
  const routeProjectId = useParams().id;
  // Tên project (chủ đề) từ bước 2 trở đi, để biết đang theo dõi project nào.
  // Trên màn có :id thì chỉ hiện khi bản nháp đang nạp đúng project đó.
  const projectName =
    currentStep && currentStep >= 2 && draft.projectId && (!routeProjectId || routeProjectId === draft.projectId)
      ? draft.authoringTopic.trim()
      : "";

  const flow = useProjectFlow();
  const nav = useStepNav(currentStep);
  const [railCollapsed, setRailCollapsed] = useState<boolean>(readRailCollapsed);
  // Chỉ xem: server sẽ từ chối sửa, nên báo trước thay vì để gõ xong mới lỗi.
  // Áp cho các màn soạn (1-5); màn 6-13 tự quản lý hành động của chúng.
  const readOnly = !!currentStep && currentStep <= 5 && nav.hasProject && !flow.editable;
  // Menu dọc thứ hai chỉ có nghĩa khi đã có một project để đặt vào 13 bước.
  const showRail = !!currentStep && nav.hasProject;

  const toggleRail = () => {
    setRailCollapsed((c) => {
      writeRailCollapsed(!c);
      return !c;
    });
  };

  return (
    <div className={styles.stage}>
      <div className={`${styles.blob} ${styles.blob1}`} />
      <div className={`${styles.blob} ${styles.blob2}`} />
      <div className={`${styles.blob} ${styles.blob3}`} />

      <aside className={styles.sidebar}>
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

        <nav className={styles.sideNav}>
          <NavLink to="/" end className={navCls}>Tạo video mới</NavLink>
          <NavLink to="/videos" className={navCls}>Danh sách video</NavLink>
          <NavLink to="/journal" className={navCls}>Nhật ký</NavLink>
          {/* CR-025 — admin entry to edit the authoring pipeline's prompt wording. */}
          <NavLink to="/settings/prompts" className={navCls}>Cài đặt prompt</NavLink>
        </nav>
      </aside>

      {showRail && currentStep && (
        <StepRail
          currentStep={currentStep}
          title={projectName || flow.projectId.slice(0, 8)}
          collapsed={railCollapsed}
          onToggle={toggleRail}
        />
      )}

      <div
        className={[
          styles.content,
          showRail ? (railCollapsed ? styles.contentRailCollapsed : styles.contentRail) : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <div className={styles.topbar}>
          <div className={styles.headerActions}>
            <AccentPicker />
            <ThemeToggle />
            {headerAction}
          </div>
        </div>

        <div className={`${styles.mainCol} ${wide ? styles.mainColWide : ""}`}>
          <div className={styles.heading}>
            <div className={styles.projectRow}>
              {projectName && (
                <div className={styles.projectChip} title={projectName} data-testid="project-name-chip">
                  <span className={styles.projectChipLabel}>Project</span>
                  <span className={styles.projectChipName}>{projectName}</span>
                </div>
              )}
              {currentStep && currentStep >= 2 && (routeProjectId || draft.projectId) && (
                <ProjectErrorBadge projectId={routeProjectId || draft.projectId} />
              )}
            </div>
            <h1>{title}</h1>
            <p>{subtitle}</p>
          </div>
          {showRail && currentStep && <StatusStrip currentStep={currentStep} />}
          {readOnly && flow.status && (
            <p className={styles.readOnlyBanner} role="status" data-testid="read-only-banner">
              {readOnlyReason(flow.status)}
            </p>
          )}
          {/* fieldset disabled khoá mọi ô nhập/nút bên trong mà không phải sửa
              từng màn; điều hướng (thanh bước, tab) nằm ngoài nên vẫn bấm được. */}
          {readOnly ? (
            <ReadOnlyContext.Provider value={true}>
              <fieldset disabled className={styles.readOnlyFieldset} data-testid="read-only-fieldset">
                {children}
              </fieldset>
            </ReadOnlyContext.Provider>
          ) : (
            children
          )}
        </div>
      </div>
    </div>
  );
}
