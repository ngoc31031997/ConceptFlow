import {getInputProps} from 'remotion';

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

const FOCUS_LAYERS = ['sky', 'far', 'mid', 'near', 'figures'];
const HEX_COLOR = /^#[0-9A-Fa-f]{6}$/;

type LeafKind = 'unit' | 'signed' | 'px' | 'color' | 'focus';
interface Shape {
  [key: string]: LeafKind | Shape;
}

/** Shape of a kit: leaves name the kind of value, branches are nested objects. */
const KIT_SHAPE: Shape = {
  outline: 'unit', radius: 'unit', rimWidth: 'unit', rimStrength: 'unit', shadowStrength: 'unit', aoStrength: 'unit',
  specular: {glossy: 'unit', metal: 'unit', glass: 'unit'},
  texture: {fur: 'unit', fabric: 'unit', wood: 'unit', stone: 'unit', organic: 'unit', leaf: 'unit'},
  sss: 'unit', glowBlur: 'unit', bloom: 'unit',
  haze: {far: 'unit', mid: 'unit'},
  dof: {sky: 'px', far: 'px', mid: 'px', near: 'px'},
  grain: 'unit', vignette: 'unit', ink: 'color',
  previewLook: {
    light: {x: 'unit', y: 'unit', color: 'color', intensity: 'unit'},
    ambient: 'color',
    atmosphere: {color: 'color', density: 'unit'},
    focus: 'focus',
    grade: {warmth: 'signed', contrast: 'signed', saturation: 'signed'},
  },
};

function checkLeaf(path: string, kind: LeafKind, value: unknown): void {
  if (kind === 'color') {
    if (typeof value !== 'string' || !HEX_COLOR.test(value)) throw new Error(`kit.${path} phải là màu #RRGGBB, nhận ${JSON.stringify(value)}`);
  } else if (kind === 'focus') {
    if (typeof value !== 'string' || !FOCUS_LAYERS.includes(value)) {
      throw new Error(`kit.${path} phải là một trong ${FOCUS_LAYERS.join(', ')}, nhận ${JSON.stringify(value)}`);
    }
  } else if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`kit.${path} phải là số, nhận ${JSON.stringify(value)}`);
  } else if (kind === 'unit' && (value < 0 || value > 1)) {
    throw new Error(`kit.${path} phải trong khoảng 0..1, nhận ${value}`);
  } else if (kind === 'signed' && (value < -1 || value > 1)) {
    throw new Error(`kit.${path} phải trong khoảng -1..1, nhận ${value}`);
  } else if (kind === 'px' && value < 0) {
    throw new Error(`kit.${path} (px) phải >= 0, nhận ${value}`);
  }
}

function checkShape(path: string, shape: Shape, value: unknown): void {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`kit${path ? '.' + path : ''} phải là object`);
  }
  const record = value as Record<string, unknown>;
  for (const [key, sub] of Object.entries(shape)) {
    const child = path ? `${path}.${key}` : key;
    if (!(key in record)) throw new Error(`kit thiếu trường ${child}`);
    if (typeof sub === 'string') checkLeaf(child, sub, record[key]);
    else checkShape(child, sub, record[key]);
  }
  const extra = Object.keys(record).filter((k) => !(k in shape));
  if (extra.length) throw new Error(`kit${path ? '.' + path : ''} có trường lạ: ${extra.join(', ')}`);
}

/** The kit if it has every field with a value in range; otherwise an error naming the field. */
export function parseKit(value: unknown): KitTokens {
  if (value === undefined || value === null) throw new Error('Thiếu kit: lượt vẽ không nhận được input prop `kit`');
  checkShape('', KIT_SHAPE, value);
  return value as KitTokens;
}

/**
 * Tokens of the kit passed as the `kit` input prop. A missing or invalid kit is
 * an error: nothing is drawn without the drawing tokens of the project's
 * reference style.
 */
export function currentKit(): KitTokens {
  const raw = (getInputProps() as {kit?: unknown}).kit;
  if (typeof raw === 'object' && raw !== null) {
    const known = checked.get(raw);
    if (known) return known;
  }
  const kit = parseKit(raw);
  checked.set(raw as object, kit);
  return kit;
}

/** Kits already validated, so components that ask for the kit on every render do not re-check it. */
const checked = new WeakMap<object, KitTokens>();
