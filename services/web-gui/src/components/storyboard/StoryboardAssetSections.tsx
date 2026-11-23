import { useEffect, useState } from "react";
import {
  describeAiRequest,
  generateStoryboardAssetDetail,
  generateStoryboardAssets,
  EXEMPLAR_FOLDER_ID,
  listIllustrationFolders,
  storyboardAssetPreviewUrl,
  type IllustrationFolder,
  type StoryboardAsset,
  type StoryboardState,
} from "../../api/client";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button, Dialog, FormField, Select, TextArea } from "../ui";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { StoryboardRunPreview } from "./StoryboardRunPreview";
import { FormDialog } from "../visualplan/PlanDialogs";
import { AssetDialog } from "./AssetDialog";
import { Foldable } from "./Foldable";
import { StoryboardAssetDetailCard } from "./StoryboardAssetDetailCard";
import {
  ASSET_KIND_LABELS,
  ASSET_TIER_LABELS,
  DEFAULT_LIBRARY_FOLDER,
  PROJECT_FOLDER_ID,
  allUnits,
  assetsToDraw,
  drawBlocker,
  listApproveBlocker,
} from "./storyboardModel";
import styles from "./storyboard.module.css";

const STATUS_CHIP: Record<string, { label: string; className: string }> = {
  approved: { label: "✓ Đã duyệt", className: researchStyles.chipOk },
  drawn: { label: "Đã vẽ, chờ duyệt", className: researchStyles.chipRun },
  planned: { label: "Chưa vẽ", className: researchStyles.chipWait },
};

function AssetStatusChip({ status, testId }: { status: string; testId: string }) {
  const chip = STATUS_CHIP[status] ?? STATUS_CHIP.planned;
  return (
    <span className={`${researchStyles.chip} ${chip.className}`} data-testid={testId}>
      {chip.label}
    </span>
  );
}

function kindTier(asset: StoryboardAsset): string {
  return `${ASSET_KIND_LABELS[asset.kind] ?? asset.kind} · ${ASSET_TIER_LABELS[asset.tier] ?? asset.tier}`;
}

/** Whether the AI is busy on this asset: drawing it, or writing its detail. */
function runningOn(storyboard: StoryboardControls, id: string): "draw" | "detail" | null {
  const r = storyboard.running;
  if (typeof r !== "object" || r === null) return null;
  if ("asset" in r && r.asset === id) return "draw";
  if ("detail" in r && r.detail === id) return "detail";
  return null;
}

interface AssetRowProps {
  projectId: string;
  asset: StoryboardAsset;
  storyboard: StoryboardControls;
  readOnly: boolean;
  onEdit: () => void;
  onDelete: () => void;
}

/**
 * One row of the asset list: what the asset is, which units it shows in,
 * whether its detail is written, and the actions that change the list entry.
 */
