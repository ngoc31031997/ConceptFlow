import { vi } from "vitest";
import * as apiClient from "../../src/api/client";

/**
 * CR-040 FR113: prompt text is rendered by authoring-service (and held to the
 * old TypeScript output by its golden tests), so component tests only need to
 * prove the wiring: which role and which draft values were sent, and that the
 * server's answer is what gets shown and copied.
 */
export function mockRenderPrompt() {
  return vi.spyOn(apiClient, "renderPrompt").mockImplementation(async (input) => ({
    prompt: `RENDERED[${input.role}] topic=${input.topic ?? ""} script=${input.script ?? ""} prev=${input.previous_output ?? ""}`,
    prompt_id: "system-x",
    prompt_name: "Mặc định",
    is_system: true,
  }));
}
