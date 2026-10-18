/**
 * conceptflow-mini/skeleton — bones, poses and motion for living things and
 * machines.
 *
 * A drawing declares WHERE its joints are (`<Bone name pivot>` around its
 * skin shapes); the rig decides HOW they turn:
 *
 *   - a named pose from the skeleton's library (`pose`), or a series of
 *     poses at moments of the shot (`actions`), eased with a small wind-up
 *     and overshoot, child bones trailing their parents by a few frames;
 *   - a repeating `cycle` (walk, run, fly, swim, crawl) and idle breathing;
 *   - two-bone IK (`reach`) so a hand arrives at a point, and `lookAt`;
 *   - `Eyes`, `Brows` and `Mouth` that follow `mood` and blink on their own.
 *
 * Angles are degrees, relative to the parent bone, 0 = the rest pose the
 * drawing was made in, positive = clockwise on screen. Sides are the viewer's:
 * `L` is on the left of the screen. A bone that hangs down at rest and
 * swings out to the left of the screen turns by a positive angle.
 *
 * `Mechanism` does the same for machines: named `Part`s that rotate, hinge or
 * slide, shown whole, in cut-away or exploded.
 */
import React from 'react';
import {useCurrentFrame, useVideoConfig} from 'remotion';
import {useKit} from './kits';
import skeletonTable from './skeletons.json';

// --- skeletons ----------------------------------------------------------------

export type SkeletonName = 'humanoid' | 'quadruped' | 'bird' | 'fish' | 'insect' | 'worm';

/** Bone name -> its parent bone (`null` for the root). */
export type BoneTree = Record<string, string | null>;

const SEGMENTS = 6;

/**
 * Bones of each skeleton; every bone listed must be declared by the drawing.
 * The table lives in skeletons.json so the rendering service's drawing checks
 * read the same bones.
 */
export const SKELETONS = skeletonTable as Record<SkeletonName, BoneTree>;

/** Depth of a bone in its tree (root 0); child bones trail their parents by this many lags. */
export function boneDepth(tree: BoneTree, bone: string): number {
  let depth = 0;
  for (let b = tree[bone]; b; b = tree[b]) depth += 1;
  return depth;
}

// --- poses -----------------------------------------------------------------------

/** Angles per bone, plus how far the whole figure drops (`lower`) or lifts (`lift`) as a share of its height. */
export interface PoseDef {
  angles: Record<string, number>;
  lower?: number;
  lift?: number;
}

export const HUMANOID_POSES = {
  stand: {angles: {}},
  sit: {angles: {thighL: -8, thighR: 8, shinL: 8, shinR: -8}, lower: 0.28},
  wave: {angles: {armR: -150, foreArmR: -15, head: 4}},
  point: {angles: {armR: -85, foreArmR: 0, head: -3}},
  think: {angles: {armR: -35, foreArmR: -125, head: 6}},
  cheer: {angles: {armL: 160, armR: -160, foreArmL: -10, foreArmR: 10, head: -4}},
  ouch: {angles: {armL: 40, armR: -40, foreArmL: -105, foreArmR: 105, spine: 8, head: 10}},
  shrug: {angles: {armL: 35, armR: -35, foreArmL: -50, foreArmR: 50, head: 5}},
  hold: {angles: {armL: 25, armR: -25, foreArmL: -70, foreArmR: 70}},
  reach: {angles: {armR: -100, foreArmR: -10, spine: -4}},
  crouch: {angles: {thighL: 60, thighR: -60, shinL: -100, shinR: 100, spine: 6}, lower: 0.2},
  jump: {angles: {armL: 140, armR: -140, thighL: 20, thighR: -20, shinL: -35, shinR: 35}, lift: 0.14},
  lookUp: {angles: {head: -16, spine: -3}},
} satisfies Record<string, PoseDef>;

export type HumanoidPose = keyof typeof HUMANOID_POSES;

