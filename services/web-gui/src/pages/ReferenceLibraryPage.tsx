import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { YoutubePromptCopy } from "../components/reference/YoutubePromptCopy";
import {
  Button,
  Card,
  CopyBox,
  Dialog,
  FormField,
  Select,
  TextArea,
  TextInput,
} from "../components/ui";
import {
  createReferenceStyle,
  deleteReferenceStyle,
  getReferenceStyleUsage,
  listProjects,
  importReferenceStyle,
  listReferenceStyles,
  updateReferenceStyle,
  type ReferenceColorRules,
  type ReferenceDrawing,
  type ReferenceStyle,
  type ReferenceStyleInput,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./ReferenceLibraryPage.module.css";

const NAME_MAX = 80;
const INTENT_MAX = 1000;
const FIELD_MAX = 300;
const PROMPT_MAX = 1000;
const LIST_MAX = 12;
const REDO_NOTE =
  "các phần đọc tham chiếu ở bước Ngôn ngữ hình ảnh sẽ cần làm lại; bản đã chốt giữ nguyên.";

/** The videos that picked a description, as the confirmation shows them. */
type Usage =
  | { state: "loading" }
  | { state: "error"; message: string }
  | { state: "ready"; videos: string[]; namesMissing: boolean };

const DRAWING_FIELDS: { key: keyof ReferenceDrawing; label: string }[] = [
  { key: "medium", label: "Chất liệu" },
  { key: "shape", label: "Hình dạng" },
  { key: "outline", label: "Đường viền" },
  { key: "fill", label: "Tô màu" },
  { key: "shading", label: "Đổ bóng" },
  { key: "light", label: "Ánh sáng" },
  { key: "depth", label: "Chiều sâu" },
  { key: "perspective", label: "Phối cảnh" },
  { key: "texture", label: "Bề mặt" },
];

const COLOR_FIELDS: { key: keyof ReferenceColorRules; label: string }[] = [
  { key: "background", label: "Nền" },
  { key: "subject", label: "Chủ thể" },
  { key: "accents", label: "Điểm nhấn" },
  { key: "note", label: "Ghi chú" },
];

const COMPOSITION_FIELDS: { key: "structure" | "layering"; label: string }[] = [
  { key: "structure", label: "Cấu trúc" },
  { key: "layering", label: "Phân lớp" },
];

const SOURCE_KINDS = [
  { value: "", label: "Chưa rõ" },
  { value: "video", label: "Video" },
  { value: "image", label: "Hình" },
  { value: "site", label: "Trang web" },
  { value: "other", label: "Khác" },
];

/**
 * The form state of the add/edit dialog: the input with the tags as the text
 * typed (comma-separated) and each list as text with one item per line.
 */
interface FormState extends Omit<
  ReferenceStyleInput,
  "tags" | "core_rules" | "avoid" | "composition"
> {
  tags: string;
  core_rules: string;
  avoid: string;
  composition: { structure: string; optional_devices: string; layering: string };
}

const EMPTY_FORM: FormState = {
  style_name: "",
  intent: "",
  tags: "",
  source: { url: "", title: "", kind: "" },
  core_rules: "",
  drawing: {
    medium: "",
    shape: "",
    outline: "",
    fill: "",
    shading: "",
    light: "",
    depth: "",
    perspective: "",
    texture: "",
  },
  color_rules: { background: "", subject: "", accents: "", note: "" },
  composition: { structure: "", optional_devices: "", layering: "" },
  avoid: "",
  prompt_template: "",
  negative_prompt: "",
};

function toForm(style: ReferenceStyleInput | null): FormState {
  if (!style) return EMPTY_FORM;
  return {
    ...style,
    tags: style.tags.join(", "),
    core_rules: style.core_rules.join("\n"),
    avoid: style.avoid.join("\n"),
    composition: {
      ...style.composition,
      optional_devices: style.composition.optional_devices.join("\n"),
    },
  };
}

/** Splits the comma-separated tag field; the server trims and drops repeats. */
function parseTags(text: string): string[] {
  return text
    .split(",")
    .map((t) => t.trim())
    .filter(Boolean);
}

/** Splits a one-item-per-line field; the server trims and drops empty lines. */
function parseLines(text: string): string[] {
  return text
    .split("\n")
    .map((t) => t.trim())
    .filter(Boolean);
}

function fromForm(form: FormState): ReferenceStyleInput {
  return {
    ...form,
    tags: parseTags(form.tags),
    core_rules: parseLines(form.core_rules),
    avoid: parseLines(form.avoid),
    composition: {
      ...form.composition,
      optional_devices: parseLines(form.composition.optional_devices),
    },
  };
}

function errorText(e: unknown): string {
  return e instanceof Error && e.message
    ? e.message
    : "Thao tác không thành công. Vui lòng thử lại.";
}

/**
 * Names the videos that picked `id`. A video is named by its topic, as the
 * video list does; when the list cannot be loaded the short ids stand in and
 * `namesMissing` says so.
 */
async function loadUsage(id: string): Promise<Usage> {
  const projectIds = await getReferenceStyleUsage(id);
  if (projectIds.length === 0) return { state: "ready", videos: [], namesMissing: false };
  try {
    const topics = new Map(
      (await listProjects()).map((p) => [p.project_id, p.topic?.trim() ?? ""]),
    );
    return {
      state: "ready",
      videos: projectIds.map((p) => topics.get(p) || p.slice(0, 8)),
      namesMissing: false,
    };
  } catch {
    return { state: "ready", videos: projectIds.map((p) => p.slice(0, 8)), namesMissing: true };
  }
}

/** Keeps the usage of the description a dialog is open for, reloaded when it changes. */
function useUsage(id: string | null): Usage | null {
  const [usage, setUsage] = useState<Usage | null>(null);
  useEffect(() => {
    if (id === null) {
      setUsage(null);
      return;
    }
    let live = true;
    setUsage({ state: "loading" });
    loadUsage(id).then(
      (u) => live && setUsage(u),
      (e) => live && setUsage({ state: "error", message: errorText(e) }),
    );
    return () => {
      live = false;
    };
  }, [id]);
  return usage;
}

/** The impact block of a confirmation: which videos use the description and what happens to them. */
function UsageImpact({ usage }: { usage: Usage | null }) {
  if (usage === null || usage.state === "loading") {
    return <p className={glass.cardHint}>Đang kiểm video nào đang chọn mô tả này…</p>;
  }
  if (usage.state === "error") {
    return (
      <p className={styles.error} role="alert" data-testid="reference-usage-error">
        Không kiểm được video nào đang chọn mô tả này: {usage.message}
      </p>
    );
  }
  if (usage.videos.length === 0) {
    return <p className={glass.cardHint}>Chưa video nào chọn mô tả này.</p>;
  }
  return (
    <div className={styles.impact} data-testid="reference-usage">
      <b>Ảnh hưởng:</b> {usage.videos.length} video đang chọn mô tả này:
      <ul>
        {usage.videos.map((v, i) => (
          <li key={`${v}-${i}`}>“{v}”</li>
        ))}
      </ul>
      Ở các video này, {REDO_NOTE}
      {usage.namesMissing && (
        <p className={glass.cardHint}>Không tải được tên video nên hiện mã video.</p>
      )}
    </div>
  );
}

const used = (usage: Usage | null) => usage?.state === "ready" && usage.videos.length > 0;
const usageKnown = (usage: Usage | null) => usage?.state === "ready";

interface EditDialogProps {
  /** The description being edited, or null to add one. */
  style: ReferenceStyle | null;
  open: boolean;
  onClose: () => void;
  onSaved: (message: string) => void;
}

/** A labelled single-line field of the form. */
function TextField({
  label,
  value,
  onChange,
  testId,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  testId: string;
}) {
  return (
    <FormField label={label}>
      <TextInput
        value={value}
        onChange={(e) => onChange(e.target.value)}
        maxLength={FIELD_MAX}
        data-testid={testId}
      />
    </FormField>
  );
}

/** A labelled multi-line field holding a list, one item per line. */
function ListField({
  label,
  value,
  onChange,
  testId,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  testId: string;
}) {
  return (
    <FormField label={`${label} · mỗi dòng một mục, tối đa ${LIST_MAX} mục`}>
      <TextArea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        data-testid={testId}
      />
    </FormField>
  );
}

/**
 * Adds or edits a description. "Dán JSON" reads a description pasted from an
 * outside AI into the form for review; saving stays a separate step. Editing
 * one that videos picked lists them above the save button.
 */
function ReferenceStyleDialog({ style, open, onClose, onSaved }: EditDialogProps) {
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pasting, setPasting] = useState(false);
  const [pasted, setPasted] = useState("");
  const [pasteError, setPasteError] = useState<string | null>(null);
  const usage = useUsage(open && style ? style.id : null);

  useEffect(() => {
    if (!open) return;
    setForm(toForm(style));
    setError(null);
    setPasting(false);
    setPasted("");
    setPasteError(null);
  }, [open, style]);

  const incomplete = form.style_name.trim() === "" || form.intent.trim() === "";
  const waitingUsage = style !== null && !usageKnown(usage);
  const setSource = (key: "url" | "title" | "kind", value: string) =>
    setForm((f) => ({ ...f, source: { ...f.source, [key]: value } }));
  const setDrawing = (key: keyof ReferenceDrawing, value: string) =>
    setForm((f) => ({ ...f, drawing: { ...f.drawing, [key]: value } }));
  const setColor = (key: keyof ReferenceColorRules, value: string) =>
    setForm((f) => ({ ...f, color_rules: { ...f.color_rules, [key]: value } }));
  const setComposition = (key: keyof FormState["composition"], value: string) =>
    setForm((f) => ({ ...f, composition: { ...f.composition, [key]: value } }));

  const readPasted = async () => {
    setBusy(true);
    setPasteError(null);
    try {
      setForm(toForm(await importReferenceStyle(pasted)));
      setPasting(false);
      setPasted("");
    } catch (e) {
      setPasteError(errorText(e));
    } finally {
      setBusy(false);
    }
  };

  const save = async () => {
    setBusy(true);
    setError(null);
    const input = fromForm(form);
    try {
      if (style) {
        await updateReferenceStyle(style.id, input);
        onSaved(`Đã lưu ${style.id}.`);
      } else {
        const created = await createReferenceStyle(input);
        onSaved(`Đã thêm ${created.id}.`);
      }
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      hasInput
      size="xl"
      title={style ? `Sửa mô tả ${style.id}` : "Thêm mô tả tham chiếu"}
      testId="reference-edit-dialog"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="reference-edit-cancel">
            Huỷ
          </Button>
          <Button
            variant={used(usage) ? "impact" : "primary"}
            onClick={save}
            disabled={busy || incomplete || waitingUsage}
            data-testid="reference-edit-save"
          >
            {busy ? "Đang lưu..." : used(usage) ? "Lưu và làm lại" : "Lưu"}
          </Button>
        </>
      }
    >
      <div className={styles.pasteBar}>
        <Button
          variant="view"
          onClick={() => setPasting((p) => !p)}
          data-testid="reference-paste-toggle"
        >
          Dán JSON
        </Button>
        <span className={glass.cardHint}>
          Dán JSON AI ngoài trả về để điền vào biểu mẫu; bạn xem lại rồi mới Lưu.
        </span>
      </div>
      {pasting && (
        <div className={`${styles.pasteBox} ${glass.reveal}`} data-testid="reference-paste-box">
          <YoutubePromptCopy />
          <FormField label="JSON đã dán">
            <TextArea
              value={pasted}
              onChange={(e) => setPasted(e.target.value)}
              rows={6}
              data-testid="reference-paste-input"
            />
          </FormField>
          <p className={glass.cardHint}>Đọc xong, các ô bên dưới được thay bằng nội dung đã dán.</p>
          {pasteError && (
            <p className={styles.error} role="alert" data-testid="reference-paste-error">
              {pasteError}
            </p>
          )}
          <div className={styles.rowActions}>
            <Button variant="ghost" onClick={() => setPasting(false)}>
              Đóng ô dán
            </Button>
            <Button
              onClick={readPasted}
              disabled={busy || pasted.trim() === ""}
              data-testid="reference-paste-read"
            >
              Điền vào biểu mẫu
            </Button>
          </div>
        </div>
      )}

      <FormField label="Tên phong cách">
        <TextInput
          value={form.style_name}
          onChange={(e) => setForm((f) => ({ ...f, style_name: e.target.value }))}
          maxLength={NAME_MAX}
          placeholder="Ví dụ: Glowing Flat Vector"
          data-testid="reference-name-input"
        />
      </FormField>
      <FormField label="Mục đích" className={glass.mtSm}>
        <TextArea
          value={form.intent}
          onChange={(e) => setForm((f) => ({ ...f, intent: e.target.value }))}
          maxLength={INTENT_MAX}
          rows={3}
          placeholder="Phong cách dùng cho loại video nào; áp dụng cho mọi chủ đề"
          data-testid="reference-intent-input"
        />
      </FormField>
      <FormField label="Nhãn · cách nhau bằng dấu phẩy, tối đa 8 nhãn" className={glass.mtSm}>
        <TextInput
          value={form.tags}
          onChange={(e) => setForm((f) => ({ ...f, tags: e.target.value }))}
          data-testid="reference-tags-input"
        />
      </FormField>

      <fieldset className={styles.group}>
        <legend>Nguồn</legend>
        <div className={styles.grid}>
          <FormField label="Địa chỉ (http hoặc https)">
            <TextInput
              value={form.source.url}
              onChange={(e) => setSource("url", e.target.value)}
              data-testid="reference-source-url-input"
            />
          </FormField>
          <TextField
            label="Tiêu đề"
            value={form.source.title}
            onChange={(v) => setSource("title", v)}
            testId="reference-source-title-input"
          />
          <FormField label="Loại">
            <Select
              value={form.source.kind}
              onChange={(e) => setSource("kind", e.target.value)}
              data-testid="reference-source-kind-select"
            >
              {SOURCE_KINDS.map((k) => (
                <option key={k.value} value={k.value}>
                  {k.label}
                </option>
              ))}
            </Select>
          </FormField>
        </div>
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Luật cốt lõi</legend>
        <ListField
          label="Luật bắt buộc"
          value={form.core_rules}
          onChange={(v) => setForm((f) => ({ ...f, core_rules: v }))}
          testId="reference-core-rules-input"
        />
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Cách vẽ</legend>
        <div className={styles.grid}>
          {DRAWING_FIELDS.map((f) => (
            <TextField
              key={f.key}
              label={f.label}
              value={form.drawing[f.key]}
              onChange={(v) => setDrawing(f.key, v)}
              testId={`reference-drawing-${f.key}-input`}
            />
          ))}
        </div>
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Màu · quan hệ giữa nền, chủ thể và điểm nhấn</legend>
        <div className={styles.grid}>
          {COLOR_FIELDS.map((f) => (
            <TextField
              key={f.key}
              label={f.label}
              value={form.color_rules[f.key]}
              onChange={(v) => setColor(f.key, v)}
              testId={`reference-color-${f.key}-input`}
            />
          ))}
        </div>
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Bố cục</legend>
        <div className={styles.grid}>
          {COMPOSITION_FIELDS.map((f) => (
            <TextField
              key={f.key}
              label={f.label}
              value={form.composition[f.key]}
              onChange={(v) => setComposition(f.key, v)}
              testId={`reference-composition-${f.key}-input`}
            />
          ))}
        </div>
        <ListField
          label="Thủ pháp tuỳ chọn"
          value={form.composition.optional_devices}
          onChange={(v) => setComposition("optional_devices", v)}
          testId="reference-composition-devices-input"
        />
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Tránh</legend>
        <ListField
          label="Những gì phá phong cách"
          value={form.avoid}
          onChange={(v) => setForm((f) => ({ ...f, avoid: v }))}
          testId="reference-avoid-input"
        />
      </fieldset>

      <fieldset className={styles.group}>
        <legend>Prompt sinh ảnh</legend>
        <FormField label="Mẫu prompt · có chỗ trống {subject} và {setting}">
          <TextArea
            value={form.prompt_template}
            onChange={(e) => setForm((f) => ({ ...f, prompt_template: e.target.value }))}
            maxLength={PROMPT_MAX}
            rows={4}
            data-testid="reference-prompt-template-input"
          />
        </FormField>
        <FormField label="Negative prompt" className={glass.mtSm}>
          <TextArea
            value={form.negative_prompt}
            onChange={(e) => setForm((f) => ({ ...f, negative_prompt: e.target.value }))}
            maxLength={PROMPT_MAX}
            rows={2}
            data-testid="reference-negative-prompt-input"
          />
        </FormField>
      </fieldset>

      {style && (
        <div className={glass.mtSm}>
          <UsageImpact usage={usage} />
        </div>
      )}
      {error && (
        <p
          className={`${styles.error} ${glass.mtSm}`}
          role="alert"
          data-testid="reference-edit-error"
        >
          {error}
        </p>
      )}
    </Dialog>
  );
}

