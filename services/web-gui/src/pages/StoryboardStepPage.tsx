import { useCallback, useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { StoryboardState } from "../api/client";
import { AppShell } from "../components/AppShell";
import lockStyles from "../components/lock/lock.module.css";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import { StoryboardAssetsTab } from "../components/storyboard/StoryboardAssetsTab";
import { StoryboardReviewTab } from "../components/storyboard/StoryboardReviewTab";
import { StoryboardRunCard } from "../components/storyboard/StoryboardRunCard";
import { StoryboardRunningBanner } from "../components/storyboard/StoryboardRunningBanner";
import { StoryboardShotsTab } from "../components/storyboard/StoryboardShotsTab";
import {
  BLOCKER_PREVIEW,
  blockersSummary,
  blockersTooltip,
  keyTarget,
} from "../components/storyboard/storyboardModel";
import styles from "../components/storyboard/storyboard.module.css";
import { Button, Card } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useRecentProject } from "../hooks/useRecentProject";
import { useStoryboard, type StoryboardNotOpen } from "../hooks/useStoryboard";
import {
  VIEW_STORYBOARD,
  comingRoute,
  lockRoute,
  readOnlyReason,
  visualConceptRoute,
  visualPlanRoute,
  visualSequenceRoute,
} from "../utils/flow";

const SUBTITLE =
  "Tài sản dùng lại và các shot của từng unit: cái gì ở đâu trong khung, chữ trên màn hình, khung hình vẽ sẵn.";

const ROLE =
  "Mỗi unit của chuỗi hình thành những shot nào, với những tài sản nào? Không đổi thứ tự unit của bước 9 và 10, không dựng chuyển động chi tiết.";

type Tab = "assets" | "shots" | "review";

/** What each closed gate says and where it leads. */
const GATES: Record<
  Exclude<StoryboardNotOpen, null>,
  { text: string; back: string; route: (id: string) => string }
> = {
  script: { text: "Chốt kịch bản ở bước 7 trước.", back: "← Rà soát và chốt", route: lockRoute },
  concept: {
    text: "Chốt Ngôn ngữ hình ảnh ở bước 8 trước.",
    back: "← Ngôn ngữ hình ảnh",
    route: visualConceptRoute,
  },
  plan: {
    text: "Chốt Kế hoạch hình ở bước 9 trước.",
    back: "← Kế hoạch hình",
    route: visualPlanRoute,
  },
  sequence: {
    text: "Chốt Chuỗi hình ở bước 10 trước.",
    back: "← Chuỗi hình",
    route: visualSequenceRoute,
  },
};

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(s: StoryboardState, running: boolean): StepStripProps {
  if (running) {
    return {
      tone: "run",
      label: "AI đang làm",
      message: "Tài sản, shot đã duyệt và trường bạn sửa tay giữ nguyên.",
    };
  }
  if (s.locked_version > 0 && s.draft.length === 0) {
    return { tone: "ok", label: `Đã chốt v${s.locked_version}`, message: "Tiếp theo: thiết kế tài nguyên." };
  }
  if (s.locked_version > 0) {
    return {
      tone: "wait",
      label: `v${s.locked_version} đã chốt · bản nháp v${s.next_version} có ${s.draft.length} thay đổi`,
      message: `Các bước sau vẫn dùng v${s.locked_version} tới khi bạn chốt bản mới.`,
    };
  }
  const assets = s.assets.items;
  if (assets.length === 0) {
    return { tone: "todo", label: "Chưa có tài sản", message: "Lập danh sách tài sản trước khi viết shot." };
  }
  if (assets.some((a) => a.status !== "approved")) {
    return {
      tone: "wait",
      label: `Tài sản ${s.assets.approved_count}/${assets.length} đã duyệt`,
      message: "Vẽ và duyệt mọi tài sản trước khi AI viết shot.",
    };
  }
  if (s.empty_units.length > 0) {
    return {
      tone: "wait",
      label: `Còn ${s.empty_units.length} unit chưa có shot`,
      message: "Cho AI viết shot các unit còn trống.",
    };
  }
  if (s.shots_approved < s.shots_total) {
    return {
      tone: "wait",
      label: `Chờ bạn duyệt · ${s.shots_approved}/${s.shots_total} shot`,
      message: "Shot đã duyệt giữ nguyên khi unit khác viết lại.",
    };
  }
  return {
    tone: "wait",
    label: `Đã duyệt ${s.shots_total}/${s.shots_total} shot`,
    message: "Xử lý các vấn đề ở tab Rà soát và chốt rồi chốt.",
  };
}

