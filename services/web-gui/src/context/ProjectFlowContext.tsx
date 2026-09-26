import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { useMatch } from "react-router-dom";
import { getProject } from "../api/client";
import { ProjectDraftContext } from "./ProjectDraftContext";
import type { Project } from "../types";
import { isAuthoringEditable } from "../utils/flow";

const POLL_MS = 5000;

/**
 * Where the project on screen stands in the 13-step flow, as the SERVER says.
 * One place fetches it so the step bar, the wizard's "next" bar and the
 * read-only banner cannot disagree; no page has to pass it around.
 *
 * The project is the one in the URL (/projects/:id/...) or, on the wizard
 * screens, the draft in progress. Before the server has that draft (a brand
 * new idea) there is nothing to read and every field is "unknown".
 */
export interface ProjectFlow {
  project: Project | null;
  projectId: string;
  status: string | undefined;
  /** 1-13, or 0 when unknown. */
  flowStep: number;
  runState: Project["run_state"] | undefined;
  /** False once the server would refuse authoring edits (see isAuthoringEditable). */
  editable: boolean;
  refetch: () => void;
}

const noFlow: ProjectFlow = {
  project: null, projectId: "", status: undefined, flowStep: 0, runState: undefined, editable: true, refetch: () => {},
};

const ProjectFlowContext = createContext<ProjectFlow>(noFlow);

export function ProjectFlowProvider({ children }: { children: ReactNode }) {
  const draft = useContext(ProjectDraftContext);
  const routeId = useMatch("/projects/:id/*")?.params.id;
  const projectId = routeId ?? draft.projectId;
  const [project, setProject] = useState<Project | null>(null);

  const load = useCallback(() => {
    if (!projectId) return;
    getProject(projectId)
      .then((p) => setProject(p))
      .catch(() => setProject(null)); // no such project on the server yet
  }, [projectId]);

  useEffect(() => {
    setProject(null);
    load();
    if (!projectId) return;
    const t = window.setInterval(load, POLL_MS);
    return () => window.clearInterval(t);
  }, [projectId, load]);

  const value: ProjectFlow = {
    project,
    projectId: projectId ?? "",
    status: project?.status,
    flowStep: project?.flow_step ?? 0,
    runState: project?.run_state,
    editable: isAuthoringEditable(project?.status),
    refetch: load,
  };
  return <ProjectFlowContext.Provider value={value}>{children}</ProjectFlowContext.Provider>;
}

export function useProjectFlow(): ProjectFlow {
  return useContext(ProjectFlowContext);
}
