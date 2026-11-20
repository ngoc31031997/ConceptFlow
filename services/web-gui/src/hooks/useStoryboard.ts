import { useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  acceptStoryboardFinding,
  addStoryboardAsset,
  addStoryboardShot,
  applyStoryboardFinding,
  approveAllStoryboard,
  approveStoryboardAsset,
  approveStoryboardAssetList,
  approveStoryboardShot,
  deleteStoryboardAsset,
  deleteStoryboardShot,
  drawStoryboardAsset,
  editStoryboardAsset,
  editStoryboardShot,
  generateStoryboard,
  generateStoryboardAssetDetail,
  generateStoryboardAssetDetails,
  generateStoryboardAssets,
  generateStoryboardUnit,
  getStoryboard,
  isRequestCancelled,
  lockStoryboard,
  newOperationId,
  redrawStoryboardAsset,
  redrawStoryboardFrame,
  reviewStoryboard,
  setStoryboardModel,
  unapproveStoryboardAsset,
  unapproveStoryboardAssetList,
  unapproveStoryboardShot,
  type StoryboardState,
} from "../api/client";
import { visualErrorText } from "./useVisualConcept";

/**
 * What the system's AI or drawer is doing now: every empty unit, the asset list, the detail of
 * every asset without one, the review, one unit's shots, one asset's detail, one asset's drawing,
 * or one shot's frame.
 */
export type StoryboardRunning =
  | "all"
  | "assets"
  | "details"
  | "review"
  | { unit: string }
  | { detail: string }
  | { asset: string }
  | { shot: string }
  | null;

/** Why step 11 cannot open yet: the step whose lock it waits for. */
export type StoryboardNotOpen = "script" | "concept" | "plan" | "sequence" | null;

const NOT_OPEN: Record<string, Exclude<StoryboardNotOpen, null>> = {
  script_not_locked: "script",
  concept_not_locked: "concept",
  plan_not_locked: "plan",
  sequence_not_locked: "sequence",
};

/** What the step 11 screen gets from useStoryboard. */
export interface StoryboardControls {
  /** null until the server answers. */
  state: StoryboardState | null;
  /** An earlier step is not locked, so step 11 cannot open. */
  notOpen: StoryboardNotOpen;
  loadError: string | null;
  running: StoryboardRunning;
  /** An action other than an AI run is in flight. */
  busy: boolean;
  actionError: string | null;
  reload: () => Promise<void>;
  /** Stops the run in flight; the server drops the call when the connection closes. */
  stop: () => void;
  setModel: (model: string) => Promise<boolean>;
  generateAssets: (note: string) => Promise<boolean>;
  generateAll: (note: string) => Promise<boolean>;
  generateAssetDetail: (assetId: string, note: string) => Promise<boolean>;
  generateAssetDetails: (note: string) => Promise<boolean>;
  generateUnit: (unitId: string, note: string) => Promise<boolean>;
  review: (note: string) => Promise<boolean>;
  drawAsset: (assetId: string) => Promise<boolean>;
  redrawAsset: (assetId: string, note: string) => Promise<boolean>;
  redrawFrame: (unitId: string, shotId: string) => Promise<boolean>;
  /** Form and confirmation actions throw so the dialog can show why they failed. */
  addAsset: (fields: Record<string, unknown>) => Promise<void>;
  editAsset: (assetId: string, fields: Record<string, unknown>) => Promise<void>;
  deleteAsset: (assetId: string) => Promise<void>;
  /** Changes fields from a control on the card (the library tick); a failure shows as the screen's error. */
  patchAsset: (assetId: string, fields: Record<string, unknown>) => Promise<boolean>;
  addShot: (unitId: string, fields: Record<string, unknown>) => Promise<void>;
  editShot: (unitId: string, shotId: string, fields: Record<string, unknown>) => Promise<void>;
  deleteShot: (unitId: string, shotId: string) => Promise<void>;
  approveAsset: (assetId: string, projectName: string) => Promise<boolean>;
  unapproveAsset: (assetId: string) => Promise<boolean>;
  approveAssetList: () => Promise<boolean>;
  unapproveAssetList: () => Promise<boolean>;
  approveShot: (unitId: string, shotId: string) => Promise<boolean>;
  unapproveShot: (unitId: string, shotId: string) => Promise<boolean>;
  approveAll: () => Promise<boolean>;
  applyFix: (findingId: string) => Promise<boolean>;
  /** Accepts a finding or a machine check; an empty reason is stored as none given. */
  accept: (findingId: string, why: string) => Promise<boolean>;
  lock: (note: string) => Promise<boolean>;
}

