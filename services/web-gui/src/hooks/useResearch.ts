import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  GENERIC_CONNECTION_ERROR,
  addFact as apiAddFact,
  addResearchQuestion,
  approveResearch,
  deleteResearchQuestion,
  editFact,
  editResearchQuestion,
  generateResearchQuestions,
  isRequestCancelled,
  getResearch,
  pasteResearch,
  setFactFlags,
  setResearchSkipped,
  unapproveResearch,
  verifyFacts,
  type FactInput,
  type FactTextField,
  type FactType,
  type ResearchState,
} from "../api/client";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useOperationRun, type OperationRun } from "./useOperationRun";

/** The operation kind the server records for the AI run that writes the questions. */
const QUESTION_RUN_KIND = "research_question";

/** How often the screen asks again while some link is still being checked. */
export const RESEARCH_POLL_MS = 2000;

/** How long a fact sent for checking keeps showing as being checked, so a check that ends at once is still seen. */
export const CHECK_HOLD_MS = 1500;

/** What reading a pasted answer said, for the line under the paste box. */
export interface PasteMessage {
  tone: "ok" | "warn";
  text: string;
}

/** The research step as the screen works with it. */
export interface ResearchControls {
  /** null until the server answers. */
  state: ResearchState | null;
  loadError: string | null;
  /** The AI is writing the questions. */
  running: boolean;
  run: OperationRun;
  /** An action other than the AI run is in flight. */
  busy: boolean;
  /** Why the last action failed, in the server's words. */
  actionError: string | null;
  pasteMessage: PasteMessage | null;
  /** Codes of the facts just sent for checking, shown as being checked for CHECK_HOLD_MS even if the answer is already in. */
  holding: ReadonlySet<string>;
  generateQuestions: (replaceEdited: boolean, note?: string) => Promise<boolean>;
  addQuestion: (text: string) => Promise<boolean>;
  saveQuestion: (id: string, text: string) => Promise<boolean>;
  deleteQuestion: (id: string) => Promise<boolean>;
  pasteResults: (raw: string) => Promise<boolean>;
  /** Add and edit throw, so the dialog that asked can show why it failed. */
  addFact: (input: FactInput) => Promise<void>;
  saveFact: (id: string, fields: Partial<Record<FactTextField, string>>) => Promise<void>;
  setFlags: (id: string, flags: { type?: FactType; must?: boolean; keep?: boolean }) => Promise<boolean>;
  recheck: (ids: string[]) => Promise<boolean>;
  skip: (skipped: boolean) => Promise<boolean>;
  approve: () => Promise<boolean>;
  unapprove: () => Promise<boolean>;
  reload: () => void;
}

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Loads and changes the project's research step; nothing loads until `enabled`.
 * Every change is followed by a reload of the whole state, since one change
 * (a pasted answer, a dropped fact) moves coverage, links and approval. While
 * any fact link is still being checked the state is asked for again every
 * RESEARCH_POLL_MS, so the checks fill in by themselves.
 */
