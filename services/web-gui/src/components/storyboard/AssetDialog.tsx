import { useEffect, useState } from "react";
import {
  listIllustrations,
  type Illustration,
  type StoryboardAsset,
  type StoryboardAssetKind,
  type StoryboardAssetTier,
} from "../../api/client";
import researchStyles from "../research/research.module.css";
import { FormField, Select, TextArea, TextInput } from "../ui";
import { FormDialog } from "../visualplan/PlanDialogs";
import { ASSET_KIND_LABELS, ASSET_TIER_LABELS, textLines } from "./storyboardModel";
import styles from "./storyboard.module.css";

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** The fields whose change sends a drawn or approved asset back to "Chưa vẽ" (the server's rule). */
const REDRAW_FIELDS = [
  "name",
  "kind",
  "tier",
  "portrayal",
  "must_not_imply",
  "palette_tokens",
  "expressions",
  "base_illustration_id",
  "units",
];

interface AssetDialogProps {
  open: boolean;
  onClose: () => void;
  /** The asset being edited; null adds a new one. */
  asset: StoryboardAsset | null;
  /** Every unit id of the plan, in sequence order: what the asset's units are picked from. */
  unitIds: string[];
  /** Stores the fields; a throw is shown in the dialog and keeps it open. */
  onSave: (fields: Record<string, unknown>) => Promise<void>;
}

/**
 * Add or edit one asset by hand: its name, what it is, how it should feel and what
 * it must not suggest. An edit sends only the fields the Creator changed, so only
 * those become the Creator's and survive the AI listing the assets again.
 */
export function AssetDialog({ open, onClose, asset, unitIds, onSave }: AssetDialogProps) {
  const [name, setName] = useState("");
  const [conceptId, setConceptId] = useState("");
  const [kind, setKind] = useState<StoryboardAssetKind>("figure");
  const [tier, setTier] = useState<StoryboardAssetTier>("support");
  const [portrayal, setPortrayal] = useState("");
  const [mustNot, setMustNot] = useState("");
  const [tokens, setTokens] = useState("");
  const [expressions, setExpressions] = useState("");
  const [base, setBase] = useState("");
  const [units, setUnits] = useState<string[]>([]);
  const [library, setLibrary] = useState<Illustration[]>([]);
  const [libraryError, setLibraryError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName(asset?.name ?? "");
    setConceptId(asset?.concept_id ?? "");
    setKind(asset?.kind ?? "figure");
    setTier(asset?.tier ?? "support");
    setPortrayal(asset?.portrayal ?? "");
    setMustNot((asset?.must_not_imply ?? []).join("\n"));
    setTokens((asset?.palette_tokens ?? []).join("\n"));
    setExpressions((asset?.expressions ?? []).join("\n"));
    setBase(asset?.base_illustration_id ?? "");
    setUnits(asset?.units ?? []);
    // The form starts from the asset when it opens; a refresh of the screen behind must not reset typed text.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, asset?.asset_id]);

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setLibraryError(null);
    listIllustrations({ status: "approved" })
      .then((items) => alive && setLibrary(items))
      .catch(() => {
        if (!alive) return;
        setLibrary([]);
        setLibraryError("Không tải được thư viện hình, nên chưa chọn được hình gốc.");
      });
    return () => {
      alive = false;
    };
  }, [open]);

  const next: Record<string, unknown> = {
    name: name.trim(),
    concept_id: conceptId.trim(),
    kind,
    tier,
    portrayal: portrayal.trim(),
    must_not_imply: textLines(mustNot),
    palette_tokens: textLines(tokens),
    expressions: textLines(expressions),
    base_illustration_id: base,
    // Kept in the plan's order whatever order the boxes were ticked in.
    units: unitIds.filter((u) => units.includes(u)).concat(units.filter((u) => !unitIds.includes(u))),
  };
  const changed = Object.keys(next).filter(
    (k) => asset === null || !same(next[k], asset[k as keyof StoryboardAsset]),
  );
  const resets =
    asset !== null &&
    asset.status !== "planned" &&
    changed.some((k) => REDRAW_FIELDS.includes(k));
  const baseOptions = library.filter((i) => (i.kind ?? "figure") === kind);
  const baseKnown = base === "" || baseOptions.some((i) => i.id === base);

  async function save() {
    const fields: Record<string, unknown> = {};
    for (const k of changed) fields[k] = next[k];
    if (asset === null) {
      fields.to_library = true;
      fields.library_folder_id = "";
    }
    if (Object.keys(fields).length > 0) await onSave(fields);
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={asset ? `Sửa tài sản ${asset.asset_id}` : "Thêm tài sản"}
      confirmLabel={asset ? "Lưu" : "Thêm tài sản"}
      confirmVariant={resets ? "impact" : "primary"}
      confirmDisabled={
        name.trim() === "" ||
        conceptId.trim() === "" ||
        (asset !== null && changed.length === 0)
      }
      onConfirm={save}
      size="lg"
      testId="sb-asset-dialog"
    >
      <div className={researchStyles.grid2}>
        <FormField label="Tên component (PascalCase, ví dụ BlackHole)">
          <TextInput
            value={name}
            onChange={(e) => setName(e.target.value)}
            data-testid="sb-asset-name"
          />
        </FormField>
        <FormField label="Khái niệm gắn (ví dụ characters.astronaut)">
          <TextInput
            value={conceptId}
            onChange={(e) => setConceptId(e.target.value)}
            data-testid="sb-asset-concept"
          />
        </FormField>
      </div>
      <div className={researchStyles.grid2}>
        <FormField label="Loại">
          <Select
            value={kind}
            onChange={(e) => setKind(e.target.value as StoryboardAssetKind)}
            data-testid="sb-asset-kind"
          >
            {Object.entries(ASSET_KIND_LABELS).map(([v, label]) => (
              <option key={v} value={v}>
                {label}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Hạng">
          <Select
            value={tier}
            onChange={(e) => setTier(e.target.value as StoryboardAssetTier)}
            data-testid="sb-asset-tier"
          >
            {Object.entries(ASSET_TIER_LABELS).map(([v, label]) => (
              <option key={v} value={v}>
                {label}
              </option>
            ))}
          </Select>
        </FormField>
      </div>
      <FormField label="Xuất hiện ở unit">
        <div className={styles.tags} data-testid="sb-asset-units">
          {unitIds.map((u) => (
            <label key={u} className={styles.tag}>
              <input
                type="checkbox"
                checked={units.includes(u)}
                onChange={(e) =>
                  setUnits((prev) => (e.target.checked ? [...prev, u] : prev.filter((x) => x !== u)))
                }
                data-testid={`sb-asset-unit-${u}`}
              />{" "}
              {u}
            </label>
          ))}
          {unitIds.length === 0 && <span className={researchStyles.hint}>Kế hoạch chưa có unit.</span>}
        </div>
      </FormField>
      <p className={researchStyles.hint}>
        Chi tiết bên dưới (cảm giác, điều không gợi ý, màu, biểu cảm) có thể để trống để AI viết sau;
        tài sản phải có chi tiết mới vẽ được.
      </p>
      <FormField label="Cảm giác khi nhìn (portrayal)">
        <TextArea
          value={portrayal}
          onChange={(e) => setPortrayal(e.target.value)}
          rows={3}
          data-testid="sb-asset-portrayal"
        />
      </FormField>
      <FormField label="Không được gợi ý (mỗi dòng một mục)">
        <TextArea
          value={mustNot}
          onChange={(e) => setMustNot(e.target.value)}
          rows={2}
          data-testid="sb-asset-must-not"
        />
      </FormField>
      <div className={researchStyles.grid2}>
        <FormField label="Màu theo bảng màu (mỗi dòng một mục)">
          <TextArea
            value={tokens}
            onChange={(e) => setTokens(e.target.value)}
            rows={2}
            data-testid="sb-asset-tokens"
          />
        </FormField>
        <FormField label="Biểu cảm (mỗi dòng một mục)">
          <TextArea
            value={expressions}
            onChange={(e) => setExpressions(e.target.value)}
            rows={2}
            data-testid="sb-asset-expressions"
          />
        </FormField>
      </div>
      <FormField label="Hình gốc trong thư viện (AI chỉnh theo hình này)">
        <Select
          value={base}
          onChange={(e) => setBase(e.target.value)}
          data-testid="sb-asset-base"
        >
          <option value="">Không dùng hình gốc</option>
          {!baseKnown && <option value={base}>{base}</option>}
          {baseOptions.map((i) => (
            <option key={i.id} value={i.id}>
              {i.title || i.name} ({i.name})
            </option>
          ))}
        </Select>
      </FormField>
      {libraryError && (
        <p className={researchStyles.error} role="status" data-testid="sb-asset-base-error">
          {libraryError}
        </p>
      )}
      {resets && (
        <div className={researchStyles.banner} data-testid="sb-asset-impact">
          Lưu đưa tài sản về “Chưa vẽ” (hình và code giữ tới lần vẽ lại).
          {asset && asset.used_by.length > 0
            ? ` Các shot đang dùng nó: ${asset.used_by.join(", ")}; khung của chúng cũ đi khi bạn vẽ lại tài sản.`
            : " Chưa shot nào dùng tài sản này."}
        </div>
      )}
      <p className={researchStyles.hint}>
        Trường bạn sửa được giữ khi AI lập lại danh sách tài sản.
      </p>
    </FormDialog>
  );
}