/** Pose library of each skeleton; every skeleton has at least `rest`. */
export const POSES: Record<SkeletonName, Record<string, PoseDef>> = {
  humanoid: HUMANOID_POSES,
  quadruped: {
    rest: {angles: {}},
    graze: {angles: {neck: 45, head: 15}},
    alert: {angles: {neck: -20, head: -10}},
    sit: {angles: {hindUpperL: -50, hindUpperR: 50, hindLowerL: 80, hindLowerR: -80}, lower: 0.15},
  },
  bird: {
    rest: {angles: {wingL: 8, wingR: -8}},
    spread: {angles: {wingL: 80, wingR: -80}},
    peck: {angles: {neck: 40, head: 20}},
  },
  fish: {rest: {angles: {}}, turn: {angles: {body: 8, tail: 18, head: -6}}},
  insect: {rest: {angles: {}}, groom: {angles: {legL1: 40, legR1: -40, head: 6}}, spread: {angles: {wingL: 60, wingR: -60}}},
  worm: {rest: {angles: {}}, curl: {angles: {seg2: 25, seg3: 25, seg4: 25, seg5: 25, seg6: 25}}},
};

/** Name of the neutral pose of a skeleton. */
export function restPoseName(skeleton: SkeletonName): string {
  return skeleton === 'humanoid' ? 'stand' : 'rest';
}

// --- cycles -----------------------------------------------------------------------

export type CycleName = 'walk' | 'run' | 'fly' | 'swim' | 'crawl';

export interface CycleSpec {
  name: CycleName;
  /** Seconds for one full cycle; defaults per cycle. */
  period?: number;
  /** Strength multiplier, 1 = default. */
  amount?: number;
}

const CYCLE_PERIOD: Record<CycleName, number> = {walk: 1, run: 0.6, fly: 0.35, swim: 1.2, crawl: 1.6};

const sin = (turns: number) => Math.sin(turns * 2 * Math.PI);

/** Bone angles a cycle adds at `seconds`; empty when the skeleton cannot do that cycle. */
export function cycleAngles(skeleton: SkeletonName, cycle: CycleSpec, seconds: number): Record<string, number> {
  const turns = seconds / (cycle.period ?? CYCLE_PERIOD[cycle.name]);
  const k = cycle.amount ?? 1;
  const out: Record<string, number> = {};
  if (skeleton === 'humanoid' && (cycle.name === 'walk' || cycle.name === 'run')) {
    const swing = (cycle.name === 'run' ? 42 : 24) * k;
    const knee = (cycle.name === 'run' ? 70 : 34) * k;
    const arms = (cycle.name === 'run' ? 38 : 18) * k;
    out.thighL = swing * sin(turns);
    out.thighR = -swing * sin(turns);
    out.shinL = knee * Math.max(0, sin(turns + 0.25));
    out.shinR = knee * Math.max(0, -sin(turns + 0.25));
    out.armL = -arms * sin(turns);
    out.armR = -arms * sin(turns);
    out.foreArmL = cycle.name === 'run' ? -50 * k : 0;
    out.foreArmR = cycle.name === 'run' ? 50 * k : 0;
    out.spine = (cycle.name === 'run' ? 8 : 2) * k * sin(turns * 2);
    return out;
  }
  if (skeleton === 'quadruped' && (cycle.name === 'walk' || cycle.name === 'run')) {
    const swing = (cycle.name === 'run' ? 38 : 20) * k;
    out.foreUpperL = swing * sin(turns);
    out.hindUpperR = swing * sin(turns);
    out.foreUpperR = -swing * sin(turns);
    out.hindUpperL = -swing * sin(turns);
    out.foreLowerL = 30 * k * Math.max(0, sin(turns + 0.25));
    out.foreLowerR = 30 * k * Math.max(0, -sin(turns + 0.25));
    out.hindLowerL = 30 * k * Math.max(0, -sin(turns + 0.25));
    out.hindLowerR = 30 * k * Math.max(0, sin(turns + 0.25));
    out.head = 4 * k * sin(turns * 2);
    return out;
  }
  if (cycle.name === 'fly' && (skeleton === 'bird' || skeleton === 'insect')) {
    const flap = (skeleton === 'bird' ? 55 : 40) * k;
    out.wingL = flap * sin(turns);
    out.wingR = -flap * sin(turns);
    return out;
  }
  if (cycle.name === 'swim' && skeleton === 'fish') {
    out.body = 5 * k * sin(turns);
    out.tail = 24 * k * sin(turns - 0.2);
    out.head = -3 * k * sin(turns);
    out.finL = 14 * k * sin(turns * 2);
    out.finR = -14 * k * sin(turns * 2);
    return out;
  }
  if (cycle.name === 'crawl' && skeleton === 'worm') {
    for (let i = 1; i <= SEGMENTS; i += 1) out[`seg${i}`] = 14 * k * sin(turns - i * 0.12);
    return out;
  }
  if (cycle.name === 'crawl' && skeleton === 'insect') {
    for (const side of ['L', 'R'] as const) {
      const m = side === 'L' ? 1 : -1;
      [1, 2, 3].forEach((n, i) => {
        out[`leg${side}${n}`] = m * 22 * k * sin(turns + (i % 2 === 0 ? 0 : 0.5) + (side === 'R' ? 0.5 : 0));
      });
    }
    return out;
  }
  return out;
}

