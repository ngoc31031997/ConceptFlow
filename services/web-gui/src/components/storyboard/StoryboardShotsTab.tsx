import { useEffect, useState } from "react";
import {
  describeAiRequest,
  generateStoryboardUnit,
  storyboardFrameUrl,
  type StoryboardBeat,
  type StoryboardShot,
  type StoryboardState,
  type StoryboardUnit,
} from "../../api/client";
import type { StoryboardControls } from "../../hooks/useStoryboard";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { AiRunDialog, AiRunPill } from "../airun/AiRunDialog";
import { StoryboardRunPreview } from "./StoryboardRunPreview";
import { FormDialog } from "../visualplan/PlanDialogs";
import { Foldable } from "./Foldable";
import { ShotDialog } from "./ShotDialog";
import {
  CAMERA_LABELS,
  POSITION_LABELS,
  allShots,
  approvableShots,
  approveBlockers,
  writeBlocker,
} from "./storyboardModel";
import styles from "./storyboard.module.css";

/** Where an asset id or free name sits in the frame, as one readable phrase. */
function placement(name: string, position: string): string {
  return `${name} @ ${POSITION_LABELS[position] ?? position}`;
}

interface FrameImageProps {
  projectId: string;
  shot: StoryboardShot;
}

/**
 * The frame last drawn for a shot. A frame that is missing, failed to draw or
 * cannot be loaded says so in the box; nothing stands in for a picture.
 */
function FrameImage({ projectId, shot }: FrameImageProps) {
  const [broken, setBroken] = useState(false);
  useEffect(() => setBroken(false), [shot.frame.path, shot.frame.stale]);
  const testId = `sb-shot-${shot.shot_id}-frame`;
  const stale = shot.frame.stale ? styles.thumbStale : "";
  if (shot.frame.path === "" || broken) {
    const failed = shot.frame.error !== "" || broken;
    return (
      <div
        className={`${styles.thumb} ${styles.placeholder} ${failed ? styles.placeholderError : ""}`}
        data-testid={`${testId}-empty`}
      >
        {broken
          ? "Không tải được ảnh khung"
          : shot.frame.error !== ""
            ? "Dựng khung lỗi"
            : "Chưa có khung"}
      </div>
    );
  }
  return (
    <img
      className={`${styles.thumb} ${stale}`}
      src={storyboardFrameUrl(projectId, shot.shot_id)}
      alt={`Khung ${shot.shot_id}`}
      onError={() => setBroken(true)}
      data-testid={testId}
    />
  );
}

interface ShotCardProps {
  projectId: string;
  state: StoryboardState;
  unit: StoryboardUnit;
  shot: StoryboardShot;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: boolean;
  onToggle: (open: boolean) => void;
}

/**
 * One shot: its frame, what it shows, its state (approved, pending, stale frame,
 * missing assets, frame error) and the actions on it. An approved shot folds to
 * one line.
 */
