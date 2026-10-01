/**
 * conceptflow-mini/primitives — the few things a Remotion composition must
 * NOT decide for itself, because they belong to the channel or to the
 * Creator's settings rather than to one video's storyboard:
 *
 *   - the frame: 1920x1080 for a long video, 1080x1920 for a vertical short,
 *     each with the safe area meaningful content stays inside.
 *   - the background: fixed, the same #080E1C the Manim theme paints
 *     (conceptflow/theme.py BRAND_BG), so both engines look like one channel.
 *     The Visual Director picks every other colour against this background.
 *   - the font: chosen per project in the settings step and handed in as the
 *     `videoFont` input prop (remotion_renderer.py). Only fonts installed in
 *     the rendering image (../../Dockerfile) are accepted.
 *
 * Everything else — colours, shapes, layout, motion — is the storyboard's,
 * drawn by the generated script in plain JSX/SVG/CSS.
 */
import React, {useState} from 'react';
import {AbsoluteFill, continueRender, delayRender, getInputProps, useVideoConfig} from 'remotion';

export const BACKGROUND = '#080E1C';
export const WIDTH = 1920;
export const HEIGHT = 1080;
/** Minimum distance between anything meaningful and the frame edge (landscape). */
export const SAFE_MARGIN = 96;

/** The rectangle anything meaningful stays inside, in px from the top-left corner. */
export type SafeArea = {left: number; top: number; right: number; bottom: number};

/** One of the two frames a video is built in. */
export type FrameBox = {width: number; height: number; safe: SafeArea};

/**
 * The landscape long-form frame and the portrait Shorts frame. The portrait
 * safe area keeps clear of the Shorts overlay: title and channel at the
 * bottom, the like/comment column on the right. authoring-service
 * (domain/frame.go), llm-service (app/frame.py) and rendering
 * (domain/frame.py) carry the same numbers; their tests compare against this
 * file.
 */
export const FRAMES: {landscape: FrameBox; portrait: FrameBox} = {
  landscape: {width: 1920, height: 1080, safe: {left: 96, top: 96, right: 1824, bottom: 984}},
  portrait: {width: 1080, height: 1920, safe: {left: 72, top: 200, right: 940, bottom: 1560}},
};

/** The frame box for a composition of `width` × `height`. */
export function frameFor(width: number, height: number): FrameBox {
  return height > width ? FRAMES.portrait : FRAMES.landscape;
}

/** The frame box of the composition being rendered. */
export function useFrameBox(): FrameBox {
  const {width, height} = useVideoConfig();
  return frameFor(width, height);
}

const INSTALLED_FONTS = ['Be Vietnam Pro', 'Montserrat', 'Cormorant Garamond'];
const DEFAULT_FONT = 'Be Vietnam Pro';
const DEFAULT_INK = '#F2F7FF';

function chosenFont(): string {
  const {videoFont} = getInputProps() as {videoFont?: unknown};
  return typeof videoFont === 'string' && INSTALLED_FONTS.includes(videoFont)
    ? videoFont
    : DEFAULT_FONT;
}

/** CSS font-family for the project's chosen font, with a safe fallback. */
export function useVideoFont(): string {
  return `'${chosenFont()}', sans-serif`;
}

// Hold the first frame until the system-installed font is actually loaded,
// so frames aren't captured with the sans-serif fallback. Module-level so it
// runs once per bundle rather than once per component instance.
let fontReady: Promise<void> | null = null;
function useFontLoaded() {
  useState(() => {
    const font = chosenFont();
    fontReady ??= Promise.all(
      [400, 700].map((weight) => document.fonts.load(`${weight} 36px '${font}'`)),
    ).then((faces) => {
      if (faces.some((f) => f.length === 0)) {
        console.warn(`conceptflow-mini: font '${font}' not found, falling back to sans-serif`);
      }
    });
    const handle = delayRender(`Loading font ${font}`);
    fontReady.then(
      () => continueRender(handle),
      () => continueRender(handle),
    );
    return null;
  });
}

/**
 * The root of every composition: paints the fixed background and sets the
 * project's font as the inherited default, so no element has to repeat it.
 */
export function Stage({children}: {children: React.ReactNode}) {
  useFontLoaded();
  const fontFamily = useVideoFont();
  return (
    <AbsoluteFill style={{backgroundColor: BACKGROUND, fontFamily, color: DEFAULT_INK}}>
      {children}
    </AbsoluteFill>
  );
}

function CenteredText({
  children,
  fontSize,
  fontWeight,
  paddingX,
  position,
  color,
}: {
  children: React.ReactNode;
  fontSize: number;
  fontWeight: number;
  paddingX: number;
  position: 'center' | 'bottom';
  color?: string;
}) {
  useFontLoaded();
  const fontFamily = useVideoFont();
  const isBottom = position === 'bottom';
  return (
    // Transparent: the background belongs to <Stage>. The old opaque fill
    // here hid every illustration drawn underneath the text.
    <AbsoluteFill
      style={{
        justifyContent: isBottom ? 'flex-end' : 'center',
        alignItems: 'center',
        padding: isBottom ? `0 ${paddingX}px 60px` : `0 ${paddingX}px`,
      }}
    >
      <div
        style={{
          fontFamily,
          fontWeight,
          fontSize,
          lineHeight: 1.4,
          color: color ?? DEFAULT_INK,
          textAlign: 'center',
          overflowWrap: 'break-word',
          maxWidth: '100%',
        }}
      >
        {children}
      </div>
    </AbsoluteFill>
  );
}

type TextProps = {children: React.ReactNode; position?: 'center' | 'bottom'; color?: string};

export function TitleText({children, position = 'center', color}: TextProps) {
  return (
    <CenteredText fontSize={64} fontWeight={700} paddingX={120} position={position} color={color}>
      {children}
    </CenteredText>
  );
}

export function BodyText({children, position = 'center', color}: TextProps) {
  return (
    <CenteredText fontSize={36} fontWeight={400} paddingX={160} position={position} color={color}>
      {children}
    </CenteredText>
  );
}
