import React from 'react';
import {registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';
import {Stage, SAFE_MARGIN, WIDTH, HEIGHT} from './conceptflow-mini/primitives';
import {LottieClip} from './conceptflow-mini/lottie';
import {Backdrop, Panel, Person, Tooth, Germ, OpenMouth, Toothbrush, Toothpaste, Drop, Shield, Heart, Candy, Lollipop, Soda, Donut, Apple, Clock, Table, Chair, Window, Plant, House, Tree, Sun, Cloud, Lightbulb, Coin, Book, Phone, Magnifier, Mark, Sparkle, Airplane, Bubble, Figure, Face, GroundShadow, useBlink, phaseOf, shadeOf, useSvgId, INK, SHADE, BLUSH, WHITE} from './conceptflow-mini/illustration';
import type {FigureProps, Mood, PersonPose} from './conceptflow-mini/illustration';

const PALETTE = {
  accent: '#F5B841', // thứ đang được chú ý
  muted: '#4A5670', // nền phụ
  ink: '#F2F7FF', // nhãn và chữ
};

const LAYOUT = {
  hero: {x: 960, y: 560, size: 460},
  tooth: {x: 620, y: 560, size: 420},
  label: {x: 1180, y: 520, w: 560},
  cat: {x: 1500, y: 600, size: 420},
};

const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

export const narrations: string[] = [
  "Bạn có biết vì sao răng bị sâu không?",
];

type ShotProps = {duration: number};

// Shot 1.1 — MÁY: đứng yên | HÌNH: mặt trời bật ra ở góc phải
function Shot1_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame, fps, config: {damping: 6}});
  return (
    <AbsoluteFill>
      <Person x={760} y={600} size={520} pose="point" mood="happy" />
      <Sun x={1640} y={300} size={300} scale={pop} />
    </AbsoluteFill>
  );
}

const SHOTS: React.FC<ShotProps>[] = [Shot1_1];

function CreatorComposition({segments = []}: {segments?: {startFrame: number; durationInFrames: number}[]}) {
  return (
    <Stage>
      <Segments segments={segments}>
        {(index, segment) => {
          const Shot = SHOTS[index];
          return Shot ? <Shot duration={segment.durationInFrames} /> : null;
        }}
      </Segments>
    </Stage>
  );
}

registerRoot(() => (
  <Composition
    id="creator"
    component={CreatorComposition}
    width={1920}
    height={1080}
    fps={30}
    durationInFrames={150}
    calculateMetadata={calculateMetadataFromSegments}
  />
));
