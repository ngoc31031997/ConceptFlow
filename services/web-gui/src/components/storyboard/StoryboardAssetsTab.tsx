import { useEffect, useState } from "react";
import {
  EXEMPLAR_FOLDER_ID,
  generateStoryboardAssetDetail,
  generateStoryboardAssets,
  listIllustrationFolders,
  storyboardAssetPreviewUrl,
  type IllustrationFolder,
  type StoryboardAsset,
  type StoryboardState,
} from "../../api/client";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, FormField, Select, TextArea } from "../ui";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
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
  drawBlocker,
} from "./storyboardModel";
import styles from "./storyboard.module.css";

const STATUS_CHIP: Record<string, { label: string; className: string }> = {
  approved: { label: "✓ Đã duyệt", className: researchStyles.chipOk },
  drawn: { label: "Đã vẽ, chờ duyệt", className: researchStyles.chipRun },
  planned: { label: "Chưa vẽ", className: researchStyles.chipWait },
};

function AssetDetailChip({ has, testId }: { has: boolean; testId: string }) {
  return (
    <span
      className={`${researchStyles.chip} ${has ? researchStyles.chipOk : researchStyles.chipWait}`}
      data-testid={testId}
    >
      {has ? "Đã có chi tiết" : "Chưa có chi tiết"}
    </span>
  );
}

function AssetStatusChip({ status, testId }: { status: string; testId: string }) {
  const chip = STATUS_CHIP[status] ?? STATUS_CHIP.planned;
  return (
    <span className={`${researchStyles.chip} ${chip.className}`} data-testid={testId}>
      {chip.label}
    </span>
  );
}

interface AssetCardProps {
  projectId: string;
  projectName: string;
  asset: StoryboardAsset;
  folders: IllustrationFolder[];
  foldersError: string | null;
  /** Whether the project has a drawing kit; assets cannot be drawn without one. */
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: boolean;
  onToggle: (open: boolean) => void;
  onEdit: () => void;
  onDelete: () => void;
}

/**
 * One asset: its preview, what the Creator described, whether approving puts it
 * in the library and where, and the actions that draw, approve or change it. An
 * approved asset folds to one line.
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
  onEdit,
  onDelete,
}: AssetCardProps) {
  const [redrawing, setRedrawing] = useState(false);
  const [detailing, setDetailing] = useState(false);
  const [note, setNote] = useState("");
  const id = asset.asset_id;
  const testId = `sb-asset-${id}`;
  const approved = asset.status === "approved";
  const running =
    typeof storyboard.running === "object" &&
    storyboard.running !== null &&
    "asset" in storyboard.running &&
    storyboard.running.asset === id;
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const editDisabled = readOnly || storyboard.busy || running;
  const hasPreview = asset.preview_path !== "";
  const drawReason = drawBlocker(asset, state);
  const detailRunning =
    typeof storyboard.running === "object" &&
    storyboard.running !== null &&
    "detail" in storyboard.running &&
    storyboard.running.detail === id;
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
        {asset.name} · {ASSET_KIND_LABELS[asset.kind] ?? asset.kind} ·{" "}
        {ASSET_TIER_LABELS[asset.tier] ?? asset.tier}
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
          <div className={styles.tags}>
            <AssetDetailChip has={asset.has_detail} testId={`${testId}-detail-state`} />
            {running || detailRunning ? (
              <span className={`${researchStyles.chip} ${researchStyles.chipRun}`}>
                {running ? "● Đang vẽ" : "● Đang viết chi tiết"}
              </span>
            ) : (
              <AssetStatusChip status={asset.status} testId={`${testId}-status`} />
            )}
          </div>
        </div>
        {hasPreview ? (
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
          <dd>
            {ASSET_KIND_LABELS[asset.kind] ?? asset.kind} ·{" "}
            {ASSET_TIER_LABELS[asset.tier] ?? asset.tier}
          </dd>
          <dt>Khái niệm</dt>
          <dd className={styles.mono}>{asset.concept_id || "(không gắn)"}</dd>
          <dt>Xuất hiện ở unit</dt>
          <dd className={styles.tags} data-testid={`${testId}-units`}>
            {asset.units.length > 0
              ? asset.units.map((u) => (
                  <span key={u} className={styles.tag}>
                    {u}
                  </span>
                ))
              : "Chưa gắn unit nào"}
          </dd>
          <dt>Cảm giác</dt>
          <dd data-testid={`${testId}-portrayal`}>
            {asset.has_detail ? asset.portrayal : "Chưa viết chi tiết"}
          </dd>
          <dt>Không gợi ý</dt>
          <dd>{asset.must_not_imply.length > 0 ? asset.must_not_imply.join("; ") : "—"}</dd>
          <dt>Màu · biểu cảm</dt>
          <dd data-testid={`${testId}-tokens`}>
            {asset.palette_tokens.length > 0 ? asset.palette_tokens.join(", ") : "—"}
            {asset.expressions.length > 0 ? ` · ${asset.expressions.join(", ")}` : ""}
          </dd>
          <dt>Hình gốc</dt>
          <dd className={styles.mono}>{asset.base_illustration_id || "Chưa có hình gốc"}</dd>
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
            {!folderKnown && <option value={asset.library_folder_id}>{asset.library_folder_id}</option>}
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
          {approved && (
            <span className={researchStyles.hint}>Bỏ duyệt để đổi thư viện.</span>
          )}
        </div>
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
                disabled={disabled}
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
        {asset.status === "planned" && drawReason !== "" && !state.kit.missing && (
          <p className={researchStyles.hint} data-testid={`${testId}-draw-reason`}>
            {drawReason}.
          </p>
        )}
        <ConceptAiDialog
          open={detailing}
          onClose={() => setDetailing(false)}
          title={
            asset.has_detail
              ? `Viết lại chi tiết của ${asset.name} kèm ghi chú`
              : `Viết chi tiết cho ${asset.name}`
          }
          impact={
            <p className={researchStyles.hint} data-testid={`${testId}-detail-impact`}>
              AI chỉ đọc mục khái niệm của tài sản, bảng màu, các unit nó xuất hiện (
              {asset.units.length > 0 ? asset.units.join(", ") : "chưa gắn unit nào"}) và tóm tắt
              các tài sản khác. Không đổi tên, loại, hạng hay hình gốc; trường bạn đã sửa tay giữ
              nguyên.
              {asset.status !== "planned"
                ? " Nếu chi tiết đổi, tài sản về “Chưa vẽ” (hình giữ tới lần vẽ lại)."
                : ""}
            </p>
          }
          onSystem={(n) => void storyboard.generateAssetDetail(id, n)}
          external={{
            run: (n, external) => generateStoryboardAssetDetail(projectId, id, n, { external }),
            onDone: storyboard.reload,
          }}
          testId={`${testId}-detail-dialog`}
        />
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

interface StoryboardAssetsTabProps {
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
 * Tab "Tài sản" of step 11: the list of assets the shots reuse, each drawn once.
 * The Creator adds or has the AI list them, draws each, approves them one by one
 * and approves the list.
 */
