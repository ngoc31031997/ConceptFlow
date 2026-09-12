import type { Ref } from "react";
import type { Scene } from "../types";
import { TranscriptViewer } from "./TranscriptViewer";
import glass from "../styles/glass.module.css";
import styles from "./VideoPlayer.module.css";

interface VideoPlayerProps {
  videoSrc: string;
  /** CR-021 FR61.2 — để báo cáo QC tua tới mốc của một phát hiện. */
  videoRef?: Ref<HTMLVideoElement>;
  scenes?: Scene[];
  contentLanguage?: "vi" | "en";
}

export function VideoPlayer({ videoSrc, videoRef, scenes, contentLanguage = "vi" }: VideoPlayerProps) {
  return (
    <>
      <div className={glass.card}>
        <div className={styles.frame}>
          {/*
            No <track>: subtitles are burnt into the frames by Rendering (CR-002),
            so there is no separate caption file to point at. Turning them on is a
            choice made in step 2, before the render.
          */}
          {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
          <video
            ref={videoRef}
            data-testid="video-player-element"
            src={videoSrc}
            controls
            className={styles.video}
          />
        </div>
      </div>

      {scenes && scenes.length > 0 && (
        <TranscriptViewer scenes={scenes} contentLanguage={contentLanguage} />
      )}
    </>
  );
}