function ShotCard({ projectId, state, unit, shot, storyboard, readOnly, expanded, onToggle }: ShotCardProps) {
  const [editing, setEditing] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const id = shot.shot_id;
  const testId = `sb-shot-${id}`;
  const approved = shot.status === "approved";
  const running =
    typeof storyboard.running === "object" &&
    storyboard.running !== null &&
    "shot" in storyboard.running &&
    storyboard.running.shot === id;
  const disabled = readOnly || storyboard.busy || storyboard.running !== null;
  const editDisabled = readOnly || storyboard.busy || running;
  const blockers = approveBlockers(shot);
  const missing = shot.missing_assets.length > 0;
  const last = unit.shots.length <= 1;
  const comp = shot.composition;

  const summary = (
    <div className={styles.folded} data-testid={testId}>
      <span className={styles.mono}>{id}</span>
      <span className={styles.summary}>
        {CAMERA_LABELS[comp.camera_view] ?? comp.camera_view} · {shot.shot_purpose}
      </span>
      <UnitStatusChip state="approved" testId={`${testId}-status`} />
      <Button variant="view" onClick={() => onToggle(true)} data-testid={`${testId}-open`}>
        Mở
      </Button>
    </div>
  );

  return (
    <Foldable folded={approved && !expanded} summary={summary}>
      <article
        className={`${styles.card} ${shot.frame.stale ? styles.cardStale : ""}`}
        id={testId}
        data-testid={testId}
      >
        <FrameImage projectId={projectId} shot={shot} />
        <div className={styles.cardHead}>
          <h4>{id}</h4>
          <div className={styles.tags}>
            {shot.frame.stale && (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipWait}`}
                data-testid={`${testId}-stale`}
              >
                Khung cũ
              </span>
            )}
            {missing && (
              <span className={`${researchStyles.chip}`} data-testid={`${testId}-missing-chip`}>
                Thiếu tài sản
              </span>
            )}
            {running ? (
              <UnitStatusChip state="running" testId={`${testId}-status`} />
            ) : (
              <UnitStatusChip state={approved ? "approved" : "pending"} testId={`${testId}-status`} />
            )}
          </div>
        </div>
        <dl className={styles.kv}>
          <dt>Mục đích</dt>
          <dd>{shot.shot_purpose}</dd>
          <dt>Bố cục</dt>
          <dd>
            <span className={styles.mono}>
              {placement(comp.subject, comp.subject_position)}
              {comp.main_object ? ` · ${placement(comp.main_object, comp.main_object_position)}` : ""}
            </span>{" "}
            · góc <span className={styles.mono}>{CAMERA_LABELS[comp.camera_view] ?? comp.camera_view}</span>
          </dd>
          <dt>Hành động</dt>
          <dd>{shot.action_summary}</dd>
          <dt>Chữ</dt>
          <dd data-testid={`${testId}-text`}>
            {shot.on_screen_text.length > 0
              ? shot.on_screen_text.map((t) => `«${t}»`).join(" ")
              : "—"}
          </dd>
          <dt>Tài sản</dt>
          <dd className={styles.tags}>
            {shot.asset_refs.length === 0 && "—"}
            {shot.asset_refs.map((r) => (
              <span
                key={r}
                className={`${styles.tag} ${shot.missing_assets.includes(r) ? styles.tagBad : ""}`}
              >
                {r}
              </span>
            ))}
          </dd>
          {shot.notes.length > 0 && (
            <>
              <dt>Ghi chú</dt>
              <dd>{shot.notes.join("; ")}</dd>
            </>
          )}
        </dl>
        {missing && (
          <div
            className={`${styles.notice} ${styles.noticeWarn}`}
            role="status"
            data-testid={`${testId}-missing`}
          >
            Cần tài sản đã duyệt trước khi dựng khung: {shot.missing_assets.join(", ")}.
          </div>
        )}
        {shot.frame.error && (
          <div
            className={`${styles.notice} ${styles.noticeBad}`}
            role="alert"
            data-testid={`${testId}-frame-error`}
          >
            Dựng khung lỗi: {shot.frame.error}
          </div>
        )}
        {shot.problems.length > 0 && (
          <ul
            className={`${styles.notice} ${styles.noticeWarn} ${styles.noticeList}`}
            role="status"
            data-testid={`${testId}-problems`}
          >
            {shot.problems.map((p) => (
              <li key={`${p.check}-${p.detail}`}>{p.detail}</li>
            ))}
          </ul>
        )}
        <div className={styles.foot}>
          <Button
            variant="ghost"
            onClick={() => setEditing(true)}
            disabled={editDisabled}
            data-testid={`${testId}-edit`}
          >
            Sửa
          </Button>
          <Button
            variant="run"
            onClick={() => void storyboard.redrawFrame(unit.unit_id, id)}
            disabled={disabled || missing || state.kit.missing}
            title={
              missing
                ? "Cần tài sản đã duyệt trước khi dựng khung"
                : state.kit.missing
                  ? state.kit.message
                  : undefined
            }
            data-testid={`${testId}-redraw`}
          >
            Vẽ lại khung
          </Button>
          {approved ? (
            <>
              <Button variant="ghost" onClick={() => onToggle(false)} data-testid={`${testId}-fold`}>
                Thu gọn
              </Button>
              <Button
                variant="ghost"
                onClick={() => {
                  onToggle(true);
                  void storyboard.unapproveShot(unit.unit_id, id);
                }}
                disabled={disabled}
                data-testid={`${testId}-unapprove`}
              >
                Bỏ duyệt
              </Button>
            </>
          ) : (
            <Button
              variant="primary"
              onClick={() => {
                onToggle(false);
                void storyboard.approveShot(unit.unit_id, id);
              }}
              disabled={disabled || blockers.length > 0}
              title={blockers.length > 0 ? blockers.join("; ") : undefined}
              data-testid={`${testId}-approve`}
            >
              Duyệt
            </Button>
          )}
          <Button
            variant="dangerGhost"
            onClick={() => setDeleting(true)}
            disabled={editDisabled || last}
            title={last ? "Không xoá được shot cuối của unit" : undefined}
            data-testid={`${testId}-delete`}
          >
            Xoá
          </Button>
        </div>
        <ShotDialog
          open={editing}
          onClose={() => setEditing(false)}
          unit={unit}
          shot={shot}
          assets={state.assets.items}
          onSave={(fields) => storyboard.editShot(unit.unit_id, id, fields)}
        />
        <FormDialog
          open={deleting}
          onClose={() => setDeleting(false)}
          title={`Xoá shot ${id}?`}
          confirmLabel="Xoá shot"
          confirmVariant="danger"
          hasInput={false}
          onConfirm={() => storyboard.deleteShot(unit.unit_id, id)}
          size="sm"
          testId={`${testId}-delete-dialog`}
        >
          <p data-testid={`${testId}-delete-impact`}>
            Xoá shot {id} “{shot.shot_purpose}” cùng khung đã vẽ.
            {approved ? ` Unit ${unit.unit_id} mất trạng thái đã duyệt nếu còn shot chưa duyệt.` : ""}{" "}
            Các shot khác giữ nguyên.
          </p>
          <p className={researchStyles.hint}>Không khôi phục được. Mã {id} không được dùng lại.</p>
        </FormDialog>
      </article>
    </Foldable>
  );
}

interface UnitBlockProps {
  projectId: string;
  state: StoryboardState;
  unit: StoryboardUnit;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: ReadonlySet<string>;
  onToggle: (id: string, open: boolean) => void;
}

/** Whether the system's AI is writing this unit now, alone or among the empty units. */
function unitRunning(storyboard: StoryboardControls, unit: StoryboardUnit): boolean {
  const r = storyboard.running;
  if (r === "all") return !unit.written;
  return typeof r === "object" && r !== null && "unit" in r && r.unit === unit.unit_id;
}

/**
 * One unit with its shots in a grid. A unit whose every shot is approved folds to
 * one line.
 */
function UnitBlock({ projectId, state, unit, storyboard, readOnly, expanded, onToggle }: UnitBlockProps) {
  const [aiOpen, setAiOpen] = useState(false);
  const [adding, setAdding] = useState(false);
  const testId = `sb-unit-${unit.unit_id}`;
  const running = unitRunning(storyboard, unit);
  const aiDisabled = readOnly || storyboard.busy || storyboard.running !== null;
  const blocker = writeBlocker(state);
  const approvedShots = unit.shots.filter((s) => s.status === "approved").length;
  const hasApproved = approvedShots > 0;

  const summary = (
    <div className={styles.unit}>
      <div className={styles.folded} data-testid={testId}>
        <span className={styles.mono}>{unit.unit_id}</span>
        <span className={styles.summary}>
          {unit.communication_goal} · {unit.shots.length} shot
        </span>
        <span className={`${researchStyles.chip} ${researchStyles.chipOk}`}>Unit đã duyệt</span>
        <Button variant="view" onClick={() => onToggle(unit.unit_id, true)} data-testid={`${testId}-open`}>
          Mở
        </Button>
      </div>
    </div>
  );

  return (
    <Foldable folded={unit.approved && !expanded.has(unit.unit_id)} summary={summary}>
      <div className={styles.unit} id={testId} data-testid={testId}>
        <div className={styles.unitHead}>
          <div className={styles.unitTitle}>
            <h3>{unit.unit_id}</h3>
            <span className={researchStyles.hint}>{unit.communication_goal}</span>
            {running ? (
              <UnitStatusChip state="running" testId={`${testId}-status`} />
            ) : unit.approved ? (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipOk}`}
                data-testid={`${testId}-status`}
              >
                Unit đã duyệt
              </span>
            ) : (
              <span
                className={`${researchStyles.chip} ${researchStyles.chipWait}`}
                data-testid={`${testId}-status`}
              >
                {unit.written ? "Chưa duyệt đủ shot" : "Chưa có shot"}
              </span>
            )}
            {unit.stale && (
              <span className={researchStyles.chip} data-testid={`${testId}-stale`}>
                lỗi thời: {unit.stale_sources.join(", ")} đã đổi
              </span>
            )}
          </div>
          <div className={styles.foot}>
            <Button
              variant={hasApproved ? "impact" : "run"}
              onClick={() => setAiOpen(true)}
              disabled={aiDisabled || blocker !== ""}
              title={blocker || undefined}
              data-testid={`${testId}-ai`}
            >
              {unit.written ? "AI viết lại unit" : "Viết shot unit này"}
            </Button>
            <AiRunPill runPath={`/v1/projects/${projectId}/storyboard/units/${unit.unit_id}/generate`} onOpen={() => setAiOpen(true)} testId={`${testId}-ai-run`} />
            <Button
              variant="ghost"
              onClick={() => setAdding(true)}
              disabled={readOnly || storyboard.busy || storyboard.running !== null}
              data-testid={`${testId}-add-shot`}
            >
              + Thêm shot
            </Button>
            {unit.approved && (
              <Button
                variant="ghost"
                onClick={() => onToggle(unit.unit_id, false)}
                data-testid={`${testId}-fold`}
              >
                Thu gọn
              </Button>
            )}
          </div>
        </div>
        {unit.shots.length === 0 && (
          <p className={researchStyles.hint} data-testid={`${testId}-empty`}>
            Chưa có shot. Cho AI viết hoặc thêm tay.
          </p>
        )}
        <div className={styles.grid}>
          {unit.shots.map((s) => (
            <ShotCard
              key={s.shot_id}
              projectId={projectId}
              state={state}
              unit={unit}
              shot={s}
              storyboard={storyboard}
              readOnly={readOnly}
              expanded={expanded.has(s.shot_id)}
              onToggle={(open) => onToggle(s.shot_id, open)}
            />
          ))}
        </div>
        <AiRunDialog
          open={aiOpen}
          onClose={() => setAiOpen(false)}
          title={
            unit.written
              ? `Viết lại shot của ${unit.unit_id} kèm ghi chú`
              : `Viết shot cho ${unit.unit_id}`
          }
          runPath={`/v1/projects/${projectId}/storyboard/units/${unit.unit_id}/generate`}
          describe={(note) => describeAiRequest((external) => generateStoryboardUnit(projectId, unit.unit_id, note, { external }))}
          renderPreview={(p) => <StoryboardRunPreview kind="unit" preview={p} />}
          impact={
            <p className={researchStyles.hint} data-testid={`${testId}-ai-impact`}>
              AI viết riêng {unit.unit_id} trong một lượt, đọc shot của unit liền trước, rồi dựng
              khung từng shot. Trường bạn đã sửa tay được giữ.
              {hasApproved
                ? ` ${approvedShots} shot đã duyệt của unit về “Chưa duyệt” và khung vẽ lại nếu đổi.`
                : ""}{" "}
              Các unit khác giữ nguyên.
            </p>
          }
          onApplied={storyboard.reload}
          testId={`${testId}-ai-dialog`}
        />
        <ShotDialog
          open={adding}
          onClose={() => setAdding(false)}
          unit={unit}
          shot={null}
          assets={state.assets.items}
          onSave={(fields) => storyboard.addShot(unit.unit_id, fields)}
        />
      </div>
    </Foldable>
  );
}

