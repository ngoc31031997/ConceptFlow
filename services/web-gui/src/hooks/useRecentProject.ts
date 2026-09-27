import { useEffect, useState } from "react";
import { listProjects } from "../api/client";
import type { ProjectSummary } from "../types";

const STORAGE_KEY = "cf.lastProjectId";

function readLast(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function writeLast(id: string) {
  try {
    localStorage.setItem(STORAGE_KEY, id);
  } catch {
    /* storage bị chặn — chỉ mất lối tắt, không sao */
  }
}

/**
 * Project để hiện ở lối tắt "Tiếp tục" trên sidebar: ưu tiên project mở lần
 * cuối trên trình duyệt này, nếu không còn thì lấy project chưa xong được cập
 * nhật gần nhất. `currentId` là project đang mở — được ghi nhớ làm "lần cuối".
 */
export function useRecentProject(currentId?: string | null): ProjectSummary | null {
  const [projects, setProjects] = useState<ProjectSummary[] | null>(null);

  useEffect(() => {
    if (currentId) writeLast(currentId);
  }, [currentId]);

  useEffect(() => {
    let alive = true;
    listProjects()
      .then((p) => alive && setProjects(p))
      .catch(() => alive && setProjects([]));
    return () => {
      alive = false;
    };
  }, []);

  if (!projects || projects.length === 0) return null;
  const last = readLast();
  const remembered = last ? projects.find((p) => p.project_id === last) : undefined;
  if (remembered) return remembered;
  const unfinished = projects.filter((p) => p.run_state !== "done" && p.run_state !== "cancelled");
  const pool = unfinished.length ? unfinished : projects;
  return [...pool].sort((a, b) => b.updated_at.localeCompare(a.updated_at))[0];
}