// --- two-bone IK ------------------------------------------------------------------

export type P2 = [number, number];

const deg = (rad: number) => (rad * 180) / Math.PI;

/** Joints of a limb in the drawing's rest pose, in drawing coordinates. */
export interface LimbJoints {
  root: P2;
  mid: P2;
  tip: P2;
}

/**
 * Rotations (degrees) of the upper bone and of the lower bone, relative to
 * the upper one, that bring the tip of a two-bone limb to `target`. A target
 * beyond the limb's reach is stretched toward as far as it goes; one closer
 * than the folded limb can get is met at the closest reach. `bend` (+1 or -1)
 * picks which way the middle joint points.
 */
export function ikTwoBone(joints: LimbJoints, target: P2, bend: 1 | -1 = 1): {upper: number; lower: number} {
  const u0: P2 = [joints.mid[0] - joints.root[0], joints.mid[1] - joints.root[1]];
  const f0: P2 = [joints.tip[0] - joints.mid[0], joints.tip[1] - joints.mid[1]];
  const l1 = Math.hypot(u0[0], u0[1]);
  const l2 = Math.hypot(f0[0], f0[1]);
  if (l1 < 1e-6 || l2 < 1e-6) throw new Error('Khớp trùng nhau: xương có độ dài 0');
  const t: P2 = [target[0] - joints.root[0], target[1] - joints.root[1]];
  const reach = Math.hypot(t[0], t[1]);
  const d = Math.min(l1 + l2 - 1e-6, Math.max(Math.abs(l1 - l2) + 1e-6, reach));
  const toTarget = Math.atan2(t[1], t[0]);
  const cosShoulder = (l1 * l1 + d * d - l2 * l2) / (2 * l1 * d);
  const shoulder = toTarget + bend * Math.acos(Math.max(-1, Math.min(1, cosShoulder)));
  const elbow: P2 = [l1 * Math.cos(shoulder), l1 * Math.sin(shoulder)];
  const aim: P2 = [d * Math.cos(toTarget), d * Math.sin(toTarget)];
  const foreWorld = Math.atan2(aim[1] - elbow[1], aim[0] - elbow[0]);
  const upper = deg(shoulder - Math.atan2(u0[1], u0[0]));
  const lower = deg(foreWorld - Math.atan2(f0[1], f0[0])) - upper;
  return {upper: wrap180(upper), lower: wrap180(lower)};
}

/** `point` turned by `degrees` around `pivot`, clockwise on screen. */
export function rotateAbout(point: P2, pivot: P2, degrees: number): P2 {
  const a = (degrees * Math.PI) / 180;
  const dx = point[0] - pivot[0];
  const dy = point[1] - pivot[1];
  return [pivot[0] + dx * Math.cos(a) - dy * Math.sin(a), pivot[1] + dx * Math.sin(a) + dy * Math.cos(a)];
}