/**
 * Step 11 of the 23-step flow: the storyboard. Reading the locked sequence, it lists
 * the assets the video reuses (drawn once, approved one by one) and gives every
 * unit its shots, each with a frame drawn from the assets. The Creator has the AI
 * write them, edits, approves, has the whole storyboard reviewed and locks a
 * version the later steps read.
 */
export function StoryboardStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const projectId = params.id ?? draft.projectId;
  const storyboard = useStoryboard(projectId);
  const recent = useRecentProject(projectId);
  const [tab, setTab] = useState<Tab>("assets");
  // Ids of approved items the Creator opened; the rest fold to one line.
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
  const [focusShot, setFocusShot] = useState<string | null>(null);
  const onFocused = useCallback(() => setFocusShot(null), []);
  const toggle = useCallback((id: string, open: boolean) => {
    setExpanded((prev) => {
      if (prev.has(id) === open) return prev;
      const next = new Set(prev);
      if (open) next.add(id);
      else next.delete(id);
      return next;
    });
  }, []);

  if (!projectId) return <Navigate to="/" replace />;

  const state = storyboard.state;
  const readOnly = !flow.editable;
  const gate = storyboard.notOpen ? GATES[storyboard.notOpen] : null;
  // The video's title goes into the description of every asset that is put in the library.
  const projectName =
    (recent?.project_id === projectId ? recent.topic?.trim() : "") ||
    draft.authoringTopic.trim() ||
    projectId.slice(0, 8);

  /** Opens what a finding or a pending unit points to, on the tab that holds it. */
  function open(key: string) {
    const target = keyTarget(key);
    if (target === "asset") {
      setTab("assets");
      toggle(key, true);
      window.setTimeout(
        () => document.getElementById(`sb-asset-${key}`)?.scrollIntoView({ behavior: "smooth", block: "start" }),
        50,
      );
      return;
    }
    setTab("shots");
    if (target === "shot") {
      setFocusShot(key);
      return;
    }
    const beat = state?.beats.find((b) => b.units.some((u) => u.unit_id === key));
    if (beat) toggle(beat.key, true);
    toggle(key, true);
    window.setTimeout(
      () => document.getElementById(`sb-unit-${key}`)?.scrollIntoView({ behavior: "smooth", block: "start" }),
      50,
    );
  }

  return (
    <div data-testid="storyboard-page">
      <AppShell
        viewStep={VIEW_STORYBOARD}
        wide
        title="Storyboard"
        subtitle={SUBTITLE}
        role={ROLE}
        strip={state ? stripOf(state, storyboard.running !== null) : undefined}
      >
        {readOnly && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="sb-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {gate && (
          <>
            <p className={researchStyles.hint} data-testid="sb-not-open">
              {gate.text}
            </p>
            <div className={researchStyles.actions}>
              <Button
                variant="ghost"
                onClick={() => navigate(gate.route(projectId))}
                data-testid="sb-go-back"
              >
                {gate.back}
              </Button>
            </div>
          </>
        )}
        {!state && !gate && storyboard.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="sb-load-error">
              Không tải được bước Storyboard: {storyboard.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={() => void storyboard.reload()} data-testid="sb-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !gate && !storyboard.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && !gate && (
          <div className={researchStyles.body}>
            <StoryboardRunningBanner storyboard={storyboard} />
            <Card data-testid="sb-head-card">
              <div className={researchStyles.body}>
                <div data-testid="sb-input">
                  <b>Đầu vào:</b> Chuỗi hình v{state.input.sequence_version} · {state.input.units_count}{" "}
                  unit · Kế hoạch hình v{state.input.plan_version} · Ngôn ngữ hình ảnh v
                  {state.input.concept_version} · Kịch bản v{state.input.script_version}
                  {state.kit.ref_id ? ` · Kit ${state.kit.ref_id}` : ""}
                </div>
                {state.kit.missing && (
                  <div className={researchStyles.banner} role="status" data-testid="sb-kit-missing">
                    ⚠ {state.kit.message || "Dự án chưa chọn phong cách có kit vẽ ở bước 8."} Chưa vẽ
                    được tài sản và khung.
                  </div>
                )}
                {state.input.outdated.length > 0 && (
                  <div className={researchStyles.banner} role="status" data-testid="sb-outdated">
                    ⚠ Chuỗi hình chưa theo bản chốt mới nhất: {state.input.outdated.join("; ")}. Vẫn
                    sửa và duyệt được, nhưng chỉ chốt được sau khi bước 10 chốt lại.
                  </div>
                )}
              </div>
            </Card>
            <StoryboardRunCard
              projectId={projectId}
              state={state}
              storyboard={storyboard}
              readOnly={readOnly}
            />
            {state.locked_version === 0 && state.blockers.length > 0 && (
              <div className={researchStyles.banner} role="status" data-testid="sb-blockers-banner">
                <strong>Chưa chốt được:</strong>
                <ul className={styles.blockerList}>
                  {state.blockers.slice(0, BLOCKER_PREVIEW).map((b, i) => (
                    <li key={`${i}-${b}`}>{b}</li>
                  ))}
                </ul>
                {state.blockers.length > BLOCKER_PREVIEW && (
                  <span data-testid="sb-blockers-more">
                    và {state.blockers.length - BLOCKER_PREVIEW} điều kiện khác (xem đủ ở tab Rà soát và
                    chốt)
                  </span>
                )}
              </div>
            )}
            <div className={lockStyles.tabs} role="tablist" aria-label="Bước 11">
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "assets"}
                onClick={() => setTab("assets")}
                data-testid="sb-tab-assets"
              >
                1 · Tài sản
                <span className={lockStyles.tabCount}>
                  {state.assets.approved_count}/{state.assets.items.length} đã duyệt
                </span>
              </button>
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "shots"}
                onClick={() => setTab("shots")}
                data-testid="sb-tab-shots"
              >
                2 · Shot
                <span className={lockStyles.tabCount}>
                  {state.shots_approved}/{state.shots_total} đã duyệt
                </span>
              </button>
              <button
                type="button"
                role="tab"
                className={lockStyles.tab}
                aria-selected={tab === "review"}
                onClick={() => setTab("review")}
                data-testid="sb-tab-review"
              >
                3 · Rà soát và chốt
                <span className={lockStyles.tabCount}>{state.review.open_count} vấn đề</span>
              </button>
            </div>
            {tab === "assets" && (
              <StoryboardAssetsTab
                projectId={projectId}
                projectName={projectName}
                state={state}
                storyboard={storyboard}
                readOnly={readOnly}
                expanded={expanded}
                onToggle={toggle}
              />
            )}
            {tab === "shots" && (
              <StoryboardShotsTab
                projectId={projectId}
                state={state}
                storyboard={storyboard}
                readOnly={readOnly}
                expanded={expanded}
                onToggle={toggle}
                focusShot={focusShot}
                onFocused={onFocused}
              />
            )}
            {tab === "review" && (
              <StoryboardReviewTab
                projectId={projectId}
                state={state}
                storyboard={storyboard}
                readOnly={readOnly}
                onOpen={open}
              />
            )}
            {storyboard.actionError && (
              <p className={researchStyles.error} role="alert" data-testid="sb-error">
                {storyboard.actionError}
              </p>
            )}
          </div>
        )}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          state && state.locked_version > 0
            ? `Đã chốt v${state.locked_version}. Sang Thiết kế tài nguyên.`
            : state && state.blockers.length > 0
              ? (
                  <span data-testid="sb-hint-blockers" title={blockersTooltip(state.blockers)}>
                    {blockersSummary(state.blockers)}
                  </span>
                )
              : "Chốt storyboard để sang bước sau."
        }
        onBack={() => navigate(visualSequenceRoute(projectId))}
        onNext={() => navigate(comingRoute(VIEW_STORYBOARD + 1, projectId))}
        nextDisabled={!state || state.locked_version === 0}
        nextLabel="Sang Thiết kế tài nguyên"
        nextTestId="sb-next"
      />
    </div>
  );
}
