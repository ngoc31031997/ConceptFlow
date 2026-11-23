import {
  ExternalRequestDescribed,
  generateSequenceUnit,
  type ExternalRequest,
} from "../../api/client";
import type { useAiQueue } from "../../hooks/useAiQueue";

/** The request that has the AI write one unit, described and not sent, so the extension can make it later. */
async function describeUnit(projectId: string, unitId: string): Promise<ExternalRequest> {
  try {
    await generateSequenceUnit(projectId, unitId, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết chuỗi cho unit này.");
}

/** What queueing units left the screen to say. */
export interface QueueOutcome {
  tone: "ok" | "error";
  text: string;
}

/**
 * Puts one queue item per unit on the AI queue, in the order given, so each is written after the
 * one before it and reads it.
 */
export async function queueUnits(
  queue: ReturnType<typeof useAiQueue>,
  projectId: string,
  unitIds: string[],
): Promise<QueueOutcome> {
  let added = 0;
  let already = 0;
  try {
    for (const [i, unitId] of unitIds.entries()) {
      const outcome = await queue.add({
        label: `Chuỗi hình · ${unitId}`,
        project: projectId,
        request: await describeUnit(projectId, unitId),
        modes: { deepThink: true, search: false },
      });
      if (!outcome.ok) {
        return {
          tone: "error",
          text: `Dừng ở unit ${i + 1}/${unitIds.length}: ${outcome.message}`,
        };
      }
      if (outcome.added) added += 1;
      else already += 1;
    }
  } catch (err) {
    return {
      tone: "error",
      text: err instanceof Error ? err.message : "Không thêm được vào hàng đợi.",
    };
  }
  return {
    tone: "ok",
    text: `Đã thêm ${added} unit vào hàng đợi AI${already > 0 ? `, ${already} unit đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi unit một prompt và đọc các unit trước nó; bấm icon extension để xem.`,
  };
}
