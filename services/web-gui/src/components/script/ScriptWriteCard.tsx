import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateScript,
  type ExternalRequest,
  type ScriptBeat,
  type ScriptState,
} from "../../api/client";
import { useAiQueue } from "../../hooks/useAiQueue";
import type { ScriptControls } from "../../hooks/useScript";
import { AiModelSelect } from "../AiModelSelect";
import { ChannelButton } from "../airun/AiRunDialog";
import { useAiRuns } from "../airun/AiRunsContext";
import researchStyles from "../research/research.module.css";
import { Card } from "../ui";

/** The request that has the AI write one beat, described and not sent, so the extension can make it later. */
async function describeWrite(projectId: string, beat: ScriptBeat): Promise<ExternalRequest> {
  try {
    await generateScript(projectId, beat.key, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết beat này.");
}

interface ScriptWriteCardProps {
  projectId: string;
  state: ScriptState;
  script: ScriptControls;
  /** A language model is configured. */
  aiAvailable: boolean;
  /** The step cannot be changed now. */
  locked: boolean;
}

/**
 * "Chạy AI" for every beat without dialogue, on the step's channel from Settings (▾ for the
 * other): on the API the system's AI writes the beats in order; on the extension one queue item
 * per beat runs on an outside AI. Each beat is stored as it is written, with no preview, because
 * the next beat reads the stored dialogue of the one before it; a single beat's dialog previews
 * before it stores.
 */
export function ScriptWriteCard({
  projectId,
  state,
  script,
  aiAvailable,
  locked,
}: ScriptWriteCardProps) {
  const ai = useAiRuns();
  const queue = useAiQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/script`)) script.reload();
  });
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const todo = state.beats.filter((b) => !b.written);
  const busy = locked || script.busy || script.running !== null;

  // One queue item per beat without dialogue, in the order of the sheet, so each beat is
  // written after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, beat] of todo.entries()) {
        const outcome = await queue.add({
          label: `B${beat.number} · ${beat.title}`,
          project: projectId,
          request: await describeWrite(projectId, beat),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({
            tone: "error",
            text: `Dừng ở beat ${i + 1}/${todo.length}: ${outcome.message}`,
          });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} beat vào hàng đợi AI${already > 0 ? `, ${already} beat đã có trong hàng đợi` : ""}. Extension chạy lần lượt; bấm icon extension để xem.`,
      });
    } catch (err) {
      setQueueNote({
        tone: "error",
        text: err instanceof Error ? err.message : "Không thêm được vào hàng đợi.",
      });
    } finally {
      setQueueing(false);
    }
  }

  return (
    <Card data-testid="script-write-card">
      <div className={researchStyles.body}>
        <p className={researchStyles.hint}>
          {aiAvailable
            ? `${todo.length} beat chưa có lời thoại. AI viết lần lượt từng beat, hoặc bạn tự viết từng beat.`
            : `${todo.length} beat chưa có lời thoại. Chưa cấu hình AI của hệ thống: dùng extension, chạy thủ công từng beat bằng AI ngoài, hoặc tự viết.`}
        </p>
        {aiAvailable && (
          <AiModelSelect
            projectId={projectId}
            step="story"
            label="Model AI cho bước Viết kịch bản"
            testId="script-model"
          />
        )}
        <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
          <ChannelButton
            channel={ai.defaultChannel}
            label={
              state.stats.written === 0 ? "Viết lời thoại cho mọi beat" : "Viết các beat còn lại"
            }
            onRun={(c) => (c === "api" ? void script.generate("") : void queueAll())}
            disabled={busy || queueing || (!aiAvailable && !queue.available)}
            unavailable={{
              manual: "Chạy hàng loạt không làm thủ công được; chạy thủ công ở từng beat bên dưới",
              ...(queue.available
                ? {}
                : { extension: "Chưa có extension AI Bridge trên trình duyệt này" }),
              ...(aiAvailable ? {} : { api: "Chưa cấu hình AI của hệ thống" }),
            }}
            testId="script-generate"
          />
          <span className={researchStyles.hint}>
            Lời thoại đã có giữ nguyên; mỗi beat viết lần lượt, đọc beat liền trước, và lưu ngay
            không xem trước. Muốn xem trước thì chạy từng beat ở phần dưới.
          </span>
        </div>
        {queue.counts &&
          (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
            <p className={researchStyles.hint} data-testid="script-queue-state">
              Hàng đợi AI: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ ·{" "}
              {queue.counts.failed} lỗi{queue.counts.paused ? " · đang tạm dừng" : ""}.
            </p>
          )}
        {queueNote && (
          <p
            className={queueNote.tone === "error" ? researchStyles.error : researchStyles.hint}
            role={queueNote.tone === "error" ? "alert" : "status"}
            data-testid="script-queue-note"
          >
            {queueNote.text}
          </p>
        )}
      </div>
    </Card>
  );
}
