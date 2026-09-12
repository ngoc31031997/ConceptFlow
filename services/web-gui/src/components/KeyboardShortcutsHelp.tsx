import { useState, useEffect } from "react";
import { formatShortcut } from "../hooks/useKeyboardShortcuts";
import glass from "../styles/glass.module.css";
import styles from "./KeyboardShortcutsHelp.module.css";

function KeyboardIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="2" y="4" width="20" height="16" rx="2" />
      <path d="M6 8h.01M10 8h.01M14 8h.01M18 8h.01M8 12h.01M12 12h.01M16 12h.01M7 16h10" />
    </svg>
  );
}

function CloseIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M18 6L6 18M6 6l12 12" />
    </svg>
  );
}

const SHORTCUTS = [
  {
    keys: { key: 'Enter', ctrlKey: true },
    description: 'Tiếp tục / Submit form',
  },
  {
    keys: { key: 'Escape' },
    description: 'Đóng modal / Dialog',
  },
  {
    keys: { key: '?' },
    description: 'Hiển thị / Ẩn keyboard shortcuts',
  },
];

/**
 * Keyboard shortcuts help overlay.
 * Shows all available keyboard shortcuts in the app.
 * Triggered by pressing "?" key.
 */
export function KeyboardShortcutsHelp() {
  const [isOpen, setIsOpen] = useState(false);

  // Listen for ? key to toggle help
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
        // Don't trigger when typing in input fields
        const target = e.target as HTMLElement;
        const isInput = target.tagName === "INPUT" || 
                       target.tagName === "TEXTAREA" || 
                       target.isContentEditable;
        
        if (!isInput) {
          e.preventDefault();
          setIsOpen(prev => !prev);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  if (!isOpen) {
    return (
      <button
        type="button"
        className={styles.trigger}
        onClick={() => setIsOpen(true)}
        title="Keyboard shortcuts (?)"
        data-testid="keyboard-shortcuts-trigger"
      >
        <KeyboardIcon />
      </button>
    );
  }

  return (
    <div className={styles.overlay} onClick={() => setIsOpen(false)} data-testid="keyboard-shortcuts-overlay">
      <div className={`${glass.card} ${styles.modal}`} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <div className={styles.headerLeft}>
            <KeyboardIcon />
            <h2 className={styles.title}>Keyboard Shortcuts</h2>
          </div>
          <button
            type="button"
            className={styles.closeBtn}
            onClick={() => setIsOpen(false)}
            aria-label="Đóng"
          >
            <CloseIcon />
          </button>
        </div>

        <div className={styles.list}>
          {SHORTCUTS.map((shortcut, index) => (
            <div key={index} className={styles.item}>
              <kbd className={styles.kbd}>{formatShortcut(shortcut.keys)}</kbd>
              <span className={styles.description}>{shortcut.description}</span>
            </div>
          ))}
        </div>

        <p className={styles.footer}>
          Nhấn <kbd className={styles.inlineKbd}>?</kbd> hoặc <kbd className={styles.inlineKbd}>Esc</kbd> để đóng
        </p>
      </div>
    </div>
  );
}
