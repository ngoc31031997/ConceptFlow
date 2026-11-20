import { useEffect, useState } from "react";
import {
  SHOT_CAMERA_VIEWS,
  SHOT_POSITIONS,
  type StoryboardAsset,
  type StoryboardShot,
  type StoryboardUnit,
} from "../../api/client";
import researchStyles from "../research/research.module.css";
import { FormField, Select, TextArea, TextInput } from "../ui";
import { FormDialog } from "../visualplan/PlanDialogs";
import { CAMERA_LABELS, POSITION_LABELS, textLines } from "./storyboardModel";
import styles from "./storyboard.module.css";

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

interface ShotDialogProps {
  open: boolean;
  onClose: () => void;
  unit: StoryboardUnit;
  /** The shot being edited; null adds a new one at the end of the unit. */
  shot: StoryboardShot | null;
  /** Every asset of the video: the subject, the main object and the asset list are picked from them. */
  assets: StoryboardAsset[];
  onSave: (fields: Record<string, unknown>) => Promise<void>;
}

/** The select of an asset (or a free name already on the shot) with an optional empty entry. */
function AssetSelect({
  value,
  assets,
  emptyLabel,
  onChange,
  testId,
}: {
  value: string;
  assets: StoryboardAsset[];
  emptyLabel?: string;
  onChange: (value: string) => void;
  testId: string;
}) {
  const known = value === "" || assets.some((a) => a.asset_id === value);
  return (
    <Select value={value} onChange={(e) => onChange(e.target.value)} data-testid={testId}>
      {emptyLabel !== undefined ? <option value="">{emptyLabel}</option> : <option value="">Chọn</option>}
      {!known && <option value={value}>{value}</option>}
      {assets.map((a) => (
        <option key={a.asset_id} value={a.asset_id}>
          {a.asset_id} · {a.name}
        </option>
      ))}
    </Select>
  );
}

/** The select of a closed list (positions or camera views), shown with its Vietnamese name. */
function ClosedSelect({
  value,
  values,
  labels,
  emptyLabel,
  onChange,
  testId,
}: {
  value: string;
  values: readonly string[];
  labels: Record<string, string>;
  emptyLabel: string;
  onChange: (value: string) => void;
  testId: string;
}) {
  return (
    <Select value={value} onChange={(e) => onChange(e.target.value)} data-testid={testId}>
      <option value="">{emptyLabel}</option>
      {values.map((v) => (
        <option key={v} value={v}>
          {labels[v] ?? v} ({v})
        </option>
      ))}
    </Select>
  );
}

/**
 * Add or edit one shot by hand: purpose, what is where in the frame (positions and
 * camera view from the closed lists), action, on-screen text, the assets it uses
 * and the script lines it covers. An edit sends only the fields the Creator
 * changed, so only those become the Creator's and survive the AI writing the unit
 * again. Saving sends the shot back to "Chưa duyệt" and draws its frame again.
 */
