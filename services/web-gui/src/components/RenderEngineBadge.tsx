import glass from "../styles/glass.module.css";

interface RenderEngineBadgeProps {
  renderEngine: string;
}

/**
 * Marks which engine rendered (or will render) a project's video, so the
 * video list distinguishes Manim output from Remotion output at a glance —
 * useful while Remotion's video quality is still being compared against
 * Manim's mature pipeline.
 */
export function RenderEngineBadge({ renderEngine }: RenderEngineBadgeProps) {
  const isRemotion = renderEngine === "remotion";
  return (
    <span
      className={`${glass.badge} ${isRemotion ? glass.badgeProgress : glass.badgeNeutral}`}
      data-testid="render-engine-badge"
      title={isRemotion ? "Video này render bằng Remotion" : "Video này render bằng Manim"}
    >
      <span className={glass.badgeDot} aria-hidden="true" />
      {isRemotion ? "Remotion" : "Manim"}
    </span>
  );
}
