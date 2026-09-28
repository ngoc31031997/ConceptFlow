/**
 * `remotion`, as the layout probe's page sees it (CR-048 T6a/T6b).
 *
 * The probe draws a merged script through @remotion/player's <Thumbnail>, and
 * Remotion refuses getInputProps() inside a Player ("the props are available
 * as React props"). conceptflow-mini/primitives calls it to pick the project
 * font, so every import of 'remotion' in the probe bundle (the kit, the
 * script, @remotion/lottie) is pointed here by layout_probe_lib.mjs, and only this
 * one function differs: it returns the props the probe was started with.
 */
export * from 'remotion';

export function getInputProps(): Record<string, unknown> {
  return ((window as unknown as {cfProbeInputProps?: Record<string, unknown>}).cfProbeInputProps ?? {});
}
