import { useContext, useEffect, useRef, useState } from "react";
import { Button } from "./ui";
import { ConfirmModal } from "./ConfirmModal";
import { ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import {
  cancelAuthoringChain,
  getAuthoringChain,
  listProjectEvents,
  type ProjectEvent,
  getAuthoringState,
  startAuthoringChain,
  type AuthoringChainState,
  type AuthoringProgress,
  type AuthoringStep,
  type LlmStatus,
} from "../api/client";
import type { AuthoringMode } from "../context/ProjectDraftContext";
import { OperationProgressCard } from "./OperationProgressCard";
import { formatChars, formatClock } from "../lib/formatProgress";
import { useAuthoringProgress } from "../hooks/useAuthoringProgress";
import { useAuthoringRun, useAuthoringRunDispatch } from "../context/AuthoringRunContext";
import { usePresence } from "../hooks/usePresence";
import { AUTHORING_STEP_FLOW, FLOW_CODE, FLOW_STORY, flowTitle } from "../utils/flow";
import glass from "../styles/glass.module.css";
import styles from "./AuthoringModeBar.module.css";

/** Thứ tự các bước; "illustrations" chỉ có ở video Remotion. */
const ALL_STEPS: AuthoringStep[] = ["story", "storyboard", "illustrations", "code"];

/** Số của từng bước trong luồng 13 bước, đúng như menu bước đánh số. */
const STEP_FLOW = AUTHORING_STEP_FLOW;

/** "Bước 4 — Hình ảnh": tên một bước soạn đúng như thanh bước gọi nó. */
function stepTitle(step: AuthoringStep): string {
  return flowTitle(STEP_FLOW[step]);
}

/** "3–6": dải bước của chuỗi, theo số trên thanh bước. */
const CHAIN_RANGE = `${FLOW_STORY}–${FLOW_CODE}`;


/** Tab của từng bước, để chuỗi AI tự đưa Creator theo đúng bước đang chạy. */
export const AUTHORING_STEP_PATHS: Record<AuthoringStep, string> = {
  story: "/create/script",
  storyboard: "/create/script/storyboard",
  illustrations: "/create/script/illustrations",
  code: "/create/script/code",
};

/** Chuỗi "chạy cả pipeline" của tab Kịch bản: Remotion có thêm bước Hình minh hoạ. */
export function authoringChainSteps(renderEngine: string): AuthoringStep[] {
  return renderEngine === "remotion" ? ["story", "storyboard", "illustrations", "code"] : ["story", "storyboard", "code"];
}

/** Kết cục của một chuỗi đã xong còn hiện bao lâu (xem `outcome` bên dưới). */
const OUTCOME_TTL_MS = 30 * 60 * 1000;

/** Kết quả lượt chạy gần nhất của từng bước, đọc từ nhật ký của dự án. */
interface LastRun {
  state: ProjectEvent["run_state"];
  durationMs: number;
  chars: number;
  tokens: number;
}

function lastRuns(events: ProjectEvent[]): Partial<Record<AuthoringStep, LastRun>> {
  const byFlow = Object.fromEntries(ALL_STEPS.map((st) => [STEP_FLOW[st], st])) as Record<number, AuthoringStep>;
  const out: Partial<Record<AuthoringStep, LastRun>> = {};
  for (const e of [...events].sort((a, b) => a.id - b.id)) {
    // Illustrations journals under its own flow number (6); legacy rows carry
    // Code's flow number (5) with source "illustrations", so the source decides.
    const step = e.source === "illustrations" ? "illustrations" : byFlow[e.flow_step];
    if ((e.source !== "authoring" && e.source !== "illustrations") || !step || e.run_state === "running") continue;
    out[step] = {
      state: e.run_state,
      durationMs: e.duration_ms ?? 0,
      chars: e.content_chars ?? 0,
      tokens: (e.prompt_tokens ?? 0) + (e.completion_tokens ?? 0),
    };
  }
  return out;
}

function formatMs(ms: number): string {
  return formatClock(ms / 1000);
}

function dismissKey(projectId: string): string {
  return `authoring-chain-dismissed:${projectId}`;
}

function readDismissed(projectId: string): string | null {
  try {
    return window.localStorage.getItem(dismissKey(projectId));
  } catch {
    return null;
  }
}

function writeDismissed(projectId: string, finishedAt: string): void {
  try {
    window.localStorage.setItem(dismissKey(projectId), finishedAt);
  } catch {
    /* per-viewer convenience only */
  }
}

interface AuthoringModeBarProps {
  /**
   * Trạng thái provider, do trang sở hữu (useLlmStatus) chứ không phải thẻ
   * này: trang cũng phải biết nó để quyết định bày ô prompt copy tay hay
   * không. Hai nơi tự hỏi riêng thì có lúc chúng trả lời khác nhau, và Creator
   * rơi vào trạng thái không có đường nào — chế độ AI đang chọn nhưng không có
   * nút chạy, mà ô prompt cũng đã bị ẩn. `null` là chưa biết.
   */
  llm: LlmStatus | null;
  mode: AuthoringMode;
  onModeChange: (mode: AuthoringMode) => void;
  projectId: string;
  /**
   * Chuỗi bước mà một lần bấm sẽ chạy, theo đúng thứ tự. Màn bước 3
   * truyền cả chuỗi (`authoringChainSteps`: story, storyboard, [illustrations],
   * code): Creator chỉ nhập chủ đề rồi bấm một lần, server chạy tuần tự, mỗi
   * bước đọc kết quả bước trước đã lưu. Màn 4/5/6 truyền đúng một bước, để
   * chạy lại riêng bước đó sau khi sửa tay.
   *
   * Để trống ở màn chọn tình huống: ở đó chưa có chủ đề, chưa có
   * artefact nào để sinh, nên chỉ có công tắc chế độ chứ không có nút chạy.
   * Chọn chế độ ngay từ đó là có ích vì nó lưu lên project và đi theo sang cả
   * ba tab.
   */
  steps?: AuthoringStep[];
  /** Bước này sinh ra cái gì, để câu chữ trên nút nói đúng việc nó làm. */
  what?: string;
  /** Kết quả từng bước, để trang nhét thẳng vào ô soạn thảo. */
  onGenerated?: (step: AuthoringStep, content: string) => void;
  /**
   * Chỉ với chuỗi nhiều bước: gọi khi server chuyển sang bước mới, và một lần
   * với `"done"` khi cả chuỗi xong thành công — trang dùng nó để chuyển tab.
   */
  onFollow?: (step: AuthoringStep | "done") => void;
  /**
   * Việc phải xong trước khi gọi — lưu chủ đề/kết quả bước trước lên server,
   * vì server render prompt từ dữ liệu của nó, không từ state trình duyệt.
   */
  beforeRun?: () => Promise<void>;
  /** Chặn nút chạy dù đã chọn chế độ AI — ví dụ chưa nhập chủ đề. */
  runDisabled?: boolean;
  runDisabledReason?: string;
  /**
   * Mặc định hiện cả chữ giải thích lẫn công tắc chế độ. `false` chỉ để lại
   * nút chạy + trạng thái/lỗi/tiến độ — PipelineSettingsBar dùng khi đang thu
   * gọn, để Creator bấm chạy luôn mà không phải mở "Đổi".
   */
  showSwitch?: boolean;
  /**
   * Nằm gọn trong thẻ của PipelineSettingsBar: các khối không tự vẽ thẻ riêng
   * mà chỉ là một phần ngăn cách bằng đường kẻ — một thẻ duy nhất cho cả
   * "đã chọn gì" lẫn "chạy".
   */
  embedded?: boolean;
}

const MODE_LABELS: Record<AuthoringMode, string> = {
  manual: "Tự làm với ChatGPT, Claude…",
  ai: "Để AI làm giúp",
};

/**
 * Cách làm **các bước soạn 3–6**, đặt ở đầu mỗi màn 3/4/5/6.
 *
 * Một lựa chọn cho toàn bộ pipeline, không phải một nút riêng mỗi màn:
 * Creator đã quyết định chạy script này bằng API thì không muốn quyết định
 * lại ở bước 4, 6. Lựa chọn nằm trong draft nên nó sống qua việc đổi màn và tải
 * lại trang, và mặc định là `manual`.
 *
 * Chế độ `manual` không bao giờ mất đi: nó là đường đi khi chưa có key, hết số
 * dư, provider sập, hoặc khi Creator muốn dùng ChatGPT/Claude/Gemini của mình.
 * Vì vậy khi máy chủ chưa cấu hình key, lựa chọn "Gọi API"
 * hiện ra ở trạng thái không chọn được kèm lý do, chứ không lẳng lặng biến mất
 * — Creator cần biết tính năng có tồn tại và thiếu gì để bật.
 *
 * Ở chế độ AI, màn bước 3 chạy cả chuỗi 3 → 6 trong một lần bấm (`steps`).
 * Chuỗi chạy ở client chứ không phải một endpoint mới, vì mỗi lượt gọi đã tự
 * lưu kết quả lên server rồi: bước sau render prompt từ đúng dữ liệu bước
 * trước vừa lưu. Đổi lại, khi một bước giữa chừng hỏng thì những bước đã xong
 * vẫn còn nguyên, và Creator chạy tiếp từ màn đang dở thay vì mất cả chuỗi.
 */
export function AuthoringModeBar({
  llm,
  mode,
  onModeChange,
  projectId,
  steps = [],
  what = "",
  onGenerated,
  onFollow,
  beforeRun,
  runDisabled,
  runDisabledReason,
  showSwitch = true,
  embedded = false,
}: AuthoringModeBarProps) {
  // Trạng thái "đang chạy" sống ở AuthoringRunContext, ngoài component này —
  // dùng chung cho cả các màn 3–6, để màn vừa mở thấy đúng một chuỗi đang
  // chạy dở ở màn khác thay vì tưởng mình rảnh và cho bấm chạy chồng lên.
  const run = useAuthoringRun();
  const dispatchRun = useAuthoringRunDispatch();
  const dispatchDraft = useContext(ProjectDraftDispatchContext);

  // Một lượt chạy ghi đè bước của nó và xóa các bước dựng trên nó (cả khi hỏng),
  // nên đọc lại cả ba từ server thay vì để bản nháp ở client lệch đi.
  async function syncFromServer(chainSteps: AuthoringStep[] = []) {
    try {
      const state = await getAuthoringState(projectId);
      dispatchDraft({
        type: "SYNC_AUTHORING",
        payload: { story: state.story, storyboard: state.storyboard, code: state.code },
      });
      // Nhét kết quả từng bước vào ô soạn thảo. Bước chưa chạy tới thì
      // server đã xóa nội dung, nên rỗng và bị bỏ qua. Chỉ nạp bước thuộc tab
      // này: chuỗi vừa xong có thể do tab khác chạy, và đẩy nội dung bước khác
      // vào ô của tab này (story vào ô storyboard/code) là lỗi từng xảy ra.
      const mine = steps.length > 0 ? chainSteps.filter((st) => steps.includes(st)) : chainSteps;
      for (const step of mine) {
        // The illustrations step writes no editor content: its result is the drawing list.
        if (step === "illustrations") continue;
        const content = step === "story" ? state.story : step === "storyboard" ? state.storyboard : state.code;
        if (content) onGenerated?.(step, content);
      }
    } catch {
      /* best-effort — lần mở lại sau vẫn đọc từ server */
    }
  }

  const [error, setError] = useState<string | null>(null);
  const [chain, setChain] = useState<AuthoringChainState | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [dismissedAt, setDismissedAt] = useState<string | null>(() => readDismissed(projectId));
  const [runs, setRuns] = useState<Partial<Record<AuthoringStep, LastRun>>>({});
  // Số đo từng bước đọc từ nhật ký: nạp lúc mở và mỗi khi một chuỗi vừa xong.
  const finishedAt = chain?.finished_at ?? null;
  useEffect(() => {
    if (!projectId) return;
    let cancelled = false;
    listProjectEvents(projectId)
      .then((rows) => {
        if (!cancelled && Array.isArray(rows)) setRuns(lastRuns(rows));
      })
      .catch(() => {
        /* the journal is a nicety: the bar works without it */
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, finishedAt]);
  // Chuỗi chạy ở SERVER, độc lập với trang này: đóng tab, tải lại, sang máy
  // khác thì nó vẫn chạy và lưu kết quả. Vì vậy trạng thái "đang chạy" và kết
  // cục đều đọc từ server (poll), không giữ trong trình duyệt — giữ ở đây thì
  // tải lại trang là mất, và Creator thấy một màn im lặng dù server đã xong.
  const starting = useRef(false);
  const handledFinish = useRef<string | null>(null);
  const runRef = useRef(run);
  runRef.current = run;
  const followRef = useRef(onFollow);
  followRef.current = onFollow;
  const lastFollowed = useRef<string | null>(null);
  const sawRunning = useRef(false);

  useEffect(() => {
    if (!projectId || !llm) return;
    let cancelled = false;
    const tick = async () => {
      let c: AuthoringChainState;
      try {
        c = await getAuthoringChain(projectId);
      } catch {
        return; // best-effort: chưa bật AI ở server, hoặc mạng chập chờn
      }
      if (cancelled) return;
      setChain(c);
      const cur = runRef.current;
      if (c.running) {
        if (!cur.running) dispatchRun({ type: "START", steps: c.steps });
        if (cur.currentIndex !== c.current_index) dispatchRun({ type: "PROGRESS", index: c.current_index });
        sawRunning.current = true;
        const at = c.steps[c.current_index];
        if (c.steps.length > 1 && at && lastFollowed.current !== at) {
          // Lần đầu thấy chuỗi đang chạy (mở/tải lại trang) chỉ ghi nhận, không
          // kéo Creator đi; chỉ những lần chuyển bước sau đó mới chuyển tab.
          if (lastFollowed.current !== null) followRef.current?.(at);
          lastFollowed.current = at;
        }
      } else if (cur.running && !starting.current) {
        dispatchRun({ type: "FINISH" });
      }
      // Kết cục mới: nạp kết quả về ô soạn thảo một lần.
      if (c.finished && c.finished_at && handledFinish.current !== c.finished_at) {
        handledFinish.current = c.finished_at;
        await syncFromServer(c.steps);
        // A chain that stopped for drawing review takes the Creator to
        // the review; one that ran through goes to its last tab.
        if (sawRunning.current && c.steps.length > 1 && !c.error && !c.cancelled) {
          followRef.current?.(c.waiting && c.waiting_step ? c.waiting_step : "done");
        }
      }
    };
    void tick();
    const id = window.setInterval(() => void tick(), 1500);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, llm !== null]);

  const activeStep = run.running && run.currentIndex >= 0 ? run.steps[run.currentIndex] ?? null : null;
  const live = useAuthoringProgress(projectId, activeStep);

  // Kết cục lần chạy gần nhất, còn hiện trong 30 phút hoặc tới khi Creator
  // đóng — đủ để thấy kết quả khi mở lại trang, mà không treo mãi một lỗi cũ.
  const outcome =
    chain?.finished && !chain.running && chain.finished_at && chain.finished_at !== dismissedAt &&
    Date.now() - new Date(chain.finished_at).getTime() < OUTCOME_TTL_MS
      ? chain
      : null;
  // Cảnh báo của bước Visual, không chặn: xem trước khi tốn
  // tiền cho bước Code. Giữ bản cuối để khối còn nội dung khi đang thu lại.
  // Shown on the screens they concern: the storyboard they are about and Code,
  // which they should be read before.
  const storyboardWarnings =
    steps[0] === "storyboard" || steps[0] === "code" ? (outcome?.warnings?.storyboard ?? []) : [];
  const shownWarnings = useRef<string[]>([]);
  if (storyboardWarnings.length > 0) shownWarnings.current = storyboardWarnings;
  const warningsPresence = usePresence(storyboardWarnings.length > 0 && !run.running);

  // Chưa biết trạng thái: chưa vẽ thẻ, để nó không nhấp nháy giữa hai hình
  // dạng ngay khi trang mở.
  if (!llm) return null;

  const aiMode = mode === "ai" && llm.enabled;
  const isChain = steps.length > 1;
  const canRun = steps.length > 0;
  const running = run.running;
  // Có chuỗi khác (tab khác) đang chạy, không phải chuỗi của chính nút này —
  // câu trạng thái phải nói rõ đang chờ cái gì, không chỉ "đang chạy" chung
  // chung khiến Creator tưởng máy đứng hình.
  const runningElsewhere = running && run.steps !== steps && run.steps.join() !== steps.join();
  const outcomeError = outcome?.error
    ? outcome.error + (outcome.error_step && outcome.steps.length > 1 ? ` (dừng ở ${stepTitle(outcome.error_step)})` : "")
    : null;
  const outcomeNote = outcome?.note ?? null;
  // The step this screen is for: a screen passes its own step first (step 3
  // passes the whole chain, which starts with "story").
  const ownStep: AuthoringStep | null = steps[0] ?? null;
  // The step the AI works on right now, when it is not this screen's.
  const runningStep = run.currentIndex >= 0 ? (run.steps[run.currentIndex] ?? null) : null;
  const runningOther = running && runningStep !== null && runningStep !== ownStep;
  const ownIndex = ownStep ? run.steps.indexOf(ownStep) : -1;
  // A chain error belongs to the screen of the step that failed; the others
  // say where it stopped and lead there.
  const errorElsewhere =
    !error && outcome?.error && outcome.error_step && outcome.steps.length > 1 && outcome.error_step !== ownStep
      ? outcome.error_step
      : null;
  const outcomeWaiting = outcome?.waiting ?? null;
  const shownError = error ?? (errorElsewhere ? null : outcomeError);

  async function handleCancel() {
    setConfirmingCancel(false);
    setCancelling(true);
    try {
      await cancelAuthoringChain(projectId);
    } catch {
      // 404: chuỗi vừa tự kết thúc — lượt poll kế tiếp sẽ cập nhật giao diện.
    } finally {
      setCancelling(false);
    }
  }

  function dismissOutcome() {
    if (!chain?.finished_at) return;
    setDismissedAt(chain.finished_at);
    writeDismissed(projectId, chain.finished_at);
    setError(null);
  }

  async function handleRun() {
    setError(null);
    starting.current = true;
    dispatchRun({ type: "START", steps });
    try {
      // Server render prompt từ dữ liệu của chính nó, nên những gì Creator vừa
      // gõ phải lên server trước, không thì prompt thiếu dữ liệu bước này cần.
      if (beforeRun) await beforeRun();
      await startAuthoringChain(projectId, steps);
      setDismissedAt(null);
      // Server đã ghi nhận chuỗi; từ đây poll là nguồn sự thật.
      const c = await getAuthoringChain(projectId).catch(() => null);
      if (c) setChain(c);
    } catch (err) {
      dispatchRun({ type: "FINISH" });
      setError(err instanceof Error ? err.message : "AI chưa chạy được. Bạn có thể thử lại hoặc chuyển sang cách tự làm.");
    } finally {
      starting.current = false;
    }
  }

  const runLabel = isChain
    ? `Chạy bằng AI các bước ${STEP_FLOW[steps[0]]}–${STEP_FLOW[steps[steps.length - 1]]}`
    : `Chạy ${what} bằng AI`;

  const modeHint = !llm.enabled
    ? llm.reason || "Chưa bật AI. Bạn có thể tự làm bằng cách sao chép prompt."
    : "AI viết sẵn nội dung vào ô soạn thảo để bạn chỉnh sửa. Có thể chạy từng bước hoặc cả chuỗi.";
  const showRunRow = aiMode && canRun;

  const cardCls = embedded ? `${styles.section} ${styles.reveal}` : `${glass.card} ${styles.card} ${styles.reveal}`;

  return (
    <div className={embedded ? styles.embedded : styles.stack} data-testid="authoring-mode-bar">
      {showSwitch && (
        <div className={`${glass.card} ${styles.card}`}>
          <div className={glass.cardTitle}>Cách làm các bước {CHAIN_RANGE}</div>
          <AuthoringModeSwitch llm={llm} mode={mode} onModeChange={onModeChange} disabled={running} />
        </div>
      )}

      {aiMode && !running && ALL_STEPS.some((st) => runs[st]) && (
        <div className={cardCls} data-testid="authoring-last-runs">
          <div className={glass.cardTitle}>Lần chạy AI gần nhất</div>
          <ul className={styles.lastRuns}>
            {ALL_STEPS.filter((st) => runs[st]).map((st) => {
              const r = runs[st] as LastRun;
              return (
                <li key={st} data-testid={`last-run-${st}`} data-state={r.state}>
                  <b>{stepTitle(st)}</b>
                  <span>{r.state === "done" ? "xong" : r.state === "failed" ? "lỗi" : r.state}</span>
                  <span>{formatMs(r.durationMs)}</span>
                  {r.chars > 0 && <span>{formatChars(r.chars)} ký tự</span>}
                  {r.tokens > 0 && <span>{r.tokens.toLocaleString("vi-VN")} token</span>}
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {(showRunRow || running) && (
        <div className={cardCls} data-testid="authoring-run-card">
          {showRunRow && (
            <>
              <div className={styles.run}>
                <Button
                  onClick={handleRun}
                  disabled={running || runDisabled}
                  data-testid={`run-with-ai-${steps[0]}`}
                  title={
                    runningElsewhere
                      ? "Một lượt chạy khác đang diễn ra, vui lòng chờ"
                      : runDisabled
                        ? runDisabledReason
                        : "Chạy bằng AI"
                  }
                >
                  {running && !runningOther ? "AI đang chạy…" : runLabel}
                </Button>
                {running && !runningOther && <span className={styles.spinner} role="status" aria-label="AI đang chạy" />}
                {running && !runningOther && (
                  <p className={styles.status} data-testid="run-with-ai-running">
                    {runningElsewhere
                      ? `Đang chạy ở ${
                          run.currentIndex >= 0 ? stepTitle(run.steps[run.currentIndex]) : "bước khác"
                        }. Vui lòng chờ hoàn tất.`
                      : run.currentIndex >= 0 && isChain
                        ? `Đang chạy ${stepTitle(steps[run.currentIndex])}. Có thể mất vài phút. Vui lòng không đóng trang.`
                        : "Có thể mất vài chục giây. Vui lòng không đóng trang."}
                  </p>
                )}
              </div>
              {!running && <p className={styles.hint}>{modeHint}</p>}
              {!running && runDisabled && runDisabledReason && (
                <p className={styles.status}>{runDisabledReason}</p>
              )}
              {shownError && (
                <p className={styles.error} data-testid="run-with-ai-error">
                  {shownError}
                </p>
              )}
              {errorElsewhere && !running && (
                <div className={styles.waitingRow} data-testid="run-with-ai-error-elsewhere">
                  <p className={styles.error}>Chuỗi AI dừng vì lỗi ở {stepTitle(errorElsewhere)}.</p>
                  {onFollow && (
                    <Button variant="ghost" onClick={() => onFollow(errorElsewhere)} data-testid="run-with-ai-open-error">
                      Mở {stepTitle(errorElsewhere)}
                    </Button>
                  )}
                </div>
              )}
              {outcomeNote && !error && (
                <p className={styles.status} data-testid="run-with-ai-note">
                  {outcomeNote}
                </p>
              )}
              {outcomeWaiting && !error && !running && (
                <div className={styles.waitingRow} data-testid="run-with-ai-waiting">
                  <p className={styles.status}>{outcomeWaiting}</p>
                  {onFollow && !(steps.length === 1 && steps[0] === "illustrations") && (
                    <Button variant="ghost" onClick={() => onFollow("illustrations")} data-testid="run-with-ai-open-review">
                      Mở bước Hình minh hoạ
                    </Button>
                  )}
                </div>
              )}
              {warningsPresence.mounted && (
                <div
                  className={`${styles.warnings} ${warningsPresence.closing ? styles.revealOut : styles.reveal}`}
                  data-testid="run-with-ai-storyboard-warnings"
                >
                  <p className={styles.warningsTitle}>
                    Cảnh báo ở bước Hình ảnh ({shownWarnings.current.length}) — không chặn, nên xem trước khi chạy Code:
                  </p>
                  <ul>
                    {shownWarnings.current.map((w, i) => (
                      <li key={i}>{w}</li>
                    ))}
                  </ul>
                </div>
              )}
              {(outcomeError || outcomeNote || outcomeWaiting || storyboardWarnings.length > 0) && !running && (
                <button type="button" className={styles.hint} onClick={dismissOutcome} data-testid="run-with-ai-dismiss">
                  Đóng thông báo
                </button>
              )}
              {outcome?.cancelled && !running && !error && (
                <p className={styles.status} data-testid="run-with-ai-cancelled">
                  Đã dừng lượt chạy AI. Bước đang dở không được lưu; các bước đã xong vẫn giữ nguyên.
                </p>
              )}
              {outcome && !outcome.cancelled && !outcomeError && !outcomeNote && !outcomeWaiting && !running && !error && (
                <p className={styles.status} data-testid="run-with-ai-done">
                  {outcome.steps.length === 1 && outcome.steps[0] === "illustrations"
                    ? "Hoàn tất. Mọi hình minh hoạ đã sẵn sàng — có thể chạy bước Code."
                    : "Hoàn tất. Kết quả đã được điền vào ô soạn thảo."}
                </p>
              )}
            </>
          )}

          {/* Mỗi màn chỉ hiện tiến độ của bước mình. Chuỗi 3 → 6 vẫn chạy liền
              và tự chuyển màn theo bước đang chạy (onFollow). Khi AI đang làm
              bước khác, màn này không trông như đang chạy (không vòng xoay,
              không nút Dừng): chỉ một dòng nói AI đang ở bước nào, bước mình
              đã xong hay chạy sau, và nút mở bước đang chạy — Dừng nằm ở đó. */}
          {running && steps.length > 0 && (
            <div className={styles.runPanel} data-testid="authoring-run-panel">
              {!runningOther && (
                <Button onClick={() => setConfirmingCancel(true)} disabled={cancelling} data-testid="run-with-ai-cancel">
                  {cancelling ? "Đang dừng…" : "Dừng"}
                </Button>
              )}
              <ConfirmModal
                isOpen={confirmingCancel}
                onClose={() => setConfirmingCancel(false)}
                onConfirm={handleCancel}
                title="Dừng lượt chạy AI?"
                message="Bước đang dở sẽ không được lưu; các bước đã xong vẫn giữ nguyên."
                confirmLabel="Dừng"
                cancelLabel="Tiếp tục chạy"
                isDangerous
              />
              {runningOther && runningStep ? (
                <div className={styles.waitingRow} data-testid="authoring-run-elsewhere">
                  <p className={styles.status}>
                    {ownIndex >= 0 && ownIndex < run.currentIndex
                      ? `${stepTitle(ownStep as AuthoringStep)} đã xong. AI đang chạy ${stepTitle(runningStep)}.`
                      : ownIndex > run.currentIndex
                        ? `AI đang chạy ${stepTitle(runningStep)}; bước này chạy sau.`
                        : `AI đang chạy ${stepTitle(runningStep)}.`}
                  </p>
                  {onFollow && (
                    <Button variant="ghost" onClick={() => onFollow(runningStep)} data-testid="authoring-run-open-current">
                      Mở {stepTitle(runningStep)}
                    </Button>
                  )}
                </div>
              ) : (
                <OperationProgressCard
                  subtitle={live?.running ? liveProgressText(live) : null}
                  {...liveCounts(live)}
                  testId="authoring-live-progress"
                />
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface AuthoringModeSwitchProps {
  llm: LlmStatus;
  mode: AuthoringMode;
  onModeChange: (mode: AuthoringMode) => void;
  disabled?: boolean;
}

/** Hai lựa chọn cách làm dạng thẻ — chỉ để chọn; nút chạy nằm ở thẻ riêng. */
export function AuthoringModeSwitch({ llm, mode, onModeChange, disabled }: AuthoringModeSwitchProps) {
  const aiMode = mode === "ai" && llm.enabled;
  const aiOff = !llm.enabled;
  return (
    <div className={styles.options} data-testid="authoring-mode-switch" role="radiogroup" aria-label={`Cách làm các bước ${CHAIN_RANGE}`}>
      <button
        type="button"
        role="radio"
        aria-checked={!aiMode}
        data-testid="authoring-mode-manual"
        className={`${styles.option} ${!aiMode ? styles.optionOn : ""}`}
        disabled={disabled}
        onClick={() => onModeChange("manual")}
      >
        <b>{MODE_LABELS.manual}</b>
        <span>Sao chép prompt, dán vào ChatGPT, Claude hoặc Gemini rồi dán kết quả về đây.</span>
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={aiMode}
        data-testid="authoring-mode-ai"
        className={`${styles.option} ${aiMode ? styles.optionOn : ""}`}
        disabled={aiOff || disabled}
        title={aiOff ? llm.reason || "Chưa bật AI" : undefined}
        onClick={() => onModeChange("ai")}
      >
        <b>{MODE_LABELS.ai}</b>
        <span>
          {aiOff
            ? llm.reason || "Chưa bật AI. Bạn có thể tự làm bằng cách sao chép prompt."
            : "AI viết sẵn nội dung để bạn chỉnh sửa."}
        </span>
      </button>
    </div>
  );
}

/**
 * Chỉ pha biết tổng mới có phần trăm: các lô của 1c và vòng sửa lỗi
 * biên dịch. Pha suy luận/viết không biết tổng nên thanh chạy không xác định.
 */
function liveCounts(p: AuthoringProgress | null): { done?: number; total?: number } {
  if (!p?.running) return {};
  if (p.phase === "chunks" && p.chunks_total) return { done: p.chunks_done ?? 0, total: p.chunks_total };
  if (p.phase === "repair" && p.repair_max) return { done: p.repair_round ?? 0, total: p.repair_max };
  if (p.phase === "draw" && p.drawings_total) return { done: p.drawings_done ?? 0, total: p.drawings_total };
  return {};
}

/** "Đang vẽ 3/7 hình · 1 lỗi · 4 hình dùng lại". */
function drawingText(p: AuthoringProgress): string {
  if (p.phase === "plan") return "Đang lập danh sách hình từ storyboard";
  const total = p.drawings_total ?? 0;
  if (total === 0) return `Không có hình nào cần vẽ${p.drawings_reused ? ` · ${p.drawings_reused} hình dùng lại` : ""}`;
  return [
    `Đang vẽ ${p.drawings_done ?? 0}/${total} hình`,
    p.drawings_failed ? `${p.drawings_failed} lỗi` : null,
    p.drawings_reused ? `${p.drawings_reused} hình dùng lại` : null,
  ]
    .filter(Boolean)
    .join(" · ");
}

/** Câu tiến độ từ luồng streaming: pha hiện tại, lượng chữ đã nhận, thời gian. */
function liveProgressText(p: AuthoringProgress): string {
  const time = formatClock(p.elapsed_seconds);
  if (p.phase === "layout" || p.phase === "cast") {
    return `Đang chuẩn bị bố cục… ${time}`;
  }
  // The illustrations step: planning, then drawing with counts.
  if (p.phase === "plan") return `${drawingText(p)}… ${time}`;
  if (p.phase === "draw") return `${drawingText(p)} · ${time}`;
  if (p.phase === "chunks") return `Đang viết code: ${p.chunks_done ?? 0}/${p.chunks_total ?? "?"} phần · ${time}`;
  if (p.phase === "merge") return `Đang ghép code… ${time}`;
  if (p.phase === "check") return `Đang kiểm tra code… ${time}`;
  if (p.phase === "repair") {
    return `Đang tự sửa lỗi (lần ${p.repair_round ?? "?"}/${p.repair_max ?? "?"}) · ${time}`;
  }
  if (p.phase === "writing") return `AI đang viết kết quả… ${formatChars(p.content_chars)} ký tự · ${time}`;
  if (p.phase === "reasoning") return `AI đang phân tích… ${formatChars(p.reasoning_chars)} ký tự · ${time}`;
  return `Đang chờ AI phản hồi… ${time}`;
}