function AssetRow({ projectId, asset, storyboard, readOnly, onEdit, onDelete }: AssetRowProps) {
  const [detailing, setDetailing] = useState(false);
  const id = asset.asset_id;
  const testId = `sb-asset-${id}`;
  const busyOn = runningOn(storyboard, id);
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const editDisabled = readOnly || storyboard.busy || busyOn !== null;
  const inUse = asset.used_by.length > 0;

  return (
    <tr data-testid={`sb-asset-row-${id}`}>
      <td className={styles.mono}>{id}</td>
      <td>
        <b>{asset.name}</b>
        <div className={researchStyles.hint} data-testid={`${testId}-portrayal`}>
          {asset.has_detail ? asset.portrayal : "Chưa viết chi tiết"}
        </div>
      </td>
      <td>{kindTier(asset)}</td>
      <td>
        <span className={styles.tags} data-testid={`${testId}-units`}>
          {asset.units.length > 0
            ? asset.units.map((u) => (
                <span key={u} className={styles.tag}>
                  {u}
                </span>
              ))
            : "Chưa gắn unit nào"}
        </span>
      </td>
      <td className={styles.mono}>{asset.base_illustration_id || "Vẽ mới"}</td>
      <td>
        {busyOn === "detail" ? (
          <span className={`${researchStyles.chip} ${researchStyles.chipRun}`}>● Đang viết</span>
        ) : (
          <span
            className={`${researchStyles.chip} ${asset.has_detail ? researchStyles.chipOk : researchStyles.chipWait}`}
            data-testid={`${testId}-detail-state`}
          >
            {asset.has_detail ? "Đã có chi tiết" : "Chưa có chi tiết"}
          </span>
        )}
      </td>
      <td>
        <div className={styles.foot}>
          <Button
            variant="ghost"
            onClick={onEdit}
            disabled={editDisabled}
            data-testid={`${testId}-edit`}
          >
            Sửa
          </Button>
          <Button
            variant="run"
            onClick={() => setDetailing(true)}
            disabled={disabled}
            data-testid={`${testId}-detail`}
          >
            {asset.has_detail ? "AI viết lại chi tiết" : "AI viết chi tiết"}
          </Button>
          <AiRunPill runPath={`/v1/projects/${projectId}/storyboard/assets/${id}/detail`} onOpen={() => setDetailing(true)} testId={`${testId}-detail-run`} />
          <Button
            variant="dangerGhost"
            onClick={onDelete}
            disabled={editDisabled || inUse}
            title={inUse ? `Shot ${asset.used_by.join(", ")} đang dùng tài sản này` : undefined}
            data-testid={`${testId}-delete`}
          >
            Xoá
          </Button>
        </div>
        <AiRunDialog
          open={detailing}
          onClose={() => setDetailing(false)}
          title={
            asset.has_detail
              ? `Viết lại chi tiết của ${asset.name} kèm ghi chú`
              : `Viết chi tiết cho ${asset.name}`
          }
          runPath={`/v1/projects/${projectId}/storyboard/assets/${id}/detail`}
          describe={(note) => describeAiRequest((external) => generateStoryboardAssetDetail(projectId, id, note, { external }))}
          renderPreview={(p) => <StoryboardRunPreview kind="asset_detail" preview={p} />}
          impact={
            <p className={researchStyles.hint} data-testid={`${testId}-detail-impact`}>
              AI chỉ đọc mục khái niệm của tài sản, bảng màu, các unit nó xuất hiện (
              {asset.units.length > 0 ? asset.units.join(", ") : "chưa gắn unit nào"}) và tóm tắt
              các tài sản khác. Không đổi tên, loại, hạng hay hình gốc; trường bạn đã sửa tay giữ
              nguyên. Danh sách tài sản về “Chờ duyệt”.
              {asset.status !== "planned"
                ? " Nếu chi tiết đổi, tài sản về “Chưa vẽ” (hình giữ tới lần vẽ lại)."
                : ""}
            </p>
          }
          onApplied={storyboard.reload}
          testId={`${testId}-detail-dialog`}
        />
      </td>
    </tr>
  );
}

interface AssetListSectionProps {
  projectId: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
}

/**
 * Section 1 of step 11, the asset list: the AI (or the Creator) names the
 * assets, the AI writes each one's detail, and the Creator approves the list,
 * which opens the drawing in section 2.
 */
