/**
 * conceptflow-mini/illustration — a kit of flat, story-book style drawings:
 * people, body parts, food, household objects, scene backdrops.
 *
 * Why it exists: a generated script that draws every picture from raw SVG
 * primitives shows circles and arrows instead of a tooth in a video about
 * tooth decay. A model that has to invent the path of a molar on the fly
 * produces something that is either abstract or ugly; a model that picks
 * <Tooth decay={0.6} /> gets a drawing that was designed once and reviewed.
 *
 * The style is deliberately one style — flat fills, no outlines, one soft
 * shade per shape, rounded forms, dot eyes — so that everything in a frame
 * looks like it was drawn by the same hand.
 *
 * Conventions shared by every figure (the same as LottieClip and the LAYOUT
 * constants of a generated script):
 *   - `x`, `y` are the CENTRE of the figure's box in the 1920x1080 frame;
 *   - `size` is the length of the box's LONGER edge, in px (the box's aspect
 *     ratio is fixed per figure and listed in the prompt catalog);
 *   - `rotate` (degrees), `flip` (mirror horizontally), `scale` and `opacity`
 *     are plain values the shot animates with interpolate()/spring().
 * Figures that stay still in real life also stay still here; living things
 * (people, germs) breathe, blink or wobble on their own so a held shot is
 * never a photograph. `still` turns that off.
 *
 * Colours: every figure has good defaults. Props that take a colour are for
 * matching the Visual Director's palette and must be given PALETTE.xxx.
 */
import React from 'react';
import {AbsoluteFill, useCurrentFrame, useVideoConfig} from 'remotion';
import {FigureFrameProvider, GroundShadow, Shape} from './material';
import {Bone, Brows, Eyes, Mechanism, Mouth, Part, Rig, isBlinking, rotateAbout, solveRig, type CycleSpec, type MechanismView, type Mood, type P2, type ReachSpec, type RigAction, type RigMotion} from './skeleton';
import {useKit} from './kits';

export {GroundShadow};

// --- shared ------------------------------------------------------------------

// Exported, with the helpers below, for the library's own figures:
// a drawing added through the review screen reuses the same box, face, blink
// and shading, so it is drawn by the same hand as the built-in kit.
export const INK = '#3A1F4B';
export const SHADE = '#000000';
export const BLUSH = '#FF7A9A';
export const WHITE = '#FFFFFF';

export interface FigureProps {
  /** Centre of the figure's box. Defaults to the frame centre. */
  x?: number;
  y?: number;
  /** Longer edge of the figure's box, in px. */
  size?: number;
  rotate?: number;
  flip?: boolean;
  scale?: number;
  opacity?: number;
  /** Turn off the figure's own idle motion (breathing, blinking, wobble). */
  still?: boolean;
}

/**
 * The box a figure is drawn in. `vx`/`vy` move the box's top-left corner
 * inside the drawing (a close-up framing of a larger figure), and `clip`
 * hides whatever falls outside the box instead of letting it spill over.
 */
export function Figure({
  x = 960,
  y = 540,
  size,
  vw,
  vh,
  vx = 0,
  vy = 0,
  clip = false,
  rotate = 0,
  flip = false,
  scale = 1,
  opacity = 1,
  children,
}: FigureProps & {size: number; vw: number; vh: number; vx?: number; vy?: number; clip?: boolean; children: React.ReactNode}) {
  const long = Math.max(vw, vh);
  const w = (size * vw) / long;
  const h = (size * vh) / long;
  return (
    <svg
      width={w}
      height={h}
      viewBox={`${vx} ${vy} ${vw} ${vh}`}
      style={{
        position: 'absolute',
        left: x - w / 2,
        top: y - h / 2,
        overflow: clip ? 'hidden' : 'visible',
        opacity,
        transformOrigin: 'center',
        transform: `rotate(${rotate}deg) scale(${flip ? -scale : scale}, ${scale})`,
      }}
    >
      <FigureFrameProvider value={{x, y, rotate, flip, vw, vh}}>{children}</FigureFrameProvider>
    </svg>
  );
}

/** Deterministic per-instance phase so two figures do not blink in unison. */
export function phaseOf(x: number, y: number): number {
  return Math.abs(Math.round(x * 7 + y * 13)) % 97;
}

export function useBlink(still: boolean, phase: number): boolean {
  const frame = useCurrentFrame();
  if (still) return false;
  return (frame + phase) % 96 < 5;
}

/** An id usable inside url(#...): React's useId contains colons. */
export function useSvgId(prefix: string): string {
  return prefix + React.useId().replace(/[^a-zA-Z0-9_-]/g, '');
}

export function shadeOf(hex: string, amount: number): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex);
  if (!m) return hex;
  const n = parseInt(m[1], 16);
  const ch = [(n >> 16) & 255, (n >> 8) & 255, n & 255].map((c) =>
    Math.max(0, Math.min(255, Math.round(amount < 0 ? c * (1 + amount) : c + (255 - c) * amount))),
  );
  return '#' + ch.map((c) => c.toString(16).padStart(2, '0')).join('');
}

/**
 * Where a point of a figure's drawing lands on the frame, in px, given the
 * box the figure was drawn in and its rotate/flip/scale — so something the
 * figure holds can be placed at that point.
 */
export function figurePoint(
  fig: FigureProps & {size: number; vw: number; vh: number; vx?: number; vy?: number},
  point: Pt,
): Pt {
  const {x = 960, y = 540, size, vw, vh, vx = 0, vy = 0, rotate = 0, flip = false, scale = 1} = fig;
  const k = (size / Math.max(vw, vh)) * scale;
  const dx = (point[0] - vx - vw / 2) * k * (flip ? -1 : 1);
  const dy = (point[1] - vy - vh / 2) * k;
  const a = (rotate * Math.PI) / 180;
  return [x + dx * Math.cos(a) - dy * Math.sin(a), y + dx * Math.sin(a) + dy * Math.cos(a)];
}

/**
 * Places `children` (kit figures, whose x/y are then measured from `at`) at a
 * point of the frame: what a hand holds rides with the hand.
 */
export function HeldAt({at, children}: {at: Pt; children?: React.ReactNode}) {
  if (!children) return null;
  return <div style={{position: 'absolute', left: at[0], top: at[1], width: 0, height: 0}}>{children}</div>;
}


// --- scene ------------------------------------------------------------------

/**
 * `data-cf-layer` value of the frame regions below: the layout check reads
 * what carries it as scenery, which may reach the frame edge.
 */
const REGION_LAYER = 'scene';

/**
 * A full-frame background for an illustrated scene, painted over the Stage's
 * default. `to` turns it into a soft top-to-bottom gradient from `color` to
 * `to` (a sky, a glowing interior). `floor` adds a flat floor band from
 * `floorY` down, the way a story-book room sits its furniture on one line.
 * It is scenery for the layout check, not an object held to the safe area.
 */
export function Backdrop({
  color = '#FFC857',
  to,
  floor,
  floorY = 820,
  opacity = 1,
}: {
  color?: string;
  to?: string;
  floor?: string;
  floorY?: number;
  opacity?: number;
}) {
  const fill = to ? {backgroundImage: `linear-gradient(180deg, ${color} 0%, ${to} 100%)`} : {backgroundColor: color};
  return (
    <AbsoluteFill data-cf-layer={REGION_LAYER} style={{opacity}}>
      <div style={{position: 'absolute', inset: 0, ...fill}} />
      {floor ? (
        <div style={{position: 'absolute', left: 0, right: 0, top: floorY, bottom: 0, backgroundColor: floor}} />
      ) : null}
    </AbsoluteFill>
  );
}

/**
 * A flat coloured rectangle: one side of a split screen, a window onto
 * another place, a stripe between two panels. `x`, `y` are its top-left
 * corner (it is a region of the frame, not a figure). Its colour is scenery
 * for the layout check; what is drawn inside it is measured like any object.
 */
