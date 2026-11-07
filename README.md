# Green Mountain Grills App

A web dashboard for monitoring and controlling Green Mountain Grills over their WiFi (UDP) protocol, with reliable alerts for events like low pellets or a target temperature being reached.

This is a Go + Svelte rewrite of [Aenima4six2/gmg](https://github.com/Aenima4six2/gmg). It keeps the same HTTP API, configuration variables and history database, so it's a drop-in replacement. The last JavaScript/C# version is on the `integration` branch.

## Features
- Grill controls: power, grill and food target temperatures
- Live status and a temperature history chart
- Alerts in the browser (with optional sounds) and to Slack
- Timers: stopwatch and countdown
- Auto-discovery of the grill on the local network
- Grill emulator for development and testing without a grill
- REST API and a live event stream for extensions

## Preview
![preview](./assets/preview.png)

## Running the binary

The server is a single static binary with the web UI built in. Nothing else is needed at run time.

```sh
make build                   # bin/gmg and bin/gmg-emulator (needs Go, Node and Make: `flox activate`)
cp .env.example .env         # set GMG_GRILL_HOST, or leave it unset to discover the grill
./bin/gmg                    # http://localhost:3001
```

`gmg` reads `.env` from the working directory if one is there; use `--env-file /path/to/file` to read another file (it must exist). Environment variables override values in the file. History is kept in `./data` unless `GMG_DB_PATH` says otherwise.

To build for another machine, for example a Raspberry Pi: `make web && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o gmg ./cmd/gmg`.

## Running with Docker

```sh
docker build -t gmg .
docker run -d -p 80:80 -v gmg-data:/data -e GMG_GRILL_HOST=192.168.1.100 gmg
```

The image is a single static binary on distroless (about 17 MB) and runs as a non-root user. History is kept in the `/data` volume.

To let the server **discover the grill automatically**, leave `GMG_GRILL_HOST` unset and use host networking so the broadcast reaches your LAN:

```sh
docker run -d --network host -v gmg-data:/data gmg
```

Alternatively, copy `.env.example` to `.env`, set `GMG_GRILL_HOST`, and run `./start`.

### Trying it with the emulator

```sh
docker compose up --build    # or: make up
```

This starts the app and the grill emulator. The app is at http://localhost:3000.

## Configuration

Settings are environment variables, or the same names in a `.env` file (see [`.env.example`](.env.example)); environment variables take precedence.

| Variable | Description | Default |
|----------|-------------|---------|
| `GMG_GRILL_HOST` | Grill IP address. A broadcast address (containing `255`) means discover the grill on the network | `255.255.255.255` |
| `GMG_GRILL_PORT` | Grill UDP port | `8080` |
| `GMG_GRILL_TRIES` | Resends per command before giving up | `5` |
| `GMG_GRILL_RETRYMS` | Wait for a reply before resending (ms) | `2000` |
| `GMG_STATUS_POLLING_INTERVAL` | Status poll interval (ms) | `5000` |
| `GMG_DB_PATH` | Directory for the history database (`grill_data.db`) | `data` (`/data` in Docker) |
| `GMG_DB_MAX_SIZE_MB` | Database size limit; the oldest 10% of rows are pruned when exceeded | `50` |
| `GMG_ALERTS_AUDIO_ENABLED` | Play a sound in the browser when an alert fires (`true`/`false`) | `true` |
| `GMG_ALERTS_SLACK_WEBHOOKURL` | Slack incoming-webhook URL; Slack alerts are off when unset | none |
| `GMG_ALERTS_SLACK_CHANNEL` | Slack channel | `#alerts` |
| `GMG_ALERTS_SLACK_USERNAME` | Slack username | `grill-bot` |
| `GMG_LOG_LEVEL` | `debug`, `info`, `warn` or `error` | `info` |
| `PORT` | HTTP port | `3001` (`80` in Docker) |

To change the alert sounds, replace the mp3 files in `web/public/alerts/` and rebuild.

## Alerts

| Alert | When it fires |
|-------|---------------|
| Target Grill Temperature Reached | The grill reaches its target. Fires again for a new target, or after the grill has been turned off |
| Target Food Temperature Reached | The food probe reaches its target. Same re-arming as above |
| Grill Pellet Alarm | The grill reports low pellets. Fires again each time the alarm clears and comes back |

## API

| Method | Path | |
|--------|------|-|
| `GET` | `/api/status` | Current grill status (JSON) |
| `PUT` | `/api/powertoggle`, `/api/poweron`, `/api/poweroff` | Power |
| `PUT` | `/api/temperature/grill/{°F}`, `/api/temperature/food/{°F}` | Set target temperatures |
| `GET` | `/api/temperature/history?since={unix seconds}` | Temperature history (JSON) |
| `GET` | `/api/events` | Live [Server-Sent Events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events): `status` on every poll, `alert` when one fires |

Commands return `OK`. They return `400` with a message when the grill can't accept the command (for example, setting a temperature while it's off), and `500` with a message when the grill doesn't respond.

## Development

**Prerequisites:** Go 1.26+, Node 24+, GNU Make, Docker (for images and end-to-end tests). With [flox](https://flox.dev), `flox activate` provides Go, Node and Make.

```sh
make run     # build and run bin/gmg with your .env (http://localhost:3001)
make dev     # emulator + server (:3001) + Vite dev server with hot reload at http://localhost:3000
make test    # Go tests (with the race detector) and web UI tests
make check   # gofmt, go vet, svelte-check
make e2e     # build the Docker images and run browser end-to-end tests against them
make build   # bin/gmg (UI embedded) and bin/gmg-emulator
make help    # everything else
```

### Layout

| Path | |
|------|-|
| `cmd/gmg` | Server: polls the grill, records history, sends alerts, serves the API and UI |
| `cmd/gmg-emulator` | Grill emulator (UDP) |
| `internal/protocol` | GMG wire protocol: commands and the 36-byte status format (shared by client and emulator) |
| `internal/grill` | UDP client: discovery, retries, command validation |
| `internal/poller`, `alerts`, `store`, `api`, `config` | Server components |
| `web/` | Svelte + TypeScript UI, built with Vite and embedded into the server binary |
| `e2e/` | Playwright checks against the Docker images |
| `doc/decode.txt` | Notes and captures from decoding the grill's status format |

## Tech stack
- **Server and emulator:** Go (standard library + pure-Go SQLite, no CGO)
- **UI:** Svelte 5, TypeScript, Vite, Chart.js
- **Tests:** Go testing, Vitest + Testing Library, Playwright