export function useResearch(projectId: string, enabled: boolean): ResearchControls {
  const { refetch: refetchFlow } = useProjectFlow();
  const [state, setState] = useState<ResearchState | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [pasteMessage, setPasteMessage] = useState<PasteMessage | null>(null);
  const [holding, setHolding] = useState<ReadonlySet<string>>(new Set());
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const load = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getResearch(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      setLoadError(messageOf(err));
    }
  }, [projectId]);

  // A run that outlived the page is followed from here; the questions it stored are loaded when it ends.
  const run = useOperationRun({ projectId: enabled ? projectId : undefined, kinds: [QUESTION_RUN_KIND], onSettled: () => void load() });

  useEffect(() => {
    setState(null);
    setLoadError(null);
    setActionError(null);
    setPasteMessage(null);
    if (enabled) void load();
  }, [enabled, load]);

  const checking = !!state?.facts.some((f) => f.link.state === "run");
  useEffect(() => {
    if (!checking) return;
    const timer = setInterval(() => void load(), RESEARCH_POLL_MS);
    return () => clearInterval(timer);
  }, [checking, load]);

  // Keeps the given facts shown as being checked for CHECK_HOLD_MS.
  const hold = useCallback((ids: string[]) => {
    if (ids.length === 0) return;
    setHolding((prev) => new Set([...prev, ...ids]));
    setTimeout(() => {
      setHolding((prev) => {
        const next = new Set(prev);
        ids.forEach((id) => next.delete(id));
        return next;
      });
    }, CHECK_HOLD_MS);
  }, []);

  // Runs a change, then reloads; reports whether it worked.
  const act = useCallback(
    async (change: () => Promise<unknown>): Promise<boolean> => {
      setBusy(true);
      setActionError(null);
      try {
        await change();
        await load();
        // The step menu opens the story step from the skip / approval just made.
        refetchFlow();
        return true;
      } catch (err) {
        setActionError(messageOf(err));
        return false;
      } finally {
        setBusy(false);
      }
    },
    [load, refetchFlow],
  );

  // Like act, but a failure is thrown for the dialog that asked.
  const actOrThrow = useCallback(
    async (change: () => Promise<unknown>): Promise<void> => {
      setActionError(null);
      try {
        await change();
      } catch (err) {
        throw new Error(messageOf(err));
      }
      await load();
    },
    [load],
  );

  const generateQuestions = useCallback(
    async (replaceEdited: boolean, note?: string) => {
      setRunning(true);
      setActionError(null);
      try {
        await generateResearchQuestions(projectId, { replaceEdited, note }, run.begin());
        run.end();
        await load();
        return true;
      } catch (err) {
        if (isRequestCancelled(err)) return false;
        // The card stays with the failure and the time already spent.
        setActionError(messageOf(err));
        return false;
      } finally {
        setRunning(false);
      }
    },
    [projectId, run, load],
  );

  const pasteResults = useCallback(
    async (raw: string) => {
      setBusy(true);
      setActionError(null);
      setPasteMessage(null);
      try {
        const result = await pasteResearch(projectId, raw);
        const read = result.added.length + result.updated.length;
        const left = result.rejected.length;
        setPasteMessage({
          tone: "ok",
          text: `Đã đọc ${read} dữ kiện${left ? `, bỏ ${left} dòng (lý do ở nhật ký)` : ""}. Đang tải từng link để kiểm đoạn trích.`,
        });
        await load();
        hold([...result.added, ...result.updated]);
        return true;
      } catch (err) {
        setPasteMessage({ tone: "warn", text: messageOf(err) });
        return false;
      } finally {
        setBusy(false);
      }
    },
    [projectId, load, hold],
  );

  return {
    state,
    loadError,
    running: running || run.resumed,
    run,
    busy,
    actionError,
    pasteMessage,
    holding,
    generateQuestions,
    addQuestion: (text) => act(() => addResearchQuestion(projectId, text)),
    saveQuestion: (id, text) => act(() => editResearchQuestion(projectId, id, text)),
    deleteQuestion: (id) => act(() => deleteResearchQuestion(projectId, id)),
    pasteResults,
    addFact: (input) => actOrThrow(() => apiAddFact(projectId, input)),
    saveFact: (id, fields) => actOrThrow(() => editFact(projectId, id, fields)),
    setFlags: (id, flags) => act(() => setFactFlags(projectId, id, flags)),
    recheck: async (ids) => {
      const ok = await act(() => verifyFacts(projectId, ids));
      if (ok) hold(ids);
      return ok;
    },
    skip: (skipped) => act(() => setResearchSkipped(projectId, skipped)),
    approve: () => act(() => approveResearch(projectId)),
    unapprove: () => act(() => unapproveResearch(projectId)),
    reload: () => void load(),
  };
}
