import { useEffect, useRef, useState } from "react";
import {
  cancelAiRun,
  describeAiRequest,
  drawStoryboardAsset,
  fixStoryboardAsset,
  redrawStoryboardAsset,
  type AiChannel,
  type AiRun,
  type AssetDrawFix,
  type StoryboardAsset,
} from "../../api/client";
import { AiRunDialog } from "../airun/AiRunDialog";
import { isOpen, useAiRuns } from "../airun/AiRunsContext";
import airStyles from "../airun/AiRun.module.css";
import researchStyles from "../research/research.module.css";
import { Button } from "../ui";
import { AssetDrawErrorView, StoryboardRunPreview } from "./StoryboardRunPreview";

/** The three AI turns that draw an asset: first drawing, redrawing with a note, and resending broken code with its error. */
export type AssetDrawKind = "draw" | "redraw" | "fix";

const KINDS: AssetDrawKind[] = ["draw", "redraw", "fix"];

/** The key of the run of one drawing turn of an asset: the path of the step request it replays. */
export function assetDrawRunPath(projectId: string, assetId: string, kind: AssetDrawKind): string {
  return `/v1/projects/${projectId}/storyboard/assets/${assetId}/${kind}`;
}

/** Whether the turn a run replays was a redraw (a resend keeps the flag of the turn it resends in its body). */
function isRedrawRun(run: AiRun): boolean {
  if (run.path.endsWith("/redraw")) return true;
  if (!run.path.endsWith("/fix")) return false;
  try {
    return (JSON.parse(run.body) as { redraw?: boolean }).redraw === true;
  } catch {
    return false;
  }
}

/** The body that sends a refused run's code back to the AI with its error; null when the run has no structured error. */
function fixBodyOf(run: AiRun | undefined): AssetDrawFix | null {
  if (!run?.error_detail) return null;
  const kept = (run.preview as { code?: unknown } | null)?.code;
  return {
    code: typeof kept === "string" && kept !== "" ? kept : run.reply,
    error: run.error_detail,
    redraw: isRedrawRun(run),
  };
}

interface ErrorPanelProps {
  run: AiRun;
  /** Redrawing needs a drawing to start from. */
  canRedraw: boolean;
  onFix: (run: AiRun) => Promise<void>;
  onRedraw: (run: AiRun) => Promise<void>;
  onDrop: (run: AiRun) => Promise<void>;
  testId: string;
}

/**
 * A refused drawing: the structured error and the three ways on. Nothing is sent again by itself on
 * any channel; the Creator picks the next turn.
 */
