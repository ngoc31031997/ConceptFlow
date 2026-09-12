import { useState } from "react";
import { approveOutline, editNarration, rejectOutline } from "../api/client";
import type { Project } from "../types";

/**
 * Shared state/actions behind the outline review screen (CR-024), pulled out
 * of the OutlineReview component so the approve/reject buttons can render in
 * RenderPage's sticky right column (next to ProgressTracker) while the
 * scrollable line list stays in the main column — bug report: the buttons
 * used to sit at the bottom of a list that can run to dozens of lines, so
 * they scrolled out of view exactly when a Creator most wanted them close by.
 */
export function useOutlineReview(project: Project, onDecided: () => void, onRejected: () => void) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<number | null>(null);
  const [draftText, setDraftText] = useState("");

  async function run(action: () => Promise<void>, after: () => void) {
    setBusy(true);
    setError(null);
    try {
      await action();
      after();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Có lỗi xảy ra");
    } finally {
      setBusy(false);
    }
  }

  function startEdit(sceneIndex: number, text: string) {
    setEditing(sceneIndex);
    setDraftText(text);
  }

  function cancelEdit() {
    setEditing(null);
  }

  function saveEdit(sceneIndex: number) {
    return run(async () => {
      await editNarration(project.project_id, sceneIndex, draftText);
      setEditing(null);
    }, onDecided);
  }

  function approve() {
    return run(() => approveOutline(project.project_id), onDecided);
  }

  function reject() {
    return run(() => rejectOutline(project.project_id), onRejected);
  }

  return { busy, error, editing, draftText, setDraftText, startEdit, cancelEdit, saveEdit, approve, reject };
}

export type UseOutlineReview = ReturnType<typeof useOutlineReview>;
