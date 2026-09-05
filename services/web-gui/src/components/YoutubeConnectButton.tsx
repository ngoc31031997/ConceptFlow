import { getYoutubeAuthStartUrl } from "../api/client";

export function YoutubeConnectButton() {
  function handleClick() {
    window.location.href = getYoutubeAuthStartUrl();
  }

  return (
    <button type="button" data-testid="youtube-connect-button" onClick={handleClick}>
      Kết nối YouTube
    </button>
  );
}
