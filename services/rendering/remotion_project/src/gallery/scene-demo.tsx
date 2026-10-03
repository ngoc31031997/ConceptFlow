/**
 * Gallery sample of the scene kit: four shots, each a place filling the frame
 * (a built-in backdrop in the Scene's depth layers), a camera move with
 * parallax, light and a keyword; then two close-ups timed by narration lines
 * (a person changing pose, a hand reaching in). The same shots are registered landscape
 * (`scene-wide`) and portrait (`scene-tall`): the backdrops and the shots
 * place everything from the frame's own size.
 *
 * Render: node render_gallery.mjs scene <outDir>
 */
import React from 'react';
import {registerRoot, Composition, AbsoluteFill, Sequence, interpolate, useCurrentFrame, useVideoConfig} from 'remotion';
import {Stage} from '../conceptflow-mini/primitives';
import {Backdrop, Book, Candy, Germ, Lightbulb, Person, Table, Tooth} from '../conceptflow-mini/illustration';
import {Glow, KeywordText, LightRays, Scene} from '../conceptflow-mini/scene';
import {InsideBodyBackdrop, MeadowBackdrop, RoomBackdrop, SpaceBackdrop} from '../conceptflow-mini/backdrops';
import {ReachingHand} from '../conceptflow-mini/rig';
import {evenLines} from '../conceptflow-mini/segments';

const SHOT = 90;
const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

function Meadow({duration}: {duration: number}) {
  const frame = useCurrentFrame();
  const {width: W, height: H} = useVideoConfig();
  const walk = interpolate(frame, [0, duration], [W * 0.35, W * 0.5], clamp);
  return (
    <Scene duration={duration} backdrop={MeadowBackdrop} camera={{from: {x: W / 2, y: H / 2, zoom: 1}, to: {x: W * 0.52, y: H * 0.55, zoom: 1.15}}}>
      <Person x={walk} y={H * 0.56} size={H * 0.5} pose="walk" mood="happy" />
    </Scene>
  );
}

function Room({duration}: {duration: number}) {
  const {width: W, height: H} = useVideoConfig();
  return (
    <Scene duration={duration} backdrop={RoomBackdrop} camera={{from: {x: W * 0.45, y: H / 2, zoom: 1.05}, to: {x: W * 0.55, y: H * 0.55, zoom: 1.2}}}>
      <Table x={W * 0.5} y={H * 0.74} size={Math.min(W, H) * 0.55} />
      <Person x={W * 0.38} y={H * 0.6} size={H * 0.45} pose="sit" age="child" mood="happy" />
      <Glow x={W * 0.56} y={H * 0.6} r={Math.min(W, H) * 0.14} color="#FFD23F" pulse />
      <Candy x={W * 0.56} y={H * 0.6} size={Math.min(W, H) * 0.16} />
    </Scene>
  );
}

function InsideBody({duration}: {duration: number}) {
  const frame = useCurrentFrame();
  const {width: W, height: H} = useVideoConfig();
  const close = interpolate(frame, [0, duration], [0, 1], clamp);
  return (
    <Scene duration={duration} backdrop={InsideBodyBackdrop} camera={{from: {x: W / 2, y: H / 2, zoom: 1.1}, to: {x: W / 2, y: H / 2, zoom: 1.3}}}>
      <Tooth x={W / 2} y={H * 0.5} size={Math.min(W, H) * 0.45} decay={close * 0.6} />
      <Germ x={W / 2 - Math.min(W, H) * (0.45 - close * 0.15)} y={H * 0.38} size={Math.min(W, H) * 0.16} color="#8BC34A" />
      <Germ x={W / 2 + Math.min(W, H) * (0.45 - close * 0.15)} y={H * 0.62} size={Math.min(W, H) * 0.14} color="#8BC34A" variant={2} />
    </Scene>
  );
}

function Space({duration}: {duration: number}) {
  const {width: W, height: H} = useVideoConfig();
  return (
    <Scene duration={duration} backdrop={SpaceBackdrop} camera={{from: {x: W / 2, y: H / 2, zoom: 1.2}, to: {x: W / 2, y: H / 2, zoom: 1}}}>
      <LightRays x={W * 0.82} y={H * 0.24} color="#FFC857" opacity={0.08} />
      <KeywordText x={W / 2} y={H / 2} width={Math.min(W, H) * 0.8} duration={duration} start={0.2} background="#7B3FC4">
        VŨ TRỤ
      </KeywordText>
    </Scene>
  );
}

/** A close-up person read in three lines: thinks, then shrugs, then holds up a book. */
function Talk({duration}: {duration: number}) {
  const frame = useCurrentFrame();
  const {width: W, height: H} = useVideoConfig();
  const lines = evenLines(duration, 3);
  const toShrug = interpolate(frame, [lines[1], lines[1] + 12], [0, 1], clamp);
  const toHold = interpolate(frame, [lines[2], lines[2] + 12], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Backdrop color="#2EC4F0" />
      <Person
        x={W / 2}
        y={H * 0.55}
        size={Math.min(W, H) * 0.9}
        framing="bust"
        pose={frame < lines[2] ? 'think' : 'shrug'}
        toPose={frame < lines[2] ? 'shrug' : 'hold'}
        poseT={frame < lines[2] ? toShrug : toHold}
        mood={frame < lines[1] ? 'worried' : 'happy'}
      >
        <Book x={0} y={-Math.min(W, H) * 0.05} size={Math.min(W, H) * 0.2} />
      </Person>
    </AbsoluteFill>
  );
}

/** A hand reaches in from below, pulls a pencil out, then counts three fingers. */
function Reach({duration}: {duration: number}) {
  const frame = useCurrentFrame();
  const {width: W, height: H} = useVideoConfig();
  const lines = evenLines(duration, 2);
  const reach = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  const pull = interpolate(frame, [duration * 0.25, lines[1]], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Backdrop color="#FFD84D" />
      <ReachingHand
        edge="bottom"
        target={{x: W * 0.5, y: H * (0.5 - pull * 0.12)}}
        reach={reach}
        size={Math.min(W, H) * 0.32}
        pose={frame < lines[1] ? 'pinch' : 'count'}
        count={3}
        sleeve="#2D5BFF"
      >
        {frame < lines[1] ? <Lightbulb x={0} y={-Math.min(W, H) * 0.08} size={Math.min(W, H) * 0.12} /> : null}
      </ReachingHand>
    </AbsoluteFill>
  );
}

const SHOTS = [Meadow, Room, InsideBody, Space, Talk, Reach];

function Demo() {
  return (
    <Stage>
      {SHOTS.map((Shot, i) => (
        <Sequence key={i} from={i * SHOT} durationInFrames={SHOT}>
          <AbsoluteFill>
            <Shot duration={SHOT} />
          </AbsoluteFill>
        </Sequence>
      ))}
    </Stage>
  );
}

registerRoot(() => (
  <>
    <Composition id="scene-wide" component={Demo} width={1920} height={1080} fps={30} durationInFrames={SHOTS.length * SHOT} />
    <Composition id="scene-tall" component={Demo} width={1080} height={1920} fps={30} durationInFrames={SHOTS.length * SHOT} />
  </>
));