export function Panel({
  x,
  y,
  w,
  h,
  color,
  radius = 0,
  opacity = 1,
  children,
}: {
  x: number;
  y: number;
  w: number;
  h: number;
  color: string;
  radius?: number;
  opacity?: number;
  children?: React.ReactNode;
}) {
  return (
    <div
      style={{
        position: 'absolute',
        left: x,
        top: y,
        width: w,
        height: h,
        borderRadius: radius,
        overflow: 'hidden',
        opacity,
      }}
    >
      <div data-cf-layer={REGION_LAYER} style={{position: 'absolute', inset: 0, backgroundColor: color}} />
      {/* Positioned like the colour so in-flow children still paint over it. */}
      <div style={{position: 'absolute', inset: 0}}>{children}</div>
    </div>
  );
}

// --- people -----------------------------------------------------------------

export type PersonPose =
  | 'stand'
  | 'sit'
  | 'wave'
  | 'point'
  | 'think'
  | 'cheer'
  | 'ouch'
  | 'shrug'
  | 'walk'
  | 'hold'
  | 'reach'
  | 'crouch'
  | 'jump'
  | 'lookUp';
export type {Mood};

export interface PersonProps extends FigureProps {
  pose?: PersonPose;
  /** The pose the person is moving into; `poseT` runs 0 (`pose`) → 1 (`toPose`). */
  toPose?: PersonPose;
  poseT?: number;
  /** `bust`: a close-up from the chest up, the head and shoulders filling the box. */
  framing?: 'full' | 'bust';
  /** What the person holds in the `hold` pose; its x/y are measured from the point between the hands. */
  children?: React.ReactNode;
  mood?: Mood;
  age?: 'child' | 'adult' | 'elder';
  outfit?: 'shirt' | 'coat' | 'dress';
  hairStyle?: 'short' | 'long' | 'bald' | 'bun';
  /** Mouth moves as if speaking. */
  talking?: boolean;
  glasses?: boolean;
  shirt?: string;
  pants?: string;
  skin?: string;
  hair?: string;
  /** Poses at moments of the shot; needs `duration`. */
  actions?: RigAction[];
  /** A repeating motion; `pose="walk"` is `stand` plus the walk cycle. */
  cycle?: CycleSpec;
  /** The shot's length in frames, for `actions`. */
  duration?: number;
  /** Bring one hand to a point of the drawing (IK). */
  reach?: ReachSpec;
  /** Turn the head toward a point of the drawing. */
  lookAt?: {to: P2; head: P2};
}

export type Pt = [number, number];

/**
 * A person, front view, story-book proportions (big head, small body), built
 * on the `humanoid` rig. Box aspect: 260 x 420 (standing adult).
 * `pose`: stand, sit (on a small stool), wave, point (to the viewer's right —
 * flip to point left), think, cheer, ouch, shrug, hold (both hands in front
 * of the chest, holding `children`), reach, crouch, jump, lookUp; `walk` is
 * standing plus the walking cycle.
 * Moving between poses: give `toPose` and animate `poseT` 0 → 1. A series of
 * poses over the shot: `actions` (needs `duration`, the shot's frames). A
 * repeating motion: `cycle` ('walk' | 'run'). `reach` brings one hand to a
 * point of the drawing with IK; `lookAt` turns the head.
 * `framing="bust"`: a close-up from the chest up, box 260 x 234.
 */
