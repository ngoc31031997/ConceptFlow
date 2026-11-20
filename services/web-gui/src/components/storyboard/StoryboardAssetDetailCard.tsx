import { useState } from "react";
import {
  ExternalRequestDescribed,
  generateStoryboardAssetDetail,
  type ExternalRequest,
  type StoryboardAsset,
  type StoryboardState,
} from "../../api/client";
import { useDeepSeekQueue } from "../../hooks/useDeepSeekQueue";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, FormField, TextArea } from "../ui";
import { assetsWithoutDetail } from "./storyboardModel";

/** The request that has the AI write one asset's detail, described and not sent, so the extension can make it later. */
async function describeDetail(projectId: string, asset: StoryboardAsset): Promise<ExternalRequest> {
  try {
    await generateStoryboardAssetDetail(projectId, asset.asset_id, "", { external: { mode: "describe" } });
  } catch (err) {
    if (err instanceof ExternalRequestDescribed) return err.request;
    throw err;
  }
  throw new Error("Không dựng được yêu cầu viết chi tiết cho tài sản này.");
}

interface StoryboardAssetDetailCardProps {
  projectId: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
}

/**
 * The second phase of the asset list: writing the detail of each asset the list named.
 * One run on the system's AI writes every asset without detail, one turn per asset; the
 * same work can be queued on DeepSeek through the extension, one item per asset without
 * detail. One asset's detail is written from its own card.
 */
export function StoryboardAssetDetailCard({
  projectId,
  state,
  storyboard,
  readOnly,
}: StoryboardAssetDetailCardProps) {
  const [open, setOpen] = useState(false);
  const [note, setNote] = useState("");
  const [queueing, setQueueing] = useState(false);
  const [queueNote, setQueueNote] = useState<{ tone: "ok" | "error"; text: string } | null>(null);
  const queue = useDeepSeekQueue((path) => {
    if (path.startsWith(`/v1/projects/${projectId}/storyboard`)) void storyboard.reload();
  });
  const missing = assetsWithoutDetail(state);
  const total = state.assets.items.length;
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;

  // One queue item per asset without detail, in list order, so each reads the details written before it.
  async function queueAll() {
    setQueueing(true);
    setQueueNote(null);
    let added = 0;
    let already = 0;
    try {
      for (const [i, asset] of missing.entries()) {
        const outcome = await queue.add({
          label: `Chi tiết tài sản · ${asset.asset_id}`,
          project: projectId,
          request: await describeDetail(projectId, asset),
          modes: { deepThink: true, search: false },
        });
        if (!outcome.ok) {
          setQueueNote({
            tone: "error",
            text: `Dừng ở tài sản ${i + 1}/${missing.length}: ${outcome.message}`,
          });
          return;
        }
        if (outcome.added) added += 1;
        else already += 1;
      }
      setQueueNote({
        tone: "ok",
        text: `Đã thêm ${added} tài sản vào hàng đợi DeepSeek${already > 0 ? `, ${already} tài sản đã có trong hàng đợi` : ""}. Extension chạy lần lượt, mỗi tài sản một prompt; bấm icon extension để xem.`,
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
    <Card data-testid="sb-detail-run">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>Viết chi tiết tài sản</h3>
          <span className={researchStyles.hint} data-testid="sb-detail-counts">
            {state.assets.with_detail}/{total} tài sản đã có chi tiết · phải có chi tiết mới vẽ được
          </span>
        </div>
        <div className={researchStyles.actions}>
          <Button
            variant="run"
            onClick={() => setOpen(true)}
            disabled={disabled || missing.length === 0}
            title={total === 0 ? "Lập danh sách tài sản trước" : undefined}
            data-testid="sb-detail-generate"
          >
            {missing.length > 0
              ? `Viết chi tiết cho ${missing.length} tài sản`
              : "Mọi tài sản đã có chi tiết"}
          </Button>
          {queue.available && missing.length > 0 && (
            <Button
              variant="run"
              onClick={() => void queueAll()}
              disabled={disabled || queueing}
              data-testid="sb-detail-queue"
            >
              Chạy tất cả bằng DeepSeek ({missing.length})
            </Button>
          )}
        </div>
        {queueNote && (
          <p
            className={queueNote.tone === "ok" ? researchStyles.hint : researchStyles.error}
            role="status"
            data-testid="sb-detail-queue-note"
          >
            {queueNote.text}
          </p>
        )}
        {queue.counts &&
          (queue.counts.waiting > 0 || queue.counts.running > 0 || queue.counts.failed > 0) && (
            <p className={researchStyles.hint} data-testid="sb-detail-queue-counts">
              Hàng đợi DeepSeek: {queue.counts.running} đang chạy · {queue.counts.waiting} chờ ·{" "}
              {queue.counts.failed} lỗi
              {queue.counts.paused ? " · đang tạm dừng" : ""}.
            </p>
          )}
        <p className={researchStyles.hint}>
          Mỗi tài sản một lượt: AI chỉ đọc mục khái niệm của tài sản, bảng màu, các unit tài sản xuất
          hiện và tóm tắt các tài sản khác. Từng tài sản cũng viết riêng được ở thẻ của nó, kể cả bằng
          AI ngoài.
        </p>
      </div>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title="Viết chi tiết cho mọi tài sản chưa có"
        hasInput
        testId="sb-detail-dialog"
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
                void storyboard.generateAssetDetails(n);
              }}
              data-testid="sb-detail-confirm"
            >
              Chạy {missing.length} tài sản
            </Button>
          </>
        }
      >
        <p>
          {missing.length} tài sản chưa có chi tiết: {missing.map((a) => a.asset_id).join(", ")}. AI
          chạy lần lượt theo thứ tự danh sách, mỗi tài sản một lượt; tài sản đã có chi tiết và trường
          bạn đã sửa tay giữ nguyên.
        </p>
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            data-testid="sb-detail-note"
          />
        </FormField>
      </Dialog>
    </Card>
  );
}
