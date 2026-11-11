import type { VisualConceptUnitKey } from "../../api/client";

/**
 * The shape of one field of a visual-concept part, as the screen shows it and
 * the hand-edit form edits it. It mirrors the part types of visual-service
 * (internal/domain/visual_concept.go).
 */
export type FieldSpec =
  | { key: string; label: string; kind: "text" | "longtext" | "lines" | "number" | "bool" }
  | { key: string; label: string; kind: "enum"; options: readonly string[] }
  | { key: string; label: string; kind: "rows"; columns: readonly FieldSpec[] }
  | { key: string; label: string; kind: "object"; fields: readonly FieldSpec[] };

/** How far a thing is drawn from how it looks. */
export const ABSTRACTION_LEVELS = [
  "realistic",
  "semi_real",
  "stylized",
  "character",
  "diagram",
  "symbol",
  "abstract",
  "hybrid",
] as const;

const thing: readonly FieldSpec[] = [
  { key: "concept_id", label: "Mã", kind: "text" },
  { key: "term", label: "Khái niệm", kind: "text" },
  { key: "draw", label: "Vẽ là", kind: "longtext" },
  { key: "must_not_imply", label: "Không được gợi", kind: "lines" },
  { key: "facts", label: "Dữ kiện", kind: "lines" },
];

