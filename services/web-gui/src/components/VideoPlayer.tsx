import glass from "../styles/glass.module.css";
import styles from "./VideoPlayer.module.css";

interface VideoPlayerProps {
  videoSrc: string;
}

export function VideoPlayer({ videoSrc }: VideoPlayerProps) {
  return (
    <div className={glass.card}>
      <div className={styles.frame}>
        {/*
          No <track>: subtitles are burnt into the frames by Rendering (CR-002),
          so there is no separate caption file to point at. Turning them on is a
          choice made in step 2, before the render.
        */}
        {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
        <video data-testid="video-player-element" src={videoSrc} controls className={styles.video} />
      </div>
    </div>
  );
}
