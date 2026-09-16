import type { ButtonHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";

type Variant = "primary" | "ghost" | "danger" | "dangerGhost";

const VARIANT_CLASS: Record<Variant, string> = {
  primary: glass.btnPrimary,
  ghost: glass.ghostBtn,
  danger: glass.dangerBtn,
  dangerGhost: glass.dangerGhostBtn,
};

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
}

/**
 * The one button vocabulary in the app (glass.module.css btnPrimary/ghostBtn/
 * dangerBtn/dangerGhostBtn) — new screens should reach for this instead of a
 * bare <button> so every CTA keeps the same shape, padding and disabled
 * treatment without each page re-deriving it from the class names.
 */
export function Button({ variant = "primary", className, type = "button", ...rest }: ButtonProps) {
  const variantClass = VARIANT_CLASS[variant];
  return (
    <button type={type} className={className ? `${variantClass} ${className}` : variantClass} {...rest} />
  );
}
