import { IDEA_UNIT, generateIdea, getIdeaState, type Idea, type IdeaState } from "../api/client";
import { useUnitDesk, type UnitDeskConfig, type UnitDeskControls } from "./useUnitDesk";

/** The operation kind the server records for the AI runs that write the idea. */
const IDEA_RUN_KIND = "idea";

const IDEA_DESK: UnitDeskConfig<Idea, IdeaState> = {
  unit: IDEA_UNIT,
  runKind: IDEA_RUN_KIND,
  load: (projectId) => getIdeaState(projectId),
  generate: (projectId, operation) => generateIdea(projectId, operation),
};

/** The idea step's idea as the screen works with it. */
export type IdeaControls = UnitDeskControls<IdeaState>;

/** Loads and changes the project's idea; nothing loads until `enabled` (the project exists on the server). */
export function useIdea(projectId: string, enabled: boolean): IdeaControls {
  return useUnitDesk(IDEA_DESK, projectId, enabled);
}
