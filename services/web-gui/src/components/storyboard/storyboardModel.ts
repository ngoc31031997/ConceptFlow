import type {
  StoryboardAsset,
  StoryboardShot,
  StoryboardState,
  StoryboardUnit,
} from "../../api/client";

/** How each position of a frame is named on the screen; the keys are the contract's closed list. */
export const POSITION_LABELS: Record<string, string> = {
  upper_left: "Trên trái",
  upper_center: "Trên giữa",
  upper_right: "Trên phải",
  center_left: "Giữa trái",
  center: "Giữa",
  center_right: "Giữa phải",
  lower_left: "Dưới trái",
  lower_center: "Dưới giữa",
  lower_right: "Dưới phải",
  split_left: "Nửa trái (chia đôi)",
  split_right: "Nửa phải (chia đôi)",
  full: "Toàn khung",
};

/** How each camera view is named on the screen. */
export const CAMERA_LABELS: Record<string, string> = {
  wide: "Toàn cảnh",
  medium: "Trung cảnh",
  close: "Cận cảnh",
  comparison: "So sánh",
};

/** The machine checks that are warnings: they can be accepted with a reason; the others must be fixed. */
export const ACCEPTABLE_CHECKS = new Set(["m-asset-unused", "m-unit-cover", "m-asset-count"]);

/** What an asset is, for a card's summary line. */
export const ASSET_KIND_LABELS: Record<string, string> = { figure: "Hình", backdrop: "Phông nền" };

/** The tier of an asset, named. */
export const ASSET_TIER_LABELS: Record<string, string> = {
  hero: "Chính",
  support: "Phụ",
  background: "Phông",
};

/** The library folder id the asset goes to when the Creator picks none: the system folder "Từ dự án". */
export const DEFAULT_LIBRARY_FOLDER = "";

/** The id rendering gives the system folder "Từ dự án"; a folder list that carries it shows as the default entry. */
export const PROJECT_FOLDER_ID = "tu-du-an";

/** Every unit of the sequence, in sequence order. */
export function allUnits(state: StoryboardState): StoryboardUnit[] {
  return state.beats.flatMap((b) => b.units);
}

/** Every shot of the video, in sequence order. */
export function allShots(state: StoryboardState): StoryboardShot[] {
  return allUnits(state).flatMap((u) => u.shots);
}

/** Why a shot cannot be approved yet; empty when it can. The machine's own reasons come first. */
export function approveBlockers(shot: StoryboardShot): string[] {
  const reasons = shot.problems.map((p) => p.detail);
  if (reasons.length > 0) return reasons;
  if (shot.missing_assets.length > 0) {
    return [`Cần tài sản đã duyệt: ${shot.missing_assets.join(", ")}`];
  }
  if (shot.frame.error) return [`Dựng khung lỗi: ${shot.frame.error}`];
  if (!shot.frame.drawn) return ["Chưa có khung, vẽ khung trước"];
  if (shot.frame.stale) return ["Khung đã cũ, vẽ lại khung trước"];
  return [];
}

/** Pending shots that can be approved now. */
export function approvableShots(state: StoryboardState): StoryboardShot[] {
  return allShots(state).filter((s) => s.status === "pending" && approveBlockers(s).length === 0);
}

/** The units the DeepSeek queue writes: every unit not approved yet, in sequence order. */
export function unitsToWrite(state: StoryboardState): StoryboardUnit[] {
  return allUnits(state).filter((u) => !u.approved);
}

/** The assets without their detail yet, in list order: what the second phase of the asset list writes. */
export function assetsWithoutDetail(state: StoryboardState): StoryboardAsset[] {
  return state.assets.items.filter((a) => !a.has_detail);
}

/** Why an asset cannot be drawn or redrawn yet; "" when it can. */
export function drawBlocker(asset: StoryboardAsset, state: StoryboardState): string {
  if (state.kit.missing) return state.kit.message || "Dự án chưa có kit vẽ";
  if (!asset.has_detail) return "Viết chi tiết cho tài sản này trước khi vẽ";
  if (!state.assets.approved) return "Duyệt danh sách tài sản ở khối 1 trước khi vẽ";
  return "";
}

