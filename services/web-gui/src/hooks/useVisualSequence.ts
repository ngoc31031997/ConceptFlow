import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  acceptVisualSequenceFinding,
  addSequenceMotif,
  applyVisualSequenceFinding,
  approveAllVisualSequence,
  approveSequenceItem,
  approveSequenceMotifs,
  confirmSequenceBeat,
  deleteSequenceMotif,
  editSequenceItem,
  editSequenceMotif,
  generateSequenceBeat,
  generateSequenceMotifs,
  generateSequenceUnit,
  generateVisualSequence,
  getVisualSequence,
  isRequestCancelled,
  lockVisualSequence,
  newOperationId,
  reviewVisualSequence,
  setVisualSequenceModel,
  unapproveSequenceItem,
  unapproveSequenceMotifs,
  type VisualSequenceState,
} from "../api/client";
import { visualErrorText } from "./useVisualConcept";

/** What the system's AI is doing now: every empty unit, one beat, one unit, the motifs, or the review. */
export type SequenceRunning =
  "all" | "review" | "motifs" | { beat: string } | { unit: string } | null;

/** Why step 10 cannot open yet: the step whose lock it waits for. */
export type SequenceNotOpen = "script" | "concept" | "plan" | null;

const NOT_OPEN: Record<string, Exclude<SequenceNotOpen, null>> = {
  script_not_locked: "script",
  concept_not_locked: "concept",
  plan_not_locked: "plan",
};

/** What the step 10 screen gets from useVisualSequence. */
export interface VisualSequenceControls {
  /** null until the server answers. */
  state: VisualSequenceState | null;
  /** An earlier step is not locked, so step 10 cannot open. */
  notOpen: SequenceNotOpen;
  loadError: string | null;
  running: SequenceRunning;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  reload: () => Promise<void>;
  /** Stops the AI run in flight; the server drops the call when the connection closes. */
  stop: () => void;
  generateMotifs: (note: string) => Promise<boolean>;
  generateAll: (note: string) => Promise<boolean>;
  generateBeat: (beatKey: string, note: string) => Promise<boolean>;
  generateUnit: (unitId: string, note: string) => Promise<boolean>;
  review: (note: string) => Promise<boolean>;
  /** Hand-edit actions throw so the form can show why they failed. */
  edit: (unitId: string, fields: Record<string, unknown>) => Promise<void>;
  addMotif: (fields: Record<string, unknown>) => Promise<void>;
  editMotif: (motifId: string, fields: Record<string, unknown>) => Promise<void>;
  deleteMotif: (motifId: string) => Promise<void>;
  approveMotifs: () => Promise<boolean>;
  unapproveMotifs: () => Promise<boolean>;
  approve: (unitId: string) => Promise<boolean>;
  unapprove: (unitId: string) => Promise<boolean>;
  approveAll: () => Promise<boolean>;
  confirmBeat: (beatKey: string) => Promise<boolean>;
  setModel: (model: string) => Promise<boolean>;
  applyFix: (findingId: string) => Promise<boolean>;
  /** Accepts a finding or a machine check; an empty reason is stored as none given. */
  accept: (findingId: string, why: string) => Promise<boolean>;
  lock: (note: string) => Promise<boolean>;
}

/**
 * Loads and changes step 10 "Chuỗi hình". Every change answers with the whole
 * screen, which replaces the state. A run of the system's AI holds the request
 * open until the answer is stored; `stop` cancels it.
 */
export function useVisualSequence(projectId: string): VisualSequenceControls {
  const [state, setState] = useState<VisualSequenceState | null>(null);
  const [notOpen, setNotOpen] = useState<SequenceNotOpen>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<SequenceRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const abort = useRef<AbortController | null>(null);
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const reload = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getVisualSequence(projectId);
      if (seq !== loadSeq.current) return;
      setState(next);
      setNotOpen(null);
      setLoadError(null);
    } catch (err) {
      if (seq !== loadSeq.current) return;
      const gate = err instanceof ApiError && err.code ? NOT_OPEN[err.code] : undefined;
      if (gate) {
        setNotOpen(gate);
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

  const act = useCallback(async (call: () => Promise<VisualSequenceState>): Promise<boolean> => {
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
  const change = useCallback(async (call: () => Promise<VisualSequenceState>): Promise<void> => {
    loadSeq.current += 1;
    setState(await call());
  }, []);

  const runAi = useCallback(
    async (
      what: Exclude<SequenceRunning, null>,
      call: (op: { id: string; signal: AbortSignal }) => Promise<VisualSequenceState>,
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
    generateMotifs: (note) => runAi("motifs", (op) => generateSequenceMotifs(projectId, note, op)),
    generateAll: (note) => runAi("all", (op) => generateVisualSequence(projectId, note, op)),
    generateBeat: (beatKey, note) =>
      runAi({ beat: beatKey }, (op) => generateSequenceBeat(projectId, beatKey, note, op)),
    generateUnit: (unitId, note) =>
      runAi({ unit: unitId }, (op) => generateSequenceUnit(projectId, unitId, note, op)),
    review: (note) => runAi("review", (op) => reviewVisualSequence(projectId, note, op)),
    edit: (unitId, fields) => change(() => editSequenceItem(projectId, unitId, fields)),
    addMotif: (fields) => change(() => addSequenceMotif(projectId, fields)),
    editMotif: (motifId, fields) => change(() => editSequenceMotif(projectId, motifId, fields)),
    deleteMotif: (motifId) => change(() => deleteSequenceMotif(projectId, motifId)),
    approveMotifs: () => act(() => approveSequenceMotifs(projectId)),
    unapproveMotifs: () => act(() => unapproveSequenceMotifs(projectId)),
    approve: (unitId) => act(() => approveSequenceItem(projectId, unitId)),
    unapprove: (unitId) => act(() => unapproveSequenceItem(projectId, unitId)),
    approveAll: () => act(() => approveAllVisualSequence(projectId)),
    confirmBeat: (beatKey) => act(() => confirmSequenceBeat(projectId, beatKey)),
    setModel: (model) => act(() => setVisualSequenceModel(projectId, model)),
    applyFix: (findingId) => act(() => applyVisualSequenceFinding(projectId, findingId)),
    accept: (findingId, why) => act(() => acceptVisualSequenceFinding(projectId, findingId, why)),
    lock: (note) => act(() => lockVisualSequence(projectId, note)),
  };
}
