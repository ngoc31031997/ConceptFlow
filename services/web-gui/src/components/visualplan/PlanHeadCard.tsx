import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateVisualPlanBeat,
  type ExternalRequest,
  type VisualPlanBeat,
  type VisualPlanState,
} from "../../api/client";
import { useDeepSeekQueue } from "../../hooks/useDeepSeekQueue";
import type { VisualPlanControls } from "../../hooks/useVisualPlan";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, Dropdown, FormField, TextArea } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { approvable, autoFixable } from "./planModel";

/** The request that has the AI plan one beat, described and not sent, so the extension can make it later. */
async function describeBeat(projectId: string, beat: VisualPlanBeat): Promise<ExternalRequest> {
  try {
    await generateVisualPlanBeat(projectId, beat.key, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu lập kế hoạch cho beat này.");
}

interface PlanHeadCardProps {
  projectId: string;
  state: VisualPlanState;
  plan: VisualPlanControls;
  readOnly: boolean;
}

/**
 * The head of step 9: the inputs the AI reads, the model, the actions on every
 * beat at once (plan the empty beats, queue them on DeepSeek, approve the rest)
 * and the state of a run in flight.
 */
export function PlanHeadCard({ projectId, state, plan, readOnly }: PlanHeadCardProps) {
  const [allOpen, setAllOpen] = useState(false);
  const [note, setNote] = useState("");
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/visual-plan`)) void plan.reload();
  });
  const disabled = readOnly || plan.busy || plan.running !== null;
  const empty = state.beats.filter((b) => !b.written);
  const notApproved = state.beats.filter((b) => b.units.length === 0 || b.units.some((u) => u.status !== "approved"));
  const ready = approvable(state);
  const fixable = autoFixable(state);
  const locked = state.locked_version > 0;
  const modelOptions = [
    { value: "", label: `Theo mặc định của bước (${state.default_model || "chưa đặt"})` },
    ...state.models.map((m) => ({ value: m.id, label: m.label || m.id, hint: m.local ? "chạy trên máy" : undefined })),
  ];

  // One queue item per beat not approved, in beat order, so each is planned after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, beat] of notApproved.entries()) {
        const outcome = await queue.add({
          label: `Beat ${beat.number} · ${beat.title}`,
          project: projectId,
          request: await describeBeat(projectId, beat),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({ tone: "error", text: `Dừng ở beat ${i + 1}/${notApproved.length}: ${outcome.message}` });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} beat vào hàng đợi DeepSeek${already > 0 ? `, ${already} beat đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi beat đọc các beat trước nó; bấm icon extension để xem.`,
      });
    } catch (err) {
      setQueueNote({ tone: "error", text: err instanceof Error ? err.message : "Không thêm được vào hàng đợi." });
    } finally {
      setQueueing(false);
    }
  }

  const runningText =
    plan.running === "all"
      ? "AI đang lập kế hoạch các beat còn trống, lần lượt theo thứ tự beat."
      : plan.running === "review"
        ? "AI đang rà soát cả kế hoạch hình."
        : "AI đang lập kế hoạch một beat. Unit bạn đã sửa tay giữ nguyên.";

  return (
    <Card data-testid="plan-head-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Kế hoạch hình</h3>
          {locked && state.draft.length === 0 ? (
            <span className={`${researchStyles.chip} ${researchStyles.chipOk}`}>✓ Đã chốt v{state.locked_version}</span>
          ) : (
            <UnitStatusChip state="pending" />
          )}
          <span className={researchStyles.hint} data-testid="plan-head-count">
            {state.units_approved}/{state.units_total} unit đã duyệt · {state.review.open_count} vấn đề cần xử lý
          </span>
        </div>
        <div data-testid="plan-input">
          <b>Đầu vào đã chốt:</b> Kịch bản v{state.input.script_version} · Ngôn ngữ hình ảnh v{state.input.concept_version} ·{" "}
          {state.beats.length} beat · {state.input.facts_count} dữ kiện · {state.input.terms_count} thuật ngữ ·{" "}
          {state.input.rule_refs_count} quy tắc hình
        </div>
        {plan.running && (
          <div className={researchStyles.banner} role="status" data-testid="plan-running">
            <UnitStatusChip state="running" /> {runningText}{" "}
            <Button variant="ghost" onClick={plan.stop} data-testid="plan-stop">
              Dừng
            </Button>
          </div>
        )}
        <div className={researchStyles.actions}>
          <Dropdown
            aria-label="Model AI của bước"
            value={state.model}
            options={modelOptions}
            onChange={(m) => void plan.setModel(m)}
            disabled={disabled}
            data-testid="plan-model"
          />
          <Button
            variant="run"
            onClick={() => setAllOpen(true)}
            disabled={disabled || empty.length === 0}
            data-testid="plan-generate"
          >
            {empty.length > 0 ? `AI lập ${empty.length} beat còn trống` : "Mọi beat đã có kế hoạch"}
          </Button>
          {queue.available && notApproved.length > 0 && (
            <Button variant="run" onClick={() => void queueAll()} disabled={disabled || queueing} data-testid="plan-queue">
              Chạy tất cả bằng DeepSeek ({notApproved.length})
            </Button>
          )}
          <span className={researchStyles.grow} />
          {fixable.length > 0 && (
            <Button variant="view" onClick={() => void plan.autoFix()} disabled={disabled} data-testid="plan-auto-fix">
              Tự sửa lỗi hiển nhiên · {fixable.length}
            </Button>
          )}
          <Button
            variant="primary"
            onClick={() => void plan.approveAll()}
            disabled={disabled || ready.length === 0}
            data-testid="plan-approve-all"
          >
            Duyệt tất cả còn lại{ready.length > 0 ? ` · ${ready.length}` : ""}
          </Button>
        </div>
        {state.auto_fixes.length > 0 && (
          <ul className={researchStyles.hint} data-testid="plan-auto-fix-list">
            {state.auto_fixes.map((line) => (
              <li key={line}>{line}</li>
            ))}
          </ul>
        )}
        {queueNote && (
          <p
            className={queueNote.tone === "ok" ? researchStyles.hint : researchStyles.error}
            role="status"
            data-testid="plan-queue-note"
          >
            {queueNote.text}
          </p>
        )}
        {queue.counts && (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
          <p className={researchStyles.hint} data-testid="plan-queue-counts">
            Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ · {queue.counts.failed} lỗi
            {queue.counts.paused ? " · đang tạm dừng" : ""}.
          </p>
        )}
      </div>

      <Dialog
        open={allOpen}
        onClose={() => setAllOpen(false)}
        title="Lập kế hoạch cho mọi beat còn trống"
        hasInput
        testId="plan-generate-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setAllOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant="run"
              onClick={() => {
                const n = note.trim();
                setAllOpen(false);
                setNote("");
                void plan.generateAll(n);
              }}
              data-testid="plan-generate-confirm"
            >
              Chạy {empty.length} beat
            </Button>
          </>
        }
      >
        <p>
          {empty.length} beat chưa có unit: {empty.map((b) => b.key).join(", ")}. AI chạy lần lượt theo thứ tự beat; beat
          đã có unit không bị đụng.
        </p>
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea value={note} onChange={(e) => setNote(e.target.value)} rows={3} data-testid="plan-generate-note" />
        </FormField>
      </Dialog>
    </Card>
  );
}
