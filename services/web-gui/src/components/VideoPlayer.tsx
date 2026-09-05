interface VideoPlayerProps {
  videoSrc: string;
}

export function VideoPlayer({ videoSrc }: VideoPlayerProps) {
  return <video data-testid="video-player-element" src={videoSrc} controls width="100%" />;
}