/**
 * Position of the middle joint of a two-bone limb of lengths `l1` and `l2`
 * whose root is at `root` and whose tip must reach `target`; the tip is
 * stretched toward the target when it is out of reach. `bend` picks the side.
 */
export function elbowPosition(root: P2, target: P2, l1: number, l2: number, bend: 1 | -1 = 1): P2 {
  const t: P2 = [target[0] - root[0], target[1] - root[1]];
  const reach = Math.hypot(t[0], t[1]);
  const d = Math.min(l1 + l2 - 1e-6, Math.max(Math.abs(l1 - l2) + 1e-6, reach));
  const cosShoulder = (l1 * l1 + d * d - l2 * l2) / (2 * l1 * d);
  const a = Math.atan2(t[1], t[0]) + bend * Math.acos(Math.max(-1, Math.min(1, cosShoulder)));
  return [root[0] + l1 * Math.cos(a), root[1] + l1 * Math.sin(a)];
}

/** `angle` folded into (-180, 180]. */
export function wrap180(angle: number): number {
  const a = ((((angle + 180) % 360) + 360) % 360) - 180;
  return a === -180 ? 180 : a;
}

// --- motion over time ----------------------------------------------------------------

/** A pose reached at a moment of the shot. */
export interface RigAction {
  /** Moment as a fraction of the shot, 0..1. */
  at: number;
  pose: string;
  /** Fraction of the shot the pose is held after it is reached. */
  hold?: number;
}

const WIND_UP = 0.15;
const WIND_UP_DEPTH = 0.08;
const OVERSHOOT = 1.15;
/** Frames by which each level of the bone tree trails its parent. */
const DRAG_FRAMES = 2;

/** Ease with a small step back before moving off and a little overshoot on arrival. */
export function easeAction(p: number): number {
  if (p <= 0) return 0;
  if (p >= 1) return 1;
  if (p < WIND_UP) return -WIND_UP_DEPTH * Math.sin((Math.PI * p) / WIND_UP);
  const q = (p - WIND_UP) / (1 - WIND_UP);
  const c1 = OVERSHOOT;
  const c3 = c1 + 1;
  return 1 + c3 * Math.pow(q - 1, 3) + c1 * Math.pow(q - 1, 2);
}

function poseOf(skeleton: SkeletonName, name: string): PoseDef {
  const pose = POSES[skeleton][name];
  if (!pose) throw new Error(`Dáng "${name}" không có ở bộ xương ${skeleton} (có: ${Object.keys(POSES[skeleton]).join(', ')})`);
  return pose;
}

/** One pose's angles for every bone of the skeleton, 0 where the pose does not say. */
function fullAngles(skeleton: SkeletonName, pose: PoseDef): Record<string, number> {
  return Object.fromEntries(Object.keys(SKELETONS[skeleton]).map((b) => [b, pose.angles[b] ?? 0]));
}

function blend(a: PoseDef, b: PoseDef, e: number, skeleton: SkeletonName): {angles: Record<string, number>; lower: number; lift: number} {
  const fa = fullAngles(skeleton, a);
  const fb = fullAngles(skeleton, b);
  const angles = Object.fromEntries(Object.keys(fa).map((k) => [k, fa[k] + (fb[k] - fa[k]) * e]));
  return {
    angles,
    lower: (a.lower ?? 0) + ((b.lower ?? 0) - (a.lower ?? 0)) * e,
    lift: (a.lift ?? 0) + ((b.lift ?? 0) - (a.lift ?? 0)) * e,
  };
}

/** Pose at fraction `t` of the shot: `base` until the first action, then each action in turn. */
export function poseAt(
  skeleton: SkeletonName,
  base: string,
  actions: RigAction[],
  t: number,
): {angles: Record<string, number>; lower: number; lift: number} {
  const sorted = [...actions].sort((x, y) => x.at - y.at);
  let from = poseOf(skeleton, base);
  let fromEnd = 0;
  for (const action of sorted) {
    const to = poseOf(skeleton, action.pose);
    const reachedAt = action.at;
    if (t < reachedAt) {
      const span = Math.max(1e-6, reachedAt - fromEnd);
      return blend(from, to, easeAction((t - fromEnd) / span), skeleton);
    }
    from = to;
    fromEnd = reachedAt + (action.hold ?? 0);
    if (t < fromEnd) return blend(from, from, 0, skeleton);
  }
  return blend(from, from, 0, skeleton);
}

