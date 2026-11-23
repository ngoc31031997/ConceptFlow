import { useCallback, useContext, useState } from "react";
import { Navigate, useNavigate, useParams } from "react-router-dom";
import type { StoryboardState } from "../api/client";
import { AiRunsProvider } from "../components/airun/AiRunsContext";
import { AppShell } from "../components/AppShell";
import researchStyles from "../components/research/research.module.css";
import type { StepStripProps } from "../components/StepStrip";
import {
  StoryboardAssetDrawSection,
  StoryboardAssetListSection,
} from "../components/storyboard/StoryboardAssetSections";
import { StoryboardReviewTab } from "../components/storyboard/StoryboardReviewTab";
import { StoryboardRunCard } from "../components/storyboard/StoryboardRunCard";
import { StoryboardRunningBanner } from "../components/storyboard/StoryboardRunningBanner";
import { StoryboardSection, type SectionState } from "../components/storyboard/StoryboardSection";
import { StoryboardShotsTab } from "../components/storyboard/StoryboardShotsTab";
import {
  blockersSummary,
  blockersTooltip,
  keyTarget,
  shotsDone,
  writeBlocker,
} from "../components/storyboard/storyboardModel";
import { Button, Card } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useRecentProject } from "../hooks/useRecentProject";
import {
  useStoryboard,
  type StoryboardControls,
  type StoryboardNotOpen,
} from "../hooks/useStoryboard";
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

/** The keys of the four sections in the Creator's set of opened items. */
const SECTION_KEYS = ["section-1", "section-2", "section-3", "section-4"] as const;

/** Where each of the four sections stands, top to bottom, and the line each one shows under its head. */
interface SectionPlan {
  states: SectionState[];
  notes: (string | undefined)[];
}

/**
 * Works out the four sections from the state. A section is done when its work
 * is approved; it is locked while it has nothing yet and the section above is
 * not done; a section that has content while one above waits stays open with a
 * note. The first section with work left is current.
 */