export function Person({
  pose: fromPose = 'stand',
  toPose,
  poseT = 0,
  framing = 'full',
  children,
  mood = 'neutral',
  age = 'adult',
  outfit = 'shirt',
  hairStyle,
  talking = false,
  glasses,
  shirt = '#2BB6A8',
  pants = '#2D3E73',
  skin = '#F9C4B4',
  hair,
  still = false,
  actions,
  cycle,
  duration,
  reach,
  lookAt,
  ...fig
}: PersonProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const kit = useKit();
  const walking = fromPose === 'walk';
  const basePose = walking ? 'stand' : fromPose;
  const target = toPose === 'walk' ? 'stand' : toPose;
  const motion: RigMotion = {
    skeleton: 'humanoid',
    pose: basePose,
    actions,
    duration,
    cycle: cycle ?? (walking ? {name: 'walk'} : undefined),
    blendTo: target ? {pose: target, t: poseT} : undefined,
    still,
  };
  const style = hairStyle ?? (age === 'elder' ? 'bald' : 'short');
  const hairColor = hair ?? (age === 'elder' ? '#B9B4C7' : '#5B2C6F');
  const wearGlasses = glasses ?? age === 'elder';

  const child = age === 'child';
  const torsoH = child ? 100 : 130;
  const sit = fromPose === 'sit';
  const cx = 130;
  const headTop = 40 + (child ? 20 : 0) + (sit ? 40 : 0);
  const headH = child ? 108 : 104;
  const headW = child ? 104 : 96;
  const torsoTop = headTop + headH + 8;
  const legTop = torsoTop + torsoH - 18;
  const floorY = 410;
  const ankleY = floorY - 12;
  const kneeY = (legTop + ankleY) / 2;
  const k = torsoH / 130;
  const sy = torsoTop + 20;
  const shirtColor = outfit === 'coat' ? '#F3F6FB' : shirt;
  const hx = cx - headW / 2;

  const shoulder = (s: -1 | 1): P2 => [cx + s * 40, sy];
  const elbow = (s: -1 | 1): P2 => [cx + s * 50, sy + 55 * k];
  const hand = (s: -1 | 1): P2 => [cx + s * 46, sy + 105 * k];
  const neck: P2 = [cx, headTop + headH + 6];
  const hips: P2 = [cx, legTop];

  const arm = (s: -1 | 1, side: 'L' | 'R') => (
    <Bone name={`arm${side}`} pivot={shoulder(s)}>
      <Shape material="fabric" color={shirtColor} capsule={{x1: shoulder(s)[0], y1: shoulder(s)[1], x2: elbow(s)[0], y2: elbow(s)[1], r: 11}} />
      <Bone name={`foreArm${side}`} pivot={elbow(s)}>
        <Shape material="fabric" color={shirtColor} capsule={{x1: elbow(s)[0], y1: elbow(s)[1], x2: hand(s)[0], y2: hand(s)[1], r: 11}} />
        <Shape material="skin" color={skin} ellipse={{cx: hand(s)[0], cy: hand(s)[1], rx: 12, ry: 12}} />
      </Bone>
    </Bone>
  );
  const leg = (s: -1 | 1, side: 'L' | 'R') => {
    const x = cx + s * 17;
    return (
      <Bone name={`thigh${side}`} pivot={[x, legTop]}>
        <Shape material="fabric" color={pants} capsule={{x1: x, y1: legTop, x2: x, y2: kneeY, r: 13}} />
        <Bone name={`shin${side}`} pivot={[x, kneeY]}>
          <Shape material="fabric" color={pants} capsule={{x1: x, y1: kneeY, x2: x, y2: ankleY, r: 13}} />
          <Shape material="matte" color="#2B2140" ellipse={{cx: x + s * 3, cy: ankleY + 4, rx: 20, ry: 10}} />
        </Bone>
      </Bone>
    );
  };

  const torso =
    outfit === 'dress' ? (
      <Shape
        material="fabric"
        color={shirt}
        d={`M${cx - 34},${torsoTop} h68 q14,0 16,24 l18,${torsoH + 10} q0,10 -12,10 h-112 q-12,0 -12,-10 l18,${-torsoH - 10} q2,-24 16,-24 Z`}
      />
    ) : (
      <g>
        <Shape material="fabric" color={shirtColor} rect={{x: cx - 44, y: torsoTop, w: 88, h: torsoH}} />
        {outfit === 'coat' ? (
          <g>
            <Shape material="fabric" color={shirt} d={`M${cx - 14},${torsoTop} L${cx},${torsoTop + 34} L${cx + 14},${torsoTop} Z`} />
            <path d={`M${cx},${torsoTop + 34} L${cx},${torsoTop + torsoH}`} stroke="#D5DCE8" strokeWidth={3} />
          </g>
        ) : null}
      </g>
    );

  const hairBack =
    style === 'long' ? <Shape material="fur" color={hairColor} rect={{x: hx - 8, y: headTop + 10, w: headW + 16, h: headH + 34}} /> : null;
  const hairFront = (() => {
    switch (style) {
      case 'bald':
        return (
          <g>
            <Shape material="fur" color={hairColor} d={`M${hx - 3},${headTop + 64} q-2,-28 14,-36 q-4,18 2,40 Z`} />
            <Shape material="fur" color={hairColor} d={`M${hx + headW + 3},${headTop + 64} q2,-28 -14,-36 q4,18 -2,40 Z`} />
          </g>
        );
      case 'bun':
        return (
          <g>
            <Shape material="fur" color={hairColor} ellipse={{cx, cy: headTop - 8, rx: 20, ry: 20}} />
            <Shape
              material="fur"
              color={hairColor}
              d={`M${hx - 2},${headTop + 50} C${hx - 4},${headTop} ${hx + headW + 4},${headTop} ${hx + headW + 2},${headTop + 50} C${cx + 20},${headTop + 22} ${cx - 20},${headTop + 22} ${hx - 2},${headTop + 50} Z`}
            />
          </g>
        );
      case 'long':
        return (
          <Shape
            material="fur"
            color={hairColor}
            d={`M${hx - 4},${headTop + 62} C${hx - 6},${headTop - 6} ${hx + headW + 6},${headTop - 6} ${hx + headW + 4},${headTop + 62} C${cx + 30},${headTop + 30} ${cx - 10},${headTop + 20} ${hx - 4},${headTop + 62} Z`}
          />
        );
      default:
        return (
          <Shape
            material="fur"
            color={hairColor}
            d={`M${hx - 3},${headTop + 52} C${hx - 8},${headTop - 2} ${cx + 10},${headTop - 16} ${hx + headW + 3},${headTop + 30} C${hx + headW + 5},${headTop + 44} ${hx + headW},${headTop + 50} ${hx + headW - 4},${headTop + 50} C${cx + 16},${headTop + 20} ${cx - 16},${headTop + 30} ${hx + 8},${headTop + 36} Z`}
          />
        );
    }
  })();

  const eyeY = headTop + headH * 0.58 - 6;
  const face = (
    <g>
      {mood !== 'angry' && mood !== 'pain' ? (
        <>
          <circle cx={cx - 24} cy={headTop + headH * 0.58 + 8} r={7} fill={BLUSH} opacity={0.45} />
          <circle cx={cx + 24} cy={headTop + headH * 0.58 + 8} r={7} fill={BLUSH} opacity={0.45} />
        </>
      ) : null}
      <Eyes cx={cx} cy={eyeY} gap={32} size={5} />
      <Brows cx={cx} cy={eyeY - 14} gap={32} />
      <Mouth cx={cx} cy={headTop + headH * 0.58 + 18} talking={talking} />
      {wearGlasses ? (
        <g stroke={kit.ink} strokeWidth={2.5} fill="none" opacity={0.9}>
          <circle cx={cx - 16} cy={eyeY} r={10} />
          <circle cx={cx + 16} cy={eyeY} r={10} />
          <path d={`M${cx - 6},${eyeY} L${cx + 6},${eyeY}`} />
        </g>
      ) : null}
    </g>
  );

  const bust = framing === 'bust';
  const box = bust ? {vx: 0, vy: headTop - 24, vw: 260, vh: 234, clip: true} : {vx: 0, vy: 0, vw: 260, vh: 420, clip: false};
  const placed = {...fig, size: fig.size ?? (bust ? 260 : 420), ...box};

  // Where the hands end up, for what the person holds.
  const solved = solveRig(motion, frame, fps);
  const chain = (p: P2, side: 'L' | 'R', s: -1 | 1): P2 => {
    let q = rotateAbout(p, elbow(s), solved.angles[`foreArm${side}`]);
    q = rotateAbout(q, shoulder(s), solved.angles[`arm${side}`]);
    q = rotateAbout(q, hips, solved.angles.spine);
    return [q[0], q[1] + (solved.lower - solved.lift) * 420];
  };
  const hl = chain(hand(-1), 'L', -1);
  const hr = chain(hand(1), 'R', 1);
  const holding: P2 = [(hl[0] + hr[0]) / 2, (hl[1] + hr[1]) / 2];

  return (
    <>
      <Figure {...placed}>
        {bust ? null : <GroundShadow cx={cx} cy={floorY} rx={sit ? 90 : 70} />}
        {sit && !bust ? (
          <g>
            <Shape material="wood" color="#B0567E" rect={{x: cx - 62, y: legTop + 32, w: 124, h: 14}} />
            <Shape material="wood" color="#8E3F63" rect={{x: cx - 52, y: legTop + 42, w: 12, h: floorY - legTop - 42}} />
            <Shape material="wood" color="#8E3F63" rect={{x: cx + 40, y: legTop + 42, w: 12, h: floorY - legTop - 42}} />
          </g>
        ) : null}
        <Rig {...motion} reach={reach} lookAt={lookAt} mood={mood} height={420}>
          <Bone name="hips" pivot={hips}>
            {leg(-1, 'L')}
            {leg(1, 'R')}
            <Bone name="spine" pivot={hips}>
              {torso}
              <Bone name="head" pivot={neck}>
                {hairBack}
                <Shape material="skin" color={shadeOf(skin, -0.12)} rect={{x: cx - 12, y: headTop + headH - 10, w: 24, h: 24}} />
                <Shape material="skin" color={skin} ellipse={{cx: hx - 2, cy: headTop + headH * 0.55, rx: 10, ry: 10}} />
                <Shape material="skin" color={skin} ellipse={{cx: hx + headW + 2, cy: headTop + headH * 0.55, rx: 10, ry: 10}} />
                <Shape material="skin" color={skin} rect={{x: hx, y: headTop, w: headW, h: headH}} />
                {hairFront}
                {face}
              </Bone>
              {arm(-1, 'L')}
              {arm(1, 'R')}
            </Bone>
          </Bone>
        </Rig>
      </Figure>
      {fromPose === 'hold' ? <HeldAt at={figurePoint(placed, holding)}>{children}</HeldAt> : null}
    </>
  );
}

// --- body & health ----------------------------------------------------------

/**
 * Level of detail each built-in figure is drawn at: `rich` (many parts,
 * for a main figure), `medium` (a few parts), `silhouette` (one block with a
 * rim of light). The library records it so a figure is only reused where its
 * detail is enough for the role it plays in a video.
 */
export type DetailLevel = 'rich' | 'medium' | 'silhouette';

export const FIGURE_DETAIL: Record<string, DetailLevel> = {
  Person: 'rich',
  Airplane: 'rich',
  Tooth: 'medium',
  Germ: 'medium',
  OpenMouth: 'medium',
  Toothbrush: 'medium',
  Toothpaste: 'medium',
  Drop: 'medium',
  Shield: 'medium',
  Candy: 'medium',
  Lollipop: 'medium',
  Soda: 'medium',
  Donut: 'medium',
  Apple: 'medium',
  Clock: 'medium',
  Table: 'medium',
  Chair: 'medium',
  Window: 'medium',
  Plant: 'medium',
  House: 'medium',
  Tree: 'medium',
  Sun: 'medium',
  Cloud: 'silhouette',
  Lightbulb: 'medium',
  Heart: 'medium',
  Coin: 'medium',
  Book: 'medium',
  Phone: 'medium',
  Magnifier: 'medium',
  Mark: 'silhouette',
  Sparkle: 'silhouette',
};

const TOOTH_PATH =
  'M40,52 C36,18 74,6 100,24 C126,6 164,18 160,52 C158,92 150,112 146,140 C142,178 132,204 120,204 C108,204 106,168 100,158 C94,168 92,204 80,204 C68,204 58,178 54,140 C50,112 42,92 40,52 Z';

