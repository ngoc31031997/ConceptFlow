/**
 * conceptflow-mini/rig — body parts that act in a close-up: a hand that
 * reaches into the frame to take, pull, point at or count something.
 *
 * Coordinates are the frame's own (0..width, 0..height of the composition),
 * like the scene kit, so the same call fits a landscape and a portrait video.
 * The arm is a two-bone limb solved with IK from a shoulder outside the
 * frame, so the elbow bends naturally as the hand reaches in. Every form is a
 * `Shape` lit by the scene's light.
 */
import React from 'react';
import {Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {HeldAt, phaseOf, type Pt} from './illustration';
import {FigureFrameProvider, Shape} from './material';
import {elbowPosition} from './skeleton';
export * from './skeleton';

export type HandPose = 'open' | 'point' | 'grip' | 'pinch' | 'count';
export type HandEdge = 'bottom' | 'left' | 'right' | 'top';

export interface ReachingHandProps {
  /** The frame edge the arm comes in from. */
  edge?: HandEdge;
  /** Where the palm arrives, in frame px. Defaults to the frame centre. */
  target?: {x: number; y: number};
  /** 0 = the hand is still outside the frame, 1 = the palm is at `target`. Animate it to reach in or pull out. */
  reach?: number;
  pose?: HandPose;
  /** Fingers raised in the `count` pose, 1–5 (index first, the thumb is the fifth). */
  count?: number;
  /** Length of the hand from wrist to fingertips, in px. */
  size?: number;
  /** Mirror the hand (thumb on the other side). */
  flip?: boolean;
  skin?: string;
  sleeve?: string;
  opacity?: number;
  /** Turn off the small idle motion. */
  still?: boolean;
  /** What the hand holds; its x/y are measured from the middle of the palm. */
  children?: React.ReactNode;
}

const FINGERS = 4;

function entryPoint(edge: HandEdge, target: Pt, size: number, width: number, height: number): Pt {
  switch (edge) {
    case 'top':
      return [target[0] - size * 0.8, -size * 1.2];
    case 'left':
      return [-size * 1.2, target[1] + size * 0.6];
    case 'right':
      return [width + size * 1.2, target[1] + size * 0.6];
    default:
      return [target[0] + size * 0.8, height + size * 1.2];
  }
}

/** Which of the four fingers (index → little) and the thumb are raised in a pose. */
function raised(pose: HandPose, count: number): {fingers: boolean[]; thumb: boolean} {
  const n = Math.min(5, Math.max(1, Math.round(count)));
  switch (pose) {
    case 'open':
      return {fingers: [true, true, true, true], thumb: true};
    case 'point':
      return {fingers: [true, false, false, false], thumb: false};
    case 'count':
      return {fingers: Array.from({length: FINGERS}, (_, i) => i < Math.min(n, FINGERS)), thumb: n === 5};
    default:
      return {fingers: [false, false, false, false], thumb: false};
  }
}

/**
 * A forearm and hand reaching in from a frame edge to `target`. Poses: open
 * (palm and spread fingers), point (index finger), grip (a fist, holding
 * `children`), pinch (thumb and index meeting, holding something small), count
 * (`count` fingers raised). Animate `reach` to bring the hand in and take it
 * out, `pose`/`count` to change what it does.
 */
export function ReachingHand({
  edge = 'bottom',
  target,
  reach = 1,
  pose = 'open',
  count = 1,
  size = 260,
  flip = false,
  skin = '#F9C4B4',
  sleeve = '#2D5BFF',
  opacity = 1,
  still = false,
  children,
}: ReachingHandProps) {
  const frame = useCurrentFrame();
  const {width, height} = useVideoConfig();
  const goal: Pt = [target?.x ?? width / 2, target?.y ?? height / 2];
  const start = entryPoint(edge, goal, size, width, height);
  const t = Easing.out(Easing.cubic)(Math.min(1, Math.max(0, reach)));
  const bob = still ? 0 : Math.sin((frame + phaseOf(goal[0], goal[1]))  / 14) * size * 0.012;
  const palm: Pt = [start[0] + (goal[0] - start[0]) * t, start[1] + (goal[1] - start[1]) * t + bob];

  const len = Math.hypot(goal[0] - start[0], goal[1] - start[1]) || 1;
  const dir: Pt = [(goal[0] - start[0]) / len, (goal[1] - start[1]) / len];
  const wrist: Pt = [palm[0] - dir[0] * size * 0.32, palm[1] - dir[1] * size * 0.32];
  const cuff: Pt = [wrist[0] - dir[0] * size * 0.28, wrist[1] - dir[1] * size * 0.28];
  // The shoulder sits behind the entry point; the limb is long enough to reach `goal` with a soft bend.
  const shoulder: Pt = [start[0] - dir[0] * size, start[1] - dir[1] * size];
  const full = Math.hypot(goal[0] - dir[0] * size * 0.32 - shoulder[0], goal[1] - dir[1] * size * 0.32 - shoulder[1]);
  const limb = Math.max(size * 0.6, full * 0.62);
  const elbow = elbowPosition(shoulder, wrist, limb, limb, edge === 'left' || edge === 'top' ? -1 : 1);
  // Fingers point along the arm: local "up" (-y) is `dir`.
  const angle = (Math.atan2(dir[1], dir[0]) * 180) / Math.PI + 90;
  const m = flip ? -1 : 1;
  const s = size;
  const {fingers, thumb} = raised(pose, count);

  // Raised fingers fan out a little so each reads on its own when counted.
  const finger = (i: number, up: boolean) => {
    const x = (-0.195 + i * 0.13) * s * m;
    const top = -0.2 * s;
    const length = up ? (i === 3 ? 0.3 : i === 0 ? 0.38 : 0.42) * s : 0.1 * s;
    const fan = up ? (i - 1.5) * 0.06 * s * m : 0;
    const bend = pose === 'pinch' && i === 0 ? 0.12 * s * m : fan;
    return <Shape key={i} material="skin" color={skin} unit={s} capsule={{x1: x, y1: top, x2: x + bend, y2: top - length, r: 0.0475 * s}} />;
  };
  const thumbEnd: Pt =
    thumb || pose === 'open'
      ? [-0.42 * s * m, -0.22 * s]
      : pose === 'pinch'
        ? [-0.08 * s * m, -0.3 * s]
        : [0.02 * s * m, -0.1 * s];
  const held: Pt = [palm[0] + dir[0] * s * 0.12, palm[1] + dir[1] * s * 0.12];

  return (
    <>
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        style={{position: 'absolute', left: 0, top: 0, overflow: 'visible', opacity}}
      >
        <FigureFrameProvider value={{x: shoulder[0], y: shoulder[1], rotate: 0, flip: false, vw: s, vh: s}}>
          <Shape material="fabric" color={sleeve} unit={s} capsule={{x1: shoulder[0], y1: shoulder[1], x2: elbow[0], y2: elbow[1], r: 0.23 * s}} />
          <Shape material="fabric" color={sleeve} unit={s} capsule={{x1: elbow[0], y1: elbow[1], x2: cuff[0], y2: cuff[1], r: 0.23 * s}} />
          <Shape material="skin" color={skin} unit={s} capsule={{x1: cuff[0], y1: cuff[1], x2: wrist[0], y2: wrist[1], r: 0.15 * s}} />
        </FigureFrameProvider>
        <FigureFrameProvider value={{x: palm[0], y: palm[1], rotate: angle, flip: false, vw: s, vh: s}}>
          <g transform={`translate(${palm[0]} ${palm[1]}) rotate(${angle})`}>
            <Shape material="skin" color={skin} unit={s} rect={{x: -0.26 * s, y: -0.24 * s, w: 0.52 * s, h: 0.46 * s}} />
            {fingers.map((up, i) => finger(i, up))}
            <Shape
              material="skin"
              color={skin}
              unit={s}
              capsule={{x1: -0.22 * s * m, y1: 0.02 * s, x2: thumbEnd[0], y2: thumbEnd[1], r: 0.06 * s}}
            />
          </g>
        </FigureFrameProvider>
      </svg>
      <HeldAt at={held}>{children}</HeldAt>
    </>
  );
}
