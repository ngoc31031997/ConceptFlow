/**
 * conceptflow-mini/primitives — a deliberately tiny set of helpers so a
 * Creator can produce an actually-visible video without hand-rolling layout
 * every time. This is NOT an equivalent of the Manim side's design system
 * (conceptflow/*.py — TitleCard, Callout, CodePanel, StepList,
 * ComparisonSplit, Recap): those are explicitly out of scope for this first
 * cut of Remotion support. No theme system, no per-category styling — just
 * sane centered text with a default font/size/color, enough to render
 * something real while the fuller component library is a separate task.
 *
 * Uses "Be Vietnam Pro" — the same family the Manim side installs into the
 * image (see ../../Dockerfile's font step) — so Vietnamese diacritics render
 * correctly here too without needing a second font source.
 */
import React, {useState} from 'react';
import {AbsoluteFill, continueRender, delayRender} from 'remotion';

const FONT_NAME = 'Be Vietnam Pro';
const FONT_FAMILY = `'${FONT_NAME}', sans-serif`;
const DEFAULT_COLOR = '#F2F2F2';
const BACKGROUND = '#0B1220';

// Hold the first frame until the system-installed font is actually loaded,
// so frames aren't captured with the sans-serif fallback. Module-level so it
// runs once per bundle rather than once per component instance.
let fontReady: Promise<void> | null = null;
function useFontLoaded() {
  useState(() => {
    fontReady ??= Promise.all(
      [400, 700].map((weight) => document.fonts.load(`${weight} 36px '${FONT_NAME}'`)),
    ).then((faces) => {
      if (faces.some((f) => f.length === 0)) {
        console.warn(`conceptflow-mini: font '${FONT_NAME}' not found, falling back to sans-serif`);
      }
    });
    const handle = delayRender(`Loading font ${FONT_NAME}`);
    fontReady.then(
      () => continueRender(handle),
      () => continueRender(handle),
    );
    return null;
  });
}

function CenteredText({
  children,
  fontSize,
  fontWeight,
  paddingX,
  position,
}: {
  children: React.ReactNode;
  fontSize: number;
  fontWeight: number;
  paddingX: number;
  position: 'center' | 'bottom';
}) {
  useFontLoaded();
  const isBottom = position === 'bottom';
  return (
    <AbsoluteFill
      style={{
        // Opaque when centered: AI-written illustrations are often full-frame
        // absolute elements with no layout coordination, and stacking them
        // under transparent text made it unreadable. 'bottom' instead keeps
        // the frame free for the illustration and only backs the caption.
        backgroundColor: isBottom ? undefined : BACKGROUND,
        justifyContent: isBottom ? 'flex-end' : 'center',
        alignItems: 'center',
        padding: isBottom ? `0 ${paddingX}px 60px` : `0 ${paddingX}px`,
      }}
    >
      <div
        style={{
          fontFamily: FONT_FAMILY,
          fontWeight,
          fontSize,
          lineHeight: 1.4,
          color: DEFAULT_COLOR,
          textAlign: 'center',
          overflowWrap: 'break-word',
          maxWidth: '100%',
          ...(isBottom && {
            backgroundColor: 'rgba(11, 18, 32, 0.8)',
            padding: '16px 32px',
            borderRadius: 12,
          }),
        }}
      >
        {children}
      </div>
    </AbsoluteFill>
  );
}

type TextProps = {children: React.ReactNode; position?: 'center' | 'bottom'};

export function TitleText({children, position = 'center'}: TextProps) {
  return (
    <CenteredText fontSize={64} fontWeight={700} paddingX={120} position={position}>
      {children}
    </CenteredText>
  );
}

export function BodyText({children, position = 'center'}: TextProps) {
  return (
    <CenteredText fontSize={36} fontWeight={400} paddingX={160} position={position}>
      {children}
    </CenteredText>
  );
}