export function StoryboardAssetListSection({
  projectId,
  state,
  storyboard,
  readOnly,
}: AssetListSectionProps) {
  const [editing, setEditing] = useState<StoryboardAsset | "new" | null>(null);
  const [deleting, setDeleting] = useState<StoryboardAsset | null>(null);
  const [aiOpen, setAiOpen] = useState(false);
  const items = state.assets.items;
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const approveBlocker = listApproveBlocker(state);

  return (
    <>
      <div className={researchStyles.actions}>
        <Button
          variant="ghost"
          onClick={() => setEditing("new")}
          disabled={disabled}
          data-testid="sb-asset-add"
        >
          + Thêm tài sản
        </Button>
        <Button
          variant="run"
          onClick={() => setAiOpen(true)}
          disabled={disabled}
          data-testid="sb-assets-generate"
        >
          {items.length > 0 ? "AI lập lại danh sách" : "AI lập danh sách"}
        </Button>
        <AiRunPill runPath={`/v1/projects/${projectId}/storyboard/assets/generate`} onOpen={() => setAiOpen(true)} testId="sb-assets-run" />
      </div>
      {items.length === 0 ? (
        <p className={researchStyles.hint} data-testid="sb-assets-empty">
          Chưa có tài sản. Thêm tay hoặc cho AI lập danh sách từ ngôn ngữ hình ảnh và các unit.
        </p>
      ) : (
        <>
          <StoryboardAssetDetailCard
            projectId={projectId}
            state={state}
            storyboard={storyboard}
            readOnly={readOnly}
          />
          <div className={styles.tableScroll}>
            <table className={styles.assetTable} data-testid="sb-asset-list">
              <thead>
                <tr>
                  <th>Mã</th>
                  <th>Tên · cảm giác</th>
                  <th>Loại · hạng</th>
                  <th>Unit</th>
                  <th>Hình gốc</th>
                  <th>Chi tiết</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {items.map((a) => (
                  <AssetRow
                    key={a.asset_id}
                    projectId={projectId}
                    asset={a}
                    storyboard={storyboard}
                    readOnly={readOnly}
                    onEdit={() => setEditing(a)}
                    onDelete={() => setDeleting(a)}
                  />
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      <div className={researchStyles.actions}>
        <span className={researchStyles.grow} />
        {state.assets.approved ? (
          <Button
            variant="ghost"
            onClick={() => void storyboard.unapproveAssetList()}
            disabled={disabled}
            data-testid="sb-assets-unapprove"
          >
            Bỏ duyệt danh sách
          </Button>
        ) : (
          <Button
            variant="primary"
            onClick={() => void storyboard.approveAssetList()}
            disabled={disabled || approveBlocker !== ""}
            title={approveBlocker || undefined}
            data-testid="sb-assets-approve"
          >
            Duyệt danh sách
          </Button>
        )}
      </div>
      {!state.assets.approved && approveBlocker !== "" && items.length > 0 && (
        <p className={researchStyles.hint} data-testid="sb-assets-approve-blocker">
          {approveBlocker}.
        </p>
      )}

      <AssetDialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        asset={editing === "new" || editing === null ? null : editing}
        unitIds={allUnits(state).map((u) => u.unit_id)}
        onSave={(fields) =>
          editing === "new" || editing === null
            ? storyboard.addAsset(fields)
            : storyboard.editAsset(editing.asset_id, fields)
        }
      />
      <FormDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        title={`Xoá tài sản ${deleting?.asset_id ?? ""} · ${deleting?.name ?? ""}?`}
        confirmLabel="Xoá tài sản"
        confirmVariant="danger"
        hasInput={false}
        onConfirm={() => (deleting ? storyboard.deleteAsset(deleting.asset_id) : Promise.resolve())}
        size="sm"
        testId="sb-asset-delete-dialog"
      >
        <p data-testid="sb-asset-delete-impact">
          Xoá hình đã vẽ và mô tả của tài sản. Chưa shot nào dùng nó.
          {deleting?.library_illustration_id ? " Hình đã đưa vào thư viện được giữ lại." : ""}
        </p>
        <p className={researchStyles.hint}>
          Không khôi phục được. Mã {deleting?.asset_id} không được dùng lại.
        </p>
      </FormDialog>
      <AiRunDialog
        open={aiOpen}
        onClose={() => setAiOpen(false)}
        title={items.length > 0 ? "AI lập lại danh sách tài sản" : "AI lập danh sách tài sản"}
        runPath={`/v1/projects/${projectId}/storyboard/assets/generate`}
        describe={(note) => describeAiRequest((external) => generateStoryboardAssets(projectId, note, { external }))}
        renderPreview={(p) => <StoryboardRunPreview kind="assets" preview={p} />}
        impact={
          <p className={researchStyles.hint}>
            AI đọc ngôn ngữ hình ảnh, kế hoạch và chuỗi hình đã chốt cùng hình đã duyệt của thư
            viện, và chỉ lập khung danh sách (tên, loại, hạng, hình gốc, unit); chi tiết từng tài
            sản viết ở lượt sau. Tài sản giữ mã khi AI giữ nó; trường bạn đã sửa tay giữ nguyên; tài
            sản đang được shot dùng không bị bỏ. Danh sách về “Chờ duyệt”; tài sản không đổi không
            phải vẽ lại.
          </p>
        }
        onApplied={storyboard.reload}
        testId="sb-assets-dialog"
      />
    </>
  );
}

interface AssetCardProps {
  projectId: string;
  projectName: string;
  asset: StoryboardAsset;
  folders: IllustrationFolder[];
  foldersError: string | null;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: boolean;
  onToggle: (open: boolean) => void;
}

/**
 * One asset to draw: its preview, what it should look like, whether approving
 * puts it in the library and where, and the actions that draw, redraw or
 * approve it. An approved asset folds to one line.
 */
function AssetCard({
  projectId,
  projectName,
  asset,
  folders,
  foldersError,
  state,
  storyboard,
  readOnly,
  expanded,
  onToggle,
}: AssetCardProps) {
  const [redrawing, setRedrawing] = useState(false);
  const [note, setNote] = useState("");
  const id = asset.asset_id;
  const testId = `sb-asset-${id}`;
  const approved = asset.status === "approved";
  const busyOn = runningOn(storyboard, id);
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const editDisabled = readOnly || storyboard.busy || busyOn !== null;
  const drawReason = drawBlocker(asset, state);
  const inUse = asset.used_by.length > 0;
  // The system folder "Từ dự án" is the default entry; the Hình mẫu folder only takes exemplars.
  const folderOptions = folders.filter(
    (f) => f.id !== EXEMPLAR_FOLDER_ID && f.id !== PROJECT_FOLDER_ID,
  );
  const folderValue = asset.library_folder_id === PROJECT_FOLDER_ID ? "" : asset.library_folder_id;
  const folderKnown = folderValue === "" || folderOptions.some((f) => f.id === folderValue);

  const summary = (
    <div className={styles.folded} data-testid={testId}>
      <span className={styles.mono}>{id}</span>
      <span className={styles.summary}>
        {asset.name} · {kindTier(asset)}
      </span>
      <AssetStatusChip status={asset.status} testId={`${testId}-status`} />
      <Button variant="view" onClick={() => onToggle(true)} data-testid={`${testId}-open`}>
        Mở
      </Button>
    </div>
  );

  return (
    <Foldable folded={approved && !expanded} summary={summary}>
      <article className={styles.card} data-testid={testId}>
        <div className={styles.cardHead}>
          <div>
            <h4>{asset.name}</h4>
            <span className={styles.mono}>{id}</span>
          </div>
          {busyOn === "draw" ? (
            <span className={`${researchStyles.chip} ${researchStyles.chipRun}`}>● Đang vẽ</span>
          ) : (
            <AssetStatusChip status={asset.status} testId={`${testId}-status`} />
          )}
        </div>
        {asset.preview_path !== "" ? (
          <img
            className={`${styles.thumb} ${styles.thumbSquare}`}
            src={storyboardAssetPreviewUrl(projectId, id, asset.version)}
            alt={`Xem trước tài sản ${asset.name}`}
            data-testid={`${testId}-preview`}
          />
        ) : (
          <div
            className={`${styles.thumb} ${styles.thumbSquare} ${styles.placeholder}`}
            data-testid={`${testId}-empty`}
          >
            Chưa vẽ
          </div>
        )}
        <dl className={styles.kv}>
          <dt>Loại · hạng</dt>
          <dd>{kindTier(asset)}</dd>
          <dt>Cảm giác</dt>
          <dd>{asset.has_detail ? asset.portrayal : "Chưa viết chi tiết"}</dd>
          <dt>Không gợi ý</dt>
          <dd>{asset.must_not_imply.length > 0 ? asset.must_not_imply.join("; ") : "—"}</dd>
          <dt>Màu · biểu cảm</dt>
          <dd data-testid={`${testId}-tokens`}>
            {asset.palette_tokens.length > 0 ? asset.palette_tokens.join(", ") : "—"}
            {asset.expressions.length > 0 ? ` · ${asset.expressions.join(", ")}` : ""}
          </dd>
          <dt>Shot dùng</dt>
          <dd>{inUse ? asset.used_by.join(", ") : "Chưa shot nào"}</dd>
        </dl>
        {asset.warnings.length > 0 && (
          <ul className={`${styles.notice} ${styles.noticeWarn} ${styles.noticeList}`}>
            {asset.warnings.map((w) => (
              <li key={`${w.line ?? ""}${w.message}`}>{w.message}</li>
            ))}
          </ul>
        )}
        <div className={styles.libRow} data-testid={`${testId}-library`}>
          <label>
            <input
              type="checkbox"
              checked={asset.to_library}
              disabled={editDisabled || approved}
              onChange={(e) => void storyboard.patchAsset(id, { to_library: e.target.checked })}
              data-testid={`${testId}-to-library`}
            />{" "}
            Đưa vào thư viện khi duyệt, thư mục
          </label>
          <Select
            aria-label="Thư mục thư viện"
            value={folderKnown ? folderValue : asset.library_folder_id}
            disabled={editDisabled || approved || !asset.to_library}
            onChange={(e) => void storyboard.patchAsset(id, { library_folder_id: e.target.value })}
            data-testid={`${testId}-folder`}
          >
            <option value={DEFAULT_LIBRARY_FOLDER}>Từ dự án</option>
            {!folderKnown && (
              <option value={asset.library_folder_id}>{asset.library_folder_id}</option>
            )}
            {folderOptions.map((f) => (
              <option key={f.id} value={f.id}>
                {f.name}
              </option>
            ))}
          </Select>
          {foldersError && (
            <span className={researchStyles.hint} data-testid={`${testId}-folders-error`}>
              {foldersError}
            </span>
          )}
          {approved && <span className={researchStyles.hint}>Bỏ duyệt để đổi thư viện.</span>}
        </div>
        <div className={styles.foot}>
          {asset.status === "planned" && (
            <Button
              variant="run"
              onClick={() => void storyboard.drawAsset(id)}
              disabled={disabled || drawReason !== ""}
              title={drawReason || undefined}
              data-testid={`${testId}-draw`}
            >
              Vẽ
            </Button>
          )}
          {asset.status === "drawn" && (
            <>
              <Button
                variant="run"
                onClick={() => setRedrawing(true)}
                disabled={disabled || drawReason !== ""}
                title={drawReason || undefined}
                data-testid={`${testId}-redraw`}
              >
                Vẽ lại
              </Button>
              <Button
                variant="primary"
                onClick={() => {
                  onToggle(false);
                  void storyboard.approveAsset(id, projectName);
                }}
                disabled={disabled}
                data-testid={`${testId}-approve`}
              >
                Duyệt
              </Button>
            </>
          )}
          {approved && (
            <>
              <Button
                variant="ghost"
                onClick={() => onToggle(false)}
                data-testid={`${testId}-fold`}
              >
                Thu gọn
              </Button>
              <Button
                variant="ghost"
                onClick={() => {
                  onToggle(true);
                  void storyboard.unapproveAsset(id);
                }}
                disabled={disabled}
                data-testid={`${testId}-unapprove`}
              >
                Bỏ duyệt
              </Button>
            </>
          )}
        </div>
        {asset.status !== "approved" && drawReason !== "" && !state.kit.missing && (
          <p className={researchStyles.hint} data-testid={`${testId}-draw-reason`}>
            {drawReason}.
          </p>
        )}
        <Dialog
          open={redrawing}
          onClose={() => setRedrawing(false)}
          title={`Vẽ lại ${asset.name} kèm ghi chú`}
          hasInput
          testId={`${testId}-redraw-dialog`}
          footer={
            <>
              <Button variant="ghost" onClick={() => setRedrawing(false)}>
                Huỷ
              </Button>
              <Button
                variant={inUse ? "impact" : "run"}
                onClick={() => {
                  const n = note.trim();
                  setRedrawing(false);
                  setNote("");
                  void storyboard.redrawAsset(id, n);
                }}
                data-testid={`${testId}-redraw-confirm`}
              >
                Vẽ lại
              </Button>
            </>
          }
        >
          <FormField label="Ghi chú cho bộ vẽ (không bắt buộc)">
            <TextArea
              value={note}
              onChange={(e) => setNote(e.target.value)}
              rows={3}
              placeholder="Viết điều chưa ổn và muốn đổi thế nào"
              data-testid={`${testId}-redraw-note`}
            />
          </FormField>
          <p className={researchStyles.hint}>
            Bộ vẽ AI của hệ thống vẽ lại từ code đang có; phiên bản hình tăng một.
            {inUse
              ? ` Khung của các shot đang dùng (${asset.used_by.join(", ")}) cũ đi và phải vẽ lại.`
              : " Chưa shot nào dùng tài sản này."}
          </p>
        </Dialog>
      </article>
    </Foldable>
  );
}

interface AssetDrawSectionProps {
  projectId: string;
  projectName: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  /** Asset ids the Creator opened; an approved asset folds unless its id is here. */
  expanded: ReadonlySet<string>;
  onToggle: (id: string, open: boolean) => void;
}

/**
 * Section 2 of step 11: drawing each asset of the approved list with the
 * project's kit and approving each drawing. "Vẽ tất cả" draws the assets not
 * drawn yet one after another and stops at the first that fails.
 */
export function StoryboardAssetDrawSection({
  projectId,
  projectName,
  state,
  storyboard,
  readOnly,
  expanded,
  onToggle,
}: AssetDrawSectionProps) {
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [foldersError, setFoldersError] = useState<string | null>(null);
  const [drawingAll, setDrawingAll] = useState(false);
  const toDraw = assetsToDraw(state);
  const disabled = readOnly || storyboard.busy || storyboard.running !== null || drawingAll;

  useEffect(() => {
    let alive = true;
    listIllustrationFolders()
      .then((f) => alive && setFolders(f))
      .catch(() => alive && setFoldersError("Không tải được các thư mục thư viện."));
    return () => {
      alive = false;
    };
  }, []);

  async function drawAll() {
    setDrawingAll(true);
    try {
      for (const a of toDraw) {
        if (!(await storyboard.drawAsset(a.asset_id))) return;
      }
    } finally {
      setDrawingAll(false);
    }
  }

  return (
    <>
      {toDraw.length > 0 && (
        <div className={researchStyles.actions}>
          <span className={researchStyles.grow} />
          <Button
            variant="run"
            onClick={() => void drawAll()}
            disabled={disabled}
            data-testid="sb-assets-draw-all"
          >
            Vẽ {toDraw.length} tài sản chưa vẽ
          </Button>
        </div>
      )}
      <div className={styles.grid} data-testid="sb-asset-grid">
        {state.assets.items.map((a) => (
          <AssetCard
            key={a.asset_id}
            projectId={projectId}
            projectName={projectName}
            asset={a}
            folders={folders}
            foldersError={foldersError}
            state={state}
            storyboard={storyboard}
            readOnly={readOnly}
            expanded={expanded.has(a.asset_id)}
            onToggle={(open) => onToggle(a.asset_id, open)}
          />
        ))}
      </div>
    </>
  );
}
