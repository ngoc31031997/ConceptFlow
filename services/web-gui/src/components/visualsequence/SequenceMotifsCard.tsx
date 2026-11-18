import { useEffect, useState } from "react";
import {
  generateSequenceMotifs,
  type SequenceMotif,
  type VisualSequenceState,
} from "../../api/client";
import type { VisualSequenceControls } from "../../hooks/useVisualSequence";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dropdown, FormField, TextArea, TextInput } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
import visualStyles from "../visual/visual.module.css";
import { FormDialog } from "../visualplan/PlanDialogs";
import styles from "./sequence.module.css";
import { motifSlot } from "./sequenceModel";

interface MotifDialogProps {
  open: boolean;
  onClose: () => void;
  state: VisualSequenceState;
  /** The motif being edited; null adds a new one. */
  motif: SequenceMotif | null;
  onSave: (fields: Record<string, unknown>) => Promise<void>;
}

/**
 * Adds or edits a motif. Only the fields the Creator changed are sent, so only
 * those become the Creator's and survive the AI setting the motifs again.
 */
function MotifDialog({ open, onClose, state, motif, onSave }: MotifDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [anchor, setAnchor] = useState("");
  const [role, setRole] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(motif?.name ?? "");
    setDescription(motif?.description ?? "");
    setAnchor(motif?.anchor_ref ?? "");
    setRole(motif?.role ?? "");
    // The form starts from the motif when it opens; a refresh of the screen behind must not reset typed text.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, motif?.motif_id]);

  const anchors = Object.entries(state.motifs.anchors)
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([path, text]) => ({ value: path, label: path, hint: text }));

  async function save() {
    const fields: Record<string, unknown> = {};
    const put = (key: string, next: string, prev: string | undefined) => {
      if (next !== (prev ?? "")) fields[key] = next;
    };
    put("name", name.trim(), motif?.name);
    put("description", description.trim(), motif?.description);
    put("anchor_ref", anchor, motif?.anchor_ref);
    put("role", role.trim(), motif?.role);
    if (Object.keys(fields).length > 0) await onSave(fields);
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={motif ? `Sửa motif ${motif.motif_id} · ${motif.name}` : "Thêm motif"}
      confirmLabel={motif ? "Lưu" : "Thêm motif"}
      confirmDisabled={name.trim() === "" || anchor === ""}
      onConfirm={save}
      size="lg"
      testId="seq-motif-dialog"
    >
      <FormField label="Tên">
        <TextInput
          value={name}
          onChange={(e) => setName(e.target.value)}
          data-testid="seq-motif-name"
        />
      </FormField>
      <FormField label="Mô tả: motif trông thế nào và biến đổi ra sao qua các lần xuất hiện">
        <TextArea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          data-testid="seq-motif-description"
        />
      </FormField>
      <FormField label="Neo vào mục cách vẽ đã chốt ở bước 8">
        <Dropdown
          aria-label="Neo bước 8"
          value={anchor}
          options={anchors}
          onChange={setAnchor}
          placeholder="Chọn mục cách vẽ"
          data-testid="seq-motif-anchor"
        />
      </FormField>
      {anchor !== "" && state.motifs.anchors[anchor] && (
        <p className={researchStyles.hint} data-testid="seq-motif-anchor-text">
          Bước 8 chốt: {state.motifs.anchors[anchor]}
        </p>
      )}
      <FormField label="Vai trò trong mạch">
        <TextInput
          value={role}
          onChange={(e) => setRole(e.target.value)}
          data-testid="seq-motif-role"
        />
      </FormField>
      <p className={researchStyles.hint}>
        Motif có ở unit nào là do chuỗi của từng unit quyết định (hộp “Sửa chuỗi”); “Xuất hiện ở”
        được máy gom lại từ đó. Lưu xong, danh sách motif về “Chờ duyệt”.
      </p>
    </FormDialog>
  );
}