/**
 * A molar with a face. `decay` runs 0 (healthy, white) → 0.3 (brown spot) →
 * 0.6 (a dark hole) → 1 (big hole, crack, yellowed). Animate it to show decay
 * spreading. Mood follows the decay unless given. Box aspect: 200 x 220.
 */
export function Tooth({
  decay = 0,
  mood,
  shine = false,
  face = true,
  still = false,
  ...fig
}: FigureProps & {decay?: number; mood?: Mood; shine?: boolean; face?: boolean}) {
  const frame = useCurrentFrame();
  const phase = phaseOf(fig.x ?? 960, fig.y ?? 540);
  const d = Math.max(0, Math.min(1, decay));
  const enamel = d > 0.75 ? '#F4E6B8' : WHITE;
  const autoMood: Mood = d < 0.15 ? 'happy' : d < 0.45 ? 'worried' : d < 0.8 ? 'sad' : 'pain';
  const spot = Math.max(0, Math.min(1, (d - 0.1) / 0.3));
  const hole = Math.max(0, Math.min(1, (d - 0.4) / 0.4));
  const clip = useSvgId('tooth');
  const m = mood ?? autoMood;
  return (
    <Figure {...fig} size={fig.size ?? 300} vw={200} vh={220}>
      <defs>
        <clipPath id={clip}>
          <path d={TOOTH_PATH} />
        </clipPath>
      </defs>
      <Shape material="glossy" color={enamel} d={TOOTH_PATH} />
      <g clipPath={`url(#${clip})`}>
        {spot > 0 ? (
          <g opacity={spot}>
            <ellipse cx={70} cy={48} rx={8 + spot * 8} ry={6 + spot * 6} fill="#9A6A45" />
            <ellipse cx={122} cy={40} rx={4 + spot * 5} ry={3 + spot * 4} fill="#9A6A45" opacity={0.8} />
          </g>
        ) : null}
        {hole > 0 ? (
          <g>
            <path
              d={`M${70 - 18 * hole},${36} q${6 * hole},${-10 * hole} ${16 * hole},${-2} q${8 * hole},${-8 * hole} ${16 * hole},${2} q${10 * hole},${4 * hole} ${6 * hole},${22 * hole} q${-10 * hole},${14 * hole} ${-22 * hole},${8 * hole} q${-14 * hole},${-6 * hole} ${-16 * hole},${-30 * hole} Z`}
              fill="#6B4228"
            />
            <ellipse cx={72} cy={44} rx={12 * hole} ry={10 * hole} fill="#2E1B12" />
          </g>
        ) : null}
        {d > 0.85 ? <path d="M112,30 l-8,26 l12,10 l-10,26" stroke="#6B4228" strokeWidth={4} fill="none" strokeLinecap="round" strokeLinejoin="round" /> : null}
      </g>
      {shine && d < 0.15 ? (
        <path d={`M172,40 l4,-12 l4,12 l12,4 l-12,4 l-4,12 l-4,-12 l-12,-4 Z`} fill={WHITE} opacity={(Math.sin(frame / 6) + 1) / 2} />
      ) : null}
      {face ? (
        <g>
          <Eyes cx={100} cy={90} gap={32} mood={m} phase={phase} still={still} />
          <Brows cx={100} cy={76} gap={32} mood={m} />
          <Mouth cx={100} cy={114} mood={m} />
        </g>
      ) : null}
    </Figure>
  );
}

/**
 * A bacterium / germ: a wobbling blob with eyes and a grin. `mood` 'evil'
 * (default, the villain of a health story), 'happy' or 'sad'. `variant` 0-3
 * changes the shape. Box aspect: 1:1.
 */
export function Germ({
  color = '#8BC34A',
  mood = 'evil',
  variant = 0,
  still = false,
  ...fig
}: FigureProps & {color?: string; mood?: 'evil' | 'happy' | 'sad'; variant?: number}) {
  const frame = useCurrentFrame();
  const ink = useKit().ink;
  const phase = phaseOf(fig.x ?? 960, fig.y ?? 540) + variant * 11;
  const t = still ? 0 : (frame + phase) / 9;
  const bumps = [7, 9, 6, 11][variant % 4];
  const pts: string[] = [];
  const N = 72;
  for (let i = 0; i <= N; i++) {
    const a = (i / N) * Math.PI * 2;
    const r = 70 + Math.sin(a * bumps + t) * 7 + Math.sin(a * 3 - t * 0.7) * 4;
    pts.push(`${(100 + Math.cos(a) * r).toFixed(1)},${(100 + Math.sin(a) * r * 0.92).toFixed(1)}`);
  }
  const spikeColor = shadeOf(color, -0.25);
  const spikes = Array.from({length: bumps}, (_, i) => {
    const a = (i / bumps) * Math.PI * 2 + 0.3;
    const r0 = 72;
    const r1 = 92 + Math.sin(t + i) * 4;
    return (
      <g key={i}>
        <Shape
          material="organic"
          color={spikeColor}
          capsule={{x1: 100 + Math.cos(a) * r0, y1: 100 + Math.sin(a) * r0 * 0.92, x2: 100 + Math.cos(a) * r1, y2: 100 + Math.sin(a) * r1 * 0.92, r: 3}}
        />
        <Shape material="organic" color={spikeColor} ellipse={{cx: 100 + Math.cos(a) * r1, cy: 100 + Math.sin(a) * r1 * 0.92, rx: 6, ry: 6}} />
      </g>
    );
  });
  const blink = isBlinking(frame, phase, still);
  const eyeY = 88;
  return (
    <Figure {...fig} size={fig.size ?? 220} vw={200} vh={200}>
      {spikes}
      <Shape material="organic" color={color} d={`M${pts.join(' L')} Z`} />
      <circle cx={132} cy={132} r={6} fill={spikeColor} opacity={0.7} />
      <circle cx={60} cy={120} r={5} fill={spikeColor} opacity={0.7} />
      {[80, 120].map((ex) => (
        <g key={ex}>
          <ellipse cx={ex} cy={eyeY} rx={13} ry={blink ? 2 : 14} fill={WHITE} />
          {!blink ? <circle cx={ex + (mood === 'evil' ? 3 : 0)} cy={eyeY + 3} r={6} fill={ink} /> : null}
        </g>
      ))}
      {mood === 'evil' ? (
        <g stroke={ink} strokeWidth={6} strokeLinecap="round">
          <line x1={64} y1={66} x2={92} y2={78} />
          <line x1={136} y1={66} x2={108} y2={78} />
        </g>
      ) : null}
      {mood === 'evil' ? (
        <g>
          <path d="M70,118 Q100,150 130,118 Z" fill={ink} />
          <path d="M80,121 l6,10 l6,-8 l6,9 l6,-9 l6,8 l6,-10 Z" fill={WHITE} />
        </g>
      ) : mood === 'happy' ? (
        <path d="M76,118 Q100,142 124,118" stroke={ink} strokeWidth={6} fill="none" strokeLinecap="round" />
      ) : (
        <path d="M78,130 Q100,114 122,130" stroke={ink} strokeWidth={6} fill="none" strokeLinecap="round" />
      )}
    </Figure>
  );
}

/**
 * A wide-open mouth seen from the front: lips, upper and lower row of teeth,
 * tongue. `decayed` lists which teeth (0-5, left to right, top row first then
 * bottom row as 6-11) show a cavity; `decay` (0-1) how far it has gone.
 * Box aspect: 400 x 260.
 */
