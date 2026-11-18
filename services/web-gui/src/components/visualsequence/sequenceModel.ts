import type { SequenceTempo, SequenceUnit, VisualSequenceState } from "../../api/client";

/** How each tempo is named on the screen; the keys are the contract's list. */
export const TEMPO_LABELS: Record<SequenceTempo, string> = {
  slow: "Chậm",
  medium: "Vừa",
  fast: "Nhanh",
  calm: "Lắng",
};

export const TEMPOS = Object.keys(TEMPO_LABELS) as SequenceTempo[];

/** The name of a tempo, or the raw value when it is outside the list. */
export function tempoLabel(tempo: string): string {
  return TEMPO_LABELS[tempo as SequenceTempo] ?? tempo;
}

/** How many motif colours the screen cycles through. */
export const MOTIF_COLOURS = 8;

/** The colour slot of a motif, by its place in the list, so it is the same dot everywhere. */
export function motifSlot(state: VisualSequenceState, motifId: string): number {
  const at = state.motifs.motifs.findIndex((m) => m.motif_id === motifId);
  return at < 0 ? -1 : at % MOTIF_COLOURS;
}

/** The name of a motif for a tag, or its id when it is unknown. */
export function motifName(state: VisualSequenceState, motifId: string): string {
  return state.motifs.motifs.find((m) => m.motif_id === motifId)?.name ?? motifId;
}

/** Every unit of the plan, in plan order. */
export function allUnits(state: VisualSequenceState): SequenceUnit[] {
  return state.beats.flatMap((b) => b.units);
}

/** Items that can be approved now: written, pending, without problems, in a beat that is not stale. */
export function approvable(state: VisualSequenceState): SequenceUnit[] {
  return state.beats
    .filter((b) => !b.stale)
    .flatMap((b) => b.units)
    .filter((u) => u.item !== null && u.item.status === "pending" && u.problems.length === 0);
}

/** The motifs are approved and still fit their inputs, so the AI can write beats. */
export function motifsReady(state: VisualSequenceState): boolean {
  return (
    state.motifs.motifs.length > 0 && state.motifs.status === "approved" && !state.motifs.stale
  );
}

/** Minutes and seconds ("6:12"). */
export function clock(s: number): string {
  const total = Math.round(s);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

export interface SequenceCriterion {
  label: string;
  ok: boolean;
  detail: string;
}

/** What must hold before the sequence can be locked, read off the screen's state. */
export function sequenceCriteria(state: VisualSequenceState): SequenceCriterion[] {
  const { units_total: total, units_sequenced: sequenced, missing, extra } = state.coverage;
  const stale = state.beats.filter((b) => b.stale);
  const failing = state.checks.filter((c) => !c.ok && !c.accepted);
  const motifProblems = state.motifs.motifs.filter((m) => m.problems.length > 0).length;
  return [
    {
      label: "Kế hoạch hình khớp phiên bản chốt mới nhất",
      ok: state.input.outdated.length === 0,
      detail: state.input.outdated.join("; "),
    },
    {
      label: "Motif đã duyệt",
      ok: motifsReady(state),
      detail: motifProblems > 0 ? `${motifProblems} motif còn lỗi` : "",
    },
    {
      label: "Mọi unit của kế hoạch hình có chuỗi",
      ok: total > 0 && missing.length === 0 && extra.length === 0,
      detail: `${sequenced}/${total}`,
    },
    {
      label: "Mọi unit đã duyệt",
      ok: total > 0 && state.items_approved === total,
      detail: `${state.items_approved}/${total}`,
    },
    {
      label: "Không beat nào lỗi thời so với kế hoạch hình",
      ok: stale.length === 0,
      detail: stale.map((b) => b.key).join(", "),
    },
    {
      label: "Đã rà soát ít nhất một lần",
      ok: state.review.reviewed,
      detail: `${state.review.rounds} lượt`,
    },
    {
      label: "Không còn vấn đề mở",
      ok: failing.length === 0 && state.review.open_count === 0,
      detail: `${failing.length} máy kiểm, ${state.review.open_count} của AI`,
    },
  ];
}
