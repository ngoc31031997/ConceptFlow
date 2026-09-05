import glass from "../styles/glass.module.css";
import styles from "./VideoPlayer.module.css";

interface VideoPlayerProps {
  videoSrc: string;
}

export function VideoPlayer({ videoSrc }: VideoPlayerProps) {
  return (
    <div className={glass.card}>
      <div className={styles.frame}>
        <video data-testid="video-player-element" src={videoSrc} controls className={styles.video} />
      </div>
    </div>
  );
}