/** Specification of a rig's motion. */
export interface RigMotion {
  skeleton: SkeletonName;
  pose?: string;
  actions?: RigAction[];
  cycle?: CycleSpec;
  /** Shot length in frames; required when `actions` is given. */
  duration?: number;
  /** Moves from `pose` toward another pose: `t` 0 = `pose`, 1 = `pose.to`. */
  blendTo?: {pose: string; t: number};
  /** Idle breathing and head drift; off for a held frame. */
  still?: boolean;
}

/** Everything the rig puts on the drawing at one frame. */
export interface RigFrame {
  angles: Record<string, number>;
  lower: number;
  lift: number;
  /** Vertical squash of the whole figure just after a landing, 1 = none. */
  squash: number;
  breath: number;
}

const LANDING_SQUASH = 0.07;
const LANDING_FRAMES = 6;

/** The angles of every bone at `frame`, with trailing child bones, cycles and breathing. */
export function solveRig(motion: RigMotion, frame: number, fps: number): RigFrame {
  const {skeleton, actions = [], cycle, still = false} = motion;
  const base = motion.pose ?? restPoseName(skeleton);
  if (actions.length > 0 && !motion.duration) throw new Error('Rig có actions thì phải có duration (số khung của shot)');
  const tree = SKELETONS[skeleton];
  const duration = motion.duration ?? 1;
  const blendTo = motion.blendTo;
  const at = (f: number) =>
    blendTo
      ? blend(poseOf(skeleton, base), poseOf(skeleton, blendTo.pose), Math.max(0, Math.min(1, blendTo.t)), skeleton)
      : poseAt(skeleton, base, actions, Math.max(0, Math.min(1, f / duration)));

  const angles: Record<string, number> = {};
  for (const bone of Object.keys(tree)) {
    const lagged = at(frame - DRAG_FRAMES * boneDepth(tree, bone));
    angles[bone] = lagged.angles[bone];
  }
  const now = at(frame);
  if (cycle) {
    const add = cycleAngles(skeleton, cycle, frame / fps);
    for (const [bone, value] of Object.entries(add)) if (bone in angles) angles[bone] += value;
  }
  const breath = still ? 0 : sin(frame / fps / 3.2);
  if (!still) {
    const chest = skeleton === 'humanoid' ? 'spine' : Object.keys(tree)[0];
    angles[chest] += breath * 0.8;
  }
  let squash = 1;
  for (let k = 0; k < LANDING_FRAMES; k += 1) {
    const before = at(frame - k - 1).lift;
    const after = at(frame - k).lift;
    if (before - after > 0.004 && after < 0.01) {
      squash = 1 - LANDING_SQUASH * (1 - k / LANDING_FRAMES);
      break;
    }
  }
  return {angles, lower: now.lower, lift: now.lift, squash, breath};
}

// --- Rig and Bone components -------------------------------------------------------------

interface RigContextValue {
  skeleton: SkeletonName;
  frame: RigFrame;
  parent: string | null;
  declared: Set<string>;
  mood: Mood;
  blinkPhase: number;
  still: boolean;
}

const RigContext = React.createContext<RigContextValue | null>(null);

export type Mood = 'neutral' | 'happy' | 'sad' | 'worried' | 'surprised' | 'pain' | 'angry';

export interface RigProps extends Omit<RigMotion, 'duration'> {
  /** Shot length in frames, for `actions`. */
  duration?: number;
  /** Two-bone IK: bring the tip of one arm to a point of the drawing. */
  reach?: ReachSpec;
  /** Turn the head toward a point of the drawing. */
  lookAt?: {to: P2; head: P2};
  mood?: Mood;
  /** Height of the drawing in its own units; `lower` and `lift` are shares of it. */
  height: number;
  children: React.ReactNode;
}