interface BeatBlockProps {
  projectId: string;
  state: StoryboardState;
  beat: StoryboardBeat;
  storyboard: StoryboardControls;
  readOnly: boolean;
  expanded: ReadonlySet<string>;
  onToggle: (id: string, open: boolean) => void;
}

/** One beat with its units. A beat whose every unit is approved folds to one line. */
function BeatBlock({ projectId, state, beat, storyboard, readOnly, expanded, onToggle }: BeatBlockProps) {
  const testId = `sb-beat-${beat.key}`;
  const shotCount = beat.units.reduce((n, u) => n + u.shots.length, 0);
  const summary = (
    <div className={styles.beat}>
      <div className={styles.folded} data-testid={testId}>
        <span className={styles.mono}>{beat.key}</span>
        <span className={styles.summary}>
          {beat.title} · {beat.units.length} unit · {shotCount} shot
        </span>
        <span className={`${researchStyles.chip} ${researchStyles.chipOk}`}>Beat đã duyệt</span>
        <Button variant="view" onClick={() => onToggle(beat.key, true)} data-testid={`${testId}-open`}>
          Mở
        </Button>
      </div>
    </div>
  );
  return (
    <Foldable folded={beat.approved && !expanded.has(beat.key)} summary={summary}>
      <div className={styles.beat} id={testId} data-testid={testId}>
        <div className={styles.beatHead}>
          <span className={styles.beatTitle}>
            {beat.key} · {beat.title}
          </span>
          {beat.approved && (
            <Button
              variant="ghost"
              onClick={() => onToggle(beat.key, false)}
              data-testid={`${testId}-fold`}
            >
              Thu gọn
            </Button>
          )}
        </div>
        {beat.units.map((u) => (
          <UnitBlock
            key={u.unit_id}
            projectId={projectId}
            state={state}
            unit={u}
            storyboard={storyboard}
            readOnly={readOnly}
            expanded={expanded}
            onToggle={onToggle}
          />
        ))}
      </div>
    </Foldable>
  );
}

