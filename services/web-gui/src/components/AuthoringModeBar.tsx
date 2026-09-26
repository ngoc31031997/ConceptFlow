import { useContext, useEffect, useRef, useState } from "react";
import { Button } from "./ui";
import { ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import {
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
import glass from "../styles/glass.module.css";
import styles from "./AuthoringModeBar.module.css";

/** Nhãn tiếng Việt của từng bước, để câu trạng thái nói đúng nó đang ở đâu. */
const STEP_LABELS: Record<AuthoringStep, string> = {
  story: "Kịch bản",
  storyboard: "Visual",
  code: "Code",
};

const ALL_STEPS: AuthoringStep[] = ["story", "storyboard", "code"];

/** Tab của từng bước, để chuỗi AI tự đưa Creator theo đúng bước đang chạy. */
export const AUTHORING_STEP_PATHS: Record<AuthoringStep, string> = {
  story: "/create/script",
  storyboard: "/create/script/storyboard",
  code: "/create/script/code",
};

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
  const byFlow: Record<number, AuthoringStep> = { 3: "story", 4: "storyboard", 5: "code" };
  const out: Partial<Record<AuthoringStep, LastRun>> = {};
  for (const e of [...events].sort((a, b) => a.id - b.id)) {
    const step = byFlow[e.flow_step];
    if (e.source !== "authoring" || !step || e.run_state === "running") continue;
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
  const s = Math.round(ms / 1000);
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
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
   * CR-030 — chuỗi bước mà một lần bấm sẽ chạy, theo đúng thứ tự. Tab 1a
   * truyền cả ba (`story`, `storyboard`, `code`): Creator chỉ nhập chủ đề rồi
   * bấm một lần, server chạy tuần tự, mỗi bước đọc kết quả bước trước đã lưu.
   * Tab 1b/1c truyền đúng một bước, để chạy lại riêng bước đó sau khi sửa tay.
   *
   * CR-031 — để trống ở màn chọn tình huống: ở đó chưa có chủ đề, chưa có
   * artefact nào để sinh, nên chỉ có công tắc chế độ chứ không có nút chạy.
   * Chọn chế độ ngay từ đó là có ích vì nó lưu lên project và đi theo sang cả
   * ba tab.
   */
  steps?: AuthoringStep[];
  /** Bước này sinh ra cái gì, để câu chữ trên nút nói đúng việc nó làm. */
  what?: string;
  /** Kết quả từng bước, để trang nhét thẳng vào ô soạn thảo (FR78.2). */
  onGenerated?: (step: AuthoringStep, content: string) => void;
  /**
   * Chỉ với chuỗi nhiều bước: gọi khi server chuyển sang bước mới, và một lần
   * với `"done"` khi cả chuỗi xong thành công — trang dùng nó để chuyển tab.
   */
  onFollow?: (step: AuthoringStep | "done") => void;
  /**
   * Việc phải xong trước khi gọi — lưu chủ đề/kết quả bước trước lên server,
   * vì server render prompt từ dữ liệu của nó, không từ state trình duyệt
   * (FR80.1).
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
}

const MODE_LABELS: Record<AuthoringMode, string> = {
  manual: "Copy prompt ra ngoài",
  ai: "Gọi API trực tiếp",
};

/**
 * CR-027 FR79 — cách làm **cả bước 3**, đặt ở đầu mỗi tab 1a/1b/1c.
 *
 * Một lựa chọn cho toàn bộ pipeline, không phải một nút riêng mỗi tab:
 * Creator đã quyết định chạy script này bằng API thì không muốn quyết định
 * lại ở 1b, 1c. Lựa chọn nằm trong draft nên nó sống qua việc đổi tab và tải
 * lại trang, và mặc định là `manual` — đúng cái mọi project vẫn làm trước
 * CR-027.
 *
 * Chế độ `manual` không bao giờ mất đi: nó là đường đi khi chưa có key, hết số
 * dư, provider sập, hoặc khi Creator muốn dùng ChatGPT/Claude/Gemini của mình
 * (FR77.4/FR83.2). Vì vậy khi máy chủ chưa cấu hình key, lựa chọn "Gọi API"
 * hiện ra ở trạng thái không chọn được kèm lý do, chứ không lẳng lặng biến mất
 * — Creator cần biết tính năng có tồn tại và thiếu gì để bật.
 *
 * CR-030 — ở chế độ AI, tab 1a chạy cả ba bước trong một lần bấm (`steps`).
 * Chuỗi chạy ở client chứ không phải một endpoint mới, vì mỗi lượt gọi đã tự
 * lưu kết quả lên server rồi: bước sau render prompt từ đúng dữ liệu bước
 * trước vừa lưu. Đổi lại, khi một bước giữa chừng hỏng thì những bước đã xong
 * vẫn còn nguyên, và Creator chạy tiếp từ tab đang dở thay vì mất cả chuỗi.
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
}: AuthoringModeBarProps) {
  // Trạng thái "đang chạy" sống ở AuthoringRunContext, ngoài component này —
  // dùng chung cho cả 3 tab 1a/1b/1c, để tab vừa mở thấy đúng một chuỗi đang
  // chạy dở ở tab khác thay vì tưởng mình rảnh và cho bấm chạy chồng lên.
  const run = useAuthoringRun();
  const dispatchRun = useAuthoringRunDispatch();
  const dispatchDraft = useContext(ProjectDraftDispatchContext);

  // Một lượt chạy ghi đè bước của nó và xoá các bước dựng trên nó (cả khi hỏng),
  // nên đọc lại cả ba từ server thay vì để bản nháp ở client lệch đi.
  async function syncFromServer(chainSteps: AuthoringStep[] = []) {
    try {
      const state = await getAuthoringState(projectId);
      dispatchDraft({
        type: "SYNC_AUTHORING",
        payload: { story: state.story, storyboard: state.storyboard, code: state.code },
      });
      // Nhét kết quả từng bước vào ô soạn thảo (FR78.2). Bước chưa chạy tới thì
      // server đã xoá nội dung, nên rỗng và bị bỏ qua. Chỉ nạp bước thuộc tab
      // này: chuỗi vừa xong có thể do tab khác chạy, và đẩy nội dung bước khác
      // vào ô của tab này (story vào ô storyboard/code) là lỗi từng xảy ra.
      const mine = steps.length > 0 ? chainSteps.filter((st) => steps.includes(st)) : chainSteps;
      for (const step of mine) {
        const content = step === "story" ? state.story : step === "storyboard" ? state.storyboard : state.code;
        if (content) onGenerated?.(step, content);
      }
    } catch {
      /* best-effort — lần mở lại sau vẫn đọc từ server */
    }
  }

  const [error, setError] = useState<string | null>(null);
  const [chain, setChain] = useState<AuthoringChainState | null>(null);
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
        if (sawRunning.current && c.steps.length > 1 && !c.error) followRef.current?.("done");
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

  // Kết cục lần chạy gần nhất, còn hiện trong 30 phút hoặc tới khi Creator
  // đóng — đủ để thấy kết quả khi mở lại trang, mà không treo mãi một lỗi cũ.
  const outcome =
    chain?.finished && !chain.running && chain.finished_at && chain.finished_at !== dismissedAt &&
    Date.now() - new Date(chain.finished_at).getTime() < OUTCOME_TTL_MS
      ? chain
      : null;
  const outcomeError = outcome?.error
    ? outcome.error + (outcome.error_step && outcome.steps.length > 1 ? ` (dừng ở ${STEP_LABELS[outcome.error_step]})` : "")
    : null;
  const outcomeNote = outcome?.note ?? null;
  const shownError = error ?? outcomeError;

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
      setError(err instanceof Error ? err.message : "Chạy bằng AI thất bại, hoặc chuyển về Copy prompt như cũ.");
    } finally {
      starting.current = false;
    }
  }

  const runLabel = isChain ? `Chạy Kịch bản → Visual → Code bằng AI` : `Chạy ${what} bằng AI`;

  const modeHint = !llm.enabled
    ? llm.reason || "Chưa cấu hình API key nên chỉ có đường copy tay."
    : `Áp dụng cho cả bước 3 (Kịch bản, Visual, Code), mỗi bước chạy riêng: hệ thống tự gọi ${llm.provider}, điền kết quả vào ô soạn thảo để bạn sửa. Chạy cả chuỗi thì tự chuyển tab theo bước đang chạy; không tự nộp render.`;
  const showRunRow = aiMode && canRun;

  return (
    <div className={styles.stack} data-testid="authoring-mode-bar">
      {showSwitch && (
        <div className={`${glass.card} ${styles.card}`}>
          <div className={glass.cardTitle}>Cách làm bước 3</div>
          <AuthoringModeSwitch llm={llm} mode={mode} onModeChange={onModeChange} disabled={running} />
        </div>
      )}

      {aiMode && !running && ALL_STEPS.some((st) => runs[st]) && (
        <div className={`${glass.card} ${styles.card}`} data-testid="authoring-last-runs">
          <div className={glass.cardTitle}>Lần chạy AI gần nhất</div>
          <ul className={styles.lastRuns}>
            {ALL_STEPS.filter((st) => runs[st]).map((st) => {
              const r = runs[st] as LastRun;
              return (
                <li key={st} data-testid={`last-run-${st}`} data-state={r.state}>
                  <b>{STEP_LABELS[st]}</b>
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
        <div className={`${glass.card} ${styles.card}`} data-testid="authoring-run-card">
          {showRunRow && (
            <>
              <div className={styles.run}>
                <Button
                  onClick={handleRun}
                  disabled={running || runDisabled}
                  data-testid={`run-with-ai-${steps[0]}`}
                  title={
                    runningElsewhere
                      ? "Một chuỗi khác đang chạy — chờ xong đã"
                      : runDisabled
                        ? runDisabledReason
                        : `Gọi trực tiếp ${llm.provider}`
                  }
                >
                  {running ? "AI đang chạy…" : runLabel}
                </Button>
                {running && <span className={styles.spinner} role="status" aria-label="AI đang chạy" />}
                {running && (
                  <p className={styles.status} data-testid="run-with-ai-running">
                    {runningElsewhere
                      ? `Đang chạy ở tab khác: ${
                          run.currentIndex >= 0 ? STEP_LABELS[run.steps[run.currentIndex]] : "..."
                        }. Chờ xong rồi mới chạy tiếp được.`
                      : run.currentIndex >= 0 && isChain
                        ? `Bước ${run.currentIndex + 1}/${steps.length} — ${STEP_LABELS[steps[run.currentIndex]]}. Có thể mất vài phút, đừng đóng trang.`
                        : "Có thể mất vài chục giây, đừng đóng trang."}
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
              {outcomeNote && !error && (
                <p className={styles.status} data-testid="run-with-ai-note">
                  {outcomeNote}
                </p>
              )}
              {(outcomeError || outcomeNote) && !running && (
                <button type="button" className={styles.hint} onClick={dismissOutcome} data-testid="run-with-ai-dismiss">
                  Đóng thông báo
                </button>
              )}
              {outcome && !outcomeError && !outcomeNote && !running && !error && (
                <p className={styles.status} data-testid="run-with-ai-done">
                  Lượt chạy gần nhất đã xong — kết quả đã nạp vào ô soạn thảo.
                </p>
              )}
            </>
          )}

          {/* Chạy cả chuỗi (1a → 1b → 1c) đụng đúng chỗ Creator từng bị lạc: bấm
              chạy ở 1a rồi lỡ chuyển sang 1b/1c xem tiến độ. Panel này hiện trên
              CẢ BA tab bất cứ khi nào một chuỗi đang chạy, nên đứng ở tab nào
              cũng thấy đủ ba bước và biết đang chờ đúng bước nào. */}
          {running && (
            <div className={styles.runPanel} data-testid="authoring-run-panel">
              {run.steps.length <= 1 && (
                <>
                  <OperationProgressCard
                    subtitle={live?.running ? liveProgressText(live) : null}
                    {...liveCounts(live)}
                    testId="authoring-live-progress"
                  />
                </>
              )}
              {run.steps.length > 1 && (
                <ol className={styles.stepper}>
                  {ALL_STEPS.map((step, index) => {
                    const status = index < run.currentIndex ? "done" : index === run.currentIndex ? "running" : "pending";
                    return (
                      <li
                        key={step}
                        className={`${styles.stepItem} ${styles[`stepItem_${status}`]}`}
                        data-testid={`authoring-run-panel-${step}`}
                        aria-current={status === "running" ? "step" : undefined}
                      >
                        <span className={styles.stepName}>{STEP_LABELS[step]}</span>
                        <span className={styles.stepNote}>
                          {status === "done"
                            ? ["xong", runs[step] ? formatClock(runs[step]!.durationMs / 1000) : null, runs[step]?.chars ? `${formatChars(runs[step]!.chars)} ký tự` : null]
                                .filter(Boolean)
                                .join(" · ")
                            : status === "running"
                            ? stepLiveNote(live)
                            : "chờ"}
                        </span>
                        {status === "running" && (
                          <OperationProgressCard variant="step" {...liveCounts(live)} />
                        )}
                      </li>
                    );
                  })}
                </ol>
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
    <div className={styles.options} data-testid="authoring-mode-switch" role="radiogroup" aria-label="Cách làm bước 3">
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
        <span>Bạn copy prompt, dán vào ChatGPT/Claude/Gemini rồi dán kết quả về. Đổi sang “Gọi API” bất cứ lúc nào.</span>
      </button>
      <button
        type="button"
        role="radio"
        aria-checked={aiMode}
        data-testid="authoring-mode-ai"
        className={`${styles.option} ${aiMode ? styles.optionOn : ""}`}
        disabled={aiOff || disabled}
        title={aiOff ? llm.reason || "Chưa cấu hình API key" : undefined}
        onClick={() => onModeChange("ai")}
      >
        <b>{MODE_LABELS.ai}</b>
        <span>
          {aiOff
            ? llm.reason || "Chưa cấu hình API key nên chỉ có đường copy tay."
            : `Hệ thống tự gọi ${llm.provider}, điền kết quả vào ô soạn thảo.`}
        </span>
      </button>
    </div>
  );
}

/**
 * Chỉ pha biết tổng mới có phần trăm (FR116.4): các lô của 1c và vòng sửa lỗi
 * biên dịch. Pha suy luận/viết không biết tổng nên thanh chạy không xác định.
 */
function liveCounts(p: AuthoringProgress | null): { done?: number; total?: number } {
  if (!p?.running) return {};
  if (p.phase === "chunks" && p.chunks_total) return { done: p.chunks_done ?? 0, total: p.chunks_total };
  if (p.phase === "repair" && p.repair_max) return { done: p.repair_round ?? 0, total: p.repair_max };
  return {};
}

/** Dòng phụ của thẻ đang chạy: "AI đang viết · 14,3k ký tự · 2m05s". */
function stepLiveNote(p: AuthoringProgress | null): string {
  if (!p?.running) return "đang chạy";
  const time = formatClock(p.elapsed_seconds);
  if (p.phase === "writing") return `AI đang viết · ${formatChars(p.content_chars)} ký tự · ${time}`;
  if (p.phase === "reasoning") return `AI đang suy luận · ${formatChars(p.reasoning_chars)} ký tự · ${time}`;
  return `${liveProgressText(p).replace(/….*$/, "").replace(/:.*$/, "")} · ${time}`;
}

/** Câu tiến độ từ luồng streaming: pha hiện tại, lượng chữ đã nhận, thời gian. */
function liveProgressText(p: AuthoringProgress): string {
  const time = `${p.elapsed_seconds}s`;
  if (p.phase === "layout" || p.phase === "cast") {
    return `Đang dựng bảng ${p.phase === "layout" ? "toạ độ chung (LAYOUT)" : "vật xuyên suốt (cast)"}… ${time}`;
  }
  if (p.phase === "chunks") return `Đang viết code theo lô: ${p.chunks_done ?? 0}/${p.chunks_total ?? "?"} lô xong · ${time}`;
  if (p.phase === "merge") return `Đang ghép code… ${time}`;
  if (p.phase === "check") return `Đang kiểm tra biên dịch… ${time}`;
  if (p.phase === "repair") {
    return `Đang sửa lỗi biên dịch: vòng ${p.repair_round ?? "?"}/${p.repair_max ?? "?"} · ${time}`;
  }
  if (p.phase === "writing") return `AI đang viết kết quả… ${formatChars(p.content_chars)} ký tự · ${time}`;
  if (p.phase === "reasoning") return `AI đang suy luận… ${formatChars(p.reasoning_chars)} ký tự · ${time}`;
  return `Đang chờ Hive phản hồi… ${time}`;
}
