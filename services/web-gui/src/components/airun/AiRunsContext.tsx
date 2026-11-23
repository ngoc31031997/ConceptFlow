import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  aiRunReplyRequest,
  getAiChannelSetting,
  listAiRuns,
  startAiRun,
  type AiChannel,
  type AiRun,
  type AiRunTask,
} from "../../api/client";
import { useAiQueue, type AiQueue } from "../../hooks/useAiQueue";
import { DEFAULT_AI_MODES } from "../../hooks/useQueuedExternalRun";

/** How often open runs are read back while one waits on the model or the extension. */
const POLL_MS = 2500;

/** The key of one task and target: one run of it is open at a time. */
export function runKey(task: AiRunTask, target = ""): string {
  return `${task}/${target}`;
}

/** A run that still waits for a reply or for the Creator to apply or drop it. */
export function isOpen(run: AiRun | undefined): run is AiRun {
  return !!run && ["waiting", "running", "ready", "unreadable", "failed"].includes(run.status);
}

/** What the AI run dialogs and pills of one step share. */
export interface AiRuns {
  /** The step's channel from Settings, until the Creator picks another for one run. */
  defaultChannel: AiChannel;
  /** The newest run of each task and target, by `runKey`. */
  runs: Record<string, AiRun>;
  /** The AI Bridge extension's queue, for runs on the extension channel. */
  queue: AiQueue;
  /** The extension queue item that carries each run, by run id. */
  queueItems: Record<string, string>;
  /** Starts a run; on the extension channel it also queues the prompt. Rejects with the reason. */
  start: (task: AiRunTask, target: string, note: string, channel: AiChannel, label: string) => Promise<AiRun>;
  /** Puts a run read back from the server in place. */
  put: (run: AiRun) => void;
  refresh: () => Promise<void>;
}

const AiRunsContext = createContext<AiRuns | null>(null);

/** The runs of the step the dialogs below it run in. */
export function useAiRuns(): AiRuns {
  const ctx = useContext(AiRunsContext);
  if (!ctx) throw new Error("useAiRuns needs an AiRunsProvider");
  return ctx;
}

interface AiRunsProviderProps {
  projectId: string;
  /** The step's id in Settings (`visual_storyboard`). */
  step: string;
  children: ReactNode;
}

/**
 * Keeps the AI runs of one step (docs/contracts/storyboard-v1.md, "Lượt chạy AI"): reads them on
 * open and, while one waits on the system's model or the extension, every few seconds, so a closed
 * dialog's run keeps going and its pill shows where it stands. A run on the extension channel is
 * queued as a fixed-prompt item whose answer the extension posts to the run's reply route.
 */
export function AiRunsProvider({ projectId, step, children }: AiRunsProviderProps) {
  const [runs, setRuns] = useState<Record<string, AiRun>>({});
  const [defaultChannel, setDefaultChannel] = useState<AiChannel>("api");
  const [queueItems, setQueueItems] = useState<Record<string, string>>({});
  const queue = useAiQueue();
  const alive = useRef(true);

  const put = useCallback((run: AiRun) => {
    setRuns((before) => ({ ...before, [runKey(run.task, run.target)]: run }));
  }, []);

  const refresh = useCallback(async () => {
    const list = await listAiRuns(projectId);
    if (!alive.current) return;
    const next: Record<string, AiRun> = {};
    for (const r of list) next[runKey(r.task, r.target)] = r;
    setRuns(next);
  }, [projectId]);

  useEffect(() => {
    alive.current = true;
    void refresh().catch(() => undefined);
    getAiChannelSetting().then(
      (s) => alive.current && setDefaultChannel(s.effective[step] ?? s.defaultChannel),
      () => undefined,
    );
    return () => {
      alive.current = false;
    };
  }, [refresh, step]);

  const waiting = Object.values(runs).some(
    (r) => r.status === "running" || (r.status === "waiting" && r.channel === "extension"),
  );
  useEffect(() => {
    if (!waiting) return;
    const timer = setInterval(() => void refresh().catch(() => undefined), POLL_MS);
    return () => clearInterval(timer);
  }, [waiting, refresh]);

  const start = useCallback(
    async (task: AiRunTask, target: string, note: string, channel: AiChannel, label: string) => {
      if (channel === "extension" && !queue.available) {
        throw new Error(
          queue.outdated
            ? "Extension ConceptFlow AI Bridge đã cũ; cập nhật lên bản mới rồi chạy lại, hoặc chọn kênh khác."
            : "Chưa thấy extension ConceptFlow AI Bridge trên trình duyệt này. Cài và bật nó, hoặc chọn kênh khác.",
        );
      }
      const run = await startAiRun(projectId, { task, target, note, channel });
      put(run);
      if (channel === "extension") {
        const outcome = await queue.add(
          {
            label,
            project: projectId,
            kind: "prompt",
            prompt: run.prompt,
            request: aiRunReplyRequest(projectId, run.id),
            modes: DEFAULT_AI_MODES,
          },
          { watch: true },
        );
        if (!outcome.ok) throw new Error(`Không đưa được prompt vào hàng đợi của extension: ${outcome.message}`);
        setQueueItems((before) => ({ ...before, [run.id]: outcome.itemId }));
      }
      return run;
    },
    [projectId, put, queue],
  );

  const value = useMemo(
    () => ({ defaultChannel, runs, queue, queueItems, start, put, refresh }),
    [defaultChannel, runs, queue, queueItems, start, put, refresh],
  );
  return <AiRunsContext.Provider value={value}>{children}</AiRunsContext.Provider>;
}
