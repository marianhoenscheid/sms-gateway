# SMS Gateway

A small containerized SMS gateway that receives text messages via a USB GSM modem and forwards them to Telegram.

## What it does

- Listens for incoming SMS on a USB GSM modem.
- Sends the sender and message text to a Telegram chat.
- Sends a monthly keepalive SMS so the SIM stays active.

## Running it

Use Docker Compose:

```bash
docker compose up -d
```

Set `TELEGRAM_TOKEN` and `TELEGRAM_CHAT_ID` in your environment or `.env` file.

## Building

A GitLab CI pipeline builds a small distroless container image, runs Go tests and static analysis, scans the image for vulnerabilities, and publishes the result to Docker Hub.

## Files

- `Dockerfile` — container build
- `compose.yml` — local deployment
- `cmd/` — Go replacement for the original shell scripts
- `systemd/` — systemd timer for the SIM keepalive
- `.gitlab-ci.yml` — CI/CD pipeline