interface StoryboardShotsTabProps {
  projectId: string;
  state: StoryboardState;
  storyboard: StoryboardControls;
  readOnly: boolean;
  /** Ids the Creator opened (beats, units, shots); an approved item folds unless its id is here. */
  expanded: ReadonlySet<string>;
  onToggle: (id: string, open: boolean) => void;
  /** Set when a finding or the grid asks to open a shot; the tab shows the list and scrolls to it. */
  focusShot: string | null;
  onFocused: () => void;
}

/**
 * Tab "Shot" of step 11: the shots grouped by beat and unit, each with its
 * frame, and a read-only grid of the whole film. Approved shots, units and
 * beats fold to one line.
 */
export function StoryboardShotsTab({
  projectId,
  state,
  storyboard,
  readOnly,
  expanded,
  onToggle,
  focusShot,
  onFocused,
}: StoryboardShotsTabProps) {
  const [grid, setGrid] = useState(false);
  const shots = allShots(state);
  const ready = approvableShots(state);
  const disabled = readOnly || storyboard.busy;

  /** Shows the list, opens the shot's unit and beat, and scrolls to the shot once it has rendered. */
  function openShot(shotId: string) {
    const shot = shots.find((s) => s.shot_id === shotId);
    if (!shot) return;
    setGrid(false);
    onToggle(shot.beat_id, true);
    onToggle(shot.unit_id, true);
    onToggle(shot.shot_id, true);
    window.setTimeout(
      () =>
        document
          .getElementById(`sb-shot-${shotId}`)
          ?.scrollIntoView({ behavior: "smooth", block: "start" }),
      50,
    );
  }

  // A request from another tab (a finding) to open a shot.
  useEffect(() => {
    if (!focusShot) return;
    openShot(focusShot);
    onFocused();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusShot]);

  return (
    <>
      <Card data-testid="sb-shots">
        <div className={researchStyles.body}>
          <div className={researchStyles.head}>
            <h3>Shot theo beat và unit</h3>
            <span className={researchStyles.chip} data-testid="sb-view-chip">
              {grid ? "Xem dạng lưới" : "Xem theo danh sách"}
            </span>
            <span className={researchStyles.grow} />
            <Button
              variant="view"
              onClick={() => setGrid((g) => !g)}
              disabled={shots.length === 0}
              data-testid="sb-view-toggle"
            >
              {grid ? "Xem dạng danh sách" : "Xem dạng lưới"}
            </Button>
            <Button
              variant="primary"
              onClick={() => void storyboard.approveAll()}
              disabled={disabled || ready.length === 0}
              data-testid="sb-approve-all"
            >
              Duyệt tất cả còn lại{ready.length > 0 ? ` · ${ready.length}` : ""}
            </Button>
          </div>
          <p className={researchStyles.hint}>
            Shot, unit và beat đã duyệt gập thành một dòng; “Mở” để xem, “Bỏ duyệt” để sửa. Shot có
            lỗi máy kiểm hoặc khung cũ ở lại khi duyệt tất cả.
          </p>
        </div>
      </Card>
      {grid ? (
        <Card data-testid="sb-sheet">
          <div className={researchStyles.body}>
            <div className={researchStyles.head}>
              <h3>Cả phim một lượt</h3>
              <span className={researchStyles.hint}>Chỉ xem. Bấm một khung để mở shot.</span>
            </div>
            <div className={styles.sheet}>
              {shots.map((s) => (
                <button
                  key={s.shot_id}
                  type="button"
                  className={styles.sheetItem}
                  onClick={() => openShot(s.shot_id)}
                  data-testid={`sb-sheet-${s.shot_id}`}
                >
                  <FrameImage projectId={projectId} shot={s} />
                  <span>
                    <strong>{s.shot_id}</strong> · {CAMERA_LABELS[s.composition.camera_view] ?? s.composition.camera_view}
                  </span>
                </button>
              ))}
            </div>
          </div>
        </Card>
      ) : (
        <div className={styles.beat} data-testid="sb-beats">
          {state.beats.map((b) => (
            <BeatBlock
              key={b.key}
              projectId={projectId}
              state={state}
              beat={b}
              storyboard={storyboard}
              readOnly={readOnly}
              expanded={expanded}
              onToggle={onToggle}
            />
          ))}
        </div>
      )}
    </>
  );
}
