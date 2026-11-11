import type { PlanMethod, PlanRelationship, VisualPlanState, VisualPlanUnit } from "../../api/client";

/** How each approach is named on the screen; the keys are the contract's list. */
export const METHOD_LABELS: Record<PlanMethod, string> = {
  spatial_composition: "Bố cục không gian",
  split_perspective_comparison: "So sánh chia góc nhìn",
  diagram: "Sơ đồ",
  analogy: "Phép so sánh",
  process_sequence: "Trình tự quá trình",
  scale_comparison: "So sánh quy mô",
  data_visual: "Hình hoá số liệu",
  character_action: "Nhân vật hành động",
  text_emphasis: "Nhấn bằng chữ",
  timeline: "Dòng thời gian",
  cross_section: "Mặt cắt",
};

export const RELATIONSHIP_LABELS: Record<PlanRelationship, string> = {
  direct_illustration: "Minh hoạ trực tiếp",
  contrast: "Đối chiếu",
  complement: "Bổ sung",
  metaphor: "Ẩn dụ",
};

export const METHODS = Object.keys(METHOD_LABELS) as PlanMethod[];
export const RELATIONSHIPS = Object.keys(RELATIONSHIP_LABELS) as PlanRelationship[];

/** Seconds as the screen writes them: whole numbers bare, others with one decimal. */
export function seconds(n: number): string {
  const rounded = Math.round(n * 10) / 10;
  return `${Number.isInteger(rounded) ? rounded : rounded.toFixed(1)}s`;
}

/** "b2.1–b2.3" for a run of lines, "b2.1" for one. */
export function refsLabel(refs: string[]): string {
  if (refs.length === 0) return "";
  return refs.length === 1 ? refs[0] : `${refs[0]}–${refs[refs.length - 1]}`;
}

/** Whether the unit has a question the Creator has not answered. */
export function hasOpenQuestion(u: VisualPlanUnit): boolean {
  return u.open_questions.some((q) => q.answer.trim() === "");
}

/** The filters over the units of step 9. */
export type PlanFilter = "all" | "pending" | "approved" | "risk" | "question";

const FILTERS: Record<PlanFilter, (u: VisualPlanUnit) => boolean> = {
  all: () => true,
  pending: (u) => u.status !== "approved",
  approved: (u) => u.status === "approved",
  risk: (u) => u.risk_notes.length > 0,
  question: hasOpenQuestion,
};

export const FILTER_LABELS: Record<PlanFilter, string> = {
  all: "Tất cả",
  pending: "Cần xem lại",
  approved: "Đã duyệt",
  risk: "Có rủi ro",
  question: "Có câu hỏi",
};

export function allUnits(state: VisualPlanState): VisualPlanUnit[] {
  return state.beats.flatMap((b) => b.units);
}

export function matchesFilter(u: VisualPlanUnit, f: PlanFilter): boolean {
  return FILTERS[f](u);
}

/** How many units each filter keeps. */
export function filterCounts(state: VisualPlanState): Record<PlanFilter, number> {
  const units = allUnits(state);
  const out = {} as Record<PlanFilter, number>;
  for (const f of Object.keys(FILTERS) as PlanFilter[]) out[f] = units.filter(FILTERS[f]).length;
  return out;
}

/** The beat a line id belongs to ("b2.4" → "b2"). */
export function beatOfLine(lineId: string): string {
  return lineId.split(".")[0];
}

/** A unit can be approved now: it is written, has no problem and its beat is not stale. */
export function approvable(state: VisualPlanState): VisualPlanUnit[] {
  return state.beats
    .filter((b) => !b.stale)
    .flatMap((b) => b.units)
    .filter((u) => u.status === "pending" && u.problems.length === 0);
}

export interface PlanCriterion {
  label: string;
  ok: boolean;
  detail: string;
}

/** What must hold before the plan can be locked, read off the screen's state. */
export function planCriteria(state: VisualPlanState): PlanCriterion[] {
  const units = allUnits(state);
  const { lines_total: total, lines_covered: covered, duplicated } = state.coverage;
  const stale = state.beats.filter((b) => b.stale);
  const openQuestions = units.filter(hasOpenQuestion).length;
  const failing = state.checks.filter((c) => !c.ok && !c.accepted);
  const openFindings = state.review.open_count;
  return [
    { label: "Phủ 100% câu kịch bản", ok: total > 0 && covered === total, detail: `${covered}/${total} câu` },
    { label: "Không câu nào thuộc hai unit", ok: duplicated.length === 0, detail: duplicated.join(", ") },
    {
      label: "Mọi unit đã duyệt",
      ok: state.units_total > 0 && state.units_approved === state.units_total,
      detail: `${state.units_approved}/${state.units_total}`,
    },
    { label: "Không beat nào lỗi thời so với kịch bản", ok: stale.length === 0, detail: stale.map((b) => b.key).join(", ") },
    { label: "Mọi câu hỏi mở đã trả lời", ok: openQuestions === 0, detail: openQuestions > 0 ? `${openQuestions} unit còn câu hỏi` : "" },
    { label: "Đã rà soát ít nhất một lần", ok: state.review.reviewed, detail: `${state.review.rounds} lượt` },
    {
      label: "Không còn vấn đề mở",
      ok: failing.length === 0 && openFindings === 0,
      detail: `${failing.length} máy kiểm, ${openFindings} của AI`,
    },
  ];
}