/** Why the asset list cannot be approved yet; "" when it can. */
export function listApproveBlocker(state: StoryboardState): string {
  if (state.assets.items.length === 0) return "Chưa có tài sản nào";
  const missing = assetsWithoutDetail(state);
  if (missing.length > 0) {
    return `Viết chi tiết trước khi duyệt danh sách: ${missing.map((a) => a.asset_id).join(", ")}`;
  }
  return "";
}

/** The assets of the approved list not drawn yet: what "Vẽ tất cả" draws, in list order. */
export function assetsToDraw(state: StoryboardState): StoryboardAsset[] {
  return state.assets.items.filter((a) => a.status === "planned" && drawBlocker(a, state) === "");
}

/** Whether every unit has shots and every shot is approved. */
export function shotsDone(state: StoryboardState): boolean {
  return (
    state.shots_total > 0 &&
    state.empty_units.length === 0 &&
    state.shots_approved === state.shots_total
  );
}

/** The assets that are not approved yet. */
export function unapprovedAssets(state: StoryboardState): StoryboardAsset[] {
  return state.assets.items.filter((a) => a.status !== "approved");
}

/** Whether the shots can be written: there are assets and every one is approved, and the kit is there. */
export function assetsReady(state: StoryboardState): boolean {
  return (
    state.assets.items.length > 0 && unapprovedAssets(state).length === 0 && !state.kit.missing
  );
}

/** Why the AI cannot write shots yet; "" when it can. */
export function writeBlocker(state: StoryboardState): string {
  if (state.kit.missing) return state.kit.message || "Dự án chưa có kit vẽ";
  if (state.assets.items.length === 0) return "Lập danh sách tài sản trước khi viết shot";
  const pending = unapprovedAssets(state);
  if (pending.length > 0) {
    return `Duyệt tài sản trước khi viết shot: ${pending.map((a) => a.asset_id).join(", ")}`;
  }
  return "";
}

/** One line of the lock conditions. */
export interface StoryboardCriterion {
  label: string;
  ok: boolean;
  detail: string;
}

/** What must hold before the storyboard can be locked, read off the screen's state. */
export function storyboardCriteria(state: StoryboardState): StoryboardCriterion[] {
  const shots = allShots(state);
  const framesToRedraw = shots.filter((s) => s.frame.stale || !s.frame.drawn || s.frame.error);
  const failing = state.checks.filter((c) => !c.ok && !c.accepted);
  const notApproved = unapprovedAssets(state);
  return [
    {
      label: `Chuỗi hình bước 10 đã chốt (v${state.input.sequence_version})`,
      ok: state.input.outdated.length === 0,
      detail: state.input.outdated.join("; "),
    },
    {
      label: "Mọi tài sản đã duyệt",
      ok: state.assets.items.length > 0 && notApproved.length === 0,
      detail:
        state.assets.items.length === 0
          ? "chưa có tài sản"
          : `${notApproved.length} tài sản chưa duyệt`,
    },
    {
      label: "Mọi unit có shot",
      ok: state.empty_units.length === 0,
      detail: state.empty_units.join(", "),
    },
    {
      label: "Mọi shot đã duyệt",
      ok: state.shots_total > 0 && state.shots_approved === state.shots_total,
      detail: `${state.shots_approved}/${state.shots_total}`,
    },
    {
      label: "Mọi khung còn mới",
      ok: framesToRedraw.length === 0,
      detail:
        framesToRedraw.length > 0
          ? `${framesToRedraw.length} khung cần vẽ lại: ${framesToRedraw.map((s) => s.shot_id).join(", ")}`
          : "",
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

/** Which screen part a finding or a unit key points to. */
export function keyTarget(key: string): "shot" | "unit" | "asset" | "other" {
  if (key.startsWith("SH-")) return "shot";
  if (key.startsWith("VP-")) return "unit";
  if (key.startsWith("AST-")) return "asset";
  return "other";
}

/** Splits a text area into its non-empty trimmed lines. */
export function textLines(text: string): string[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}

/** How many lock blockers the banner lists before "và k điều kiện khác". */
export const BLOCKER_PREVIEW = 3;

/** Compact reason for the bottom bar: the count only, the full list is in the tooltip. */
export function blockersSummary(blockers: readonly string[]): string {
  return `Chưa chốt được: ${blockers.length} điều kiện chưa đạt`;
}

/** The full blocker list, one per line, for a tooltip. */
export function blockersTooltip(blockers: readonly string[]): string {
  return blockers.join("\n");
}