/** A read-only list of a description, or a dash when it is empty. */
function DetailList({ items, testId }: { items: string[]; testId: string }) {
  if (items.length === 0) return <p className={glass.cardHint}>—</p>;
  return (
    <ul className={styles.detailList} data-testid={testId}>
      {items.map((it, i) => (
        <li key={`${i}-${it}`}>{it}</li>
      ))}
    </ul>
  );
}

/** A prompt of a description in a copy box, or a dash when it is empty. */
function DetailPrompt({ text, what, testId }: { text: string; what: string; testId: string }) {
  if (!text) return <p className={glass.cardHint}>—</p>;
  return (
    <CopyBox text={text} what={what} copyTestId={`${testId}-copy`}>
      <pre className={styles.prompt} data-testid={testId}>
        {text}
      </pre>
    </CopyBox>
  );
}

/** Read-only view of one description with every field. */
function ReferenceDetailDialog({
  style,
  onClose,
}: {
  style: ReferenceStyle | null;
  onClose: () => void;
}) {
  const rows = (pairs: { label: string; value: string }[]) =>
    pairs.map((p) => (
      <div key={p.label} className={styles.detailRow}>
        <dt>{p.label}</dt>
        <dd>{p.value || <span className={glass.cardHint}>—</span>}</dd>
      </div>
    ));
  return (
    <Dialog
      open={style !== null}
      onClose={onClose}
      size="lg"
      title={style ? `${style.id} · ${style.style_name}` : ""}
      testId="reference-detail-dialog"
      footer={
        <Button variant="ghost" onClick={onClose} data-testid="reference-detail-close">
          Đóng
        </Button>
      }
    >
      {style && (
        <>
          <p className={styles.description}>{style.intent}</p>
          <h3 className={styles.detailHead}>Luật cốt lõi</h3>
          <DetailList items={style.core_rules} testId="reference-detail-core-rules" />
          <h3 className={styles.detailHead}>Cách vẽ</h3>
          <dl className={styles.detail} data-testid="reference-detail-drawing">
            {rows(DRAWING_FIELDS.map((f) => ({ label: f.label, value: style.drawing[f.key] })))}
          </dl>
          <h3 className={styles.detailHead}>Màu</h3>
          <dl className={styles.detail} data-testid="reference-detail-color">
            {rows(COLOR_FIELDS.map((f) => ({ label: f.label, value: style.color_rules[f.key] })))}
          </dl>
          <h3 className={styles.detailHead}>Bố cục</h3>
          <dl className={styles.detail} data-testid="reference-detail-composition">
            {rows(
              COMPOSITION_FIELDS.map((f) => ({ label: f.label, value: style.composition[f.key] })),
            )}
          </dl>
          <h3 className={styles.detailHead}>Thủ pháp tuỳ chọn</h3>
          <DetailList
            items={style.composition.optional_devices}
            testId="reference-detail-devices"
          />
          <h3 className={styles.detailHead}>Tránh</h3>
          <DetailList items={style.avoid} testId="reference-detail-avoid" />
          <h3 className={styles.detailHead}>Mẫu prompt</h3>
          <DetailPrompt
            text={style.prompt_template}
            what="mẫu prompt"
            testId="reference-detail-prompt-template"
          />
          <h3 className={styles.detailHead}>Negative prompt</h3>
          <DetailPrompt
            text={style.negative_prompt}
            what="negative prompt"
            testId="reference-detail-negative-prompt"
          />
          <h3 className={styles.detailHead}>Nguồn</h3>
          <SourceLink style={style} />
        </>
      )}
    </Dialog>
  );
}

