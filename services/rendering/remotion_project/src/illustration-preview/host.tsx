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

export const PREVIEW_SIZE = 720;

type HostProps = {
  /** CommonJS output of esbuild for the asset's TSX; empty for a built-in. */
  js?: string;
  /** Exported component name to render. */
  name?: string;
  /** Extra props for the figure (mood, decay...). */
  props?: Record<string, unknown>;
  background?: string;
};

const MODULES: Record<string, unknown> = {
  react: ReactNS,
  'react/jsx-runtime': JSXRuntime,
  remotion: Remotion,
  './conceptflow-mini/illustration': Kit,
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
  const {js = '', name = '', props = {}, background = '#FFF4D6'} = getInputProps() as HostProps;
  const exports = js ? load(js) : (Kit as unknown as Record<string, unknown>);
  const Figure = exports[name] as React.ComponentType<Record<string, unknown>> | undefined;
  if (typeof Figure !== 'function') {
    throw new Error(`component '${name}' is not exported`);
  }
  return (
    <AbsoluteFill style={{backgroundColor: background}}>
      <Figure x={PREVIEW_SIZE / 2} y={PREVIEW_SIZE / 2} size={PREVIEW_SIZE * 0.8} {...props} />
    </AbsoluteFill>
  );
}

registerRoot(() => (
  <Composition id="asset" component={Preview} width={PREVIEW_SIZE} height={PREVIEW_SIZE} fps={30} durationInFrames={60} />
));