/** The fields of each part, in the order the screen shows them. */
export const PART_SCHEMAS: Record<VisualConceptUnitKey, readonly FieldSpec[]> = {
  creative_direction: [
    { key: "style_name", label: "Tên phong cách", kind: "text" },
    { key: "description", label: "Mô tả", kind: "longtext" },
    { key: "why", label: "Vì sao hợp", kind: "longtext" },
    { key: "feeling", label: "Cảm giác", kind: "longtext" },
    { key: "tone", label: "Giọng hình", kind: "lines" },
    { key: "principles", label: "Nguyên tắc", kind: "lines" },
    {
      key: "abstraction_level",
      label: "Mức trừu tượng",
      kind: "enum",
      options: ABSTRACTION_LEVELS,
    },
    {
      key: "abstraction_map",
      label: "Mức trừu tượng theo loại",
      kind: "rows",
      columns: [
        { key: "for", label: "Dùng cho", kind: "text" },
        { key: "level", label: "Mức", kind: "enum", options: ABSTRACTION_LEVELS },
        { key: "example", label: "Ví dụ", kind: "longtext" },
      ],
    },
    { key: "refs_used", label: "Tham chiếu dựa vào", kind: "lines" },
    {
      key: "examples",
      label: "Nên và không nên",
      kind: "rows",
      columns: [
        { key: "do", label: "Nên", kind: "longtext" },
        { key: "dont", label: "Không nên", kind: "longtext" },
      ],
    },
  ],
  color_system: [
    { key: "background", label: "Nền", kind: "text" },
    {
      key: "palette",
      label: "Bảng màu",
      kind: "rows",
      columns: [
        { key: "token", label: "Token", kind: "text" },
        { key: "hex", label: "Mã màu", kind: "text" },
        { key: "name", label: "Tên", kind: "text" },
        { key: "role", label: "Dùng cho", kind: "text" },
      ],
    },
    {
      key: "journey",
      label: "Theo hành trình",
      kind: "rows",
      columns: [
        { key: "beat_keys", label: "Beat", kind: "lines" },
        { key: "mood", label: "Cảm xúc", kind: "text" },
        { key: "accent_token", label: "Màu nhấn", kind: "text" },
        { key: "background_token", label: "Nền (trống = nền chung)", kind: "text" },
      ],
    },
    {
      key: "contrast",
      label: "Tương phản",
      kind: "object",
      fields: [
        { key: "text_token", label: "Màu chữ", kind: "text" },
        { key: "text_min", label: "Tương phản chữ tối thiểu (x:1)", kind: "number" },
        { key: "subject_vs_bg", label: "Chủ thể so với nền", kind: "longtext" },
        { key: "accent_per_frame", label: "Số màu nhấn tối đa mỗi khung", kind: "number" },
      ],
    },
    {
      key: "exceptions",
      label: "Ngoại lệ",
      kind: "rows",
      columns: [
        { key: "token", label: "Token", kind: "text" },
        { key: "beat_keys", label: "Chỉ ở beat", kind: "lines" },
        { key: "why", label: "Vì sao", kind: "longtext" },
      ],
    },
    { key: "examples", label: "Ví dụ", kind: "lines" },
  ],
  visual_language: [
    { key: "shape_rules", label: "Hình dạng", kind: "lines" },
    { key: "line_rules", label: "Đường nét", kind: "lines" },
    { key: "lighting_rules", label: "Ánh sáng", kind: "lines" },
    { key: "depth_rules", label: "Chiều sâu và phối cảnh", kind: "lines" },
    { key: "surface_rules", label: "Bề mặt: tô, kết cấu, chất liệu", kind: "lines" },
    { key: "layout_rules", label: "Layout: cấu trúc, lớp, thành phần cố định", kind: "lines" },
    { key: "composition_rules", label: "Bố cục (theo chủ đề và cảnh)", kind: "lines" },
    { key: "examples", label: "Ví dụ", kind: "lines" },
  ],
  representation_rules: [
    {
      key: "characters",
      label: "Nhân vật",
      kind: "object",
      fields: [
        { key: "approach", label: "Cách làm nhân vật", kind: "longtext" },
        { key: "allowed_variations", label: "Biến thể được phép", kind: "lines" },
        { key: "items", label: "Từng nhân vật", kind: "rows", columns: thing },
      ],
    },
    { key: "objects", label: "Vật thể", kind: "rows", columns: thing },
    { key: "invisible", label: "Khái niệm vô hình", kind: "rows", columns: thing },
    {
      key: "scientific_concepts",
      label: "Hiện tượng khoa học",
      kind: "rows",
      columns: [
        { key: "concept_id", label: "Mã", kind: "text" },
        { key: "term", label: "Khái niệm", kind: "text" },
        { key: "representation", label: "Thể hiện", kind: "longtext" },
        { key: "must_not_imply", label: "Không được gợi", kind: "lines" },
        { key: "facts", label: "Dữ kiện", kind: "lines" },
      ],
    },
    { key: "examples", label: "Ví dụ", kind: "lines" },
  ],
  typography: [
    { key: "font", label: "Font chữ trên video", kind: "text" },
    { key: "font_direction", label: "Hướng chữ", kind: "text" },
    { key: "label_style", label: "Kiểu nhãn", kind: "text" },
    {
      key: "labels",
      label: "Nhãn",
      kind: "object",
      fields: [
        { key: "max_words", label: "Số từ tối đa", kind: "number" },
        { key: "when", label: "Lúc hiện", kind: "longtext" },
        { key: "min_px", label: "Cỡ chữ nhỏ nhất (px)", kind: "number" },
      ],
    },
    { key: "captions", label: "Chú thích", kind: "longtext" },
    {
      key: "terms",
      label: "Thuật ngữ",
      kind: "rows",
      columns: [
        { key: "say", label: "Cách nói", kind: "text" },
        { key: "show_on_screen", label: "Hiện chữ trên hình", kind: "bool" },
      ],
    },
    { key: "formulas", label: "Công thức", kind: "longtext" },
  ],
  motion_language: [
    { key: "principles", label: "Nguyên tắc", kind: "lines" },
    {
      key: "pace",
      label: "Nhịp theo beat",
      kind: "rows",
      columns: [
        { key: "beat_keys", label: "Beat", kind: "lines" },
        { key: "pace", label: "Nhịp", kind: "enum", options: ["slow", "medium", "fast"] },
      ],
    },
    { key: "preferred_transitions", label: "Chuyển cảnh ưa dùng", kind: "lines" },
    { key: "character_motion", label: "Nhân vật", kind: "longtext" },
    { key: "avoid", label: "Tránh", kind: "lines" },
    { key: "examples", label: "Ví dụ", kind: "lines" },
  ],
  do_not_use: [
    {
      key: "items",
      label: "Không dùng",
      kind: "rows",
      columns: [
        { key: "text", label: "Không dùng", kind: "longtext" },
        { key: "why", label: "Vì sao", kind: "longtext" },
        { key: "source", label: "Loại", kind: "enum", options: ["accuracy", "style"] },
        { key: "facts", label: "Dữ kiện", kind: "lines" },
      ],
    },
  ],
  feasibility: [
    {
      key: "scope",
      label: "Phạm vi",
      kind: "object",
      fields: [
        { key: "beats", label: "Beat (máy đếm)", kind: "number" },
        { key: "est_shots", label: "Số shot ước lượng", kind: "number" },
        { key: "settings", label: "Số bối cảnh", kind: "number" },
        { key: "characters", label: "Nhân vật (máy đếm)", kind: "number" },
      ],
    },
    {
      key: "checks",
      label: "Kiểm khả thi",
      kind: "rows",
      columns: [
        { key: "item", label: "Hạng mục", kind: "text" },
        { key: "assessment", label: "Đánh giá", kind: "longtext" },
        { key: "verdict", label: "Kết luận", kind: "enum", options: ["ok", "risk"] },
        { key: "mitigation", label: "Cách giảm rủi ro", kind: "longtext" },
      ],
    },
  ],
};

