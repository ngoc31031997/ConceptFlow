import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { useMatch } from "react-router-dom";
import { getBeats, getProject, getResearch, getScript, getStory } from "../api/client";
import { ProjectDraftContext } from "./ProjectDraftContext";
import type { Project } from "../types";
import { FLOW_STORY, isAuthoringEditable } from "../utils/flow";

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
  /**
   * The research step no longer holds the flow back (its table is approved or
   * it was skipped). Only read while the project waits at the story step; it
   * is false before that and moot after.
   */
  researchResolved: boolean;
  /** The story is approved. Only read once research is resolved; false before that. */
  storyApproved: boolean;
  /** Every beat is approved. Only read once the story is approved; false before that. */
  beatsApproved: boolean;
  /** The dialogue of every beat is approved. Only read once the beats are approved; false before that. */
  scriptApproved: boolean;
  /**
   * The server has answered for this project at least once, so a null
   * `project` means "not on the server yet" rather than "still asking".
   */
  loaded: boolean;
  refetch: () => void;
}

const noFlow: ProjectFlow = {
  project: null, projectId: "", status: undefined, flowStep: 0, runState: undefined, editable: true, researchResolved: false, storyApproved: false, beatsApproved: false, scriptApproved: false, loaded: true,
  refetch: () => {},
};

const ProjectFlowContext = createContext<ProjectFlow>(noFlow);

export function ProjectFlowProvider({ children }: { children: ReactNode }) {
  const draft = useContext(ProjectDraftContext);
  const routeId = useMatch("/projects/:id/*")?.params.id;
  const projectId = routeId ?? draft.projectId;
  const [project, setProject] = useState<Project | null>(null);
  const [researchResolved, setResearchResolved] = useState(false);
  const [storyApproved, setStoryApproved] = useState(false);
  const [beatsApproved, setBeatsApproved] = useState(false);
  const [scriptApproved, setScriptApproved] = useState(false);
  const [loaded, setLoaded] = useState(false);

  const load = useCallback(() => {
    if (!projectId) return;
    getProject(projectId)
      .then(async (p) => {
        // Each step opens the next only while the project waits at the story step,
        // and each read is made only when the one before it passed. A read that
        // fails leaves the step it gates closed.
        const resolved =
          p.flow_step === FLOW_STORY ? await getResearch(projectId).then((r) => r.gate.resolved, () => false) : false;
        const storyDone = resolved ? await getStory(projectId).then((s) => s.status === "approved", () => false) : false;
        const beatsDone = storyDone ? await getBeats(projectId).then((b) => b.status === "approved", () => false) : false;
        const scriptDone = beatsDone ? await getScript(projectId).then((c) => c.status === "approved", () => false) : false;
        setProject(p);
        setResearchResolved(resolved);
        setStoryApproved(storyDone);
        setBeatsApproved(beatsDone);
        setScriptApproved(scriptDone);
      })
      .catch(() => setProject(null)) // no such project on the server yet
      .finally(() => setLoaded(true));
  }, [projectId]);

  useEffect(() => {
    setProject(null);
    setResearchResolved(false);
    setStoryApproved(false);
    setBeatsApproved(false);
    setScriptApproved(false);
    setLoaded(!projectId);
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
    researchResolved,
    storyApproved,
    beatsApproved,
    scriptApproved,
    loaded,
    refetch: load,
  };
  return <ProjectFlowContext.Provider value={value}>{children}</ProjectFlowContext.Provider>;
}

export function useProjectFlow(): ProjectFlow {
  return useContext(ProjectFlowContext);
}
