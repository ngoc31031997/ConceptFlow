import { useState } from "react";
import type { Scene } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./TranscriptViewer.module.css";

interface TranscriptViewerProps {
  scenes: Scene[];
  contentLanguage: "vi" | "en";
}

function ChevronIcon({ isOpen }: { isOpen: boolean }) {
  return (
    <svg
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ transform: isOpen ? "rotate(180deg)" : "rotate(0deg)", transition: "transform 0.2s" }}
    >
      <path d="M6 9l6 6 6-6" />
    </svg>
  );
}

function DownloadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 4v12m0 0l4-4m-4 4l-4-4" />
      <path d="M4 17v2a2 2 0 002 2h12a2 2 0 002-2v-2" />
    </svg>
  );
}

/**
 * Displays video transcript for accessibility.
 * 
 * Provides a text version of all narration content that:
 * - Is readable by screen readers
 * - Can be searched with Ctrl+F
 * - Can be downloaded as a plain text file
 * - Shows scene-by-scene breakdown
 * 
 * This addresses WCAG 1.2.8 (Media Alternative) for videos with burnt-in captions.
 */
export function TranscriptViewer({ scenes, contentLanguage }: TranscriptViewerProps) {
  const [isOpen, setIsOpen] = useState(false);

  // Extract all narration from scenes
  const transcriptLines = scenes
    .filter((scene) => scene.narration_text && scene.narration_text.trim().length > 0)
    .map((scene, index) => ({
      sceneIndex: index + 1,
      text: scene.narration_text,
      duration: scene.duration_seconds,
    }));

  const hasNarration = transcriptLines.length > 0;

  const handleDownload = () => {
    const header = contentLanguage === "vi" 
      ? "TRANSCRIPT - VIDEO CONCEPTFLOW\n\n"
      : "TRANSCRIPT - CONCEPTFLOW VIDEO\n\n";
    
    const content = transcriptLines
      .map((line, index) => {
        const speakerLabel = contentLanguage === "vi"
          ? `[Đoạn ${index + 1}]`
          : `[Segment ${index + 1}]`;
        return `${speakerLabel} ${line.text}`;
      })
      .join("\n\n");

    const fullText = header + content;
    const blob = new Blob([fullText], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "transcript.txt";
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  if (!hasNarration) {
    return null;
  }

  return (
    <div className={`${glass.card} ${styles.wrapper}`}>
      <button
        type="button"
        className={styles.header}
        onClick={() => setIsOpen(!isOpen)}
        aria-expanded={isOpen}
        data-testid="transcript-viewer-toggle"
      >
        <div className={styles.headerLeft}>
          <span className={styles.title}>
            {contentLanguage === "vi" ? "Phiên bản văn bản (Transcript)" : "Text Transcript"}
          </span>
          <span className={styles.hint}>
            {contentLanguage === "vi" 
              ? `${transcriptLines.length} đoạn lời thoại`
              : `${transcriptLines.length} narration segments`}
          </span>
        </div>
        <ChevronIcon isOpen={isOpen} />
      </button>

      {isOpen && (
        <div className={styles.content} data-testid="transcript-viewer-content">
          <div className={styles.toolbar}>
            <p className={styles.description}>
              {contentLanguage === "vi"
                ? "Toàn bộ nội dung lời thoại trong video, giúp người dùng screen reader và công cụ tìm kiếm truy cập dễ dàng."
                : "Full narration content for screen readers and search accessibility."}
            </p>
            <button
              type="button"
              className={glass.ghostBtn}
              onClick={handleDownload}
              data-testid="transcript-download-button"
            >
              <DownloadIcon />
              {contentLanguage === "vi" ? "Tải transcript" : "Download"}
            </button>
          </div>

          <ol className={styles.list} aria-label={contentLanguage === "vi" ? "Danh sách lời thoại" : "Narration list"}>
            {transcriptLines.map((line, index) => (
              <li key={index} className={styles.item}>
                <span className={styles.itemNumber} aria-hidden="true">
                  {index + 1}
                </span>
                <span className={styles.itemText}>{line.text}</span>
              </li>
            ))}
          </ol>
        </div>
      )}
    </div>
  );
}
