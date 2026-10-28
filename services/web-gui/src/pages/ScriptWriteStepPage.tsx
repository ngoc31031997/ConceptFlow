import { useContext, useEffect, useState } from "react";
import { Navigate, useNavigate, useParams, useSearchParams } from "react-router-dom";
import { ExternalRequestDescribed, generateScript, type ExternalRequest, type ScriptBeat, type ScriptState } from "../api/client";
import { AiModelSelect } from "../components/AiModelSelect";
import { AppShell } from "../components/AppShell";
import researchStyles from "../components/research/research.module.css";
import { ScriptBeatCard } from "../components/script/ScriptBeatCard";
import { ScriptOverviewCard } from "../components/script/ScriptOverviewCard";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card } from "../components/ui";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import { useDeepSeekQueue } from "../hooks/useDeepSeekQueue";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useScript, type ScriptControls } from "../hooks/useScript";
import { VIEW_SCRIPT, beatsRoute, lockRoute, readOnlyReason } from "../utils/flow";

const SUBTITLE =
  "Lời thoại chia theo beat. Nghe thử từng câu bằng giọng đã chọn; máy chỉ ra câu dài, câu khó nghe và chỗ nối chưa mượt.";

const ROLE =
  "Viết toàn bộ lời dẫn theo beat: chuyển tiếp tự nhiên, đủ rõ mà không dài dòng, đúng nhịp với thời lượng mục tiêu. Nghe thử bằng giọng đã chọn.";

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

/** Where the step stands and what comes next, for the line under the heading. */
function stripOf(state: ScriptState, running: ScriptControls["running"]): StepStripProps {
  if (!state.beatsApproved) {
    return { tone: "todo", label: "Chưa mở bước", message: "Duyệt beat sheet ở bước Beat sheet trước." };
  }
  if (running !== null) return { tone: "run", label: "AI đang viết", message: "Lời thoại bạn đã viết giữ nguyên." };
  if (state.stats.written === 0) {
    return { tone: "todo", label: "Chưa có lời thoại", message: "AI viết lời thoại từng beat, hoặc bạn tự viết." };
  }
  const done = state.beats.filter((b) => b.written && b.status === "approved").length;
  if (state.status === "approved") return { tone: "ok", label: "Kịch bản đã duyệt", message: "Tiếp theo: rà soát và chốt." };
  return {
    tone: "wait",
    label: `Chờ bạn duyệt · ${done}/${state.beats.length} beat`,
    message: "Beat đã duyệt giữ nguyên khi beat khác làm lại.",
  };
}

/**
 * Writing the script. The AI writes the dialogue of
 * each beat from the approved beat sheet, one beat after another; the Creator
 * edits, redoes and approves each beat and can listen to any line, a beat or the
 * whole script in the project's voice. Checks over the opening, the
 * closing, long or hard-to-hear lines and the joins between beats only warn.
 * With the AI Bridge extension, one click queues every beat without dialogue on DeepSeek.
 */