/** The source as a link to its address (opened in a new tab), or plain text when it has none. */
function SourceLink({ style }: { style: ReferenceStyle }) {
  const { url, title, kind } = style.source;
  const kindLabel = SOURCE_KINDS.find((k) => k.value === kind && k.value !== "")?.label;
  const text = title || url;
  if (!text) return <span className={glass.cardHint}>—</span>;
  return (
    <span>
      {url ? (
        <a
          href={url}
          target="_blank"
          rel="noreferrer noopener"
          onClick={(e) => e.stopPropagation()}
        >
          {text}
        </a>
      ) : (
        text
      )}
      {kindLabel && <span className={glass.cardHint}> · {kindLabel}</span>}
    </span>
  );
}

interface DeleteDialogProps {
  style: ReferenceStyle | null;
  onClose: () => void;
  onDeleted: (message: string) => void;
}

/** Confirms deleting a description, naming the videos that picked it. */
function DeleteReferenceDialog({ style, onClose, onDeleted }: DeleteDialogProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const usage = useUsage(style ? style.id : null);

  useEffect(() => setError(null), [style]);

  const remove = async () => {
    if (!style) return;
    setBusy(true);
    setError(null);
    try {
      await deleteReferenceStyle(style.id);
      onDeleted(`Đã xoá ${style.id}.`);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog
      open={style !== null}
      onClose={onClose}
      role="alertdialog"
      size="sm"
      title={style ? `Xoá mô tả ${style.id}?` : ""}
      testId="reference-delete-dialog"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid="reference-delete-cancel">
            Huỷ
          </Button>
          <Button
            variant="danger"
            onClick={remove}
            disabled={busy || !usageKnown(usage)}
            data-testid="reference-delete-confirm"
          >
            {busy ? "Đang xoá..." : "Xoá mô tả"}
          </Button>
        </>
      }
    >
      {style && (
        <p>
          Xoá mô tả <b>{style.style_name}</b>. Không khôi phục được; mã {style.id} không dùng lại.
        </p>
      )}
      <div className={glass.mtSm}>
        <UsageImpact usage={usage} />
      </div>
      {error && (
        <p
          className={`${styles.error} ${glass.mtSm}`}
          role="alert"
          data-testid="reference-delete-error"
        >
          {error}
        </p>
      )}
    </Dialog>
  );
}