/** The empty value of a field. */
export function emptyOf(spec: FieldSpec): unknown {
  switch (spec.kind) {
    case "number":
      return 0;
    case "bool":
      return false;
    case "lines":
    case "rows":
      return [];
    case "object":
      return Object.fromEntries(spec.fields.map((f) => [f.key, emptyOf(f)]));
    case "enum":
      return spec.options[0];
    default:
      return "";
  }
}

const asList = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);
const asObj = (v: unknown): Record<string, unknown> =>
  v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
const asText = (v: unknown): string => (typeof v === "string" ? v : v == null ? "" : String(v));

/** The one-line summary a folded part shows. */
export function partSummary(
  key: VisualConceptUnitKey,
  content: Record<string, unknown> | null,
): string {
  if (!content) return "Chưa viết";
  const c = content;
  switch (key) {
    case "creative_direction":
      return [
        asText(c.style_name),
        `mức trừu tượng: ${asText(c.abstraction_level)}`,
        asList(c.tone).map(asText).join(", "),
      ]
        .filter(Boolean)
        .join(" · ");
    case "color_system": {
      const contrast = asObj(c.contrast);
      return `${asList(c.palette).length} token màu · chữ ≥ ${asText(contrast.text_min)}:1 · ${asList(c.journey).length} đoạn hành trình`;
    }
    case "visual_language":
      return [asList(c.shape_rules)[0], asList(c.lighting_rules)[0], asList(c.composition_rules)[0]]
        .map(asText)
        .filter(Boolean)
        .join(" · ");
    case "representation_rules":
      return `${asList(asObj(c.characters).items).length} nhân vật · ${asList(c.objects).length} vật thể · ${asList(c.invisible).length} khái niệm vô hình · ${asList(c.scientific_concepts).length} hiện tượng`;
    case "typography":
      return `${asText(c.font)} · nhãn tối đa ${asText(asObj(c.labels).max_words)} từ`;
    case "motion_language":
      return asList(c.pace)
        .map((p) => `${asList(asObj(p).beat_keys).map(asText).join(", ")} ${asText(asObj(p).pace)}`)
        .join(" · ");
    case "do_not_use": {
      const items = asList(c.items).map(asObj);
      const acc = items.filter((i) => i.source === "accuracy").length;
      return `${items.length} điều cấm: ${acc} vì độ chính xác, ${items.length - acc} vì phong cách`;
    }
    case "feasibility": {
      const s = asObj(c.scope);
      const risks = asList(c.checks).filter((x) => asObj(x).verdict === "risk").length;
      return `${asText(s.beats)} beat · khoảng ${asText(s.est_shots)} shot · ${asText(s.settings)} bối cảnh · ${asText(s.characters)} nhân vật · ${risks} rủi ro`;
    }
  }
}

export { asList, asObj, asText };