export function OpenMouth({
  decayed = [],
  decay = 0.6,
  lips = '#E0607E',
  ...fig
}: FigureProps & {decayed?: number[]; decay?: number; lips?: string}) {
  const d = Math.max(0, Math.min(1, decay));
  const tooth = (i: number, x: number, y: number, up: boolean) => {
    const bad = decayed.includes(i);
    const h = 48;
    return (
      <g key={i}>
        <Shape material="glossy" color={bad && d > 0.75 ? '#F4E6B8' : WHITE} rect={{x, y: up ? y : y - h, w: 40, h}} />
        {bad ? <ellipse cx={x + 20} cy={up ? y + h - 16 : y - h + 16} rx={6 + d * 8} ry={5 + d * 6} fill={d > 0.45 ? '#3A2217' : '#9A6A45'} /> : null}
      </g>
    );
  };
  return (
    <Figure {...fig} size={fig.size ?? 520} vw={400} vh={260}>
      <Shape material="skin" color={lips} ellipse={{cx: 200, cy: 130, rx: 190, ry: 120}} />
      <ellipse cx={200} cy={130} rx={160} ry={96} fill="#5A1530" />
      <Shape material="organic" color="#F2728F" ellipse={{cx: 200, cy: 196, rx: 96, ry: 36}} />
      <path d="M200,176 L200,214" stroke="#D9587A" strokeWidth={4} strokeLinecap="round" />
      {[0, 1, 2, 3, 4, 5].map((i) => tooth(i, 78 + i * 42, 42, true))}
      {[0, 1, 2, 3, 4, 5].map((i) => tooth(i + 6, 78 + i * 42, 222, false))}
    </Figure>
  );
}

/**
 * A toothbrush lying horizontally, head on the right, bristles up
 * (`bristlesDown` turns them towards a tooth below). `paste` adds a blob of
 * toothpaste. Animate `x` / `rotate` for brushing. Box aspect: 400 x 100.
 */
export function Toothbrush({
  color = '#3D7BFF',
  paste = true,
  bristlesDown = false,
  ...fig
}: FigureProps & {color?: string; paste?: boolean; bristlesDown?: boolean}) {
  return (
    <Figure {...fig} size={fig.size ?? 360} vw={400} vh={100}>
      <g transform={bristlesDown ? 'translate(0 100) scale(1 -1)' : undefined}>
        <Shape material="glossy" color={color} rect={{x: 10, y: 56, w: 300, h: 24}} />
        <Shape material="glossy" color={color} rect={{x: 290, y: 52, w: 100, h: 30}} />
        {Array.from({length: 8}, (_, i) => (
          <Shape key={i} material="matte" color={i % 2 ? '#DDE7F5' : WHITE} rect={{x: 298 + i * 11, y: 22, w: 8, h: 32}} />
        ))}
        {paste ? <Shape material="glossy" color="#E9FAFF" d="M296,24 q10,-18 30,-10 q12,-12 28,-2 q16,-8 26,6 q4,10 -6,12 h-72 q-10,-2 -6,-6 Z" /> : null}
        {paste ? <path d="M310,16 q18,-6 36,2 q14,-6 28,4" stroke="#6FD0E8" strokeWidth={5} fill="none" strokeLinecap="round" /> : null}
      </g>
    </Figure>
  );
}

/** A tube of toothpaste, cap on the right. Box aspect: 300 x 120. */
export function Toothpaste({color = '#2BB6A8', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 280} vw={300} vh={120}>
      <Shape material="glossy" color={WHITE} d="M20,20 L230,32 Q244,34 244,48 L244,72 Q244,86 230,88 L20,100 Q10,100 10,90 L10,30 Q10,20 20,20 Z" />
      <Shape material="matte" color={color} d="M40,40 L200,46 L200,74 L40,80 Z" />
      <Shape material="glossy" color={color} rect={{x: 244, y: 40, w: 44, h: 40}} />
      <path d="M90,52 h70 M90,66 h50" stroke={WHITE} strokeWidth={6} strokeLinecap="round" />
    </Figure>
  );
}

/** A drop of liquid (acid, water, sweat, a tear). `face` adds eyes and a grin. Box aspect: 140 x 180. */
export function Drop({
  color = '#B6E34A',
  face = false,
  ...fig
}: FigureProps & {color?: string; face?: boolean}) {
  const ink = useKit().ink;
  return (
    <Figure {...fig} size={fig.size ?? 120} vw={140} vh={180}>
      <Shape material="liquid" color={color} d="M70,8 C92,52 128,86 128,118 C128,152 102,172 70,172 C38,172 12,152 12,118 C12,86 48,52 70,8 Z" />
      {face ? (
        <g>
          <circle cx={54} cy={118} r={6} fill={ink} />
          <circle cx={88} cy={118} r={6} fill={ink} />
          <path d="M52,138 Q70,154 90,138" stroke={ink} strokeWidth={5} fill="none" strokeLinecap="round" />
          <path d="M44,102 l16,6 M98,102 l-16,6" stroke={ink} strokeWidth={5} strokeLinecap="round" />
        </g>
      ) : null}
    </Figure>
  );
}

/** A shield (protection, defence, fluoride on enamel). Box aspect: 160 x 190. */
export function Shield({color = '#3D7BFF', check = true, ...fig}: FigureProps & {color?: string; check?: boolean}) {
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={160} vh={190}>
      <Shape material="metal" color={color} d="M80,8 L150,32 C150,110 124,160 80,184 C36,160 10,110 10,32 Z" />
      {check ? <path d="M48,96 L72,120 L114,70" stroke={WHITE} strokeWidth={14} strokeLinecap="round" strokeLinejoin="round" fill="none" /> : null}
    </Figure>
  );
}

// --- food -------------------------------------------------------------------

/** A wrapped candy, twists left and right. Box aspect: 240 x 110. */
export function Candy({color = '#FF5C8A', ...fig}: FigureProps & {color?: string}) {
  const dark = shadeOf(color, -0.2);
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={240} vh={110}>
      <Shape material="glossy" color={dark} d="M70,55 L14,14 Q4,55 14,96 Z" />
      <Shape material="glossy" color={dark} d="M170,55 L226,14 Q236,55 226,96 Z" />
      <Shape material="glossy" color={color} ellipse={{cx: 120, cy: 55, rx: 56, ry: 44}} />
      <path d="M86,24 q20,30 0,62 M118,12 q20,40 0,86 M150,22 q18,32 0,66" stroke={WHITE} strokeWidth={8} fill="none" opacity={0.4} />
    </Figure>
  );
}

/** A spiral lollipop on a stick. Box aspect: 140 x 260. */
export function Lollipop({color = '#FF5C8A', ...fig}: FigureProps & {color?: string}) {
  const clip = useSvgId('lolly');
  return (
    <Figure {...fig} size={fig.size ?? 240} vw={140} vh={260}>
      <Shape material="matte" color="#F3F6FB" rect={{x: 64, y: 120, w: 12, h: 132}} />
      <Shape material="glossy" color={color} ellipse={{cx: 70, cy: 70, rx: 62, ry: 62}} />
      <clipPath id={clip}>
        <circle cx={70} cy={70} r={56} />
      </clipPath>
      <path
        clipPath={`url(#${clip})`}
        d="M70,70 m0,-8 a8,8 0 1,1 -8,8 a16,16 0 1,1 16,16 a24,24 0 1,1 -24,-24 a32,32 0 1,1 32,32 a40,40 0 1,1 -40,-40 a48,48 0 1,1 48,48"
        stroke={WHITE}
        strokeWidth={9}
        fill="none"
        opacity={0.6}
        strokeLinecap="round"
      />
    </Figure>
  );
}

/** A soft-drink cup with lid and straw. Box aspect: 160 x 260. */
export function Soda({color = '#E8453C', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 240} vw={160} vh={260}>
      <g transform="rotate(14 99 40)">
        <Shape material="glossy" color="#FFD23F" rect={{x: 92, y: 6, w: 14, h: 70}} />
      </g>
      <Shape material="glossy" color={color} d="M24,70 L136,70 L120,250 Q118,256 110,256 L50,256 Q42,256 40,250 Z" />
      <Shape material="matte" color="#F3F6FB" rect={{x: 16, y: 56, w: 128, h: 22}} />
      <path d="M44,120 h72 l-4,50 h-64 Z" fill={WHITE} opacity={0.9} />
      <circle cx={80} cy={145} r={14} fill={color} />
    </Figure>
  );
}