/**
 * Cài đặt › Thư viện tham chiếu: text descriptions of a visual style, shared by
 * every video. The Ngôn ngữ hình ảnh step lets a video pick the ones its AI
 * reads; this page adds, edits and deletes them.
 */
export function ReferenceLibraryPage() {
  const [rows, setRows] = useState<ReferenceStyle[]>([]);
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState<string | null>(null);
  const [editing, setEditing] = useState<{ style: ReferenceStyle | null } | null>(null);
  const [deleting, setDeleting] = useState<ReferenceStyle | null>(null);
  const [viewing, setViewing] = useState<ReferenceStyle | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      setRows(await listReferenceStyles());
    } catch (e) {
      setStatus(`Không tải được thư viện tham chiếu: ${errorText(e)}`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const done = (message: string) => {
    setEditing(null);
    setDeleting(null);
    setStatus(message);
    void reload();
  };

  return (
    <div data-testid="reference-library-page">
      <AppShell
        title="Thư viện tham chiếu"
        subtitle="Mô tả phong cách để AI đọc ở bước Ngôn ngữ hình ảnh. Mỗi mục có mục đích, luật cốt lõi, cách vẽ, quan hệ màu, bố cục, điều cần tránh và prompt sinh ảnh của một phong cách. Dùng chung cho mọi video; ở bước 8 chỉ tick chọn. Bấm một dòng để xem đủ."
        wide
      >
        <div className={styles.toolbar}>
          <Button
            onClick={() => setEditing({ style: null })}
            disabled={loading}
            data-testid="reference-add-button"
          >
            + Thêm mô tả
          </Button>
        </div>
        {status && (
          <p
            className={`${glass.cardHint} ${glass.mtXs}`}
            role="status"
            data-testid="reference-status"
          >
            {status}
          </p>
        )}
        <Card>
          {loading && rows.length === 0 ? (
            <p className={glass.cardHint}>Đang tải…</p>
          ) : rows.length === 0 ? (
            <p className={glass.cardHint} data-testid="reference-empty">
              Thư viện chưa có mô tả nào. Bấm “+ Thêm mô tả” để thêm.
            </p>
          ) : (
            <div className={styles.tableWrap}>
              <table className={styles.table} data-testid="reference-table">
                <thead>
                  <tr>
                    <th>Mã</th>
                    <th>Tên</th>
                    <th>Mục đích</th>
                    <th>Nhãn</th>
                    <th>Nguồn</th>
                    <th aria-label="Thao tác" />
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr
                      key={r.id}
                      className={styles.clickRow}
                      onClick={() => setViewing(r)}
                      data-testid={`reference-row-${r.id}`}
                    >
                      <td className={styles.id}>{r.id}</td>
                      <td>
                        <b>{r.style_name}</b>
                      </td>
                      <td className={styles.description}>{r.intent}</td>
                      <td>
                        <span className={styles.tags}>
                          {r.tags.map((t) => (
                            <span key={t} className={styles.tag}>
                              {t}
                            </span>
                          ))}
                        </span>
                      </td>
                      <td>
                        <SourceLink style={r} />
                      </td>
                      <td>
                        <div className={styles.rowActions}>
                          <Button
                            variant="ghost"
                            onClick={(e) => {
                              e.stopPropagation();
                              setEditing({ style: r });
                            }}
                            data-testid={`reference-edit-${r.id}`}
                          >
                            Sửa
                          </Button>
                          <Button
                            variant="danger"
                            onClick={(e) => {
                              e.stopPropagation();
                              setDeleting(r);
                            }}
                            data-testid={`reference-delete-${r.id}`}
                          >
                            Xoá
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      </AppShell>
      <ReferenceStyleDialog
        style={editing?.style ?? null}
        open={editing !== null}
        onClose={() => setEditing(null)}
        onSaved={done}
      />
      <ReferenceDetailDialog style={viewing} onClose={() => setViewing(null)} />
      <DeleteReferenceDialog style={deleting} onClose={() => setDeleting(null)} onDeleted={done} />
    </div>
  );
}
