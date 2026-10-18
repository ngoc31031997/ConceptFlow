/**
 * Sample scenes of a visual language: the inside of a fig (bright, pink) and
 * outer space (dark), each in landscape and portrait. Two palettes that could
 * not be further apart, drawn with the same kit, to show that the language is
 * a way of painting and not a colour scheme.
 *
 * Registered by language-sample-entry.tsx (gallery) and by the preview host (stills API).
 * Render: node render_gallery.mjs sample <outDir> [--kit <key>]
 */
import React from 'react';
import {Composition, useVideoConfig} from 'remotion';
import {Airplane, Person, Germ} from '../conceptflow-mini/illustration';
import {Shot} from '../conceptflow-mini/film';
import type {Look} from '../conceptflow-mini/kits';
import {InsideBodyBackdrop, SpaceBackdrop} from '../conceptflow-mini/backdrops';
import {Scene} from '../conceptflow-mini/scene';

const SHOT_FRAMES = 90;

const FIG_LOOK: Look = {
  light: {x: 0.7, y: 0.15, color: '#FFD9C2', intensity: 0.8},
  ambient: '#8A4372',
  atmosphere: {color: '#F7B3B8', density: 0.5},
  focus: 'figures',
  grade: {warmth: 0.1, contrast: 0.05, saturation: 0.05},
};

const SPACE_LOOK: Look = {
  light: {x: 0.82, y: 0.24, color: '#FFC857', intensity: 0.9},
  ambient: '#2A2060',
  atmosphere: {color: '#1A1650', density: 0.3},
  focus: 'figures',
  grade: {warmth: -0.05, contrast: 0.1, saturation: 0},
};

function FigInterior() {
  const {width: W, height: H} = useVideoConfig();
  const unit = Math.min(W, H);
  return (
    <Shot look={FIG_LOOK}>
      <Scene
        duration={SHOT_FRAMES}
        backdrop={InsideBodyBackdrop}
        backdropProps={{color: '#D4566F', ground: '#8A4372', light: '#FFD9C2'}}
        camera={{from: {x: W / 2, y: H / 2, zoom: 1.05}, to: {x: W * 0.52, y: H * 0.5, zoom: 1.2}}}
      >
        <Person x={W / 2} y={H * 0.55} size={unit * 0.62} pose="walk" mood="happy" />
        <Germ x={W * 0.2} y={H * 0.35} size={unit * 0.18} color="#F2C14E" mood="happy" />
      </Scene>
    </Shot>
  );
}

function OuterSpace() {
  const {width: W, height: H} = useVideoConfig();
  const unit = Math.min(W, H);
  return (
    <Shot look={SPACE_LOOK}>
      <Scene
        duration={SHOT_FRAMES}
        backdrop={SpaceBackdrop}
        backdropProps={{color: '#0B0A2A', ground: '#4B2A7A', light: '#FFC857'}}
        camera={{from: {x: W / 2, y: H / 2, zoom: 1.2}, to: {x: W / 2, y: H / 2, zoom: 1}}}
      >
        <Airplane x={W / 2} y={H * 0.5} size={unit * 0.7} color="#8FD3FF" />
        <Person x={W * 0.2} y={H * 0.68} size={unit * 0.42} pose="lookUp" mood="surprised" />
      </Scene>
    </Shot>
  );
}

/** The four sample compositions: fig interior and outer space, landscape and portrait. */
export function SampleCompositions() {
  return (
    <>
      <Composition id="sample-fig-wide" component={FigInterior} width={1920} height={1080} fps={30} durationInFrames={SHOT_FRAMES} />
      <Composition id="sample-fig-tall" component={FigInterior} width={1080} height={1920} fps={30} durationInFrames={SHOT_FRAMES} />
      <Composition id="sample-space-wide" component={OuterSpace} width={1920} height={1080} fps={30} durationInFrames={SHOT_FRAMES} />
      <Composition id="sample-space-tall" component={OuterSpace} width={1080} height={1920} fps={30} durationInFrames={SHOT_FRAMES} />
    </>
  );
}

/** Ids of the sample compositions, for the stills API and the gallery. */
export const SAMPLE_IDS = ['sample-fig-wide', 'sample-fig-tall', 'sample-space-wide', 'sample-space-tall'] as const;