/** A glazed doughnut with sprinkles. Box aspect: 1:1. */
export function Donut({color = '#FF7AB6', ...fig}: FigureProps & {color?: string}) {
  const sprinkles = [
    [60, 60, 20, '#FFD23F'],
    [110, 44, -30, '#3D7BFF'],
    [148, 74, 60, '#FFFFFF'],
    [150, 124, -10, '#2BB6A8'],
    [112, 156, 40, '#FFD23F'],
    [58, 140, -50, '#FFFFFF'],
    [44, 100, 80, '#3D7BFF'],
    [88, 32, 10, '#2BB6A8'],
  ] as const;
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <Shape material="organic" color="#D9975B" ellipse={{cx: 100, cy: 104, rx: 90, ry: 90}} />
      <Shape
        material="glossy"
        color={color}
        d="M100,18 C150,18 186,50 184,96 C182,118 170,112 164,128 C156,150 140,140 128,160 C116,176 96,164 82,170 C60,178 50,156 36,146 C18,134 16,110 16,96 C16,50 52,18 100,18 Z"
      />
      <ellipse cx={100} cy={100} rx={26} ry={26} fill="#8A5A33" />
      {sprinkles.map(([x, y, r, c], i) => (
        <rect key={i} x={x - 9} y={y - 3} width={18} height={6} rx={3} fill={c} transform={`rotate(${r} ${x} ${y})`} />
      ))}
    </Figure>
  );
}

/** An apple with a leaf. Box aspect: 1:1. */
export function Apple({color = '#E8453C', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 180} vw={200} vh={200}>
      <Shape material="glossy" color={color} d="M100,52 C70,30 22,40 22,96 C22,150 60,190 84,190 C94,190 96,184 100,184 C104,184 106,190 116,190 C140,190 178,150 178,96 C178,40 130,30 100,52 Z" />
      <Shape material="wood" color="#6B4228" rect={{x: 95, y: 16, w: 10, h: 40}} />
      <Shape material="leaf" color="#4CAF50" d="M104,34 C120,12 150,14 158,22 C146,40 120,44 104,34 Z" />
    </Figure>
  );
}

// --- objects ----------------------------------------------------------------

/** A square wall clock whose hands show `hour`:`minute` (animate them). Box aspect: 1:1. */
export function Clock({
  hour = 4,
  minute = 0,
  frameColor = '#C2477E',
  handColor = '#7B3FC4',
  ...fig
}: FigureProps & {hour?: number; minute?: number; frameColor?: string; handColor?: string}) {
  const ha = ((hour % 12) + minute / 60) * 30;
  const ma = minute * 6;
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <Shape material="glossy" color={frameColor} rect={{x: 4, y: 4, w: 180, h: 180}} />
      <Shape material="matte" color={WHITE} rect={{x: 16, y: 16, w: 156, h: 156}} />
      {Array.from({length: 12}, (_, i) => {
        const a = (i / 12) * Math.PI * 2;
        return (
          <rect
            key={i}
            x={94 - 3 + Math.sin(a) * 62}
            y={94 - 7 - Math.cos(a) * 62}
            width={6}
            height={14}
            rx={3}
            fill="#E8B6F0"
            transform={`rotate(${i * 30} ${94 + Math.sin(a) * 62} ${94 - Math.cos(a) * 62})`}
          />
        );
      })}
      <line x1={94} y1={94} x2={94 + Math.sin((ha * Math.PI) / 180) * 36} y2={94 - Math.cos((ha * Math.PI) / 180) * 36} stroke={handColor} strokeWidth={10} strokeLinecap="round" />
      <line x1={94} y1={94} x2={94 + Math.sin((ma * Math.PI) / 180) * 52} y2={94 - Math.cos((ma * Math.PI) / 180) * 52} stroke={handColor} strokeWidth={8} strokeLinecap="round" />
    </Figure>
  );
}

/** A table seen from the front. Box aspect: 400 x 200. */
export function Table({color = '#C9709A', ...fig}: FigureProps & {color?: string}) {
  const dark = shadeOf(color, -0.18);
  return (
    <Figure {...fig} size={fig.size ?? 420} vw={400} vh={200}>
      <GroundShadow cx={200} cy={194} rx={190} />
      <Shape material="wood" color={dark} rect={{x: 30, y: 40, w: 20, h: 150}} />
      <Shape material="wood" color={dark} rect={{x: 350, y: 40, w: 20, h: 150}} />
      <Shape material="wood" color={dark} rect={{x: 260, y: 40, w: 16, h: 130}} opacity={0.7} />
      <Shape material="wood" color={color} rect={{x: 0, y: 20, w: 400, h: 30}} />
    </Figure>
  );
}

/** A simple chair, seat on the left, back on the right. Box aspect: 200 x 260. */
export function Chair({color = '#C9709A', ...fig}: FigureProps & {color?: string}) {
  const dark = shadeOf(color, -0.18);
  return (
    <Figure {...fig} size={fig.size ?? 260} vw={200} vh={260}>
      <GroundShadow cx={100} cy={254} rx={90} />
      <Shape material="wood" color={dark} rect={{x: 150, y: 20, w: 24, h: 236}} />
      <Shape material="wood" color={dark} rect={{x: 28, y: 150, w: 18, h: 106}} />
      <Shape material="wood" color={color} rect={{x: 20, y: 134, w: 160, h: 24}} />
    </Figure>
  );
}

/** A window with a sky (day: sun, night: moon and stars). Box aspect: 220 x 260. */
export function Window({time = 'day', frameColor = '#FFFFFF', ...fig}: FigureProps & {time?: 'day' | 'night'; frameColor?: string}) {
  const night = time === 'night';
  return (
    <Figure {...fig} size={fig.size ?? 260} vw={220} vh={260}>
      <Shape material="wood" color={frameColor} rect={{x: 0, y: 0, w: 220, h: 260}} />
      <Shape material="glass" color={night ? '#1B1650' : '#8FD3FF'} rect={{x: 14, y: 14, w: 192, h: 232}} />
      {night ? (
        <g>
          <circle cx={150} cy={70} r={24} fill="#FFF3B0" />
          <circle cx={162} cy={62} r={22} fill="#1B1650" />
          {[
            [50, 50],
            [90, 120],
            [60, 190],
            [150, 170],
          ].map(([x, y], i) => (
            <circle key={i} cx={x} cy={y} r={3} fill="#FFF3B0" />
          ))}
        </g>
      ) : (
        <g>
          <Shape material="emissive" color="#FFD23F" ellipse={{cx: 150, cy: 70, rx: 28, ry: 28}} />
          <ellipse cx={70} cy={170} rx={40} ry={16} fill={WHITE} />
          <ellipse cx={96} cy={160} rx={26} ry={16} fill={WHITE} />
        </g>
      )}
      <rect x={104} y={14} width={12} height={232} fill={frameColor} />
      <rect x={14} y={124} width={192} height={12} fill={frameColor} />
    </Figure>
  );
}

/** A potted plant. Box aspect: 160 x 240. */
export function Plant({color = '#2BB673', pot = '#E07A5F', ...fig}: FigureProps & {color?: string; pot?: string}) {
  const frame = useCurrentFrame();
  const sway = fig.still ? 0 : Math.sin(frame / 20) * 2;
  return (
    <Figure {...fig} size={fig.size ?? 220} vw={160} vh={240}>
      <GroundShadow cx={80} cy={236} rx={52} />
      <g transform={`rotate(${sway} 80 150)`}>
        <Shape material="leaf" color={color} d="M80,150 C40,130 20,80 36,40 C60,60 76,100 80,150 Z" />
        <Shape material="leaf" color={shadeOf(color, -0.15)} d="M80,150 C120,130 140,80 124,40 C100,60 84,100 80,150 Z" />
        <Shape material="leaf" color={shadeOf(color, 0.1)} d="M80,150 C70,100 70,50 80,10 C92,50 92,100 80,150 Z" />
      </g>
      <Shape material="stone" color={pot} d="M36,150 L124,150 L112,234 L48,234 Z" />
      <Shape material="stone" color={shadeOf(pot, -0.12)} rect={{x: 30, y: 142, w: 100, h: 18}} />
    </Figure>
  );
}

