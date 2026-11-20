import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateSequenceUnit,
  type ExternalRequest,
  type SequenceUnit,
  type VisualSequenceState,
} from "../../api/client";
import { useAiQueue } from "../../hooks/useAiQueue";
import type { VisualSequenceControls } from "../../hooks/useVisualSequence";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, Dropdown, FormField, TextArea } from "../ui";
import { motifsReady } from "./sequenceModel";

/** The request that has the AI write one unit, described and not sent, so the extension can make it later. */
async function describeUnit(projectId: string, unit: SequenceUnit): Promise<ExternalRequest> {
  try {
    await generateSequenceUnit(projectId, unit.unit_id, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết chuỗi cho unit này.");
}

interface SequenceRunCardProps {
  projectId: string;
  state: VisualSequenceState;
  sequence: VisualSequenceControls;
  readOnly: boolean;
}

/**
 * Section 3 of step 10: the model the step runs on, then the run that writes
 * every beat with no sequence yet in plan order, on the system's AI or queued
 * on an outside AI through the extension.
 */
export function SequenceRunCard({ projectId, state, sequence, readOnly }: SequenceRunCardProps) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState("");
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const queue = useAiQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/visual-sequence`)) void sequence.reload();
  });
  // Units with no item or one still waiting for review, in plan order.
  const notApproved = state.beats
    .flatMap((b) => b.units)
    .filter((u) => u.item?.status !== "approved");
  const empty = state.beats.flatMap((b) => b.units).filter((u) => u.item === null);
  const disabled = readOnly || sequence.busy || sequence.running !== null;
  const ready = motifsReady(state);
  const modelOptions = [
    { value: "", label: `Theo mặc định của bước (${state.default_model || "chưa đặt"})` },
    ...state.models.map((m) => ({
      value: m.id,
      label: m.label || m.id,
      hint: m.local ? "chạy trên máy" : undefined,
    })),
  ];

  // One queue item per unit not approved, in plan order, so each is written after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, unit] of notApproved.entries()) {
        const outcome = await queue.add({
          label: `Chuỗi hình · ${unit.unit_id}`,
          project: projectId,
          request: await describeUnit(projectId, unit),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({
            tone: "error",
            text: `Dừng ở unit ${i + 1}/${notApproved.length}: ${outcome.message}`,
          });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} unit vào hàng đợi AI${already > 0 ? `, ${already} unit đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi unit một prompt và đọc các unit trước nó; bấm icon extension để xem.`,
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
    <Card data-testid="seq-run">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>3. Chạy AI</h3>
          <span className={researchStyles.hint}>AI viết chuỗi từng unit, mỗi unit một lượt</span>
        </div>
        <div className={researchStyles.actions}>
          <Dropdown
            aria-label="Model AI của bước"
            value={state.model}
            options={modelOptions}
            onChange={(m) => void sequence.setModel(m)}
            disabled={disabled}
            data-testid="seq-model"
          />
          <Button
            variant="run"
            onClick={() => setOpen(true)}
            disabled={disabled || !ready || empty.length === 0}
            title={ready ? undefined : "Duyệt motif trước khi AI viết chuỗi"}
            data-testid="seq-generate"
          >
            {empty.length > 0
              ? `Viết chuỗi cho ${empty.length} unit còn trống`
              : "Mọi unit đã có chuỗi"}
          </Button>
          {queue.available && notApproved.length > 0 && (
            <Button
              variant="run"
              onClick={() => void queueAll()}
              disabled={disabled || queueing || !ready}
              title={ready ? undefined : "Duyệt motif trước khi AI viết chuỗi"}
              data-testid="seq-queue"
            >
              Thêm tất cả vào hàng đợi AI ({notApproved.length})
            </Button>
          )}
        </div>
        {queueNote && (
          <p
            className={queueNote.tone === "ok" ? researchStyles.hint : researchStyles.error}
            role="status"
            data-testid="seq-queue-note"
          >
            {queueNote.text}
          </p>
        )}
        {queue.counts &&
          (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
            <p className={researchStyles.hint} data-testid="seq-queue-counts">
              Hàng đợi AI: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ ·{" "}
              {queue.counts.failed} lỗi
              {queue.counts.paused ? " · đang tạm dừng" : ""}.
            </p>
          )}
        {!ready && (
          <p className={researchStyles.hint} data-testid="seq-run-needs-motifs">
            Duyệt motif ở mục 2 trước khi AI viết chuỗi.
          </p>
        )}
        <p className={researchStyles.hint}>
          Mỗi lượt AI viết đúng một unit: đọc các unit khác của beat, unit liền trước và liền sau
          (để viết chuyển cảnh nối), motif đã duyệt, nhịp chung và ngôn ngữ chuyển động của bước 8.
          Nhịp và chuyển cảnh AI tự viết, không giới hạn trong danh sách nào.
        </p>
      </div>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title="Viết chuỗi cho mọi unit còn trống"
        hasInput
        testId="seq-generate-dialog"
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
                void sequence.generateAll(n);
              }}
              data-testid="seq-generate-confirm"
            >
              Chạy {empty.length} unit
            </Button>
          </>
        }
      >
        <p>
          {empty.length} unit chưa có chuỗi: {empty.map((u) => u.unit_id).join(", ")}. AI chạy lần
          lượt theo thứ tự kế hoạch hình, mỗi unit một lượt; unit đã có chuỗi giữ nguyên.
        </p>
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            data-testid="seq-generate-note"
          />
        </FormField>
      </Dialog>
    </Card>
  );
}
