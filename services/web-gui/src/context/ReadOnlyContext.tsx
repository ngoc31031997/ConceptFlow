import { createContext, useContext } from "react";

/**
 * True while a screen is shown view-only (see AppShell). The shared form
 * fields read it and draw their value as plain text — selectable and
 * copyable, and unmistakably not an input — instead of a greyed-out box that
 * looks broken. Anything that does not read it is still locked by AppShell's
 * disabled fieldset, and the server refuses the write regardless.
 */
export const ReadOnlyContext = createContext<boolean>(false);

export function useReadOnly(): boolean {
  return useContext(ReadOnlyContext);
}