/** Reaching with a limb. */
export interface ReachSpec {
  /** The pair of bones: `armR` + `foreArmR`, `armL` + `foreArmL`. */
  limb: 'R' | 'L';
  to: P2;
  /** Rest-pose joints of that arm in the drawing. */
  joints: LimbJoints;
  /** 0 = pose as is, 1 = fully reaching. */
  amount?: number;
  bend?: 1 | -1;
}

const HEAD_LOOK_LIMIT = 30;
const HEAD_LOOK_SOFTNESS = 100;

/** Head angle that turns toward `to` from `head`, capped. */
export function lookAngle(head: P2, to: P2): number {
  const dx = to[0] - head[0];
  const dy = to[1] - head[1];
  const a = deg(Math.atan2(dx, Math.abs(dy) + HEAD_LOOK_SOFTNESS));
  return Math.max(-HEAD_LOOK_LIMIT, Math.min(HEAD_LOOK_LIMIT, a));
}

/**
 * A skeleton around a drawing made of `<Bone>`s. All bones of the skeleton
 * must be declared; a missing one is an error, so a drawing cannot silently
 * lose a limb.
 */
export function Rig(props: RigProps) {
  const {skeleton, reach, lookAt, mood = 'neutral', height, children, still = false} = props;
  const frameNo = useCurrentFrame();
  const {fps} = useVideoConfig();
  const solved = solveRig(props, frameNo, fps);
  const angles = {...solved.angles};
  if (reach) {
    const amount = reach.amount ?? 1;
    const ik = ikTwoBone(reach.joints, reach.to, reach.bend ?? 1);
    const upper = `arm${reach.limb}`;
    const lower = `foreArm${reach.limb}`;
    if (!(upper in angles)) throw new Error(`reach cần xương ${upper}, bộ xương ${skeleton} không có`);
    angles[upper] += (ik.upper - angles[upper]) * amount;
    angles[lower] += (ik.lower - angles[lower]) * amount;
  }
  if (lookAt && 'head' in angles) angles.head = lookAngle(lookAt.head, lookAt.to);

  const declared = React.useMemo(() => new Set<string>(), []);
  declared.clear();
  const blinkPhase = React.useMemo(() => Math.abs(Math.round(height * 7)) % 97, [height]);
  const required = Object.keys(SKELETONS[skeleton]);
  React.useLayoutEffect(() => {
    const missing = required.filter((bone) => !declared.has(bone));
    if (missing.length > 0) throw new Error(`Rig ${skeleton} thiếu xương: ${missing.join(', ')}`);
  });

  const dy = (solved.lower - solved.lift) * height;
  return (
    <RigContext.Provider
      value={{skeleton, frame: {...solved, angles}, parent: null, declared, mood, blinkPhase, still}}
    >
      <g transform={`translate(0 ${dy}) translate(0 ${height}) scale(1 ${solved.squash}) translate(0 ${-height})`}>{children}</g>
    </RigContext.Provider>
  );
}

/**
 * A bone: everything inside turns around `pivot` (in the drawing's units) by
 * the angle the rig computed for `name`. Bones nest like the skeleton tree:
 * `foreArmR` goes inside `armR`.
 */
export function Bone({name, pivot, children}: {name: string; pivot: P2; children?: React.ReactNode}) {
  const rig = React.useContext(RigContext);
  if (!rig) throw new Error(`<Bone name="${name}"> phải nằm trong <Rig>`);
  const tree = SKELETONS[rig.skeleton];
  if (!(name in tree)) throw new Error(`Xương "${name}" không có ở bộ xương ${rig.skeleton} (có: ${Object.keys(tree).join(', ')})`);
  if (tree[name] !== rig.parent) {
    throw new Error(`Xương "${name}" phải nằm trong xương "${tree[name] ?? 'Rig (gốc)'}", không phải "${rig.parent ?? 'Rig (gốc)'}"`);
  }
  rig.declared.add(name);
  const angle = rig.frame.angles[name] ?? 0;
  return (
    <RigContext.Provider value={{...rig, parent: name}}>
      <g data-cf-bone={name} transform={`rotate(${angle} ${pivot[0]} ${pivot[1]})`}>
        {children}
      </g>
    </RigContext.Provider>
  );
}

