/**
 * Preview host for one illustration of the library.
 *
 * Bundled ONCE by illustration_preview.mjs; each preview hands the figure in as
 * input props — already-compiled JavaScript for a library drawing, or the name
 * of a built-in kit component — so a preview costs a render, not a webpack build.
 */
import React from 'react';
import * as ReactNS from 'react';
import * as JSXRuntime from 'react/jsx-runtime';
import * as Remotion from 'remotion';
import {registerRoot, Composition, AbsoluteFill, getInputProps} from 'remotion';
import * as Kit from '../conceptflow-mini/illustration';
import * as SceneKit from '../conceptflow-mini/scene';
import * as Backdrops from '../conceptflow-mini/backdrops';
import * as FilmKit from '../conceptflow-mini/film';
import * as MaterialKit from '../conceptflow-mini/material';
import * as SkeletonKit from '../conceptflow-mini/skeleton';
import * as ProceduralKit from '../conceptflow-mini/procedural';
import * as ShaderKit from '../conceptflow-mini/shader';
import * as RigKit from '../conceptflow-mini/rig';
import * as KitsModule from '../conceptflow-mini/kits';
import {SampleCompositions} from '../gallery/language-sample';
import {LookProvider} from '../conceptflow-mini/film';
import {useKit} from '../conceptflow-mini/kits';

export const PREVIEW_SIZE = 720;
/** A backdrop is previewed as a whole landscape frame, scaled down. */
const BACKDROP_PREVIEW = {width: 1280, height: 720};
const PREVIEW_FRAMES = 60;

type HostProps = {
  /** CommonJS output of esbuild for the asset's TSX; empty for a built-in. */
  js?: string;
  /** Exported component name to render. */
  name?: string;
  /** Extra props for the figure (mood, decay...). */
  props?: Record<string, unknown>;
  background?: string;
  /** "figure" (default) or "backdrop": a backdrop is drawn in a Scene's layers. */
  kind?: 'figure' | 'backdrop';
};

const MODULES: Record<string, unknown> = {
  react: ReactNS,
  'react/jsx-runtime': JSXRuntime,
  remotion: Remotion,
  './conceptflow-mini/illustration': Kit,
  './conceptflow-mini/scene': SceneKit,
  './conceptflow-mini/backdrops': Backdrops,
  './conceptflow-mini/film': FilmKit,
  './conceptflow-mini/material': MaterialKit,
  './conceptflow-mini/skeleton': SkeletonKit,
  './conceptflow-mini/procedural': ProceduralKit,
  './conceptflow-mini/shader': ShaderKit,
  './conceptflow-mini/rig': RigKit,
  './conceptflow-mini/kits': KitsModule,
};

function load(js: string): Record<string, unknown> {
  const module = {exports: {} as Record<string, unknown>};
  const require = (id: string) => {
    if (!(id in MODULES)) throw new Error(`import '${id}' is not allowed in a library drawing`);
    return MODULES[id];
  };
  // eslint-disable-next-line no-new-func
  new Function('require', 'module', 'exports', js)(require, module, module.exports);
  return module.exports;
}

function Preview() {
  const {js = '', name = '', props = {}, background = '#FFF4D6', kind = 'figure'} = getInputProps() as HostProps;
  const builtin = {...(Kit as unknown as Record<string, unknown>), ...(Backdrops as unknown as Record<string, unknown>)};
  const kit = useKit();
  const exports = js ? load(js) : builtin;
  const Figure = exports[name] as React.ComponentType<Record<string, unknown>> | undefined;
  if (typeof Figure !== 'function') {
    throw new Error(`component '${name}' is not exported`);
  }
  if (kind === 'backdrop') {
    // A loaded drawing's props are unknown to the type system; the style check
    // already required it to take `layer`.
    const backdrop = Figure as React.ComponentType<SceneKit.BackdropInput>;
    const colors = props as Omit<SceneKit.BackdropInput, 'layer'>;
    return (
      <LookProvider look={kit.previewLook}>
        <SceneKit.Scene duration={PREVIEW_FRAMES} backdrop={backdrop} backdropProps={colors} />
      </LookProvider>
    );
  }
  return (
    <LookProvider look={kit.previewLook}>
      <AbsoluteFill style={{backgroundColor: background}}>
        <Figure x={PREVIEW_SIZE / 2} y={PREVIEW_SIZE / 2} size={PREVIEW_SIZE * 0.8} {...props} />
      </AbsoluteFill>
    </LookProvider>
  );
}

registerRoot(() => (
  <>
  <SampleCompositions />
  <Composition
    id="asset"
    component={Preview}
    width={PREVIEW_SIZE}
    height={PREVIEW_SIZE}
    fps={30}
    durationInFrames={PREVIEW_FRAMES}
    calculateMetadata={({props}) => {
      const {kind} = props as HostProps;
      return kind === 'backdrop' ? BACKDROP_PREVIEW : {width: PREVIEW_SIZE, height: PREVIEW_SIZE};
    }}
  />
  </>
));
