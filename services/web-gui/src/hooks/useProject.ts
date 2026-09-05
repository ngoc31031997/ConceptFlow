import { useCallback, useEffect, useState } from "react";
import { getProject } from "../api/client";
import type { Project } from "../types";

const POLL_INTERVAL_MS = 5000;

export function useProject(projectId: string) {
  const [project, setProject] = useState<Project | null>(null);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    try {
      const result = await getProject(projectId);
      setProject(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [projectId]);

  useEffect(() => {
    refetch();
    const interval = setInterval(refetch, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [refetch]);

  return { project, error, refetch };
}