// --- expression ----------------------------------------------------------------------------

/** Mood and blink state of a face: the rig's when inside a `Rig`, else the `mood` prop. */
function useFace(mood: Mood | undefined, phase: number | undefined, still: boolean | undefined): {mood: Mood; blinkPhase: number; still: boolean} {
  const rig = React.useContext(RigContext);
  return {mood: mood ?? rig?.mood ?? 'neutral', blinkPhase: phase ?? rig?.blinkPhase ?? 0, still: still ?? rig?.still ?? false};
}

/** Overrides for a face drawn outside a `Rig`: the mood, and the blink phase and stillness of the figure. */
export interface FaceProps {
  mood?: Mood;
  phase?: number;
  still?: boolean;
}

const BLINK_PERIOD = 96;
const BLINK_FRAMES = 5;

/** Whether the eyes are closed at this frame of a blink cycle that is offset per figure. */
export function isBlinking(frame: number, phase: number, still: boolean): boolean {
  return !still && (frame + phase) % BLINK_PERIOD < BLINK_FRAMES;
}

/** A pair of eyes centred on (`cx`, `cy`), `gap` apart; closed while blinking, arcs when happy. */
export function Eyes({cx, cy, gap, size = 5, mood, phase, still}: FaceProps & {cx: number; cy: number; gap: number; size?: number}) {
  const rig = useFace(mood, phase, still);
  const frame = useCurrentFrame();
  const ink = useKit().ink;
  const closed = isBlinking(frame, rig.blinkPhase, rig.still);
  const happy = rig.mood === 'happy';
  const surprised = rig.mood === 'surprised';
  return (
    <g>
      {[-1, 1].map((side) => {
        const x = cx + (side * gap) / 2;
        if (closed || happy) {
          const up = happy && !closed ? -1 : 1;
          return (
            <path key={side} d={`M${x - size} ${cy} Q${x} ${cy + size * up} ${x + size} ${cy}`} stroke={ink} strokeWidth={size * 0.7} strokeLinecap="round" fill="none" />
          );
        }
        return <circle key={side} cx={x} cy={cy} r={surprised ? size * 1.15 : size * 0.8} fill={ink} />;
      })}
    </g>
  );
}

/** Eyebrows over eyes at (`cx`, `cy`), `gap` apart, tilted by mood. */
export function Brows({cx, cy, gap, size = 7, mood}: FaceProps & {cx: number; cy: number; gap: number; size?: number}) {
  const rig = useFace(mood, undefined, undefined);
  const ink = useKit().ink;
  // Positive tilt lifts the inner ends (sad, worried), negative lowers them (angry); `lift` raises both brows.
  const tilt: Record<Mood, number> = {neutral: 0, happy: 2, sad: 5, worried: 6, surprised: 2, pain: 5, angry: -6};
  const lift: Partial<Record<Mood, number>> = {surprised: -7, happy: -3};
  const t = tilt[rig.mood];
  cy += lift[rig.mood] ?? 0;
  return (
    <g stroke={ink} strokeWidth={size * 0.4} strokeLinecap="round" fill="none">
      <path d={`M${cx - gap / 2 - size} ${cy + t} L${cx - gap / 2 + size} ${cy - t}`} />
      <path d={`M${cx + gap / 2 - size} ${cy - t} L${cx + gap / 2 + size} ${cy + t}`} />
    </g>
  );
}

