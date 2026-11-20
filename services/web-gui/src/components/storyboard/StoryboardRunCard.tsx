import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateStoryboardUnit,
  type ExternalRequest,
  type StoryboardState,
  type StoryboardUnit,
} from "../../api/client";
import { useDeepSeekQueue } from "../../hooks/useDeepSeekQueue";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, Dropdown, FormField, TextArea } from "../ui";
import { unitsToWrite, writeBlocker } from "./storyboardModel";

/** The request that has the AI write one unit's shots, described and not sent, so the extension can make it later. */
async function describeUnit(projectId: string, unit: StoryboardUnit): Promise<ExternalRequest> {
  try {
    await generateStoryboardUnit(projectId, unit.unit_id, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết shot cho unit này.");
}

interface StoryboardRunCardProps {
  projectId: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
}

/**
 * "Chạy AI" of step 11: the model the step runs on, the run that writes the
 * shots of every unit with none in sequence order on the system's AI, and the
 * same work queued on DeepSeek through the extension, one item per unit not
 * approved yet. Drawing assets and frames runs on the system and is not queued.
 */
export function StoryboardRunCard({ projectId, state, storyboard, readOnly }: StoryboardRunCardProps) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState("");
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/storyboard`)) void storyboard.reload();
  });
  const toWrite = unitsToWrite(state);
  const empty = state.empty_units;
  const blocker = writeBlocker(state);
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const modelOptions = [
    { value: "", label: `Theo mặc định của bước (${state.default_model || "chưa đặt"})` },
    ...state.models.map((m) => ({
      value: m.id,
      label: m.label || m.id,
      hint: m.local ? "chạy trên máy" : undefined,
    })),
  ];

  // One queue item per unit not approved, in sequence order, so each is written after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, unit] of toWrite.entries()) {
        const outcome = await queue.add({
          label: `Storyboard · ${unit.unit_id}`,
          project: projectId,
          request: await describeUnit(projectId, unit),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({
            tone: "error",
            text: `Dừng ở unit ${i + 1}/${toWrite.length}: ${outcome.message}`,
          });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} unit vào hàng đợi DeepSeek${already > 0 ? `, ${already} unit đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi unit một prompt và đọc shot của unit liền trước; bấm icon extension để xem.`,
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
    <Card data-testid="sb-run">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Chạy AI</h3>
          <span className={researchStyles.hint}>
            AI viết từng unit, mỗi unit một lượt, đọc shot của unit liền trước để giữ nhất quán
          </span>
        </div>
        <div className={researchStyles.actions}>
          <Dropdown
            aria-label="Model của bước"
            value={state.model}
            options={modelOptions}
            onChange={(m) => void storyboard.setModel(m)}
            disabled={disabled}
            data-testid="sb-model"
          />
          <Button
            variant="run"
            onClick={() => setOpen(true)}
            disabled={disabled || blocker !== "" || empty.length === 0}
            title={blocker || undefined}
            data-testid="sb-generate"
          >
            {empty.length > 0
              ? `Viết shot cho ${empty.length} unit còn trống`
              : "Mọi unit đã có shot"}
          </Button>
          {queue.available && toWrite.length > 0 && (
            <Button
              variant="run"
              onClick={() => void queueAll()}
              disabled={disabled || queueing || blocker !== ""}
              title={blocker || undefined}
              data-testid="sb-queue"
            >
              Chạy tất cả bằng DeepSeek ({toWrite.length})
            </Button>
          )}
        </div>
        {queueNote && (
          <p
            className={queueNote.tone === "ok" ? researchStyles.hint : researchStyles.error}
            role="status"
            data-testid="sb-queue-note"
          >
            {queueNote.text}
          </p>
        )}
        {queue.counts &&
          (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
            <p className={researchStyles.hint} data-testid="sb-queue-counts">
              Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ ·{" "}
              {queue.counts.failed} lỗi
              {queue.counts.paused ? " · đang tạm dừng" : ""}.
            </p>
          )}
        {blocker && (
          <p className={researchStyles.hint} data-testid="sb-run-blocker">
            {blocker}.
          </p>
        )}
        <p className={researchStyles.hint}>
          Vẽ tài sản và dựng khung chạy trên hệ thống (bộ vẽ AI và Remotion), không đi qua AI ngoài.
          Chỉ các lượt AI viết chữ (lập tài sản, viết shot, rà soát) dùng được AI ngoài và DeepSeek.
        </p>
      </div>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title="Viết shot cho mọi unit còn trống"
        hasInput
        testId="sb-generate-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant="run"
              onClick={() => {
                const n = note.trim();
                setOpen(false);
                setNote("");
                void storyboard.generateAll(n);
              }}
              data-testid="sb-generate-confirm"
            >
              Chạy {empty.length} unit
            </Button>
          </>
        }
      >
        <p>
          {empty.length} unit chưa có shot: {empty.join(", ")}. AI chạy lần lượt theo thứ tự chuỗi
          hình, mỗi unit một lượt, rồi dựng khung từng shot; unit đã có shot giữ nguyên.
        </p>
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            data-testid="sb-generate-note"
          />
        </FormField>
      </Dialog>
    </Card>
  );
}
