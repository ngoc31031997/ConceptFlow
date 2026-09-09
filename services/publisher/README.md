# Publisher Service (Unit 7)

Uploads a finished video to YouTube, and owns the OAuth connection to the
Creator's channels.

## Two tiers: OAuth apps and channels (CR-012, ADR-0026)

These are independent, and conflating them is the mistake this design exists
to avoid:

| Tier | What it is | Where it lives |
|---|---|---|
| **OAuth app** | one `client_secret*.json` = one GCP project = one quota bucket | the `secrets/` directory, scanned at startup |
| **Channel** | one consented YouTube channel | the `youtube_accounts` table |

One app connects any number of channels. What an extra app buys is **quota**:
the YouTube Data API allows 10,000 units/day per GCP project and
`videos.insert` costs 1,600, so each app is worth roughly **6 uploads/day** no
matter how many channels hang off it.

A channel row records the `client_id` that minted it. That is required, not
bookkeeping — a refresh token is only valid against the exact client_id/secret
pair that issued it, so a single global pair breaks as soon as a second app
exists.

### Each channel needs its own consent

A Google account can own many channels (its personal one plus Brand Accounts),
but a YouTube OAuth token is bound to **one** channel — the one picked on
Google's chooser — and `channels.list(mine=true)` returns only that channel.
There is no way to connect an account once and enumerate its other channels
(`onBehalfOfContentOwner` is for YouTube CMS/partner accounts only). So
connecting N channels means N consents, and `prompt="select_account consent"`
is what makes the chooser appear each time.

## Configuration

| Variable | Purpose |
|---|---|
| `GOOGLE_OAUTH_CLIENT_SECRETS_DIR` | directory of `client_secret*.json` (compose mounts `./secrets` read-only) |
| `GOOGLE_OAUTH_REDIRECT_URI` | one redirect for every app; each app must register it verbatim |
| `GOOGLE_OAUTH_CLIENT_ID` / `_SECRET` | legacy single-app fallback, used only when the directory yields nothing |
| `DATABASE_URL`, `RABBITMQ_URL`, `UPLOAD_TIMEOUT_SECONDS` | as before |

A malformed file in the secrets directory is logged and skipped rather than
crashing startup. Adding a file requires a service restart.

## REST API

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/auth/youtube/apps` | configured OAuth clients, each with a `redirect_ok` flag |
| `GET` | `/v1/auth/youtube/start?app=&state=` | 302 to Google; `400` (not a Google error page) when the redirect URI is unregistered |
| `GET` | `/v1/auth/youtube/callback?code=&state=` | exchanges the code, returns the connected channel |
| `GET` | `/v1/auth/youtube/status` | `connected` plus the full channel list |
| `GET` | `/v1/auth/youtube/accounts` | connected channels |
| `DELETE` | `/v1/auth/youtube/accounts/{channel_id}` | disconnect; promotes another channel if the default went |
| `POST` | `/v1/auth/youtube/accounts/{channel_id}/default` | choose the channel used when a publish names none |

`state` is base64url JSON carrying the client id, the project id and a
single-use nonce. The nonce is checked on callback: without it a forged
callback could attach someone else's channel to this installation.

## Publishing

The `publish_video` command payload may carry `channel_id`. Absent means the
default channel, which is how projects created before CR-012 keep working. A
`channel_id` naming an unknown channel raises `MissingCredentialError` rather
than falling back to the default — publishing to a channel the Creator did not
choose is public and cannot be undone.

## Migration

The pre-CR-012 `oauth_credentials` table (a single row pinned by
`CHECK (id = 1)`) is migrated into `youtube_accounts` at startup, attributed to
`GOOGLE_OAUTH_CLIENT_ID`. The old table is left in place rather than dropped.
If that client's secret file is not in `secrets/`, the migrated channel cannot
be refreshed and the Creator is told to reconnect it.

## Tests

    .venv/bin/python -m pytest tests -q