/** A mouth at (`cx`, `cy`) shaped by mood; opens and closes while `talking`. */
export function Mouth({cx, cy, width = 18, talking = false, mood}: FaceProps & {cx: number; cy: number; width?: number; talking?: boolean}) {
  const rig = useFace(mood, undefined, undefined);
  const frame = useCurrentFrame();
  const ink = useKit().ink;
  const open = talking ? Math.abs(Math.sin(frame / 3)) : 0;
  const curve: Record<Mood, number> = {neutral: 0, happy: 0.5, sad: -0.4, worried: -0.2, surprised: 0, pain: -0.5, angry: -0.3};
  if (talking || rig.mood === 'surprised') {
    const h = (rig.mood === 'surprised' ? 0.7 : 0.2 + open * 0.6) * width * 0.5;
    return <ellipse cx={cx} cy={cy} rx={width * 0.32} ry={h} fill={ink} />;
  }
  const c = curve[rig.mood] * width * 0.5;
  return <path d={`M${cx - width / 2} ${cy} Q${cx} ${cy + c * 2} ${cx + width / 2} ${cy}`} stroke={ink} strokeWidth={width * 0.14} strokeLinecap="round" fill="none" />;
}

// --- machines ----------------------------------------------------------------------------------

export type PartMotion = 'rotate' | 'hinge' | 'slide' | 'none';
export type MechanismView = 'outside' | 'cutaway' | 'exploded';

interface MechanismContextValue {
  view: MechanismView;
  seconds: number;
}

const MechanismContext = React.createContext<MechanismContextValue | null>(null);

/** Distance an exploded part moves from its place, as a multiple of `explode` units. */
const EXPLODE_STEP = 1;

/** Position and angle of a part at `seconds`, for the motion it declares. */
export function partTransform(
  motion: PartMotion,
  pivot: P2,
  seconds: number,
  opts: {speed?: number; angle?: number; offset?: P2},
): string {
  if (motion === 'rotate') return `rotate(${(opts.speed ?? 360) * seconds} ${pivot[0]} ${pivot[1]})`;
  if (motion === 'hinge') {
    const range = opts.angle ?? 30;
    return `rotate(${range * (0.5 + 0.5 * sin(seconds / 2))} ${pivot[0]} ${pivot[1]})`;
  }
  if (motion === 'slide') {
    const [ox, oy] = opts.offset ?? [0, 0];
    const k = 0.5 + 0.5 * sin(seconds / 2);
    return `translate(${ox * k} ${oy * k})`;
  }
  return '';
}

/**
 * A machine made of `<Part>`s. `view` shows it whole (`outside`), with the
 * parts marked `inside` revealed and the casing parts hidden (`cutaway`), or
 * with every part moved apart along its `explode` direction (`exploded`).
 */
export function Mechanism({view = 'outside', children}: {view?: MechanismView; children: React.ReactNode}) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  return <MechanismContext.Provider value={{view, seconds: frame / fps}}>{children}</MechanismContext.Provider>;
}

/**
 * One part of a machine. `motion` says how it moves (rotate: `speed` deg/s
 * around `pivot`; hinge: swings up to `angle` deg; slide: travels `offset`).
 * `inside` marks parts that show only in the cut-away; `casing` marks parts
 * that the cut-away removes. `explode` is the direction the part moves in the
 * exploded view, in drawing units.
 */
export function Part({
  name,
  pivot = [0, 0],
  motion = 'none',
  speed,
  angle,
  offset,
  inside = false,
  casing = false,
  explode = [0, 0],
  children,
}: {
  name: string;
  pivot?: P2;
  motion?: PartMotion;
  speed?: number;
  angle?: number;
  offset?: P2;
  inside?: boolean;
  casing?: boolean;
  explode?: P2;
  children?: React.ReactNode;
}) {
  const machine = React.useContext(MechanismContext);
  if (!machine) throw new Error(`<Part name="${name}"> phải nằm trong <Mechanism>`);
  if (inside && casing) throw new Error(`Bộ phận "${name}" không thể vừa nằm trong vừa là vỏ`);
  const {view, seconds} = machine;
  if (view === 'outside' && inside) return null;
  if (view === 'cutaway' && casing) return null;
  const move = partTransform(motion, pivot, seconds, {speed, angle, offset});
  const apart = view === 'exploded' ? `translate(${explode[0] * EXPLODE_STEP} ${explode[1] * EXPLODE_STEP})` : '';
  return (
    <g data-cf-part={name} transform={`${apart} ${move}`.trim()}>
      {children}
    </g>
  );
}
