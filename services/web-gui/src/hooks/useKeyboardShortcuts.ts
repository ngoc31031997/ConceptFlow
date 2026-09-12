import { useEffect } from "react";

export interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  action: () => void;
  description: string;
  preventDefault?: boolean;
}

/**
 * Register global keyboard shortcuts.
 * 
 * @param shortcuts - Array of keyboard shortcuts to register
 * @param enabled - Whether shortcuts are enabled (default: true)
 * 
 * @example
 * useKeyboardShortcuts([
 *   { key: 'Enter', ctrlKey: true, action: handleSubmit, description: 'Submit form' },
 *   { key: 'Escape', action: handleClose, description: 'Close modal' },
 * ]);
 */
export function useKeyboardShortcuts(shortcuts: KeyboardShortcut[], enabled: boolean = true) {
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      for (const shortcut of shortcuts) {
        const ctrlOrMeta = shortcut.ctrlKey || shortcut.metaKey;
        const matchesCtrl = ctrlOrMeta ? (event.ctrlKey || event.metaKey) : true;
        const matchesShift = shortcut.shiftKey ? event.shiftKey : !event.shiftKey;
        const matchesAlt = shortcut.altKey ? event.altKey : !event.altKey;
        const matchesKey = event.key.toLowerCase() === shortcut.key.toLowerCase();

        if (matchesKey && matchesCtrl && matchesShift && matchesAlt) {
          // Don't trigger shortcuts when typing in input fields
          const target = event.target as HTMLElement;
          const isInput = target.tagName === "INPUT" || 
                         target.tagName === "TEXTAREA" || 
                         target.isContentEditable;
          
          // Allow Escape to work even in input fields
          if (isInput && shortcut.key.toLowerCase() !== "escape") {
            continue;
          }

          if (shortcut.preventDefault !== false) {
            event.preventDefault();
          }
          
          shortcut.action();
          break; // Only trigger first matching shortcut
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [shortcuts, enabled]);
}

/**
 * Format keyboard shortcut for display in tooltips.
 * 
 * @example
 * formatShortcut({ key: 'Enter', ctrlKey: true }) // "Ctrl+Enter" on Windows/Linux, "⌘+Enter" on Mac
 */
export function formatShortcut(shortcut: Pick<KeyboardShortcut, 'key' | 'ctrlKey' | 'metaKey' | 'shiftKey' | 'altKey'>): string {
  const isMac = navigator.platform.toUpperCase().indexOf('MAC') >= 0;
  const parts: string[] = [];

  if (shortcut.ctrlKey || shortcut.metaKey) {
    parts.push(isMac ? '⌘' : 'Ctrl');
  }
  if (shortcut.shiftKey) {
    parts.push(isMac ? '⇧' : 'Shift');
  }
  if (shortcut.altKey) {
    parts.push(isMac ? '⌥' : 'Alt');
  }

  // Format key name
  const keyName = shortcut.key === ' ' ? 'Space' : 
                  shortcut.key.length === 1 ? shortcut.key.toUpperCase() :
                  shortcut.key.charAt(0).toUpperCase() + shortcut.key.slice(1);
  
  parts.push(keyName);

  return parts.join(isMac ? '' : '+');
}