export function ScriptWriteStepPage() {
  const params = useParams<{ id?: string }>();
  const draft = useContext(ProjectDraftContext);
  const flow = useProjectFlow();
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const projectId = params.id ?? draft.projectId;
  const script = useScript(projectId, !!projectId);
  // A line opened from another screen (the lock step's "Mở câu đó") is marked and brought into view.
  const highlight = useSearchParams()[0].get("line");
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/script`)) script.reload();
  });
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const loaded = script.state !== null;
  useEffect(() => {
    if (highlight && loaded) document.getElementById(`ln-${highlight}`)?.scrollIntoView({ block: "center" });
  }, [highlight, loaded]);

  if (!projectId) return <Navigate to="/" replace />;

  const state = script.state;
  const locked = !flow.editable;
  const aiAvailable = llm?.enabled === true;

  // One queue item per beat without dialogue, in the order of the sheet, so each beat is
  // written after the one before it and reads it.
  async function queueAll(todo: ScriptBeat[]) {
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
          setQueueNote({ tone: "error", text: `Dừng ở beat ${i + 1}/${todo.length}: ${outcome.message}` });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} beat vào hàng đợi DeepSeek${already > 0 ? `, ${already} beat đã có trong hàng đợi` : ""}. Extension chạy lần lượt; bấm icon extension để xem.`,
      });
    } catch (err) {
      setQueueNote({ tone: "error", text: err instanceof Error ? err.message : "Không thêm được vào hàng đợi." });
    } finally {
      setQueueing(false);
    }
  }

  function content(s: ScriptState) {
    const busy = locked || script.busy || script.running !== null;
    const unwrittenBeats = s.beats.filter((b) => !b.written);
    const unwritten = unwrittenBeats.length;
    const pending = s.beats.filter((b) => b.written && b.status === "pending").length;
    return (
      <>
        {!s.beatsApproved && (
          <p className={researchStyles.hint} data-testid="script-not-open">
            Duyệt beat sheet ở bước Beat sheet trước.
          </p>
        )}
        {s.beatsApproved && unwritten > 0 && (
          <Card data-testid="script-write-card">
            <div className={researchStyles.body}>
              <p className={researchStyles.hint}>
                {aiAvailable
                  ? `${unwritten} beat chưa có lời thoại. AI viết lần lượt từng beat, hoặc bạn tự viết từng beat.`
                  : `${unwritten} beat chưa có lời thoại. Chưa cấu hình AI nên bạn tự viết từng beat.`}
              </p>
              {aiAvailable && <AiModelSelect projectId={projectId} step="story" label="Model AI cho bước Viết kịch bản" testId="script-model" />}
              {aiAvailable && (
                <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                  <Button variant="run" onClick={() => void script.generate("", "")} disabled={busy} data-testid="script-generate">
                    {s.stats.written === 0 ? "AI viết lời thoại" : "AI viết các beat còn lại"}
                  </Button>
                  {queue.available && (
                    <Button variant="run" onClick={() => void queueAll(unwrittenBeats)} disabled={busy || queueing} data-testid="script-queue-all">
                      {queueing ? "Đang thêm…" : `Chạy tất cả bằng DeepSeek (${unwritten})`}
                    </Button>
                  )}
                  <span className={researchStyles.hint}>Lời thoại đã có giữ nguyên; mỗi beat viết lần lượt, đọc beat liền trước.</span>
                </div>
              )}
              {queue.counts && (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
                <p className={researchStyles.hint} data-testid="script-queue-state">
                  Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ · {queue.counts.failed} lỗi{queue.counts.paused ? " · đang tạm dừng" : ""}.
                </p>
              )}
              {queueNote && (
                <p className={queueNote.tone === "error" ? researchStyles.error : researchStyles.hint} role={queueNote.tone === "error" ? "alert" : "status"} data-testid="script-queue-note">
                  {queueNote.text}
                </p>
              )}
            </div>
          </Card>
        )}
        {s.beatsApproved && s.stats.written > 0 && <ScriptOverviewCard projectId={projectId} state={s} player={script.player} disabled={locked} onReload={script.reload} />}
        {s.beatsApproved &&
          s.beats.map((beat) => <ScriptBeatCard key={beat.key} projectId={projectId} beat={beat} script={script} aiAvailable={aiAvailable} locked={locked} highlight={highlight} />)}
        {s.beatsApproved && s.stats.written > 0 && (
          <div className={`${researchStyles.actions} ${researchStyles.actionsEnd}`}>
            <Button onClick={() => void script.approveAll()} disabled={busy || pending === 0} data-testid="script-approve-all">
              Duyệt tất cả beat còn lại
            </Button>
          </div>
        )}
        {script.actionError && (
          <p className={researchStyles.error} role="alert" data-testid="script-error">
            {script.actionError}
          </p>
        )}
      </>
    );
  }

  return (
    <div data-testid="script-step-page">
      <AppShell viewStep={VIEW_SCRIPT} wide title="Viết kịch bản" subtitle={SUBTITLE} role={ROLE} strip={state ? stripOf(state, script.running) : undefined}>
        {locked && flow.status && (
          <p className={researchStyles.banner} role="status" data-testid="script-read-only">
            {readOnlyReason(flow.status)}
          </p>
        )}
        {!state && script.loadError && (
          <>
            <p className={researchStyles.error} role="alert" data-testid="script-load-error">
              Không tải được bước Viết kịch bản: {script.loadError}
            </p>
            <div className={researchStyles.actions}>
              <Button variant="ghost" onClick={script.reload} data-testid="script-reload">
                Tải lại
              </Button>
            </div>
          </>
        )}
        {!state && !script.loadError && (
          <p className={researchStyles.hint} role="status">
            Đang tải…
          </p>
        )}
        {state && <div className={researchStyles.body}>{content(state)}</div>}
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={state?.status === "approved" ? "Kịch bản đã duyệt. Sang bước Rà soát và chốt." : "Duyệt lời thoại của mọi beat để sang bước Rà soát và chốt."}
        onBack={() => navigate(beatsRoute(projectId))}
        onNext={() => navigate(lockRoute(projectId))}
        nextDisabled={state?.status !== "approved"}
        nextLabel="Lưu và sang rà soát và chốt"
        nextTestId="script-next"
      />
    </div>
  );
}
