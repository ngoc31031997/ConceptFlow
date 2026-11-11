import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  acceptVisualPlanFinding,
  answerVisualPlanQuestion,
  applyVisualPlanFinding,
  approveAllVisualPlan,
  approveVisualPlanUnit,
  editVisualPlanUnit,
  generateVisualPlan,
  generateVisualPlanBeat,
  getVisualPlan,
  isRequestCancelled,
  lockVisualPlan,
  mergeVisualPlanUnit,
  newOperationId,
  reviewVisualPlan,
  setVisualPlanModel,
  splitVisualPlanUnit,
  unapproveVisualPlanUnit,
  type VisualPlanState,
} from "../api/client";
import { visualErrorText } from "./useVisualConcept";

/** What the system's AI is doing now: every empty beat, one beat (its key), or the review. */
export type PlanRunning = "all" | "review" | { beat: string } | null;

/** Why step 9 cannot open yet. */
export type PlanNotOpen = "script" | "concept" | null;

/** What the step 9 screen gets from useVisualPlan. */
export interface VisualPlanControls {
  /** null until the server answers. */
  state: VisualPlanState | null;
  /** The step before it is not locked, so step 9 cannot open. */
  notOpen: PlanNotOpen;
  loadError: string | null;
  running: PlanRunning;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  reload: () => Promise<void>;
  /** Stops the AI run in flight; the server drops the call when the connection closes. */
  stop: () => void;
  generateAll: (note: string) => Promise<boolean>;
  generateBeat: (beatKey: string, note: string) => Promise<boolean>;
  review: (note: string) => Promise<boolean>;
  /** Hand-edit actions throw so the form can show why they failed. */
  edit: (unitId: string, fields: Record<string, unknown>) => Promise<void>;
  answer: (unitId: string, questionId: string, answer: string) => Promise<void>;
  split: (unitId: string, atLine: string) => Promise<void>;
  merge: (unitId: string, withId: string) => Promise<void>;
  approve: (unitId: string) => Promise<boolean>;
  unapprove: (unitId: string) => Promise<boolean>;
  approveAll: () => Promise<boolean>;
  setModel: (model: string) => Promise<boolean>;
  applyFix: (findingId: string) => Promise<boolean>;
  accept: (findingId: string, why: string) => Promise<boolean>;
  lock: (note: string) => Promise<boolean>;
}

/**
 * Loads and changes step 9 "Kế hoạch hình". Every change answers with the
 * whole screen, which replaces the state. A run of the system's AI holds the
 * request open until the answer is stored; `stop` cancels it.
 */
export function useVisualPlan(projectId: string): VisualPlanControls {
  const [state, setState] = useState<VisualPlanState | null>(null);
  const [notOpen, setNotOpen] = useState<PlanNotOpen>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<PlanRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const abort = useRef<AbortController | null>(null);
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const reload = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getVisualPlan(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setNotOpen(null);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      if (err instanceof ApiError && (err.code === "script_not_locked" || err.code === "concept_not_locked")) {
        setNotOpen(err.code === "script_not_locked" ? "script" : "concept");
        setLoadError(null);
        return;
      }
      setLoadError(visualErrorText(err));
    }
  }, [projectId]);

  useEffect(() => {
    setState(null);
    void reload();
  }, [reload]);

  useEffect(() => () => abort.current?.abort(), []);

  const act = useCallback(async (call: () => Promise<VisualPlanState>): Promise<boolean> => {
    setBusy(true);
    setActionError(null);
    try {
      loadSeq.current += 1;
      setState(await call());
      return true;
    } catch (err) {
      setActionError(visualErrorText(err));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  /** A change made from a form: the answer replaces the state, a failure is thrown to the form. */
  const change = useCallback(async (call: () => Promise<VisualPlanState>): Promise<void> => {
    loadSeq.current += 1;
    setState(await call());
  }, []);

  const runAi = useCallback(
    async (
      what: Exclude<PlanRunning, null>,
      call: (op: { id: string; signal: AbortSignal }) => Promise<VisualPlanState>,
    ) => {
      const controller = new AbortController();
      abort.current = controller;
      setRunning(what);
      setActionError(null);
      try {
        loadSeq.current += 1;
        setState(await call({ id: newOperationId(), signal: controller.signal }));
        return true;
      } catch (err) {
        if (!isRequestCancelled(err)) setActionError(visualErrorText(err));
        // A run stopped or failed half way may have stored the beats written before it.
        await reload();
        return false;
      } finally {
        abort.current = null;
        setRunning(null);
      }
    },
    [reload],
  );

  return {
    state,
    notOpen,
    loadError,
    running,
    busy,
    actionError,
    reload,
    stop: () => abort.current?.abort(),
    generateAll: (note) => runAi("all", (op) => generateVisualPlan(projectId, note, op)),
    generateBeat: (beatKey, note) =>
      runAi({ beat: beatKey }, (op) => generateVisualPlanBeat(projectId, beatKey, note, op)),
    review: (note) => runAi("review", (op) => reviewVisualPlan(projectId, note, op)),
    edit: (unitId, fields) => change(() => editVisualPlanUnit(projectId, unitId, fields)),
    answer: (unitId, qid, text) => change(() => answerVisualPlanQuestion(projectId, unitId, qid, text)),
    split: (unitId, atLine) => change(() => splitVisualPlanUnit(projectId, unitId, atLine)),
    merge: (unitId, withId) => change(() => mergeVisualPlanUnit(projectId, unitId, withId)),
    approve: (unitId) => act(() => approveVisualPlanUnit(projectId, unitId)),
    unapprove: (unitId) => act(() => unapproveVisualPlanUnit(projectId, unitId)),
    approveAll: () => act(() => approveAllVisualPlan(projectId)),
    setModel: (model) => act(() => setVisualPlanModel(projectId, model)),
    applyFix: (findingId) => act(() => applyVisualPlanFinding(projectId, findingId)),
    accept: (findingId, why) => act(() => acceptVisualPlanFinding(projectId, findingId, why)),
    lock: (note) => act(() => lockVisualPlan(projectId, note)),
  };
}