export function ShotDialog({ open, onClose, unit, shot, assets, onSave }: ShotDialogProps) {
  const [purpose, setPurpose] = useState("");
  const [subject, setSubject] = useState("");
  const [subjectPosition, setSubjectPosition] = useState("");
  const [mainObject, setMainObject] = useState("");
  const [mainObjectPosition, setMainObjectPosition] = useState("");
  const [camera, setCamera] = useState("");
  const [action, setAction] = useState("");
  const [text, setText] = useState("");
  const [refs, setRefs] = useState<string[]>([]);
  const [scriptRefs, setScriptRefs] = useState<string[]>([]);
  const [notes, setNotes] = useState("");

  useEffect(() => {
    if (!open) return;
    setPurpose(shot?.shot_purpose ?? "");
    setSubject(shot?.composition.subject ?? "");
    setSubjectPosition(shot?.composition.subject_position ?? "");
    setMainObject(shot?.composition.main_object ?? "");
    setMainObjectPosition(shot?.composition.main_object_position ?? "");
    setCamera(shot?.composition.camera_view ?? "");
    setAction(shot?.action_summary ?? "");
    setText((shot?.on_screen_text ?? []).join("\n"));
    setRefs(shot?.asset_refs ?? []);
    setScriptRefs(shot?.script_refs ?? []);
    setNotes((shot?.notes ?? []).join("\n"));
    // The form starts from the shot when it opens; a refresh of the screen behind must not reset typed text.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, shot?.shot_id, unit.unit_id]);

  const composition = {
    subject,
    subject_position: subjectPosition,
    main_object: mainObject,
    main_object_position: mainObject === "" ? "" : mainObjectPosition,
    camera_view: camera,
  };
  const next: Record<string, unknown> = {
    script_refs: unit.lines.map((l) => l.id).filter((id) => scriptRefs.includes(id)),
    shot_purpose: purpose.trim(),
    composition,
    action_summary: action.trim(),
    on_screen_text: textLines(text),
    asset_refs: assets.map((a) => a.asset_id).filter((id) => refs.includes(id)),
    notes: textLines(notes),
  };
  const prev: Record<string, unknown> = shot
    ? {
        script_refs: shot.script_refs,
        shot_purpose: shot.shot_purpose,
        composition: shot.composition,
        action_summary: shot.action_summary,
        on_screen_text: shot.on_screen_text,
        asset_refs: shot.asset_refs,
        notes: shot.notes,
      }
    : {};
  const changed = Object.keys(next).filter((k) => shot === null || !same(next[k], prev[k]));
  const complete =
    purpose.trim() !== "" &&
    action.trim() !== "" &&
    subject !== "" &&
    subjectPosition !== "" &&
    camera !== "" &&
    (mainObject === "" || mainObjectPosition !== "");
  const approved = shot?.status === "approved";
  const toggle = (list: string[], set: (next: string[]) => void, id: string) =>
    set(list.includes(id) ? list.filter((x) => x !== id) : [...list, id]);

  async function save() {
    const fields: Record<string, unknown> = {};
    for (const k of changed) fields[k] = next[k];
    if (Object.keys(fields).length > 0) await onSave(fields);
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={shot ? `Sửa shot ${shot.shot_id}` : `Thêm shot vào ${unit.unit_id}`}
      confirmLabel={shot ? "Lưu và vẽ lại khung" : "Thêm shot và vẽ khung"}
      confirmVariant={approved ? "impact" : "primary"}
      confirmDisabled={!complete || (shot !== null && changed.length === 0)}
      onConfirm={save}
      size="xl"
      testId="sb-shot-dialog"
    >
      <FormField label="Mục đích">
        <TextInput
          value={purpose}
          onChange={(e) => setPurpose(e.target.value)}
          data-testid="sb-shot-purpose"
        />
      </FormField>
      <div className={researchStyles.grid2}>
        <FormField label="Chủ thể">
          <AssetSelect
            value={subject}
            assets={assets}
            onChange={setSubject}
            testId="sb-shot-subject"
          />
        </FormField>
        <FormField label="Vị trí chủ thể">
          <ClosedSelect
            value={subjectPosition}
            values={SHOT_POSITIONS}
            labels={POSITION_LABELS}
            emptyLabel="Chọn vị trí"
            onChange={setSubjectPosition}
            testId="sb-shot-subject-position"
          />
        </FormField>
      </div>
      <div className={researchStyles.grid2}>
        <FormField label="Vật chính">
          <AssetSelect
            value={mainObject}
            assets={assets}
            emptyLabel="(không có)"
            onChange={setMainObject}
            testId="sb-shot-main-object"
          />
        </FormField>
        <FormField label="Vị trí vật chính">
          <ClosedSelect
            value={mainObject === "" ? "" : mainObjectPosition}
            values={SHOT_POSITIONS}
            labels={POSITION_LABELS}
            emptyLabel={mainObject === "" ? "(không có vật chính)" : "Chọn vị trí"}
            onChange={setMainObjectPosition}
            testId="sb-shot-main-object-position"
          />
        </FormField>
      </div>
      <FormField label="Góc máy">
        <ClosedSelect
          value={camera}
          values={SHOT_CAMERA_VIEWS}
          labels={CAMERA_LABELS}
          emptyLabel="Chọn góc máy"
          onChange={setCamera}
          testId="sb-shot-camera"
        />
      </FormField>
      <FormField label="Hành động">
        <TextArea
          value={action}
          onChange={(e) => setAction(e.target.value)}
          rows={2}
          data-testid="sb-shot-action"
        />
      </FormField>
      <FormField label="Chữ trên màn hình (mỗi dòng một mục, tối đa theo luật chữ của bước 8)">
        <TextArea
          value={text}
          onChange={(e) => setText(e.target.value)}
          rows={2}
          data-testid="sb-shot-text"
        />
      </FormField>
      <div className={researchStyles.grid2}>
        <FormField label="Tài sản shot dùng">
          <div className={styles.checks}>
            {assets.length === 0 && <span className={researchStyles.hint}>Chưa có tài sản.</span>}
            {assets.map((a) => (
              <label key={a.asset_id}>
                <input
                  type="checkbox"
                  checked={refs.includes(a.asset_id)}
                  onChange={() => toggle(refs, setRefs, a.asset_id)}
                  data-testid={`sb-shot-ref-${a.asset_id}`}
                />
                {a.asset_id} · {a.name}
              </label>
            ))}
          </div>
        </FormField>
        <FormField label="Dòng lời đọc shot phủ">
          <div className={styles.checks}>
            {unit.lines.map((l) => (
              <label key={l.id}>
                <input
                  type="checkbox"
                  checked={scriptRefs.includes(l.id)}
                  onChange={() => toggle(scriptRefs, setScriptRefs, l.id)}
                  data-testid={`sb-shot-line-${l.id}`}
                />
                {l.id} · {l.text}
              </label>
            ))}
          </div>
        </FormField>
      </div>
      <FormField label="Ghi chú (mỗi dòng một mục)">
        <TextArea
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          rows={2}
          data-testid="sb-shot-notes"
        />
      </FormField>
      <div className={researchStyles.banner} data-testid="sb-shot-impact">
        {shot
          ? approved
            ? `Lưu đưa shot về “Chưa duyệt”, khung vẽ lại và unit ${unit.unit_id} mất trạng thái đã duyệt.`
            : "Lưu đưa shot về “Chưa duyệt” và vẽ lại khung nếu bố cục, chữ hay tài sản đổi."
          : `Shot mới vào cuối ${unit.unit_id} ở trạng thái “Chưa duyệt”; unit mất trạng thái đã duyệt.`}
      </div>
      <p className={researchStyles.hint}>
        Trường bạn sửa được giữ khi AI viết lại các shot của unit.
      </p>
    </FormDialog>
  );
}
