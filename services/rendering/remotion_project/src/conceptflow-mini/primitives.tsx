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
import React from 'react';
import {AbsoluteFill} from 'remotion';

const FONT_FAMILY = "'Be Vietnam Pro', sans-serif";
const DEFAULT_COLOR = '#F2F2F2';
const BACKGROUND = '#0B1220';

export function TitleText({children}: {children: React.ReactNode}) {
  return (
    <AbsoluteFill
      style={{
        backgroundColor: BACKGROUND,
        justifyContent: 'center',
        alignItems: 'center',
        padding: '0 120px',
      }}
    >
      <div
        style={{
          fontFamily: FONT_FAMILY,
          fontWeight: 700,
          fontSize: 64,
          color: DEFAULT_COLOR,
          textAlign: 'center',
        }}
      >
        {children}
      </div>
    </AbsoluteFill>
  );
}

export function BodyText({children}: {children: React.ReactNode}) {
  return (
    <AbsoluteFill
      style={{
        justifyContent: 'center',
        alignItems: 'center',
        padding: '0 160px',
      }}
    >
      <div
        style={{
          fontFamily: FONT_FAMILY,
          fontSize: 36,
          color: DEFAULT_COLOR,
          textAlign: 'center',
        }}
      >
        {children}
      </div>
    </AbsoluteFill>
  );
}
