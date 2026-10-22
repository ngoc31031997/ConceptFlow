import { BRIEF_UNIT, generateBrief, getBriefState, type BriefState, type CreativeBrief } from "../api/client";
import { useUnitDesk, type UnitDeskConfig, type UnitDeskControls } from "./useUnitDesk";

/** The operation kind the server records for the AI runs that write the brief. */
const BRIEF_RUN_KIND = "brief";

const BRIEF_DESK: UnitDeskConfig<CreativeBrief, BriefState> = {
  unit: BRIEF_UNIT,
  runKind: BRIEF_RUN_KIND,
  load: (projectId) => getBriefState(projectId),
  generate: (projectId, operation) => generateBrief(projectId, operation),
};

/** The brief step's brief as the screen works with it. */
export type CreativeBriefControls = UnitDeskControls<BriefState>;

/**
 * Loads and changes the project's creative brief; nothing loads until
 * `enabled` (the project exists on the server). Every change is followed by a
 * reload of the whole state, since whether the story outline was written from
 * this brief can change with it.
 */
export function useCreativeBrief(projectId: string, enabled: boolean): CreativeBriefControls {
  return useUnitDesk(BRIEF_DESK, projectId, enabled);
}
