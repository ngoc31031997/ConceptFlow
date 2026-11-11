import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateVisualConceptUnit,
  type ExternalRequest,
  type VisualConceptState,
  type VisualConceptUnit,
} from "../../api/client";
import { useDeepSeekQueue } from "../../hooks/useDeepSeekQueue";
import type { VisualConceptControls } from "../../hooks/useVisualConcept";
import researchStyles from "../research/research.module.css";
import { Button, Card, CopyBox, Dialog, Dropdown } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { asList, asObj, asText } from "./conceptSchema";
import styles from "./visual.module.css";

/** The request that has the AI write one part, described and not sent, so the extension can make it later. */
async function describeWrite(projectId: string, unit: VisualConceptUnit): Promise<ExternalRequest> {
  try {
    await generateVisualConceptUnit(projectId, unit.key, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết phần này.");
}

/** Dark or light text, whichever reads better on `hex`. */
function inkOn(hex: string): string {
  const n = parseInt(hex.slice(1), 16);
  const [r, g, b] = [(n >> 16) & 255, (n >> 8) & 255, n & 255];
  return 0.299 * r + 0.587 * g + 0.114 * b > 150 ? "#0a0a0a" : "#ffffff";
}

/** The colour of each beat: the accent of its stretch of the colour journey. */
function beatColours(state: VisualConceptState): Record<string, string> {
  const color = state.units.find((u) => u.key === "color_system")?.content;
  if (!color) return {};
  const hexOf: Record<string, string> = {};
  for (const s of asList(color.palette).map(asObj)) hexOf[asText(s.token)] = asText(s.hex);
  const out: Record<string, string> = {};
  for (const j of asList(color.journey).map(asObj)) {
    for (const b of asList(j.beat_keys)) out[asText(b)] = hexOf[asText(j.accent_token)] ?? "";
  }
  return out;
}

interface VisualHeadCardProps {
  projectId: string;
  state: VisualConceptState;
  concept: VisualConceptControls;
  readOnly: boolean;
}

/**
 * The head of step 8: what the AI reads, the colour strip over the beats, the
 * machine checks of the whole concept, the model, and the actions on every part
 * at once (write the empty parts, rewrite all, queue on DeepSeek, approve the rest).
 */
export function VisualHeadCard({ projectId, state, concept, readOnly }: VisualHeadCardProps) {
  const [inputOpen, setInputOpen] = useState(false);
  const [rewriteOpen, setRewriteOpen] = useState(false);
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/visual-concept`)) void concept.reload();
  });
  const total = state.units.length;
  const notApproved = state.units.filter((u) => u.status !== "approved");
  const unwritten = state.units.filter((u) => !u.written);
  const approvable = state.units.filter(
    (u) => u.written && u.status === "pending" && u.problems.length === 0 && !u.stale,
  );
  const disabled = readOnly || concept.busy || concept.running !== null;
  const colours = beatColours(state);
  const totalDur = state.input.beats.reduce((s, b) => s + Math.max(b.dur, 1), 0);
  const openFindings = state.review.open_count;
  const locked = state.locked_version > 0;

  // One queue item per part not approved, in part order, so each is written after the one before it and reads it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, unit] of notApproved.entries()) {
        const outcome = await queue.add({
          label: `Phần ${unit.number} · ${unit.title}`,
          project: projectId,
          request: await describeWrite(projectId, unit),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({
            tone: "error",
            text: `Dừng ở phần ${i + 1}/${notApproved.length}: ${outcome.message}`,
          });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} phần vào hàng đợi DeepSeek${already > 0 ? `, ${already} phần đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi phần đọc các phần trước nó; bấm icon extension để xem.`,
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

  const approvedWritten = state.units.filter((u) => u.status === "approved").length;
  const modelOptions = [
    { value: "", label: `Theo mặc định của bước (${state.default_model || "chưa đặt"})` },
    ...state.models.map((m) => ({
      value: m.id,
      label: m.label || m.id,
      hint: m.local ? "chạy trên máy" : undefined,
    })),
  ];

  return (
    <Card data-testid="concept-head-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Ngôn ngữ hình ảnh</h3>
          {locked && state.draft.length === 0 ? (
            <span className={`${researchStyles.chip} ${researchStyles.chipOk}`}>
              ✓ Đã chốt v{state.locked_version}
            </span>
          ) : (
            <UnitStatusChip state="pending" />
          )}
          <span className={researchStyles.hint} data-testid="concept-head-count">
            {state.approved_count}/{total} phần đã duyệt · {openFindings} vấn đề cần xử lý
          </span>
        </div>
        <div className={researchStyles.actions}>
          <span>
            <b>Đầu vào:</b> Kịch bản v{state.input.script_version} đã chốt ·{" "}
            {state.input.beats.length} beat ·{" "}
            {state.input.research_skipped
              ? "bỏ qua Nghiên cứu"
              : `${state.input.facts_count} dữ kiện`}{" "}
            · {state.input.terms_count} thuật ngữ · {state.input.refs_count} mô tả tham chiếu
          </span>
          <span className={researchStyles.grow} />
          <Button
            variant="view"
            onClick={() => setInputOpen(true)}
            data-testid="concept-input-open"
          >
            Xem đầu vào của AI
          </Button>
        </div>
        {state.input.beats.length > 0 && (
          <>
            <div className={researchStyles.scroll}>
              <div
                className={styles.timeline}
                title="Màu theo hành trình câu chuyện"
                data-testid="concept-timeline"
              >
                {state.input.beats.map((b) => {
                  const bg = colours[b.key] || "var(--muted)";
                  return (
                    <div
                      key={b.key}
                      style={{
                        flex: Math.max(b.dur, 1) / totalDur,
                        background: bg,
                        color: colours[b.key] ? inkOn(colours[b.key]) : undefined,
                      }}
                    >
                      B{b.number} {b.fn}
                    </div>
                  );
                })}
              </div>
            </div>
            <span className={researchStyles.hint}>
              Dải màu theo hành trình câu chuyện, rộng theo thời lượng beat (đọc từ phần 2).
            </span>
          </>
        )}
        <div className={styles.checks} data-testid="concept-checks">
          {state.checks.map((c) => (
            <span
              key={c.id}
              className={`${styles.check} ${c.ok || c.accepted ? "" : styles.checkWarn}`}
              title={c.detail}
              data-testid={`concept-check-${c.id}`}
            >
              {c.ok ? "✓ " : c.accepted ? "✓ chấp nhận · " : "! "}
              {c.label}
            </span>
          ))}
          {state.fonts_error && (
            <span
              className={`${styles.check} ${styles.checkWarn}`}
              title={state.fonts_error}
              data-testid="concept-fonts-error"
            >
              ! Không lấy được danh sách font từ rendering: phần 5 chưa duyệt và chốt được (
              {state.fonts_error})
            </span>
          )}
        </div>
        {concept.running && (
          <div className={researchStyles.banner} role="status" data-testid="concept-running">
            <UnitStatusChip state="running" />{" "}
            {concept.running === "all"
              ? "AI đang viết các phần, lần lượt từ phần 1. Phần đã duyệt và trường bạn sửa tay giữ nguyên."
              : concept.running === "review"
                ? "AI đang rà soát cả bộ quy tắc."
                : "AI đang viết một phần."}{" "}
            <Button variant="ghost" onClick={concept.stop} data-testid="concept-stop">
              Dừng
            </Button>
          </div>
        )}
        <div className={researchStyles.actions}>
          <Dropdown
            aria-label="Model AI của bước"
            value={state.model}
            options={modelOptions}
            onChange={(m) => void concept.setModel(m)}
            disabled={disabled}
            data-testid="concept-model"
          />
          {unwritten.length > 0 ? (
            <Button
              variant="run"
              onClick={() => void concept.generateAll("", false)}
              disabled={disabled}
              data-testid="concept-generate"
            >
              AI viết {unwritten.length} phần còn trống
            </Button>
          ) : (
            <Button
              variant={approvedWritten > 0 ? "impact" : "run"}
              onClick={() => setRewriteOpen(true)}
              disabled={disabled}
              data-testid="concept-rewrite-all"
            >
              Viết lại cả bộ quy tắc
            </Button>
          )}
          {queue.available && notApproved.length > 0 && (
            <Button
              variant="run"
              onClick={() => void queueAll()}
              disabled={disabled || queueing}
              data-testid="concept-queue"
            >
              Chạy tất cả bằng DeepSeek ({notApproved.length})
            </Button>
          )}
          <span className={researchStyles.grow} />
          <Button
            variant="primary"
            onClick={() => void concept.approveAll()}
            disabled={disabled || approvable.length === 0}
            data-testid="concept-approve-all"
          >
            Duyệt tất cả phần còn lại
          </Button>
        </div>
        {queueNote && (
          <p
            className={queueNote.tone === "ok" ? researchStyles.hint : researchStyles.error}
            role="status"
            data-testid="concept-queue-note"
          >
            {queueNote.text}
          </p>
        )}
        {queue.counts &&
          (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
            <p className={researchStyles.hint} data-testid="concept-queue-counts">
              Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ ·{" "}
              {queue.counts.failed} lỗi
              {queue.counts.paused ? " · đang tạm dừng" : ""}.
            </p>
          )}
      </div>

      <Dialog
        open={inputOpen}
        onClose={() => setInputOpen(false)}
        title="Đầu vào AI đọc ở bước Ngôn ngữ hình ảnh"
        size="lg"
        testId="concept-input-dialog"
        footer={
          <Button variant="ghost" onClick={() => setInputOpen(false)}>
            Đóng
          </Button>
        }
      >
        {state.input_view.map((s) => (
          <div key={s.key} className={researchStyles.body}>
            <b>{s.label}</b>
            <CopyBox text={s.text} what={s.label}>
              <pre className={styles.pre}>{s.text}</pre>
            </CopyBox>
          </div>
        ))}
      </Dialog>

      <Dialog
        open={rewriteOpen}
        onClose={() => setRewriteOpen(false)}
        title="Viết lại cả bộ quy tắc?"
        size="sm"
        testId="concept-rewrite-dialog"
        footer={
          <>
            <Button variant="ghost" onClick={() => setRewriteOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant={approvedWritten > 0 ? "impact" : "run"}
              onClick={() => {
                setRewriteOpen(false);
                void concept.generateAll("", true);
              }}
              data-testid="concept-rewrite-confirm"
            >
              Viết lại {total} phần
            </Button>
          </>
        }
      >
        <p>AI viết lại {total} phần từ đầu, theo đầu vào hiện tại, lần lượt từ phần 1.</p>
        <p className={researchStyles.banner}>
          <b>Sẽ làm lại:</b> {approvedWritten} phần đã duyệt về “chờ duyệt”; trường bạn sửa tay giữ
          nguyên; các vấn đề rà soát cần rà soát lại.
          {locked
            ? ` Bản đã chốt v${state.locked_version} giữ nguyên tới khi bạn chốt bản mới.`
            : ""}
        </p>
      </Dialog>
    </Card>
  );
}