/** A little house. Box aspect: 240 x 220. */
export function House({color = '#FFB84D', roof = '#E8453C', ...fig}: FigureProps & {color?: string; roof?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 260} vw={240} vh={220}>
      <GroundShadow cx={120} cy={216} rx={110} />
      <Shape material="stone" color={color} rect={{x: 30, y: 90, w: 180, h: 126}} />
      <Shape material="stone" color={roof} d="M10,100 L120,14 L230,100 Z" />
      <Shape material="wood" color={shadeOf(color, -0.35)} rect={{x: 100, y: 140, w: 44, h: 76}} />
      <Shape material="glass" color="#8FD3FF" rect={{x: 50, y: 120, w: 36, h: 36}} />
      <Shape material="glass" color="#8FD3FF" rect={{x: 160, y: 120, w: 36, h: 36}} />
    </Figure>
  );
}

/** A round tree. Box aspect: 180 x 260. */
export function Tree({color = '#2BB673', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 280} vw={180} vh={260}>
      <GroundShadow cx={90} cy={256} rx={60} />
      <Shape material="wood" color="#8A5A3C" rect={{x: 78, y: 150, w: 24, h: 106}} />
      <Shape material="leaf" color={color} ellipse={{cx: 90, cy: 90, rx: 80, ry: 80}} />
      <Shape material="leaf" color={shadeOf(color, -0.12)} ellipse={{cx: 120, cy: 110, rx: 50, ry: 50}} />
    </Figure>
  );
}

/** A sun with slowly turning rays. Box aspect: 1:1. */
export function Sun({color = '#FFD23F', ...fig}: FigureProps & {color?: string}) {
  const frame = useCurrentFrame();
  const spin = fig.still ? 0 : frame * 0.4;
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <g transform={`rotate(${spin} 100 100)`}>
        {Array.from({length: 12}, (_, i) => (
          <rect key={i} x={95} y={4} width={10} height={30} rx={5} fill={color} transform={`rotate(${i * 30} 100 100)`} />
        ))}
      </g>
      <Shape material="emissive" color={color} ellipse={{cx: 100, cy: 100, rx: 56, ry: 56}} />
    </Figure>
  );
}

/** A fluffy cloud. Box aspect: 260 x 140. */
export function Cloud({color = '#FFFFFF', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 240} vw={260} vh={140}>
      <Shape material="matte" color={color} d="M40,130 C8,130 4,86 36,80 C30,40 80,24 104,50 C116,10 190,10 196,60 C236,56 256,100 230,124 C226,128 220,130 214,130 Z" />
    </Figure>
  );
}

/** A lightbulb; `on` makes it glow (an idea, understanding). Box aspect: 160 x 240. */
export function Lightbulb({on = true, color = '#FFD23F', ...fig}: FigureProps & {on?: boolean; color?: string}) {
  const frame = useCurrentFrame();
  const glow = on ? 0.35 + (fig.still ? 0 : Math.sin(frame / 8) * 0.1) : 0;
  return (
    <Figure {...fig} size={fig.size ?? 220} vw={160} vh={240}>
      {on ? <circle cx={80} cy={80} r={78} fill={color} opacity={glow} /> : null}
      <Shape material={on ? 'emissive' : 'glass'} color={on ? color : '#DDE3EC'} d="M80,16 C120,16 146,46 146,82 C146,112 124,126 116,150 L44,150 C36,126 14,112 14,82 C14,46 40,16 80,16 Z" />
      <path d="M58,150 L58,112 Q80,90 102,112 L102,150" stroke={on ? '#E89B10' : '#9AA5B5'} strokeWidth={6} fill="none" />
      <Shape material="metal" color="#8F9BB3" rect={{x: 46, y: 150, w: 68, h: 40}} />
      <path d="M46,164 h68 M46,178 h68" stroke="#6E7A92" strokeWidth={5} />
      <path d="M62,194 h36 l-8,14 h-20 Z" fill="#6E7A92" />
    </Figure>
  );
}

/** A heart (love, health, a pulse). `beat` makes it pulse. Box aspect: 200 x 180. */
export function Heart({color = '#FF4D6D', beat = false, ...fig}: FigureProps & {color?: string; beat?: boolean}) {
  const frame = useCurrentFrame();
  const s = beat && !fig.still ? 1 + Math.max(0, Math.sin(frame / 4)) * 0.08 : 1;
  return (
    <Figure {...fig} size={fig.size ?? 180} vw={200} vh={180}>
      <g transform={`translate(100 90) scale(${s}) translate(-100 -90)`}>
        <Shape
          material="organic"
          color={color}
          d="M100,170 C60,140 10,110 10,62 C10,30 34,10 62,10 C80,10 94,20 100,34 C106,20 120,10 138,10 C166,10 190,30 190,62 C190,110 140,140 100,170 Z"
        />
      </g>
    </Figure>
  );
}

/** A gold coin (money, cost, price). Box aspect: 1:1. */
export function Coin({color = '#FFC72C', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 140} vw={160} vh={160}>
      <Shape material="metal" color={shadeOf(color, -0.25)} ellipse={{cx: 80, cy: 86, rx: 70, ry: 70}} />
      <Shape material="metal" color={color} ellipse={{cx: 80, cy: 78, rx: 70, ry: 70}} />
      <Shape material="metal" color={shadeOf(color, -0.1)} ellipse={{cx: 80, cy: 78, rx: 52, ry: 52}} />
      <path d="M92,56 q-12,-10 -24,-2 q-12,10 4,18 q18,6 10,20 q-10,10 -26,0 M80,46 v64" stroke={shadeOf(color, -0.35)} strokeWidth={8} fill="none" strokeLinecap="round" />
    </Figure>
  );
}

/** A closed book. Box aspect: 200 x 160. */
export function Book({color = '#3D7BFF', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={160}>
      <Shape material="matte" color="#F3F6FB" rect={{x: 20, y: 20, w: 170, h: 130}} />
      <Shape material="fabric" color={color} rect={{x: 10, y: 10, w: 170, h: 130}} />
      <Shape material="fabric" color={shadeOf(color, -0.2)} rect={{x: 10, y: 10, w: 22, h: 130}} />
      <rect x={60} y={50} width={90} height={16} rx={8} fill={WHITE} opacity={0.6} />
      <rect x={60} y={76} width={60} height={12} rx={6} fill={WHITE} opacity={0.4} />
    </Figure>
  );
}

/** A smartphone; `glow` lights the screen. Box aspect: 120 x 220. */
export function Phone({color = '#2B2140', glow = true, ...fig}: FigureProps & {color?: string; glow?: boolean}) {
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={120} vh={220}>
      <Shape material="glossy" color={color} rect={{x: 4, y: 4, w: 112, h: 212}} />
      <Shape material={glow ? 'emissive' : 'glass'} color={glow ? '#8FD3FF' : '#3A3F5C'} rect={{x: 12, y: 20, w: 96, h: 170}} />
      {glow ? (
        <g fill={WHITE} opacity={0.8}>
          <rect x={22} y={34} width={76} height={40} rx={6} />
          <rect x={22} y={84} width={56} height={10} rx={5} />
          <rect x={22} y={102} width={70} height={10} rx={5} />
          <rect x={22} y={120} width={44} height={10} rx={5} />
        </g>
      ) : null}
      <rect x={46} y={198} width={28} height={8} rx={4} fill="#5A5F7C" />
    </Figure>
  );
}

