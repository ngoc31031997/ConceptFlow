import { forwardRef, type ButtonHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";

/**
 * How far the action reaches, from the highest level down
 * (docs/ux-ui-design-rules.md, rule 6): `danger` deletes for good, `impact`
 * sends approved or already-run later work back, `run` starts an AI or long
 * job, `primary` moves forward or saves, `view` only opens information, `ghost`
 * is neutral (cancel, back, copy). `dangerGhost` is the outlined delete used
 * beside other controls.
 */
export type ButtonVariant = "primary" | "ghost" | "danger" | "dangerGhost" | "impact" | "run" | "view";

const VARIANT_CLASS: Record<ButtonVariant, string> = {
  primary: glass.btnPrimary,
  ghost: glass.ghostBtn,
  danger: glass.dangerBtn,
  dangerGhost: glass.dangerGhostBtn,
  impact: glass.impactBtn,
  run: glass.runBtn,
  view: glass.viewBtn,
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
}

/**
 * The one button vocabulary in the app (glass.module.css) — new screens should
 * reach for this instead of a bare <button> so every action keeps the same
 * shape, padding and disabled treatment, and its fill says how far it reaches.
 * The ref reaches the <button>, for a dialog that focuses its main action.
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = "primary", className, type = "button", ...rest },
  ref,
) {
  const variantClass = VARIANT_CLASS[variant];
  return (
    <button
      ref={ref}
      type={type}
      className={className ? `${variantClass} ${className}` : variantClass}
      {...rest}
    />
  );
});
