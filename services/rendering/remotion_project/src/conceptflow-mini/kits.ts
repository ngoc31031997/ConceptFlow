import {getInputProps} from 'remotion';
import kitVersions from './kits.json';

/** Lighting and colour state of one scene; every colour comes from the video's palette. */
export interface Look {
  light: {
    /** Position of the main light source in the frame, 0..1 on both axes. */
    x: number;
    y: number;
    color: string;
    /** 0..1 */
    intensity: number;
  };
  /** Fill light colour reaching the shadow side of every form. */
  ambient: string;
  atmosphere: {color: string; density: number};
  /** Which depth layer stays sharp. */
  focus: 'sky' | 'far' | 'mid' | 'near' | 'figures';
  grade: {warmth: number; contrast: number; saturation: number};
}

export type MaterialName =
  | 'matte'
  | 'glossy'
  | 'metal'
  | 'glass'
  | 'skin'
  | 'fur'
  | 'fabric'
  | 'wood'
  | 'stone'
  | 'organic'
  | 'leaf'
  | 'liquid'
  | 'emissive';

/**
 * A kit is a set of tokens that say HOW forms are painted (rim width, shadow
 * strength, grain...). It never carries the colours of a scene: those arrive
 * through `Look` and the video's palette.
 */
export interface KitTokens {
  outline: number;
  /** Corner radius as a fraction of a shape's shorter side. */
  radius: number;
  rimWidth: number;
  rimStrength: number;
  shadowStrength: number;
  aoStrength: number;
  specular: {glossy: number; metal: number; glass: number};
  texture: {fur: number; fabric: number; wood: number; stone: number; organic: number; leaf: number};
  sss: number;
  glowBlur: number;
  bloom: number;
  haze: {far: number; mid: number};
  /** Blur radius in px per depth layer. */
  dof: {sky: number; far: number; mid: number; near: number};
  grain: number;
  vignette: number;
  /** Colour of fine detail strokes (eyes, mouths, thin lines). */
  ink: string;
  /** Neutral look for previews of a single illustration and the gallery; never used in a video. */
  previewLook: Look;
}

export const KITS: Record<string, KitTokens> = {
  phatsang: {
    outline: 0,
    radius: 0.18,
    rimWidth: 0.035,
    rimStrength: 0.7,
    shadowStrength: 0.35,
    aoStrength: 0.25,
    specular: {glossy: 0.5, metal: 0.8, glass: 0.9},
    texture: {fur: 0.25, fabric: 0.12, wood: 0.2, stone: 0.22, organic: 0.15, leaf: 0.12},
    sss: 0.4,
    glowBlur: 0.08,
    bloom: 0.5,
    haze: {far: 0.35, mid: 0.15},
    dof: {sky: 6, far: 3, mid: 1, near: 10},
    grain: 0.05,
    vignette: 0.3,
    ink: '#141A3A',
    previewLook: {
      light: {x: 0.7, y: 0.2, color: '#FFF4D6', intensity: 0.8},
      ambient: '#8A93B8',
      atmosphere: {color: '#B8C0DC', density: 0.2},
      focus: 'figures',
      grade: {warmth: 0, contrast: 0, saturation: 0},
    },
  },
};

/**
 * Version of each kit (kits.json, also read by the rendering service): a still
 * drawn by one version is not reused for another.
 */
export const KIT_VERSIONS: Record<string, {version: number}> = kitVersions;

/** Keys of the kits this bundle can draw with. */
export function kitKeys(): string[] {
  return Object.keys(KITS);
}

/**
 * Tokens of the kit named by the `kitKey` input prop. A missing or unknown key
 * is an error: the project must be pinned to a visual language before it is
 * drawn or rendered.
 */
export function useKit(): KitTokens {
  const {kitKey} = getInputProps() as {kitKey?: unknown};
  if (typeof kitKey !== 'string' || kitKey === '') {
    throw new Error('Thiếu kitKey: project chưa ghim ngôn ngữ hình');
  }
  const kit = KITS[kitKey];
  if (!kit) {
    throw new Error(`kitKey "${kitKey}" không có trong bộ kit (${kitKeys().join(', ')})`);
  }
  return kit;
}
