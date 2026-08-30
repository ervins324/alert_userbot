# Neptun → Telegram air-alert forwarder

Forwards new posts from a Telegram channel (`mon1tor_ua` by default) to your
chat **only while a Kyiv city air alert is active** (state comes from the
NEPTUN WebSocket stream).

How it works:

- A **userbot** (MTProto, logged in as your real account) reads the channel in
  real time — required because a bot cannot read a channel it isn't an admin of.
- A **bot** (@BotFather) sends the message to your chat.

## Configuration

Edit `.env` (copy from `.env.example` if present):

| Variable | Required | Meaning |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | yes | Bot that sends the messages (from @BotFather) |
| `TG_API_ID` | yes | App ID from https://my.telegram.org |
| `TG_API_HASH` | yes | App hash from https://my.telegram.org |
| `TG_PHONE` | only first login | Your phone, e.g. `+380501234567` |
| `TG_PASSWORD` | no | 2FA password, if enabled |
| `DESTINATION_CHAT_ID` | yes | Chat the bot sends to (your user ID works) |
| `SOURCE_CHANNEL` | no | Channel username (`mon1tor_ua`) or `-100...` ID |
| `SESSION_FILE` | no | Session path (default `session.bin`) |
| `FORCE_ALERT` | no | `true` = treat alert as always active (testing) |
| `SKIP_PATTERNS` | no | Extra comma-separated substrings to skip |

## Local run

```sh
go build -o monitor.exe ./cmd/monitor

# first run only: will ask for the SMS login code on stdin
./monitor.exe

# or skip the code prompt with
#   set TG_AUTH_CODE=<code> in .env and delete it after first login

# diagnostics
./monitor.exe -test-notify     # send a test message via the bot
./monitor.exe -test-forward    # send the latest channel post (alert forced on)
./monitor.exe -test-map        # generate and send a test Kyiv map to chat

## Interactive Commands

- `/map` (in reply to any post): Extracts Kyiv districts, neighborhoods, or landmarks from the replied message and replies with a generated Kyiv visual map highlighting the affected area and placing a pin marker.
- `/map <location>`: Renders and sends a map for the specified Kyiv location (e.g. `/map Позняки` or `/map Оболонь`).
```

The session is saved to `session.bin` (git-ignored). Keep it safe — it grants
full access to the account.

## Docker deployment (Ubuntu server)

Prerequisites: Docker with `docker compose` plugin.

### 1. Configure

Fill in `.env` on the server (see table above).

### 2. Build

```sh
docker compose build
```

### 3. First login (one time)

The container needs to log in once, so run it interactively — it will prompt
for the SMS code on stdin and save the session to the volume:

```sh
docker compose run --rm -it monitor
```

Wait for the code prompt, enter the code, then stop it with `Ctrl+C`. The
session is now stored in the `neptun-session` volume.

> Alternatively set `TG_AUTH_CODE=<code>` in `.env`, run once, then remove it.

### 4. Run as a service

```sh
docker compose up -d
docker compose logs -f
```

The container restarts automatically (`restart: unless-stopped`). The session
is persisted in the `neptun-session` volume, so no re-login on restarts.

### Updating

```sh
git pull
docker compose build
docker compose up -d --force-recreate
```

## Security

- `.env` holds your bot token, API hash and phone — keep it out of git.
- `session.bin` is full access to your account. Never share it.
- Telegram does not allow revoking an `api_id`/`api_hash`. If leaked: terminate
  all sessions (Settings → Privacy and Security → Active Sessions), enable 2FA.