function DrawErrorPanel({ run, canRedraw, onFix, onRedraw, onDrop, testId }: ErrorPanelProps) {
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);

  async function act(call: (r: AiRun) => Promise<void>) {
    setBusy(true);
    setFailure(null);
    try {
      await call(run);
    } catch (err) {
      setFailure(err instanceof Error ? err.message : "Không làm được việc này.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={airStyles.alert} role="alert" data-testid={`${testId}-error`}>
      {run.error_detail && <AssetDrawErrorView error={run.error_detail} testId={`${testId}-draw-error`} />}
      <p className={airStyles.previewSub}>
        Hệ thống không tự gửi lại. Chọn bước tiếp theo, hoặc sửa câu trả lời ở tab Kết quả rồi bấm Đọc kết quả.
      </p>
      {failure && (
        <p className={researchStyles.error} role="alert" data-testid={`${testId}-draw-error-failure`}>
          {failure}
        </p>
      )}
      <div className={airStyles.row}>
        <Button variant="run" onClick={() => void act(onFix)} disabled={busy} data-testid={`${testId}-fix`}>
          Gửi lại kèm lỗi
        </Button>
        {canRedraw && (
          <Button variant="ghost" onClick={() => void act(onRedraw)} disabled={busy} data-testid={`${testId}-redraw-note`}>
            Vẽ lại kèm ghi chú
          </Button>
        )}
        <Button variant="ghost" onClick={() => void act(onDrop)} disabled={busy} data-testid={`${testId}-drop-error`}>
          Bỏ
        </Button>
      </div>
    </div>
  );
}

interface AssetDrawDialogProps {
  projectId: string;
  asset: StoryboardAsset;
  /** The turn the dialog opens on, unless another turn of this asset already has an open run. */
  initialKind: AssetDrawKind;
  /** Starts the first drawing at once on this channel (the "Vẽ tất cả" queue) instead of showing the form. */
  autoChannel?: AiChannel;
  /** The screen loads what an applied drawing stored. */
  onApplied: () => void | Promise<void>;
  onClose: () => void;
  /**
   * Set by the queue: called when this asset's run was applied or dropped, so the next asset can
   * open. Closing the dialog without either does not call it and so ends the queue.
   */
  onSettled?: () => void;
  testId: string;
}

/**
 * One asset's drawing on an AI channel (docs/ux-ui-design-rules.md, rule 10): the first drawing,
 * redrawing with a note, and resending refused code with its error. It follows one run at a time;
 * choosing a different turn cancels the run it replaces so no run stays open behind it. The trial
 * render is previewed and nothing is stored before "Áp dụng kết quả".
 */
export function AssetDrawDialog({
  projectId,
  asset,
  initialKind,
  autoChannel,
  onApplied,
  onClose,
  onSettled,
  testId,
}: AssetDrawDialogProps) {
  const ai = useAiRuns();
  const id = asset.asset_id;
  const hasDrawing = asset.preview_path !== "";
  const [kind, setKind] = useState<AssetDrawKind>(
    () => [initialKind, ...KINDS].find((k) => isOpen(ai.runs[assetDrawRunPath(projectId, id, k)])) ?? initialKind,
  );
  const [starting, setStarting] = useState(autoChannel !== undefined);
  const [startError, setStartError] = useState<string | null>(null);
  const started = useRef(false);
  const settled = useRef(false);

  const titles: Record<AssetDrawKind, string> = {
    draw: `Vẽ ${asset.name}`,
    redraw: `Vẽ lại ${asset.name} kèm ghi chú`,
    fix: `Gửi lại ${asset.name} kèm lỗi`,
  };

  const describe = {
    draw: () => describeAiRequest((external) => drawStoryboardAsset(projectId, id, { external })),
    redraw: (note: string) =>
      describeAiRequest((external) => redrawStoryboardAsset(projectId, id, note, { external })),
    fix: () => {
      const body = fixBodyOf(ai.runs[assetDrawRunPath(projectId, id, "fix")]);
      if (!body) throw new Error("Không còn lượt lỗi nào để gửi lại; hãy vẽ lại kèm ghi chú hoặc vẽ từ đầu.");
      return describeAiRequest((external) => fixStoryboardAsset(projectId, id, body, { external }));
    },
  };

  // The queue starts this asset's first drawing at once, unless one of its turns already has an open run.
  useEffect(() => {
    if (autoChannel === undefined || started.current) return;
    started.current = true;
    if (KINDS.some((k) => isOpen(ai.runs[assetDrawRunPath(projectId, id, k)]))) {
      setStarting(false);
      return;
    }
    describe.draw().then(
      (request) => ai.start(request, autoChannel, titles.draw),
    ).catch((err: unknown) => {
      setStartError(err instanceof Error ? err.message : "Không bắt đầu được lượt vẽ.");
    }).finally(() => setStarting(false));
    // Only on opening.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  /** Cancels a run the Creator moves on from. */
  async function cancelRun(run: AiRun) {
    ai.put(await cancelAiRun(ai.service, ai.projectId, run.id));
  }

  function settle() {
    settled.current = true;
    if (onSettled) onSettled();
    else onClose();
  }

  // A resend is a new run on the failed run's channel; the failed one is cancelled once the new one is open.
  async function resend(run: AiRun) {
    const body = fixBodyOf(run);
    if (!body) throw new Error("Lượt này không có lỗi có cấu trúc để gửi lại.");
    const request = await describeAiRequest((external) => fixStoryboardAsset(projectId, id, body, { external }));
    const next = await ai.start(request, run.channel, titles.fix);
    // A new run on the same path replaces the old one on the server; any other path needs the cancel.
    if (next.path !== run.path) await cancelRun(run);
    setKind("fix");
  }

  async function redrawWithNote(run: AiRun) {
    await cancelRun(run);
    setKind("redraw");
  }

  async function drop(run: AiRun) {
    await cancelRun(run);
    settle();
  }

  if (starting) return null;
  return (
    <AiRunDialog
      key={kind}
      open
      onClose={() => {
        if (!(settled.current && onSettled)) onClose();
      }}
      title={titles[kind]}
      runPath={assetDrawRunPath(projectId, id, kind)}
      describe={describe[kind]}
      noNote={kind !== "redraw"}
      impact={
        <>
          <p className={researchStyles.hint} data-testid={`${testId}-impact`}>
            {kind === "redraw"
              ? `AI vẽ lại từ code đang có; phiên bản hình tăng một. ${
                  asset.used_by.length > 0
                    ? `Khung của các shot đang dùng (${asset.used_by.join(", ")}) cũ đi và phải vẽ lại.`
                    : "Chưa shot nào dùng tài sản này."
                }`
              : kind === "fix"
                ? "AI nhận lại code hỏng cùng lỗi có số dòng và sửa. Chưa lưu gì cho tới khi bạn áp dụng."
                : "AI vẽ tài sản bằng code theo kit của dự án. Hình dựng thử hiện ra để xem; chưa lưu gì cho tới khi bạn áp dụng."}
            {" "}Code lỗi thì mọi kênh đều dừng, không tự gửi lại.
          </p>
          {startError && (
            <p className={researchStyles.error} role="alert" data-testid={`${testId}-start-error`}>
              {startError}
            </p>
          )}
        </>
      }
      renderPreview={(p) => <StoryboardRunPreview kind="asset_draw" preview={p} />}
      renderError={(run) => (
        <DrawErrorPanel
          run={run}
          canRedraw={hasDrawing}
          onFix={resend}
          onRedraw={redrawWithNote}
          onDrop={drop}
          testId={testId}
        />
      )}
      onApplied={async () => {
        await onApplied();
        settled.current = true;
        onSettled?.();
      }}
      onDropped={() => {
        if (kind === "fix") setKind(hasDrawing ? "redraw" : "draw");
        settled.current = true;
        onSettled?.();
      }}
      testId={testId}
    />
  );
}
