import { useEffect, useState } from "react";
import { subscribeProgress } from "../api/client";
import type { ProgressMessage } from "../types";

export interface ProgressState {
  currentStep: string | null;
  sceneIndex: number | null;
  sceneTotal: number | null;
  elapsedSeconds: number | null;
  animationIndex: number | null;
  // CR-029
  stageIndex: number | null;
  stageTotal: number | null;
  clipIndex: number | null;
  clipTotal: number | null;
  status: "in_progress" | "completed" | "failed" | null;
  errorMessage: string | null;
}

const initialState: ProgressState = {
  currentStep: null,
  sceneIndex: null,
  sceneTotal: null,
  elapsedSeconds: null,
  animationIndex: null,
  stageIndex: null,
  stageTotal: null,
  clipIndex: null,
  clipTotal: null,
  status: null,
  errorMessage: null,
};

export function useSSE(projectId: string): ProgressState {
  const [state, setState] = useState<ProgressState>(initialState);

  useEffect(() => {
    const onMessage = (msg: ProgressMessage) => {
      setState({
        currentStep: msg.step,
        sceneIndex: msg.scene_index ?? null,
        sceneTotal: msg.scene_total ?? null,
        elapsedSeconds: msg.elapsed_seconds ?? null,
        animationIndex: msg.animation_index ?? null,
        stageIndex: msg.stage_index ?? null,
        stageTotal: msg.stage_total ?? null,
        clipIndex: msg.clip_index ?? null,
        clipTotal: msg.clip_total ?? null,
        status: msg.status,
        errorMessage: msg.status === "failed" ? (msg.error_message ?? null) : null,
      });
    };
    const unsubscribe = subscribeProgress(projectId, onMessage);
    return unsubscribe;
  }, [projectId]);

  return state;
}