function sectionsOf(s: StoryboardState): SectionPlan {
  const items = s.assets.items;
  const listDone = s.assets.approved;
  const drawDone = listDone && items.length > 0 && items.every((a) => a.status === "approved");
  const shotsAllDone = shotsDone(s);
  const lockDone = s.locked_version > 0 && s.draft.length === 0;
  const done = [listDone, drawDone, shotsAllDone, lockDone];
  const reachable = [
    true,
    listDone || items.some((a) => a.status !== "planned"),
    drawDone || s.shots_total > 0,
    shotsAllDone || s.locked_version > 0 || s.review.rounds > 0,
  ];
  const lockedWhy = [
    "",
    "Mở khi danh sách tài sản ở khối 1 đã duyệt.",
    "Mở khi mọi tài sản ở khối 2 đã vẽ và duyệt.",
    "Mở khi mọi unit ở khối 3 đã có shot và mọi shot đã duyệt.",
  ];
  const waitingWhy = [
    "",
    "Danh sách tài sản ở khối 1 đang chờ duyệt lại; duyệt lại để vẽ tiếp.",
    `${writeBlocker(s) || "Tài sản ở khối 2 chưa duyệt hết"}. Shot mới chỉ viết được khi khối 2 xong.`,
    "Khối 3 chưa xong: còn unit chưa có shot hoặc shot chưa duyệt.",
  ];
  let currentSet = false;
  const states: SectionState[] = done.map((d, i) => {
    if (d) return "done";
    if (!reachable[i]) return "locked";
    if (!currentSet) {
      currentSet = true;
      return "current";
    }
    return "open";
  });
  const notes = states.map((st, i) => {
    if (st === "locked") return lockedWhy[i];
    if (st === "done" || i === 0 || done[i - 1]) return undefined;
    return waitingWhy[i];
  });
  return { states, notes };
}

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
    return {
      tone: "ok",
      label: `Đã chốt v${s.locked_version}`,
      message: "Tiếp theo: thiết kế tài nguyên.",
    };
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
    return {
      tone: "todo",
      label: "Chưa có tài sản",
      message: "Lập danh sách tài sản trước khi viết shot.",
    };
  }
  if (!s.assets.approved) {
    return {
      tone: "wait",
      label: "Danh sách tài sản chờ duyệt",
      message: "Viết chi tiết từng tài sản rồi duyệt danh sách để vẽ.",
    };
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
    message: "Xử lý các vấn đề ở khối Rà soát và chốt rồi chốt.",
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

  /** Opens what a finding or a pending unit points to, in the section that holds it. */
  function open(key: string) {
    const target = keyTarget(key);
    if (target === "asset") {
      toggle(SECTION_KEYS[1], true);
      toggle(key, true);
      window.setTimeout(
        () =>
          document
            .getElementById(`sb-asset-${key}`)
            ?.scrollIntoView({ behavior: "smooth", block: "start" }),
        50,
      );
      return;
    }
    toggle(SECTION_KEYS[2], true);
    if (target === "shot") {
      setFocusShot(key);
      return;
    }
    const beat = state?.beats.find((b) => b.units.some((u) => u.unit_id === key));
    if (beat) toggle(beat.key, true);
    toggle(key, true);
    window.setTimeout(
      () =>
        document
          .getElementById(`sb-unit-${key}`)
          ?.scrollIntoView({ behavior: "smooth", block: "start" }),
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
              <Button
                variant="ghost"
                onClick={() => void storyboard.reload()}
                data-testid="sb-reload"
              >
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
                  <b>Đầu vào:</b> Chuỗi hình v{state.input.sequence_version} ·{" "}
                  {state.input.units_count} unit · Kế hoạch hình v{state.input.plan_version} · Ngôn
                  ngữ hình ảnh v{state.input.concept_version} · Kịch bản v
                  {state.input.script_version}
                  {state.kit.ref_id ? ` · Kit ${state.kit.ref_id}` : ""}
                </div>
                {state.kit.missing && (
                  <div className={researchStyles.banner} role="status" data-testid="sb-kit-missing">
                    ⚠ {state.kit.message || "Dự án chưa chọn phong cách có kit vẽ ở bước 8."} Chưa
                    vẽ được tài sản và khung.
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
            <AiRunsProvider projectId={projectId} step="visual_storyboard">
              <StoryboardSections
                projectId={projectId}
                projectName={projectName}
                state={state}
                storyboard={storyboard}
                readOnly={readOnly}
                expanded={expanded}
                onToggle={toggle}
                focusShot={focusShot}
                onFocused={onFocused}
                onOpen={open}
              />
            </AiRunsProvider>
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
          state && state.locked_version > 0 ? (
            `Đã chốt v${state.locked_version}. Sang Thiết kế tài nguyên.`
          ) : state && state.blockers.length > 0 ? (
            <span data-testid="sb-hint-blockers" title={blockersTooltip(state.blockers)}>
              {blockersSummary(state.blockers)}
            </span>
          ) : (
            "Chốt storyboard để sang bước sau."
          )
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

interface StoryboardSectionsProps {
  projectId: string;
  projectName: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: ReadonlySet<string>;
  onToggle: (id: string, open: boolean) => void;
  focusShot: string | null;
  onFocused: () => void;
  onOpen: (key: string) => void;
}

/**
 * The four sections of step 11 in the order the work is done: the asset list,
 * drawing the assets, the shots of each unit, then the review and the lock.
 */
function StoryboardSections({
  projectId,
  projectName,
  state,
  storyboard,
  readOnly,
  expanded,
  onToggle,
  focusShot,
  onFocused,
  onOpen,
}: StoryboardSectionsProps) {
  const { states, notes } = sectionsOf(state);
  const items = state.assets.items;
  const chip = (ok: boolean, text: string, testId: string) => (
    <span
      className={`${researchStyles.chip} ${ok ? researchStyles.chipOk : researchStyles.chipWait}`}
      data-testid={testId}
    >
      {text}
    </span>
  );
  const section = (i: number) => ({
    number: i + 1,
    state: states[i],
    note: notes[i],
    expanded: expanded.has(SECTION_KEYS[i]),
    onToggle: (open: boolean) => onToggle(SECTION_KEYS[i], open),
  });

  return (
    <>
      <StoryboardSection
        {...section(0)}
        title="Danh sách tài sản cần vẽ"
        intro="AI đọc ngôn ngữ hình ảnh (bước 8) và kế hoạch hình (bước 9), liệt kê nhân vật, vật thể, phông nền sẽ xuất hiện, rồi viết chi tiết từng cái. Mỗi tài sản vẽ một lần, dùng lại ở nhiều shot. Duyệt danh sách để sang vẽ."
        chip={chip(
          state.assets.approved,
          items.length === 0
            ? "Chưa có"
            : state.assets.approved
              ? `Đã duyệt · ${items.length} tài sản`
              : `${state.assets.with_detail}/${items.length} có chi tiết`,
          "sb-chip-assets",
        )}
        testId="sb-section-assets"
      >
        <StoryboardAssetListSection
          projectId={projectId}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
        />
      </StoryboardSection>
      <StoryboardSection
        {...section(1)}
        title="Vẽ tài sản"
        intro="Vẽ từng tài sản của danh sách đã duyệt theo kit của phong cách dự án, rồi duyệt hình từng cái."
        chip={chip(
          states[1] === "done",
          `${state.assets.approved_count}/${items.length} đã duyệt`,
          "sb-chip-draw",
        )}
        testId="sb-section-draw"
      >
        <StoryboardAssetDrawSection
          projectId={projectId}
          projectName={projectName}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
          expanded={expanded}
          onToggle={onToggle}
        />
      </StoryboardSection>
      <StoryboardSection
        {...section(2)}
        title="Shot theo từng unit"
        intro="Mỗi unit của chuỗi hình thành 1 hoặc vài shot, viết lần lượt từ unit đầu; AI đọc shot của unit liền trước để giữ nhất quán. Khung hình ghép từ tài sản ở khối 2, không vẽ bằng AI."
        chip={chip(
          states[2] === "done",
          state.empty_units.length > 0
            ? `${state.shots_approved}/${state.shots_total} đã duyệt · ${state.empty_units.length} unit trống`
            : `${state.shots_approved}/${state.shots_total} đã duyệt`,
          "sb-chip-shots",
        )}
        testId="sb-section-shots"
      >
        <StoryboardRunCard
          projectId={projectId}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
        />
        <StoryboardShotsTab
          projectId={projectId}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
          expanded={expanded}
          onToggle={onToggle}
          focusShot={focusShot}
          onFocused={onFocused}
        />
      </StoryboardSection>
      <StoryboardSection
        {...section(3)}
        title="Rà soát và chốt"
        intro="Máy tự kiểm các điều kiện; AI rà soát khi bạn bấm. Đủ điều kiện thì chốt thành phiên bản cho bước sau."
        chip={chip(
          states[3] === "done",
          state.locked_version > 0
            ? `Đã chốt v${state.locked_version}`
            : `${state.review.open_count} vấn đề`,
          "sb-chip-review",
        )}
        testId="sb-section-review"
      >
        <StoryboardReviewTab
          projectId={projectId}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
          onOpen={onOpen}
        />
      </StoryboardSection>
    </>
  );
}
