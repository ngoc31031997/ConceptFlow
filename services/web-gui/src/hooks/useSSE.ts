import { useEffect, useState } from "react";
import { subscribeProgress } from "../api/client";
import type { ProgressMessage } from "../types";

export interface ProgressState {
  currentStep: string | null;
  sceneIndex: number | null;
  sceneTotal: number | null;
  status: "in_progress" | "completed" | "failed" | null;
  errorMessage: string | null;
}

const initialState: ProgressState = {
  currentStep: null,
  sceneIndex: null,
  sceneTotal: null,
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
        status: msg.status,
        errorMessage: msg.status === "failed" ? (msg.error_message ?? null) : null,
      });
    };
    const unsubscribe = subscribeProgress(projectId, onMessage);
    return unsubscribe;
  }, [projectId]);

  return state;
}