/**
 * Loads and changes step 11 "Storyboard". Every change answers with the whole
 * screen, which replaces the state. A run of the system's AI or drawer holds
 * the request open until the answer is stored; `stop` cancels it.
 */
export function useStoryboard(projectId: string): StoryboardControls {
  const [state, setState] = useState<StoryboardState | null>(null);
  const [notOpen, setNotOpen] = useState<StoryboardNotOpen>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [running, setRunning] = useState<StoryboardRunning>(null);
  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const abort = useRef<AbortController | null>(null);
  // Only the newest load may land: an older answer must not overwrite a newer one.
  const loadSeq = useRef(0);

  const reload = useCallback(async () => {
    if (!projectId) return;
    const seq = ++loadSeq.current;
    try {
      const next = await getStoryboard(projectId);
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

  const act = useCallback(async (call: () => Promise<StoryboardState>): Promise<boolean> => {
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

  /** A change made from a form or a confirmation: the answer replaces the state, a failure is thrown to the dialog. */
  const change = useCallback(async (call: () => Promise<StoryboardState>): Promise<void> => {
    loadSeq.current += 1;
    setState(await call());
  }, []);

  const runAi = useCallback(
    async (
      what: Exclude<StoryboardRunning, null>,
      call: (op: { id: string; signal: AbortSignal }) => Promise<StoryboardState>,
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
        // A run stopped or failed half way may have stored the units written before it.
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
    setModel: (model) => act(() => setStoryboardModel(projectId, model)),
    generateAssets: (note) => runAi("assets", (op) => generateStoryboardAssets(projectId, note, op)),
    generateAll: (note) => runAi("all", (op) => generateStoryboard(projectId, note, op)),
    generateAssetDetail: (assetId, note) =>
      runAi({ detail: assetId }, (op) => generateStoryboardAssetDetail(projectId, assetId, note, op)),
    generateAssetDetails: (note) =>
      runAi("details", (op) => generateStoryboardAssetDetails(projectId, note, op)),
    generateUnit: (unitId, note) =>
      runAi({ unit: unitId }, (op) => generateStoryboardUnit(projectId, unitId, note, op)),
    review: (note) => runAi("review", (op) => reviewStoryboard(projectId, note, op)),
    drawAsset: (assetId) => runAi({ asset: assetId }, (op) => drawStoryboardAsset(projectId, assetId, op)),
    redrawAsset: (assetId, note) =>
      runAi({ asset: assetId }, (op) => redrawStoryboardAsset(projectId, assetId, note, op)),
    redrawFrame: (unitId, shotId) =>
      runAi({ shot: shotId }, (op) => redrawStoryboardFrame(projectId, unitId, shotId, op)),
    addAsset: (fields) => change(() => addStoryboardAsset(projectId, fields)),
    editAsset: (assetId, fields) => change(() => editStoryboardAsset(projectId, assetId, fields)),
    deleteAsset: (assetId) => change(() => deleteStoryboardAsset(projectId, assetId)),
    patchAsset: (assetId, fields) => act(() => editStoryboardAsset(projectId, assetId, fields)),
    addShot: (unitId, fields) => change(() => addStoryboardShot(projectId, unitId, fields)),
    editShot: (unitId, shotId, fields) =>
      change(() => editStoryboardShot(projectId, unitId, shotId, fields)),
    deleteShot: (unitId, shotId) => change(() => deleteStoryboardShot(projectId, unitId, shotId)),
    approveAsset: (assetId, projectName) =>
      act(() => approveStoryboardAsset(projectId, assetId, projectName)),
    unapproveAsset: (assetId) => act(() => unapproveStoryboardAsset(projectId, assetId)),
    approveAssetList: () => act(() => approveStoryboardAssetList(projectId)),
    unapproveAssetList: () => act(() => unapproveStoryboardAssetList(projectId)),
    approveShot: (unitId, shotId) => act(() => approveStoryboardShot(projectId, unitId, shotId)),
    unapproveShot: (unitId, shotId) => act(() => unapproveStoryboardShot(projectId, unitId, shotId)),
    approveAll: () => act(() => approveAllStoryboard(projectId)),
    applyFix: (findingId) => act(() => applyStoryboardFinding(projectId, findingId)),
    accept: (findingId, why) => act(() => acceptStoryboardFinding(projectId, findingId, why)),
    lock: (note) => act(() => lockStoryboard(projectId, note)),
  };
}