interface SequenceMotifsCardProps {
  projectId: string;
  state: VisualSequenceState;
  sequence: VisualSequenceControls;
  readOnly: boolean;
}

/**
 * Section 2 of step 10: the video's motifs, each anchored to a drawing rule of
 * step 8, with the units the machine found them in. The motifs are set by the
 * AI or by hand and approved together before the AI writes any beat.
 */
export function SequenceMotifsCard({
  projectId,
  state,
  sequence,
  readOnly,
}: SequenceMotifsCardProps) {
  const [aiOpen, setAiOpen] = useState(false);
  const [editing, setEditing] = useState<SequenceMotif | "new" | null>(null);
  const [deleting, setDeleting] = useState<SequenceMotif | null>(null);
  const disabled = readOnly || sequence.busy || sequence.running !== null;
  const m = state.motifs;
  const approved = m.status === "approved" && !m.stale && m.motifs.length > 0;
  const hasProblems = m.motifs.some((x) => x.problems.length > 0);

  return (
    <Card data-testid="seq-motifs">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>2. Motif</h3>
          {m.motifs.length > 0 && (
            <UnitStatusChip
              state={sequence.running === "motifs" ? "running" : approved ? "approved" : "pending"}
            />
          )}
          <span className={researchStyles.grow} />
          <Button
            variant="run"
            onClick={() => setAiOpen(true)}
            disabled={disabled}
            data-testid="seq-motifs-generate"
          >
            {m.motifs.length > 0 ? "AI đặt lại motif" : "AI đặt motif"}
          </Button>
        </div>
        <p className={researchStyles.hint}>
          Motif là hình lặp lại có chủ ý để nối mạch. Mỗi motif neo vào một mục cách vẽ đã chốt ở
          bước 8. Phải duyệt motif trước khi AI viết chuỗi.
        </p>
        {m.stale && (
          <p className={researchStyles.banner} role="status" data-testid="seq-motifs-stale">
            ⚠ Đã đổi từ lúc đặt motif: {m.stale_sources.join(", ")}. Xem lại rồi duyệt motif để xác
            nhận vẫn hợp.
          </p>
        )}
        {m.motifs.length === 0 && (
          <p className={researchStyles.hint} data-testid="seq-motifs-empty">
            Chưa có motif. Cho AI đặt motif hoặc thêm tay.
          </p>
        )}
        <div className={styles.motifs}>
          {m.motifs.map((x) => {
            const few = x.unit_refs.length < 2;
            return (
              <div
                key={x.motif_id}
                className={`${styles.motif} ${x.problems.length > 0 ? styles.motifWarn : ""}`}
                id={`seq-motif-${x.motif_id}`}
                data-testid={`seq-motif-${x.motif_id}`}
              >
                <span className={styles.motifHead}>
                  <i
                    className={`${styles.dot} ${styles[`c${motifSlot(state, x.motif_id)}`]}`}
                    aria-hidden="true"
                  />
                  {x.motif_id} · {x.name}
                </span>
                <span>{x.description}</span>
                <span className={visualStyles.kv}>Neo bước 8</span>
                <span className={styles.anchor}>{x.anchor_ref}</span>
                {x.anchor_text && <span className={researchStyles.hint}>{x.anchor_text}</span>}
                <span className={visualStyles.kv}>Vai trò</span>
                <span>{x.role}</span>
                <span className={visualStyles.kv}>
                  Xuất hiện ở <span className={researchStyles.hint}>(máy gom từ chuỗi)</span>
                </span>
                <span data-testid={`seq-motif-${x.motif_id}-units`}>
                  {x.unit_refs.length > 0 ? x.unit_refs.join(", ") : "Chưa unit nào"}
                  {x.planned_units.length > 0 && (
                    <span className={researchStyles.hint}>
                      {" "}
                      · AI dự định: {x.planned_units.join(", ")}
                    </span>
                  )}
                </span>
                {few && state.coverage.units_sequenced > 0 && (
                  <span className={researchStyles.chip} data-testid={`seq-motif-${x.motif_id}-few`}>
                    ⚠ Mới ở {x.unit_refs.length} unit, cần ít nhất 2 unit
                  </span>
                )}
                {x.problems.length > 0 && (
                  <ul className={visualStyles.problems} role="status">
                    {x.problems.map((p) => (
                      <li key={p}>{p}</li>
                    ))}
                  </ul>
                )}
                <div className={styles.motifActions}>
                  <Button
                    variant="ghost"
                    onClick={() => setEditing(x)}
                    disabled={disabled}
                    data-testid={`seq-motif-${x.motif_id}-edit`}
                  >
                    Sửa
                  </Button>
                  <Button
                    variant="danger"
                    onClick={() => setDeleting(x)}
                    disabled={disabled}
                    data-testid={`seq-motif-${x.motif_id}-delete`}
                  >
                    Xoá
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
        <div className={researchStyles.actions}>
          <Button
            variant="ghost"
            onClick={() => setEditing("new")}
            disabled={disabled}
            data-testid="seq-motif-add"
          >
            + Thêm motif
          </Button>
          <span className={researchStyles.grow} />
          {approved ? (
            <Button
              variant="ghost"
              onClick={() => void sequence.unapproveMotifs()}
              disabled={disabled}
              data-testid="seq-motifs-unapprove"
            >
              Bỏ duyệt motif
            </Button>
          ) : (
            <Button
              variant="primary"
              onClick={() => void sequence.approveMotifs()}
              disabled={disabled || m.motifs.length === 0 || hasProblems}
              title={hasProblems ? "Còn motif chưa đạt điều kiện kiểm" : undefined}
              data-testid="seq-motifs-approve"
            >
              Duyệt motif
            </Button>
          )}
        </div>
      </div>

      <ConceptAiDialog
        open={aiOpen}
        onClose={() => setAiOpen(false)}
        title={m.motifs.length > 0 ? "AI đặt lại motif" : "AI đặt motif"}
        impact={
          <p className={researchStyles.hint}>
            AI đọc kế hoạch hình, nhịp chung, ngôn ngữ chuyển động và các mục cách vẽ của bước 8.
            Motif giữ mã khi AI giữ nó; trường bạn đã sửa tay giữ nguyên; motif đang được unit dùng
            không bị bỏ. Danh sách motif về “Chờ duyệt”, các beat đã viết sẽ cần xác nhận lại nếu
            motif đổi.
          </p>
        }
        onSystem={(note) => void sequence.generateMotifs(note)}
        external={{
          run: (note, external) => generateSequenceMotifs(projectId, note, { external }),
          onDone: sequence.reload,
        }}
        testId="seq-motifs-dialog"
      />
      <MotifDialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        state={state}
        motif={editing === "new" ? null : editing}
        onSave={(fields) =>
          editing === "new" || editing === null
            ? sequence.addMotif(fields)
            : sequence.editMotif(editing.motif_id, fields)
        }
      />
      <FormDialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        title={`Xoá motif ${deleting?.motif_id ?? ""} · ${deleting?.name ?? ""}?`}
        confirmLabel="Xoá motif"
        confirmVariant="danger"
        hasInput={false}
        onConfirm={() => (deleting ? sequence.deleteMotif(deleting.motif_id) : Promise.resolve())}
        size="sm"
        testId="seq-motif-delete-dialog"
      >
        <p data-testid="seq-motif-delete-impact">
          {deleting && deleting.unit_refs.length > 0
            ? `Motif bị gỡ khỏi các unit đang dùng: ${deleting.unit_refs.join(", ")}. Các unit đó về “Chờ duyệt”; chuyển cảnh nối qua motif này mất motif nối.`
            : "Chưa unit nào dùng motif này."}
        </p>
        <p className={researchStyles.hint}>
          Không khôi phục được. Mã {deleting?.motif_id} không được dùng lại.
        </p>
      </FormDialog>
    </Card>
  );
}
