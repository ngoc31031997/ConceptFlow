import { useCallback, useEffect, useState } from "react";
import {
  disconnectYoutubeAccount,
  getYoutubeAuthStartUrl,
  listYoutubeAccounts,
  listYoutubeApps,
  makeYoutubeAccountDefault,
} from "../api/client";
import type { YoutubeAccount, YoutubeApp } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./YoutubeChannels.module.css";

function YoutubeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="6" width="18" height="12" rx="3" />
      <path d="M11 10l4 2-4 2z" fill="currentColor" stroke="none" />
    </svg>
  );
}

interface Props {
  projectId: string;
  /** Reports which connected channel the video should publish to. */
  onSelectedChannelChange: (channelId: string | null) => void;
}

/**
 * Replaces the old connected/not-connected button (CR-012 FR34).
 *
 * A boolean was the right shape while exactly one channel could exist. Now
 * that several can, the Creator needs to see *which* channels are connected
 * and pick one — publishing to the wrong channel is public and cannot be
 * undone.
 */
export function YoutubeChannels({ projectId, onSelectedChannelChange }: Props) {
  const [accounts, setAccounts] = useState<YoutubeAccount[] | null>(null);
  const [apps, setApps] = useState<YoutubeApp[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [choosingApp, setChoosingApp] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const [rawAccounts, rawApps] = await Promise.all([listYoutubeAccounts(), listYoutubeApps()]);
    // Normalised rather than trusted: this panel sits on the result page, and
    // an unexpected response shape from the gateway must not blank out the
    // video the Creator just made.
    const loadedAccounts = Array.isArray(rawAccounts) ? rawAccounts : [];
    setAccounts(loadedAccounts);
    setApps(Array.isArray(rawApps) ? rawApps : []);
    setSelected((current) => {
      // Keep the Creator's pick across refreshes; otherwise fall to the
      // default channel, which is what the backend would choose anyway.
      if (current && loadedAccounts.some((a) => a.channel_id === current)) return current;
      return loadedAccounts.find((a) => a.is_default)?.channel_id ?? loadedAccounts[0]?.channel_id ?? null;
    });
  }, []);

  useEffect(() => {
    let cancelled = false;
    load().catch((err) => {
      if (!cancelled) setError(err instanceof Error ? err.message : String(err));
    });
    return () => {
      cancelled = true;
    };
  }, [load]);

  useEffect(() => {
    onSelectedChannelChange(selected);
  }, [selected, onSelectedChannelChange]);

  function connect(clientId?: string) {
    window.location.href = getYoutubeAuthStartUrl(projectId, clientId);
  }

  function handleAddChannel() {
    // One app needs no picker; several do, since each is a separate quota
    // bucket and the Creator may care which one a channel lands in.
    if (apps.length <= 1) connect(apps[0]?.client_id);
    else setChoosingApp(true);
  }

  async function handleDisconnect(account: YoutubeAccount) {
    const name = account.channel_title || account.channel_id;
    if (!window.confirm(`Ngắt kết nối kênh "${name}"? Bạn sẽ phải cấp quyền lại nếu muốn đăng lên kênh này.`)) {
      return;
    }
    setError(null);
    try {
      await disconnectYoutubeAccount(account.channel_id);
      await load();
      // Clear error on successful action
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleMakeDefault(account: YoutubeAccount) {
    setError(null);
    try {
      await makeYoutubeAccountDefault(account.channel_id);
      await load();
      // Clear error on successful action
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function handleAddChannelClick() {
    // Clear error when user tries to add channel
    setError(null);
    handleAddChannel();
  }

  const unusableApps = apps.filter((app) => !app.redirect_ok);

  return (
    <div className={glass.card} style={{ padding: 0 }}>
      <div className={styles.header}>
        <div className={styles.left}>
          <div className={styles.icon}>
            <YoutubeIcon />
          </div>
          <div className={styles.title}>Kênh YouTube</div>
        </div>
        <button type="button" data-testid="youtube-add-channel" className={styles.connectBtn} onClick={handleAddChannelClick}>
          Thêm kênh
        </button>
      </div>

      {choosingApp && (
        <div className={styles.appPicker} data-testid="youtube-app-picker">
          <p className={styles.appPickerHint}>
            Chọn OAuth client để nối kênh. Mỗi client là một GCP project với hạn mức riêng
            (khoảng 6 video/ngày) — nối thêm kênh vào cùng một client không làm tăng hạn mức đó.
          </p>
          {apps.map((app) => (
            <button
              key={app.client_id}
              type="button"
              className={styles.appOption}
              disabled={!app.redirect_ok}
              onClick={() => connect(app.client_id)}
            >
              <span className={styles.appLabel}>{app.label}</span>
              <span className={styles.appFile}>{app.source_file}</span>
            </button>
          ))}
          <button type="button" className={styles.linkBtn} onClick={() => setChoosingApp(false)}>
            Huỷ
          </button>
        </div>
      )}

      {unusableApps.length > 0 && (
        // Surfaced here rather than after the Creator has been bounced to
        // Google's redirect_uri_mismatch page, which names neither the client
        // nor the URI to add (CR-012 FR30.3).
        <div role="alert" className={styles.warning} data-testid="youtube-redirect-warning">
          {unusableApps.map((app) => (
            <p key={app.client_id}>
              OAuth client <strong>{app.label}</strong> ({app.source_file}) chưa khai báo redirect
              URI. Vào Google Cloud Console → Credentials → client này → Authorized redirect URIs,
              thêm đúng chuỗi <code>{app.redirect_uri_hint}</code>, rồi tải lại file client secret.
            </p>
          ))}
        </div>
      )}

      {error && (
        <p role="alert" className={glass.helperText} style={{ padding: "0 24px 16px" }}>
          {error}
        </p>
      )}

      {accounts !== null && accounts.length === 0 && (
        <p className={styles.empty} data-testid="youtube-no-channels">
          Chưa nối kênh nào. Bấm “Thêm kênh” — bạn sẽ chọn tài khoản Google và kênh ngay trên màn
          hình của Google. Mỗi kênh cần cấp quyền một lần riêng, kể cả khi chúng thuộc cùng một tài
          khoản Google.
        </p>
      )}

      {accounts !== null && accounts.length > 0 && (
        <ul className={styles.list} data-testid="youtube-channel-list">
          {accounts.map((account) => (
            <li key={account.channel_id} className={styles.item}>
              <label className={styles.itemMain}>
                <input
                  type="radio"
                  name="youtube-channel"
                  value={account.channel_id}
                  checked={selected === account.channel_id}
                  onChange={() => setSelected(account.channel_id)}
                />
                <span className={styles.channelName}>
                  {account.channel_title || account.channel_id}
                </span>
                {account.is_default && <span className={styles.defaultBadge}>mặc định</span>}
                {!account.has_caption_scope && (
                  // CR-015 FR40.2: learned here, before publish, rather than
                  // from a video that quietly has no CC afterward.
                  <span
                    className={styles.captionBadge}
                    title="Kênh này được nối trước khi tính năng phụ đề YouTube ra mắt — video vẫn đăng bình thường, chỉ không có phụ đề. Ngắt kết nối rồi nối lại để bật."
                  >
                    thiếu quyền phụ đề
                  </span>
                )}
                <span className={styles.appTag}>{account.app_label}</span>
              </label>
              <div className={styles.itemActions}>
                {!account.is_default && (
                  <button type="button" className={styles.linkBtn} onClick={() => handleMakeDefault(account)}>
                    Đặt mặc định
                  </button>
                )}
                <button type="button" className={styles.dangerLink} onClick={() => handleDisconnect(account)}>
                  Ngắt
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