/** A magnifying glass (looking closer, investigating). Box aspect: 1:1. */
export function Magnifier({color = '#2B2140', ...fig}: FigureProps & {color?: string}) {
  return (
    <Figure {...fig} size={fig.size ?? 200} vw={200} vh={200}>
      <g transform="rotate(-45 133 154)">
        <Shape material="wood" color={color} rect={{x: 118, y: 112, w: 30, h: 84}} />
      </g>
      <Shape material="metal" color={color} ellipse={{cx: 84, cy: 84, rx: 66, ry: 66}} />
      <Shape material="glass" color="#CFEFFF" ellipse={{cx: 84, cy: 84, rx: 52, ry: 52}} />
    </Figure>
  );
}

/** A round badge with a tick (right, allowed) or a cross (wrong, not allowed). Box aspect: 1:1. */
export function Mark({kind = 'check', color, ...fig}: FigureProps & {kind?: 'check' | 'cross'; color?: string}) {
  const c = color ?? (kind === 'check' ? '#2BB673' : '#E8453C');
  return (
    <Figure {...fig} size={fig.size ?? 120} vw={120} vh={120}>
      <Shape material="matte" color={c} ellipse={{cx: 60, cy: 60, rx: 56, ry: 56}} />
      {kind === 'check' ? (
        <path d="M34,62 L52,80 L88,42" stroke={WHITE} strokeWidth={14} strokeLinecap="round" strokeLinejoin="round" fill="none" />
      ) : (
        <path d="M40,40 L80,80 M80,40 L40,80" stroke={WHITE} strokeWidth={14} strokeLinecap="round" />
      )}
    </Figure>
  );
}

/** A four-pointed sparkle (clean, new, magic, a highlight). Twinkles. Box aspect: 1:1. */
export function Sparkle({color = '#FFFFFF', ...fig}: FigureProps & {color?: string}) {
  const frame = useCurrentFrame();
  const s = fig.still ? 1 : 0.8 + ((Math.sin(frame / 5 + phaseOf(fig.x ?? 0, fig.y ?? 0)) + 1) / 2) * 0.3;
  return (
    <Figure {...fig} size={fig.size ?? 60} vw={100} vh={100}>
      <g transform={`translate(50 50) scale(${s}) translate(-50 -50)`}>
        <Shape material="emissive" color={color} d="M50,2 C54,38 62,46 98,50 C62,54 54,62 50,98 C46,62 38,54 2,50 C38,46 46,38 50,2 Z" />
      </g>
    </Figure>
  );
}

/**
 * A propeller plane seen from the side, nose to the right, built as a
 * mechanism: the propeller spins, the flap hinges, and `view` shows it whole
 * (`outside`), with its skin removed to show the cabin, engine and fuel tank
 * (`cutaway`), or taken apart (`exploded`). Box aspect: 320 x 120. Flip to
 * fly left.
 */
export function Airplane({color = '#2D5BFF', view = 'outside', ...fig}: FigureProps & {color?: string; view?: MechanismView}) {
  const frame = useCurrentFrame();
  const prop = fig.still ? 1 : Math.abs(Math.sin(frame * 1.3));
  const dark = shadeOf(color, -0.3);
  const glass = '#8FD3FF';
  return (
    <Figure {...fig} size={fig.size ?? 320} vw={320} vh={120}>
      <Mechanism view={view}>
        <Part name="tail" explode={[-24, -14]}>
          <Shape material="metal" color={dark} d="M40,40 L20,6 L52,6 L84,44 Z" />
        </Part>
        <Part name="skin" casing explode={[0, -16]}>
          <Shape material="metal" color={color} d="M30,52 C30,36 60,32 110,32 L250,34 C282,36 296,48 296,60 C296,72 282,82 250,84 L110,86 C60,86 30,70 30,52 Z" />
          <Shape material="glass" color={glass} d="M220,40 C236,40 250,46 256,56 L214,56 Z" />
        </Part>
        <Part name="cabin" inside explode={[0, 0]}>
          <Shape material="matte" color="#F3F6FB" d="M30,52 C30,36 60,32 110,32 L250,34 C282,36 296,48 296,60 C296,72 282,82 250,84 L110,86 C60,86 30,70 30,52 Z" />
          {[70, 100, 130, 160].map((x) => (
            <Shape key={x} material="fabric" color={dark} rect={{x, y: 50, w: 18, h: 22}} />
          ))}
          <Shape material="glass" color={glass} rect={{x: 210, y: 42, w: 36, h: 18}} />
        </Part>
        <Part name="engine" inside explode={[16, 20]}>
          <Shape material="metal" color="#8F9BB3" rect={{x: 262, y: 44, w: 32, h: 32}} />
          <Shape material="metal" color="#6E7A92" rect={{x: 270, y: 50, w: 16, h: 20}} />
        </Part>
        <Part name="fuel-tank" inside explode={[0, 28]}>
          <Shape material="glossy" color="#E89B10" rect={{x: 176, y: 60, w: 70, h: 18}} />
        </Part>
        <Part name="wing" explode={[0, 24]}>
          <Shape material="metal" color={dark} d="M120,60 L200,60 L150,112 L110,112 Z" />
        </Part>
        <Part name="flap" motion="hinge" pivot={[122, 70]} angle={25} explode={[-18, 34]}>
          <Shape material="metal" color={color} d="M110,100 L150,100 L146,112 L106,112 Z" />
        </Part>
        <Part name="propeller" explode={[22, 0]}>
          <Shape material="metal" color="#6E7A92" ellipse={{cx: 298, cy: 60, rx: 8, ry: 8}} />
          <ellipse cx={304} cy={60} rx={5} ry={40 * prop + 6} fill="#FF4DA6" opacity={0.85} />
        </Part>
      </Mechanism>
    </Figure>
  );
}

// --- speech -----------------------------------------------------------------

/**
 * A speech bubble (or a thought bubble with `thought`) holding a SHORT label
 * or a figure. The tail points down-left towards the speaker; `flip` points it
 * down-right. `x`, `y` are the bubble's centre, `w` x `h` its size in px.
 */
export function Bubble({
  x = 960,
  y = 400,
  w = 360,
  h = 180,
  thought = false,
  flip = false,
  color = '#FFFFFF',
  textColor = INK,
  fontSize = 40,
  opacity = 1,
  scale = 1,
  children,
}: {
  x?: number;
  y?: number;
  w?: number;
  h?: number;
  thought?: boolean;
  flip?: boolean;
  color?: string;
  textColor?: string;
  fontSize?: number;
  opacity?: number;
  scale?: number;
  children?: React.ReactNode;
}) {
  const tx = flip ? w * 0.75 : w * 0.25;
  const dir = flip ? 1 : -1;
  return (
    <div
      style={{
        position: 'absolute',
        left: x - w / 2,
        top: y - h / 2,
        width: w,
        height: h,
        opacity,
        transform: `scale(${scale})`,
        transformOrigin: `${tx}px ${h + 40}px`,
      }}
    >
      <svg width={w} height={h + 60} style={{position: 'absolute', left: 0, top: 0, overflow: 'visible'}}>
        {thought ? (
          <g fill={color}>
            <circle cx={tx + dir * 10} cy={h + 16} r={14} />
            <circle cx={tx + dir * 30} cy={h + 44} r={8} />
          </g>
        ) : (
          <path d={`M${tx - 24},${h - 10} L${tx + dir * 34},${h + 44} L${tx + 24},${h - 10} Z`} fill={color} />
        )}
        <rect x={0} y={0} width={w} height={h} rx={thought ? h / 2 : 36} fill={color} />
      </svg>
      <div
        style={{
          position: 'absolute',
          inset: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          textAlign: 'center',
          padding: '0 28px',
          fontSize,
          fontWeight: 700,
          lineHeight: 1.25,
          color: textColor,
        }}
      >
        {children}
      </div>
    </div>
  );
}
