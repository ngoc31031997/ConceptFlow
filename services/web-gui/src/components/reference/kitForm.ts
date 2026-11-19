import type { KitFocus, ReferenceKit } from "../../api/client";

/** What a kit field holds: a 0..1 number, a -1..1 number, a px blur radius (0 or more) or a #RRGGBB colour. */
export type KitFieldKind = "unit" | "signed" | "px" | "color" | "focus";

export interface KitFieldDef {
  /** Dotted path into the kit, e.g. "specular.glossy". */
  path: string;
  label: string;
  kind: KitFieldKind;
}

export interface KitFieldGroup {
  title: string;
  fields: KitFieldDef[];
}

export const KIT_FOCUS_LAYERS: { value: KitFocus; label: string }[] = [
  { value: "sky", label: "Trời" },
  { value: "far", label: "Xa" },
  { value: "mid", label: "Giữa" },
  { value: "near", label: "Gần" },
  { value: "figures", label: "Nhân vật" },
];

const u = (path: string, label: string): KitFieldDef => ({ path, label, kind: "unit" });
const signed = (path: string, label: string): KitFieldDef => ({ path, label, kind: "signed" });
const px = (path: string, label: string): KitFieldDef => ({ path, label, kind: "px" });
const color = (path: string, label: string): KitFieldDef => ({ path, label, kind: "color" });

/** Every field of a kit, grouped the way the card shows them. */
export const KIT_FIELD_GROUPS: KitFieldGroup[] = [
  {
    title: "Hình khối",
    fields: [
      u("outline", "Viền"),
      u("radius", "Bo góc"),
      u("rimWidth", "Bề rộng ánh viền"),
      u("rimStrength", "Độ mạnh ánh viền"),
      u("shadowStrength", "Độ mạnh bóng"),
      u("aoStrength", "Độ mạnh bóng khuất"),
      u("sss", "Tán xạ dưới da"),
      color("ink", "Màu nét chi tiết"),
    ],
  },
  {
    title: "Độ bóng",
    fields: [
      u("specular.glossy", "Bóng"),
      u("specular.metal", "Kim loại"),
      u("specular.glass", "Kính"),
    ],
  },
  {
    title: "Bề mặt",
    fields: [
      u("texture.fur", "Lông"),
      u("texture.fabric", "Vải"),
      u("texture.wood", "Gỗ"),
      u("texture.stone", "Đá"),
      u("texture.organic", "Hữu cơ"),
      u("texture.leaf", "Lá"),
    ],
  },
  {
    title: "Hiệu ứng ảnh",
    fields: [
      u("glowBlur", "Độ nhoè phát sáng"),
      u("bloom", "Bloom"),
      u("grain", "Hạt"),
      u("vignette", "Tối góc"),
      u("haze.far", "Sương lớp xa"),
      u("haze.mid", "Sương lớp giữa"),
    ],
  },
  {
    title: "Nhoè theo độ sâu (px)",
    fields: [
      px("dof.sky", "Trời"),
      px("dof.far", "Xa"),
      px("dof.mid", "Giữa"),
      px("dof.near", "Gần"),
    ],
  },
  {
    title: "Cảnh xem trước",
    fields: [
      u("previewLook.light.x", "Ánh sáng chính · vị trí ngang"),
      u("previewLook.light.y", "Ánh sáng chính · vị trí dọc"),
      color("previewLook.light.color", "Ánh sáng chính · màu"),
      u("previewLook.light.intensity", "Ánh sáng chính · cường độ"),
      color("previewLook.ambient", "Ánh sáng nền · màu"),
      color("previewLook.atmosphere.color", "Không khí · màu"),
      u("previewLook.atmosphere.density", "Không khí · độ đặc"),
      { path: "previewLook.focus", label: "Lớp nét", kind: "focus" },
      signed("previewLook.grade.warmth", "Chỉnh màu · ấm"),
      signed("previewLook.grade.contrast", "Chỉnh màu · tương phản"),
      signed("previewLook.grade.saturation", "Chỉnh màu · bão hoà"),
    ],
  },
];

const ALL_FIELDS = KIT_FIELD_GROUPS.flatMap((g) => g.fields);

/** The kit as the text typed in each field, keyed by dotted path. */
export type KitDraft = Record<string, string>;

function read(obj: unknown, path: string): unknown {
  return path
    .split(".")
    .reduce<unknown>((o, k) => (o as Record<string, unknown> | undefined)?.[k], obj);
}

/** A kit's fields as text; an absent kit gives the starting values of a new one. */
export function kitToDraft(kit: ReferenceKit | null): KitDraft {
  const draft: KitDraft = {};
  for (const f of ALL_FIELDS) {
    const v = kit ? read(kit, f.path) : undefined;
    if (v !== undefined) draft[f.path] = String(v);
    else if (f.kind === "color") draft[f.path] = "#000000";
    else if (f.kind === "focus") draft[f.path] = "figures";
    else draft[f.path] = "0";
  }
  return draft;
}

const COLOR = /^#[0-9A-Fa-f]{6}$/;

/** Checks one field's text; returns the problem in Vietnamese, or null when fine. */
export function kitFieldProblem(f: KitFieldDef, text: string): string | null {
  if (f.kind === "color") return COLOR.test(text.trim()) ? null : `${f.label}: cần là màu #RRGGBB`;
  if (f.kind === "focus") {
    return KIT_FOCUS_LAYERS.some((l) => l.value === text) ? null : `${f.label}: chọn một lớp`;
  }
  const n = Number(text);
  if (text.trim() === "" || !Number.isFinite(n)) return `${f.label}: cần là một số`;
  if (f.kind === "unit" && (n < 0 || n > 1)) return `${f.label}: cần từ 0 đến 1`;
  if (f.kind === "signed" && (n < -1 || n > 1)) return `${f.label}: cần từ -1 đến 1`;
  if (f.kind === "px" && n < 0) return `${f.label}: cần từ 0 trở lên`;
  return null;
}

/** The first problem of a draft, or null when every field is valid. */
export function kitDraftProblem(draft: KitDraft): string | null {
  for (const f of ALL_FIELDS) {
    const problem = kitFieldProblem(f, draft[f.path] ?? "");
    if (problem) return problem;
  }
  return null;
}

/** Builds the kit from a valid draft; call only when kitDraftProblem is null. */
export function draftToKit(draft: KitDraft): ReferenceKit {
  const kit: Record<string, unknown> = {};
  for (const f of ALL_FIELDS) {
    const text = draft[f.path].trim();
    const value = f.kind === "unit" || f.kind === "signed" || f.kind === "px" ? Number(text) : text;
    const keys = f.path.split(".");
    let node = kit;
    for (const k of keys.slice(0, -1)) {
      node[k] = (node[k] as Record<string, unknown> | undefined) ?? {};
      node = node[k] as Record<string, unknown>;
    }
    node[keys[keys.length - 1]] = value;
  }
  return kit as unknown as ReferenceKit;
}
