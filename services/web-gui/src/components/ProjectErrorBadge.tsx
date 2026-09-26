import { useCallback, useEffect, useState } from "react";
import { listProjectErrors, type ProjectErrorEntry } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./ProjectErrorBadge.module.css";

const POLL_MS = 10_000;

/** Plain text of one entry — what gets pasted into a bug report or a chat. */
export function formatProjectError(e: ProjectErrorEntry): string {
  const head = [e.at, e.source, e.step, e.kind, e.provider].filter(Boolean).join(" | ");
  const lines = [head, e.message];
  if (e.usage) {
    lines.push(
      `usage: model=${e.usage.model ?? "?"} prompt=${e.usage.prompt_tokens} completion=${e.usage.completion_tokens} reasoning=${e.usage.reasoning_tokens}`,
    );
  }
  if (e.partial_chars) lines.push(`partial_chars: ${e.partial_chars}`);
  if (e.elapsed_seconds) lines.push(`elapsed: ${e.elapsed_seconds}s`);
  if (e.detail) lines.push(e.detail);
  return lines.join("\n");
}

async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/**
 * Shows nothing while the project has no recorded failures; otherwise a red
 * badge with the count. Clicking lists them, newest first, with a Copy button
 * per entry and one for all — the trace exists to be pasted somewhere.
 */
export function ProjectErrorBadge({ projectId }: { projectId: string }) {
  const [errors, setErrors] = useState<ProjectErrorEntry[]>([]);
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);

  const load = useCallback(() => {
    listProjectErrors(projectId)
      .then((rows) => setErrors(Array.isArray(rows) ? rows : []))
      .catch(() => {});
  }, [projectId]);

  useEffect(() => {
    setErrors([]);
    load();
    const timer = window.setInterval(load, POLL_MS);
    return () => window.clearInterval(timer);
  }, [load]);

  if (errors.length === 0) return null;

  const newestFirst = [...errors].reverse();
  const copy = async (key: string, text: string) => {
    if (await copyText(text)) {
      setCopied(key);
      window.setTimeout(() => setCopied(null), 1500);
    }
  };

  return (
    <div className={styles.wrap}>
      <button
        type="button"
        className={`${glass.badge} ${glass.badgeFailed} ${styles.badge}`}
        onClick={() => {
          setOpen((v) => !v);
          load();
        }}
        aria-expanded={open}
        data-testid="project-error-badge"
      >
        <span className={glass.badgeDot} aria-hidden="true" />
        {errors.length} lỗi
      </button>
      {open && (
        <div className={styles.panel} role="dialog" aria-label="Nhật ký lỗi của project">
          <div className={styles.panelHead}>
            <strong>Nhật ký lỗi</strong>
            <button
              type="button"
              className={styles.copy}
              onClick={() => copy("all", newestFirst.map(formatProjectError).join("\n\n----\n\n"))}
            >
              {copied === "all" ? "Đã sao chép" : "Sao chép tất cả"}
            </button>
          </div>
          <ul className={styles.list}>
            {newestFirst.map((e, i) => {
              const key = `${e.at}-${i}`;
              return (
                <li key={key} className={styles.item}>
                  <div className={styles.meta}>
                    {new Date(e.at).toLocaleString()} · {[e.source, e.step, e.kind].filter(Boolean).join(" · ")}
                    <button type="button" className={styles.copy} onClick={() => copy(key, formatProjectError(e))}>
                      {copied === key ? "Đã sao chép" : "Sao chÃ©p"}
                    </button>
                  </div>
                  <div className={styles.message}>{e.message}</div>
                  {e.detail && <pre className={styles.detail}>{e.detail}</pre>}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
