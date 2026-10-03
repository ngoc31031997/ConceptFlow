/**
 * conceptflow-mini/rig — body parts that act in a close-up: a hand that
 * reaches into the frame to take, pull, point at or count something.
 *
 * Coordinates are the frame's own (0..width, 0..height of the composition),
 * like the scene kit, so the same call fits a landscape and a portrait video.
 * The drawing follows the illustration style: solid shapes without outlines,
 * one soft shade on the right, round ends.
 */
import React from 'react';
import {Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {HeldAt, phaseOf, shadeOf, type Pt} from './illustration';

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
  const bob = still ? 0 : Math.sin((frame + phaseOf(goal[0], goal[1])) / 14) * size * 0.012;
  const palm: Pt = [start[0] + (goal[0] - start[0]) * t, start[1] + (goal[1] - start[1]) * t + bob];

  const len = Math.hypot(goal[0] - start[0], goal[1] - start[1]) || 1;
  const dir: Pt = [(goal[0] - start[0]) / len, (goal[1] - start[1]) / len];
  const wrist: Pt = [palm[0] - dir[0] * size * 0.32, palm[1] - dir[1] * size * 0.32];
  const cuff: Pt = [wrist[0] - dir[0] * size * 0.28, wrist[1] - dir[1] * size * 0.28];
  // Fingers point along the arm: local "up" (-y) is `dir`.
  const angle = (Math.atan2(dir[1], dir[0]) * 180) / Math.PI + 90;
  const m = flip ? -1 : 1;
  const s = size;
  const shade = shadeOf(skin, -0.14);
  const {fingers, thumb} = raised(pose, count);

  // Raised fingers fan out a little so each reads on its own when counted.
  const finger = (i: number, up: boolean) => {
    const x = (-0.195 + i * 0.13) * s * m;
    const top = -0.2 * s;
    const length = up ? (i === 3 ? 0.3 : i === 0 ? 0.38 : 0.42) * s : 0.1 * s;
    const fan = up ? (i - 1.5) * 0.06 * s * m : 0;
    const bend = pose === 'pinch' && i === 0 ? 0.12 * s * m : fan;
    return (
      <path
        key={i}
        d={`M${x},${top} Q${x + bend * 0.3},${top - length * 0.6} ${x + bend},${top - length}`}
        stroke={skin}
        strokeWidth={0.095 * s}
        strokeLinecap="round"
        fill="none"
      />
    );
  };
  const thumbPath = thumb || pose === 'open'
    ? `M${-0.22 * s * m},${0.02 * s} Q${-0.4 * s * m},${-0.08 * s} ${-0.42 * s * m},${-0.22 * s}`
    : pose === 'pinch'
      ? `M${-0.22 * s * m},${0.02 * s} Q${-0.26 * s * m},${-0.2 * s} ${-0.08 * s * m},${-0.3 * s}`
      : `M${-0.22 * s * m},${0.04 * s} Q${-0.12 * s * m},${-0.08 * s} ${0.02 * s * m},${-0.1 * s}`;
  const held: Pt = [palm[0] + dir[0] * s * 0.12, palm[1] + dir[1] * s * 0.12];

  return (
    <>
      <svg
        width={width}
        height={height}
        viewBox={`0 0 ${width} ${height}`}
        style={{position: 'absolute', left: 0, top: 0, overflow: 'visible', opacity}}
      >
        <path
          d={`M${start[0] - dir[0] * s},${start[1] - dir[1] * s} L${cuff[0]},${cuff[1]}`}
          stroke={sleeve}
          strokeWidth={0.46 * s}
          strokeLinecap="round"
          fill="none"
        />
        <path d={`M${cuff[0]},${cuff[1]} L${wrist[0]},${wrist[1]}`} stroke={skin} strokeWidth={0.3 * s} strokeLinecap="round" fill="none" />
        <g transform={`translate(${palm[0]} ${palm[1]}) rotate(${angle})`}>
          <rect x={-0.26 * s} y={-0.24 * s} width={0.52 * s} height={0.46 * s} rx={0.18 * s} fill={skin} />
          <rect x={0.1 * s * m - (m < 0 ? 0.12 * s : 0)} y={-0.08 * s} width={0.12 * s} height={0.24 * s} rx={0.06 * s} fill={shade} opacity={0.35} />
          {fingers.map((up, i) => finger(i, up))}
          <path d={thumbPath} stroke={skin} strokeWidth={0.12 * s} strokeLinecap="round" fill="none" />
        </g>
      </svg>
      <HeldAt at={held}>{children}</HeldAt>
    </>
  );
}