export function StoryboardAssetsTab({
  projectId,
  projectName,
  state,
  storyboard,
  readOnly,
  expanded,
  onToggle,
}: StoryboardAssetsTabProps) {
  const [editing, setEditing] = useState<StoryboardAsset | "new" | null>(null);
  const [deleting, setDeleting] = useState<StoryboardAsset | null>(null);
  const [aiOpen, setAiOpen] = useState(false);
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [foldersError, setFoldersError] = useState<string | null>(null);
  const items = state.assets.items;
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const allApproved = items.length > 0 && items.every((a) => a.status === "approved");
  const hasApproved = items.some((a) => a.status === "approved");

  useEffect(() => {
    let alive = true;
    listIllustrationFolders()
      .then((f) => alive && setFolders(f))
      .catch(() => alive && setFoldersError("Không tải được các thư mục thư viện."));
    return () => {
      alive = false;
    };
  }, []);

  return (
    <>
      <Card data-testid="sb-assets">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Danh sách tài sản</h3>
            <span className={researchStyles.hint}>
              Mỗi tài sản vẽ một lần, dùng lại ở nhiều shot. AI lập khung danh sách trước (tên, loại,
              hạng, unit), rồi viết chi tiết từng tài sản từ ngôn ngữ hình ảnh của bước 8.
            </span>
          </div>
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
                disabled={disabled || !allApproved}
                title={allApproved ? undefined : "Duyệt từng tài sản trước"}
                data-testid="sb-assets-approve"
              >
                Duyệt cả danh sách
              </Button>
            )}
          </div>
          {items.length === 0 && (
            <p className={researchStyles.hint} data-testid="sb-assets-empty">
              Chưa có tài sản. Thêm tay hoặc cho AI lập danh sách từ ngôn ngữ hình ảnh và các unit.
            </p>
          )}
          {hasApproved && (
            <p className={researchStyles.hint}>
              Tài sản đã duyệt gập thành một dòng; bấm “Mở” để xem hoặc “Bỏ duyệt” để sửa.
            </p>
          )}
        </div>
      </Card>
      {items.length > 0 && (
        <StoryboardAssetDetailCard
          projectId={projectId}
          state={state}
          storyboard={storyboard}
          readOnly={readOnly}
        />
      )}
      <div className={styles.grid} data-testid="sb-asset-grid">
        {items.map((a) => (
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
            onEdit={() => setEditing(a)}
            onDelete={() => setDeleting(a)}
          />
        ))}
      </div>

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
        onConfirm={() =>
          deleting ? storyboard.deleteAsset(deleting.asset_id) : Promise.resolve()
        }
        size="sm"
        testId="sb-asset-delete-dialog"
      >
        <p data-testid="sb-asset-delete-impact">
          Xoá hình đã vẽ và mô tả của tài sản. Chưa shot nào dùng nó.
          {deleting?.library_illustration_id
            ? " Hình đã đưa vào thư viện được giữ lại."
            : ""}
        </p>
        <p className={researchStyles.hint}>
          Không khôi phục được. Mã {deleting?.asset_id} không được dùng lại.
        </p>
      </FormDialog>
      <ConceptAiDialog
        open={aiOpen}
        onClose={() => setAiOpen(false)}
        title={items.length > 0 ? "AI lập lại danh sách tài sản" : "AI lập danh sách tài sản"}
        impact={
          <p className={researchStyles.hint}>
            AI đọc ngôn ngữ hình ảnh, kế hoạch và chuỗi hình đã chốt cùng hình đã duyệt của thư
            viện, và chỉ lập khung danh sách (tên, loại, hạng, hình gốc, unit); chi tiết từng tài
            sản viết ở lượt sau. Tài sản giữ mã khi AI giữ nó; trường bạn đã sửa tay giữ nguyên; tài sản đang được
            shot dùng không bị bỏ. Danh sách về “Chờ duyệt”; tài sản không đổi không phải vẽ lại.
          </p>
        }
        onSystem={(note) => void storyboard.generateAssets(note)}
        external={{
          run: (note, external) => generateStoryboardAssets(projectId, note, { external }),
          onDone: storyboard.reload,
        }}
        testId="sb-assets-dialog"
      />
    </>
  );
}
